package errors

import "errors"

var (
	ErrCloudflare         = errors.New("cloudflare error")
	ErrChallenge          = errors.New("challenge error")
	ErrUnknownChallenge   = errors.New("unknown cloudflare challenge")
	ErrChallengeTimeout   = errors.New("challenge solving timeout")
	ErrNoCaptchaSolver    = errors.New("captcha provider not configured")
	ErrAllProxiesBanned   = errors.New("all proxies are currently banned")
	ErrMaxRetriesExceeded = errors.New("failed after max retries")
)
