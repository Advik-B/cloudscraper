package js

import (
	"fmt"
	"net/url"
	"time"

	"github.com/Advik-B/cloudscraper/lib/errors"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

// GojaEngine uses the embedded goja interpreter with a Node.js-style event loop.
type GojaEngine struct{}

// NewGojaEngine creates a new engine that uses the built-in goja interpreter.
func NewGojaEngine() *GojaEngine {
	return &GojaEngine{}
}

// Run executes a script in goja using an event loop to support async operations like setTimeout.
func (e *GojaEngine) Run(script string, _ *url.URL, body string) (string, error) {
	// --- CHANGE: Use goja_nodejs event loop for a cleaner implementation ---
	loop := eventloop.NewEventLoop()

	var result string
	var scriptErr error
	var program *goja.Program

	// First, compile the script.
	program, err := goja.Compile("challenge-script", script, true)
	if err != nil {
		return "", fmt.Errorf("goja: failed to compile script: %w", err)
	}

	// Use a channel to signal completion.
	done := make(chan struct{})

	loop.Run(func(vm *goja.Runtime) {
		// Ensure we signal completion even on panic.
		defer func() {
			if r := recover(); r != nil {
				scriptErr = fmt.Errorf("goja: script panicked: %v", r)
			}
			close(done)
		}()

		// Set up console.log to capture the result.
		_ = vm.Set("console", map[string]interface{}{
			"log": func(call goja.FunctionCall) goja.Value {
				if len(call.Arguments) > 0 {
					result = call.Argument(0).String()
				}
				// The script has produced its result, we can stop the loop.
				loop.Stop()
				return vm.ToValue(nil)
			},
		})

		// Execute the compiled script.
		_, err := vm.RunProgram(program)
		if err != nil {
			scriptErr = err
		}
	})

	// Wait for the event loop to finish or for a timeout.
	select {
	case <-done:
		// Event loop finished.
	case <-time.After(10 * time.Second):
		loop.Stop() // Attempt to stop the loop.
		<-done      // Wait for it to actually stop.
		return "", errors.ErrChallengeTimeout
	}

	if scriptErr != nil {
		return "", fmt.Errorf("goja: script execution failed: %w", scriptErr)
	}

	return result, nil
}
