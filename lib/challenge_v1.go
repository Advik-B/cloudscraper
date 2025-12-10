package cloudscraper

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Advik-B/cloudscraper/lib/errors"
	"github.com/Advik-B/cloudscraper/lib/js"
)

var (
	jsV1ChallengeRegex = regexp.MustCompile(`setTimeout\(function\(\){\s+(var s,t,o,p,b,r,e,a,k,i,n,g,f.+?a\.value =.+?)\r?\n`)
	jsV1PassRegex      = regexp.MustCompile(`a\.value = (.+?)\.toFixed\(10\)`)
)

// sanitizeDomain ensures the domain string is safe to use in JavaScript context
// by escaping special characters that could break out of string literals.
func sanitizeDomain(domain string) string {
	// Replace backslash first to prevent double-escaping
	domain = strings.ReplaceAll(domain, `\`, `\\`)
	// Escape single quotes to prevent breaking out of string literals
	domain = strings.ReplaceAll(domain, `'`, `\'`)
	// Escape newlines and other control characters
	domain = strings.ReplaceAll(domain, "\n", `\n`)
	domain = strings.ReplaceAll(domain, "\r", `\r`)
	domain = strings.ReplaceAll(domain, "\t", `\t`)
	return domain
}

// solveV1Logic prepares and executes the v1 JS challenge using the configured engine.
func solveV1Logic(body, domain string, engine js.Engine) (string, error) {
	matches := jsV1ChallengeRegex.FindStringSubmatch(body)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not find Cloudflare v1 JS challenge script: %w", errors.ErrChallenge)
	}
	challengeScript := matches[1]

	passMatches := jsV1PassRegex.FindStringSubmatch(challengeScript)
	if len(passMatches) < 2 {
		return "", fmt.Errorf("could not find JS v1 pass expression: %w", errors.ErrChallenge)
	}
	// finalExpression is the core calculation, e.g., `(+((!![]+!![]...))) + t.length`
	finalExpression := passMatches[1]

	// Security: Sanitize domain to prevent injection attacks
	safeDomain := sanitizeDomain(domain)

	// Create a self-contained script that can be executed by any JS runtime.
	// It prints the final answer to stdout, which is captured by the engine.
	fullScript := fmt.Sprintf(`
        var t = '%s';
        var result = (%s).toFixed(10);
        console.log(result);
    `, safeDomain, finalExpression)

	return engine.Run(fullScript)
}
