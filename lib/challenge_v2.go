package cloudscraper

import (
	"fmt"
	"github.com/Advik-B/cloudscraper/lib/js"
	"log"
	"net/url"
	"regexp"
	"strings"
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

	if gojaEngine, ok := engine.(*js.GojaEngine); ok {
		// Pass the full URL to the goja solver so it can also generate the correct shim.
		return gojaEngine.SolveV2Challenge(body, pageURL, challengeScripts, logger)
	}

	return solveV2WithExternal(pageURL, challengeScripts, engine)
}

func solveV2WithExternal(pageURL *url.URL, scriptMatches [][]string, engine js.Engine) (string, error) {
	// Generate the shim from our unified template.
	setupScript, err := js.GenerateShim(pageURL)
	if err != nil {
		return "", fmt.Errorf("failed to generate JS shim: %w", err)
	}

	var fullScript strings.Builder
	fullScript.WriteString(setupScript)

	for _, match := range scriptMatches {
		if len(match) > 1 {
			scriptContent := match[1]
			scriptContent = strings.ReplaceAll(scriptContent, `document.getElementById('challenge-form');`, "({})")
			fullScript.WriteString(scriptContent)
			fullScript.WriteString(";\n")
		}
	}

	answerExtractor := `
        setTimeout(function() {
            try {
                var answer = document.getElementById('jschl-answer').value;
                console.log(answer);
            } catch (e) {
                // Ignore errors if the element isn't found.
            }
        }, 4100);
    `
	fullScript.WriteString(answerExtractor)

	return engine.Run(fullScript.String())
}
