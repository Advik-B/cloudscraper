package cloudscraper

import (
	useragent "github.com/Advik-B/cloudscraper/lib/user_agent"
	"log"
	"time"

	"github.com/Advik-B/cloudscraper/lib/captcha"
	"github.com/Advik-B/cloudscraper/lib/js"
	"github.com/Advik-B/cloudscraper/lib/proxy"
	"github.com/Advik-B/cloudscraper/lib/stealth"
)

// Options holds all configuration for the scraper.
type Options struct {
	MaxRetries             int
	Delay                  time.Duration
	AutoRefreshOn403       bool
	SessionRefreshInterval time.Duration
	Max403Retries          int
	Browser                useragent.Config
	RotateTlsCiphers       bool
	CaptchaSolver          captcha.Solver
	Proxies                []string
	ProxyOptions           struct {
		Strategy proxy.Strategy
		BanTime  time.Duration
	}
	Stealth       stealth.Options
	JSRuntime     js.Runtime // "goja", "node", "deno", "bun"
	CustomJSEngine js.Engine  // Custom JS engine implementation (overrides JSRuntime if set)
	Logger        *log.Logger
}

// ScraperOption configures a Scraper.
type ScraperOption func(*Options)

// WithBrowser configures the browser profile to use.
func WithBrowser(cfg useragent.Config) ScraperOption {
	return func(o *Options) {
		o.Browser = cfg
	}
}

// WithCaptchaSolver configures a captcha solver.
func WithCaptchaSolver(solver captcha.Solver) ScraperOption {
	return func(o *Options) {
		o.CaptchaSolver = solver
	}
}

// WithProxies configures the proxy manager.
func WithProxies(proxyURLs []string, strategy proxy.Strategy, banTime time.Duration) ScraperOption {
	return func(o *Options) {
		o.Proxies = proxyURLs
		o.ProxyOptions.Strategy = strategy
		o.ProxyOptions.BanTime = banTime
	}
}

// WithStealth configures the stealth mode options.
func WithStealth(opts stealth.Options) ScraperOption {
	return func(o *Options) {
		o.Stealth = opts
	}
}

// WithSessionConfig configures session handling.
func WithSessionConfig(refreshOn403 bool, interval time.Duration, maxRetries int) ScraperOption {
	return func(o *Options) {
		o.AutoRefreshOn403 = refreshOn403
		o.SessionRefreshInterval = interval
		o.Max403Retries = maxRetries
	}
}

// WithDelay sets a fixed delay between requests (used by StealthMode if HumanLikeDelays is false).
func WithDelay(d time.Duration) ScraperOption {
	return func(o *Options) {
		o.Delay = d
	}
}

// WithJSRuntime sets the JavaScript runtime to use for solving challenges.
// Supported values are js.Goja (default, recommended), js.Node, js.Deno, js.Bun.
// The selected runtime must be available in the system's PATH for external runtimes.
func WithJSRuntime(runtime js.Runtime) ScraperOption {
	return func(o *Options) {
		o.JSRuntime = runtime
	}
}

// WithCustomJSEngine sets a custom JavaScript engine implementation.
// This overrides the JSRuntime setting and allows you to provide your own engine.
// The engine must implement the js.Engine interface.
func WithCustomJSEngine(engine js.Engine) ScraperOption {
	return func(o *Options) {
		o.CustomJSEngine = engine
	}
}

// WithLogger sets a logger for the scraper to use for debug output.
// By default, logging is disabled.
func WithLogger(logger *log.Logger) ScraperOption {
	return func(o *Options) {
		o.Logger = logger
	}
}
