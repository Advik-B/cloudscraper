package cloudscraper

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Advik-B/cloudscraper/lib/captcha"
	"github.com/Advik-B/cloudscraper/lib/errors"
	"github.com/Advik-B/cloudscraper/lib/js"
	"github.com/Advik-B/cloudscraper/lib/proxy"
	"github.com/Advik-B/cloudscraper/lib/stealth"
	"github.com/Advik-B/cloudscraper/lib/transport"
	"github.com/Advik-B/cloudscraper/lib/user_agent"

	"github.com/andybalholm/brotli"
	"golang.org/x/net/publicsuffix"
)

// Scraper is the main struct for making requests.
type Scraper struct {
	client *http.Client
	opts   Options
	logger *log.Logger

	UserAgent     *useragent.Agent
	CaptchaSolver captcha.Solver
	ProxyManager  *proxy.Manager
	StealthMode   *stealth.Mode
	jsEngine      js.Engine

	mu               sync.Mutex
	sessionStartTime time.Time
	requestCount     int32
	last403Time      time.Time
	in403Retry       bool
}

// New creates a new Scraper instance with the given options.
func New(opts ...ScraperOption) (*Scraper, error) {
	options := Options{
		MaxRetries:             3,
		AutoRefreshOn403:       true,
		SessionRefreshInterval: 1 * time.Hour,
		Max403Retries:          3,
		RotateTlsCiphers:       true,
		Stealth: stealth.Options{
			Enabled:          true,
			HumanLikeDelays:  true,
			RandomizeHeaders: true,
			BrowserQuirks:    true,
		},
		JSRuntime: js.Goja, // Default to the built-in Goja engine
	}

	for _, opt := range opts {
		opt(&options)
	}

	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	agent, err := useragent.New(options.Browser)
	if err != nil {
		return nil, fmt.Errorf("failed to create user agent: %w", err)
	}

	tr := transport.NewTransport()
	tr.SetCipherSuites(agent.CipherSuites)

	var pm *proxy.Manager
	if len(options.Proxies) > 0 {
		pm, err = proxy.NewManager(options.Proxies, options.ProxyOptions.Strategy, options.ProxyOptions.BanTime)
		if err != nil {
			return nil, err
		}
	}

	var logger *log.Logger
	if options.Logger != nil {
		logger = options.Logger
	} else {
		// Default to a silent logger if none is provided.
		logger = log.New(io.Discard, "", 0)
	}

	var jsEngine js.Engine
	
	// Check if a custom engine was provided
	if options.CustomJSEngine != nil {
		jsEngine = options.CustomJSEngine
	} else {
		// Use the configured runtime or default to Goja
		switch options.JSRuntime {
		case js.Node, js.Deno, js.Bun:
			jsEngine, err = js.NewExternalEngine(string(options.JSRuntime))
			if err != nil {
				return nil, fmt.Errorf("failed to initialize JS runtime: %w", err)
			}
		case js.Goja, "": // Default to Goja
			jsEngine = js.NewGojaEngine()
		default:
			return nil, fmt.Errorf("unsupported JS runtime: %s", options.JSRuntime)
		}
	}

	s := &Scraper{
		client: &http.Client{
			Jar:       jar,
			Transport: tr,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Timeout: 30 * time.Second, // Add a default timeout
		},
		opts:             options,
		UserAgent:        agent,
		CaptchaSolver:    options.CaptchaSolver,
		ProxyManager:     pm,
		StealthMode:      stealth.New(options.Stealth),
		jsEngine:         jsEngine,
		logger:           logger,
		sessionStartTime: time.Now(),
	}

	return s, nil
}

// Get performs a GET request.
func (s *Scraper) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return s.do(req)
}

// Post performs a POST request.
func (s *Scraper) Post(url, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return s.do(req)
}

func (s *Scraper) do(req *http.Request) (*http.Response, error) {
	s.mu.Lock()
	if s.shouldRefreshSession() {
		if err := s.refreshSession(req.URL); err != nil {
			s.logger.Printf("Warning: session refresh failed: %v\n", err)
		}
	}
	s.mu.Unlock()

	for key, values := range s.UserAgent.Headers {
		if req.Header.Get(key) == "" {
			req.Header[key] = values
		}
	}

	s.StealthMode.Apply(req, s.UserAgent.Browser)

	var currentProxy *url.URL
	var err error
	if s.ProxyManager != nil {
		currentProxy, err = s.ProxyManager.GetProxy()
		if err != nil {
			return nil, err
		}
		if tr, ok := s.client.Transport.(*transport.CipherSuiteTransport); ok {
			tr.Transport.Proxy = http.ProxyURL(currentProxy)
		}
	}

	atomic.AddInt32(&s.requestCount, 1)

	resp, err := s.client.Do(req)
	if err != nil {
		if currentProxy != nil {
			s.ProxyManager.ReportFailure(currentProxy)
		}
		return nil, err
	}

	if currentProxy != nil {
		s.ProxyManager.ReportSuccess(currentProxy)
	}

	switch resp.Header.Get("Content-Encoding") {
	case "br":
		resp.Body = io.NopCloser(brotli.NewReader(resp.Body))
		resp.Header.Del("Content-Encoding")
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	resp.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	if isChallengeResponse(resp, bodyBytes) {
		s.logger.Println("Cloudflare protection detected, attempting to bypass...")
		return s.handleChallenge(resp)
	}

	if resp.StatusCode == http.StatusForbidden && s.opts.AutoRefreshOn403 {
		return s.handle403(req)
	}

	if resp.StatusCode >= 300 && resp.StatusCode <= 399 {
		loc, err := resp.Location()
		if err != nil {
			return resp, nil
		}
		redirectReq, _ := http.NewRequest("GET", loc.String(), nil)
		return s.do(redirectReq)
	}

	return resp, nil
}

func (s *Scraper) handle403(req *http.Request) (*http.Response, error) {
	s.mu.Lock()
	if s.in403Retry {
		s.mu.Unlock()
		return nil, errors.ErrMaxRetriesExceeded
	}
	s.in403Retry = true
	s.last403Time = time.Now()
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.in403Retry = false
		s.mu.Unlock()
	}()

	for i := 0; i < s.opts.Max403Retries; i++ {
		s.logger.Printf("Received 403. Refreshing session (attempt %d/%d)...\n", i+1, s.opts.Max403Retries)
		if err := s.refreshSession(req.URL); err != nil {
			return nil, fmt.Errorf("failed to refresh session after 403: %w", err)
		}

		resp, err := s.do(req)
		if err == nil && resp.StatusCode != http.StatusForbidden {
			return resp, nil
		}
	}

	return nil, errors.ErrMaxRetriesExceeded
}

func (s *Scraper) shouldRefreshSession() bool {
	return time.Since(s.sessionStartTime) > s.opts.SessionRefreshInterval
}

func (s *Scraper) refreshSession(currentURL *url.URL) error {
	s.logger.Println("Refreshing session...")
	s.sessionStartTime = time.Now()
	atomic.StoreInt32(&s.requestCount, 0)

	agent, err := useragent.New(s.opts.Browser)
	if err != nil {
		return err
	}
	s.UserAgent = agent

	if s.opts.RotateTlsCiphers {
		if tr, ok := s.client.Transport.(*transport.CipherSuiteTransport); ok {
			tr.SetCipherSuites(s.UserAgent.CipherSuites)
		}
	}

	if s.client.Jar != nil {
		s.client.Jar.SetCookies(currentURL, []*http.Cookie{})
	}

	rootURL := &url.URL{Scheme: currentURL.Scheme, Host: currentURL.Host}
	_, err = s.Get(rootURL.String())
	return err
}
