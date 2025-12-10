package cloudscraper

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/Advik-B/cloudscraper/lib/js"
	"github.com/Advik-B/cloudscraper/lib/security"
)

// Regex to find and extract the modern challenge script content.
var v2ScriptRegex = regexp.MustCompile(`(?s)<script[^>]*>(.*?window\._cf_chl_opt.*?)<\/script>`)

// solveV2Logic solves modern v2/v3 challenges by delegating to the appropriate JS engine implementation.
func solveV2Logic(body, domain string, engine js.Engine, logger *log.Logger) (string, error) {
	scriptMatches := v2ScriptRegex.FindAllStringSubmatch(body, -1)
	if len(scriptMatches) == 0 {
		return "", fmt.Errorf("could not find modern JS challenge scripts")
	}

	// Security: Check total script size to prevent DoS
	if err := security.ValidateTotalScriptSize(scriptMatches, security.MaxChallengeScriptSize); err != nil {
		return "", err
	}

	// Use a special synchronous path for Goja, which can't handle async setTimeout.
	if gojaEngine, ok := engine.(*js.GojaEngine); ok {
		return gojaEngine.SolveV2Challenge(body, domain, scriptMatches, logger)
	}

	// Use a modern asynchronous path for external runtimes (node, deno, bun).
	return solveV2WithExternal(domain, scriptMatches, engine)
}

// solveV2WithExternal builds a full script with shims and an async callback to solve the challenge.
func solveV2WithExternal(domain string, scriptMatches [][]string, engine js.Engine) (string, error) {
	// Security: Sanitize domain to prevent injection
	safeDomain := security.SanitizeDomainForJS(domain)
	if safeDomain == "" {
		// Don't expose the original domain in error message for security
		return "", fmt.Errorf("invalid domain: contains only filtered characters or is empty")
	}

	// This DOM shim is required for the challenge script to run in a non-browser environment.
	atobImpl := `
        var chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=';
        var a, b, c, d, e, f, g, i = 0, result = '';
        str = str.replace(/[^A-Za-z0-9\+\/\=]/g, '');
        do {
            a = chars.indexOf(str.charAt(i++)); b = chars.indexOf(str.charAt(i++));
            c = chars.indexOf(str.charAt(i++)); d = chars.indexOf(str.charAt(i++));
            e = a << 18 | b << 12 | c << 6 | d; f = e >> 16 & 255; g = e >> 8 & 255; a = e & 255;
            result += String.fromCharCode(f);
            if (c != 64) result += String.fromCharCode(g);
            if (d != 64) result += String.fromCharCode(a);
        } while (i < str.length);
        return result;
    `

	setupScript := `
		var window = globalThis;
		var navigator = { userAgent: "" };
		var document = {
			getElementById: function(id) {
				if (!this.elements) this.elements = {};
				if (!this.elements[id]) this.elements[id] = { value: "" };
				return this.elements[id];
			},
			createElement: function(tag) {
				return {
					firstChild: { href: "https://` + safeDomain + `/" }
				};
			},
			cookie: ""
		};
		var atob = function(str) {` + atobImpl + `};
	`

	var fullScript strings.Builder
	fullScript.WriteString(setupScript)

	for _, match := range scriptMatches {
		if len(match) > 1 {
			scriptContent := match[1]
			// The script expects a 'challenge-form' to exist for submission. We stub it.
			scriptContent = strings.ReplaceAll(scriptContent, `document.getElementById('challenge-form');`, "({})")
			fullScript.WriteString(scriptContent)
			fullScript.WriteString(";\n")
		}
	}

	// The Cloudflare script uses a setTimeout of 4000ms. We'll wait a little longer
	// and then extract the answer, printing it to stdout for Go to capture.
	answerExtractor := `
        setTimeout(function() {
            try {
                var answer = document.getElementById('jschl-answer').value;
                console.log(answer);
            } catch (e) {
                // Ignore errors if the element isn't found, the process will just exit.
            }
        }, 4100);
    `
	fullScript.WriteString(answerExtractor)

	return engine.Run(fullScript.String())
}
