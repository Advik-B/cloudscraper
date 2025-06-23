package cloudscraper

import (
	"fmt"
	"github.com/Advik-B/cloudscraper/errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	jsV1DetectRegex    = regexp.MustCompile(`(?i)cdn-cgi/images/trace/jsch/`)
	jsV2DetectRegex    = regexp.MustCompile(`(?i)/cdn-cgi/challenge-platform/`)
	captchaDetectRegex = regexp.MustCompile(`data-sitekey="([^\"]+)"`)
	challengeFormRegex = regexp.MustCompile(`<form class="challenge-form" id="challenge-form" action="(.+?)" method="POST">`)
	jschlVcRegex       = regexp.MustCompile(`name="jschl_vc" value="(\w+)"`)
	passRegex          = regexp.MustCompile(`name="pass" value="(.+?)"`)
)

func (s *Scraper) handleChallenge(resp *http.Response) (*http.Response, error) {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read challenge response body: %w", err)
	}
	bodyStr := string(body)

	// Check for modern v2/v3 JS VM challenge first
	if jsV2DetectRegex.MatchString(bodyStr) {
		fmt.Println("Modern (v2/v3) JavaScript challenge detected. Solving with Otto...")
		return s.solveModernJSChallenge(resp, bodyStr)
	}

	// Check for classic v1 JS challenge
	if jsV1DetectRegex.MatchString(bodyStr) {
		fmt.Println("Classic (v1) JavaScript challenge detected. Solving with Otto...")
		return s.solveClassicJSChallenge(resp.Request.URL, bodyStr)
	}

	// Check for Captcha/Turnstile
	if siteKeyMatch := captchaDetectRegex.FindStringSubmatch(bodyStr); len(siteKeyMatch) > 1 {
		fmt.Println("Captcha/Turnstile challenge detected...")
		return s.solveCaptchaChallenge(resp, bodyStr, siteKeyMatch[1])
	}

	return nil, errors.ErrUnknownChallenge
}

func (s *Scraper) solveClassicJSChallenge(originalURL *url.URL, body string) (*http.Response, error) {
	time.Sleep(4 * time.Second)

	answer, err := solveV1Challenge(body, originalURL.Host)
	if err != nil {
		return nil, fmt.Errorf("v1 challenge solver failed: %w", err)
	}

	formMatch := challengeFormRegex.FindStringSubmatch(body)
	if len(formMatch) < 2 {
		return nil, fmt.Errorf("v1: could not find challenge form")
	}
	vcMatch := jschlVcRegex.FindStringSubmatch(body)
	if len(vcMatch) < 2 {
		return nil, fmt.Errorf("v1: could not find jschl_vc")
	}
	passMatch := passRegex.FindStringSubmatch(body)
	if len(passMatch) < 2 {
		return nil, fmt.Errorf("v1: could not find pass")
	}

	fullSubmitURL, _ := originalURL.Parse(formMatch[1])
	formData := url.Values{
		"r":            {s.extractRValue(body)},
		"jschl_vc":     {vcMatch[1]},
		"pass":         {passMatch[1]},
		"jschl_answer": {answer},
	}

	return s.submitChallengeForm(fullSubmitURL.String(), originalURL.String(), formData)
}

func (s *Scraper) solveModernJSChallenge(resp *http.Response, body string) (*http.Response, error) {
	answer, err := solveModernChallenge(body, resp.Request.URL.Host)
	if err != nil {
		return nil, fmt.Errorf("modern challenge solver failed: %w", err)
	}

	formMatch := challengeFormRegex.FindStringSubmatch(body)
	if len(formMatch) < 2 {
		return nil, fmt.Errorf("v2: could not find challenge form")
	}
	vcMatch := jschlVcRegex.FindStringSubmatch(body)
	if len(vcMatch) < 2 {
		return nil, fmt.Errorf("v2: could not find jschl_vc")
	}
	passMatch := passRegex.FindStringSubmatch(body)
	if len(passMatch) < 2 {
		return nil, fmt.Errorf("v2: could not find pass")
	}

	fullSubmitURL, _ := resp.Request.URL.Parse(formMatch[1])
	formData := url.Values{
		"r":            {s.extractRValue(body)},
		"jschl_vc":     {vcMatch[1]},
		"pass":         {passMatch[1]},
		"jschl_answer": {answer},
	}

	return s.submitChallengeForm(fullSubmitURL.String(), resp.Request.URL.String(), formData)
}

func (s *Scraper) solveCaptchaChallenge(resp *http.Response, body, siteKey string) (*http.Response, error) {
	if s.CaptchaSolver == nil {
		return nil, errors.ErrNoCaptchaSolver
	}

	token, err := s.CaptchaSolver.Solve("turnstile", resp.Request.URL.String(), siteKey)
	if err != nil {
		return nil, fmt.Errorf("captcha solver failed: %w", err)
	}

	formMatch := challengeFormRegex.FindStringSubmatch(body)
	if len(formMatch) < 2 {
		return nil, fmt.Errorf("captcha: could not find challenge form")
	}
	submitURL, _ := resp.Request.URL.Parse(formMatch[1])

	formData := url.Values{
		"r":                     {s.extractRValue(body)},
		"cf-turnstile-response": {token},
		"g-recaptcha-response":  {token},
	}

	return s.submitChallengeForm(submitURL.String(), resp.Request.URL.String(), formData)
}

func (s *Scraper) submitChallengeForm(submitURL, refererURL string, formData url.Values) (*http.Response, error) {
	req, _ := http.NewRequest("POST", submitURL, strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", refererURL)

	// Use the main `do` method to ensure all headers and logic are applied
	return s.do(req)
}

func (s *Scraper) extractRValue(body string) string {
	rValMatch := regexp.MustCompile(`name="r" value="([^"]+)"`).FindStringSubmatch(body)
	if len(rValMatch) > 1 {
		return rValMatch[1]
	}
	return ""
}

func isChallengeResponse(resp *http.Response, body []byte) bool {
	if !strings.HasPrefix(resp.Header.Get("Server"), "cloudflare") {
		return false
	}
	if resp.StatusCode != http.StatusServiceUnavailable && resp.StatusCode != http.StatusForbidden {
		return false
	}

	bodyStr := string(body)
	return jsV1DetectRegex.MatchString(bodyStr) || jsV2DetectRegex.MatchString(bodyStr) || captchaDetectRegex.MatchString(bodyStr)
}
