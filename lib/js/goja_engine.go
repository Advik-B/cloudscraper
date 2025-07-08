package js

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Advik-B/cloudscraper/lib/errors"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

// GojaEngine uses the embedded goja interpreter.
type GojaEngine struct{}

// NewGojaEngine creates a new engine that uses the built-in goja interpreter.
func NewGojaEngine() *GojaEngine {
	return &GojaEngine{}
}

// Run executes a script in goja. It uses an event loop to support async operations like setTimeout.
func (e *GojaEngine) Run(script string) (string, error) {
	loop := eventloop.NewEventLoop()
	var result string
	var scriptErr error
	var wg sync.WaitGroup
	wg.Add(1)

	loop.Run(func(vm *goja.Runtime) {
		// Set up console.log to capture the result.
		_ = vm.Set("console", map[string]interface{}{
			"log": func(call goja.FunctionCall) goja.Value {
				if len(call.Arguments) > 0 {
					result = call.Argument(0).String()
				}
				// The script might be finished, let's signal the wait group.
				// We also set a short timeout to exit if the script doesn't explicitly.
				time.AfterFunc(100*time.Millisecond, wg.Done)
				return vm.ToValue(nil)
			},
		})

		// Run the script.
		_, err := vm.RunString(script)
		if err != nil {
			scriptErr = err
			wg.Done()
		}
	})

	// Wait for the script to finish or for a timeout.
	waitChan := make(chan struct{})
	go func() {
		defer close(waitChan)
		wg.Wait()
	}()

	select {
	case <-waitChan:
		// Script finished normally.
	case <-time.After(10 * time.Second):
		return "", errors.ErrChallengeTimeout
	}

	if scriptErr != nil {
		return "", fmt.Errorf("goja: script execution failed: %w", scriptErr)
	}

	return result, nil
}

// SolveV2Challenge is now deprecated in favor of the unified Run method,
// as goja's event loop can handle the asynchronous challenge script directly.
// This function is kept for API compatibility but simply delegates.
func (e *GojaEngine) SolveV2Challenge(body string, pageURL *url.URL, scriptMatches [][]string, _ any) (string, error) {
	// Generate the shim from our unified template.
	setupScript, err := GenerateShim(pageURL)
	if err != nil {
		return "", fmt.Errorf("goja: failed to generate JS shim: %w", err)
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

	// The answer extractor is now the only way for the script to signal it's done.
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

	return e.Run(fullScript.String())
}
