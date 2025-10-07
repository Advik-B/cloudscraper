package cloudscraper

import (
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"

	"github.com/Advik-B/cloudscraper/lib/js"
)

var v2ScriptRegex = regexp.MustCompile(`(?s)<script[^>]*?>(.*?)</script>`)

func solveV2Logic(body string, pageURL *url.URL, engine js.Engine, logger *log.Logger) (string, error) {
	allMatches := v2ScriptRegex.FindAllStringSubmatch(body, -1)
	if len(allMatches) == 0 {
		return "", fmt.Errorf("could not find any JS scripts on challenge page")
	}

	var challengeScripts [][]string
	for _, match := range allMatches {
		if len(match) > 1 && (strings.Contains(match[0], "challenge-platform") || strings.Contains(match[1], "_cf_chl_opt")) {
			challengeScripts = append(challengeScripts, match)
		}
	}

	if len(challengeScripts) == 0 {
		return "", fmt.Errorf("could not find the primary Cloudflare JS challenge script")
	}

	// Use the external solving approach for all engines now
	return solveV2WithExternal(pageURL, challengeScripts, engine)
}

func solveV2WithExternal(pageURL *url.URL, scriptMatches [][]string, engine js.Engine) (string, error) {
	var fullScript strings.Builder

	// --- THE CORE FIX ---
	// Only generate and prepend our custom shim if the engine is NOT node.
	// The node engine uses JSDOM, which is its own, complete shim.
	if extEngine, ok := engine.(*js.ExternalEngine); !ok || extEngine.Command != "node" {
		setupScript, err := js.GenerateShim(pageURL)
		if err != nil {
			return "", fmt.Errorf("failed to generate JS shim: %w", err)
		}
		fullScript.WriteString(setupScript)
	}

	// Concatenate the raw challenge scripts.
	for _, match := range scriptMatches {
		if len(match) > 1 {
			scriptContent := match[1]
			scriptContent = strings.ReplaceAll(scriptContent, `document.getElementById('challenge-form');`, "({})")
			fullScript.WriteString(scriptContent)
			fullScript.WriteString(";\n")
		}
	}

	// Append the answer extractor, which is needed by all engines.
	answerExtractor := `
        setTimeout(function() {
            try {
                var answer = document.getElementById('jschl-answer').value;
                console.log(answer);
            } catch (e) {
                console.log(""); // Log empty string on failure to prevent hanging
            }
        }, 4100);
    `
	fullScript.WriteString(answerExtractor)

	// Pass the final script and the real pageURL to the engine.
	return engine.Run(fullScript.String(), pageURL, "")
}
