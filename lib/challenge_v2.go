package cloudscraper

import (
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"

	"github.com/Advik-B/cloudscraper/lib/js"
)

// Use a simple, robust regex to capture the content of ANY script tag.
var v2ScriptRegex = regexp.MustCompile(`(?s)<script[^>]*?>(.*?)</script>`)

// --- CHANGE 1: Update function signature to accept a full *url.URL ---
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

	if ottoEngine, ok := engine.(*js.OttoEngine); ok {
		// Note: Otto will likely fail this advanced challenge, but we keep the path for compatibility.
		return ottoEngine.SolveV2Challenge(body, pageURL.Host, challengeScripts, logger)
	}

	// --- CHANGE 2: Pass the full URL object to the external solver ---
	return solveV2WithExternal(pageURL, challengeScripts, engine)
}

// --- CHANGE 3: Update signature and implement a new, powerful DOM shim ---
func solveV2WithExternal(pageURL *url.URL, scriptMatches [][]string, engine js.Engine) (string, error) {
	// A comprehensive DOM shim that defines `location` dynamically and intercepts dynamic script loading.
	setupScript := fmt.Sprintf(`
		(function (global) {
			var window = global;
			global.self = window;
			global.window = window;
			global.top = window;
			global.parent = window;

			global.location = {
				href: "%s",
				protocol: "%s:",
				host: "%s",
				hostname: "%s",
				port: "%s",
				pathname: "%s",
				search: "?%s",
				hash: "%s",
				assign: function() {},
				reload: function() {},
				replace: function() {}
			};

			global.navigator = {
				userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
				platform: "Win32",
				language: "en-US"
			};

			global.history = {
				replaceState: function() {}
			};

			global.document = {
				elements: {},
				getElementById: function(id) {
					if (!this.elements[id]) { this.elements[id] = { value: "" }; }
					return this.elements[id];
				},
				createElement: function(tag) {
					return { setAttribute: function() {}, src: "" };
				},
				getElementsByTagName: function(name) {
					if (name === 'head' || name === 'body') {
						return [{
							appendChild: function(element) {
								// CRITICAL: Intercept and neutralize dynamic script loading.
							}
						}];
					}
					return [];
				},
				cookie: ""
			};

			var atobImpl = 'var chars="ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=";var a,b,c,d,e,f,g,i=0,result="";str=str.replace(/[^A-Za-z0-9\\+\\/\\=]/g,"");do{a=chars.indexOf(str.charAt(i++));b=chars.indexOf(str.charAt(i++));c=chars.indexOf(str.charAt(i++));d=chars.indexOf(str.charAt(i++));e=a<<18|b<<12|c<<6|d;f=e>>16&255;g=e>>8&255;a=e&255;result+=String.fromCharCode(f);if(c!=64)result+=String.fromCharCode(g);if(d!=64)result+=String.fromCharCode(a)}while(i<str.length);return result;';
			global.atob = new Function('str', atobImpl);

		})(typeof globalThis !== 'undefined' ? globalThis : this);
	`,
		pageURL.String(),
		pageURL.Scheme,
		pageURL.Host,
		pageURL.Hostname(),
		pageURL.Port(),
		pageURL.Path,
		pageURL.RawQuery,
		pageURL.Fragment,
	)

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
