package js

import (
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/Advik-B/cloudscraper/lib/errors"
	"github.com/robertkrimen/otto"
)

// OttoEngine uses the embedded otto interpreter.
type OttoEngine struct{}

// NewOttoEngine creates a new engine that uses the built-in otto interpreter.
func NewOttoEngine() *OttoEngine {
	return &OttoEngine{}
}

// Run executes a script in otto. It captures output by overriding console.log.
func (e *OttoEngine) Run(script string) (string, error) {
	vm := otto.New()
	var result string

	err := vm.Set("console", map[string]interface{}{
		"log": func(call otto.FunctionCall) otto.Value {
			result = call.Argument(0).String()
			return otto.Value{}
		},
	})
	if err != nil {
		return "", fmt.Errorf("otto: failed to set console.log: %w", err)
	}

	const maxExecutionTime = 3 * time.Second
	vm.Interrupt = make(chan func(), 1)
	watchdogDone := make(chan struct{})
	defer close(watchdogDone)

	go func() {
		select {
		case <-time.After(maxExecutionTime):
			vm.Interrupt <- func() {
				panic(errors.ErrExecutionTimeout)
			}
		case <-watchdogDone:
		}
	}()

	defer func() {
		if r := recover(); r != nil {
			if r == errors.ErrExecutionTimeout {
				err = fmt.Errorf("otto: script execution timed out after %v", maxExecutionTime)
			} else {
				panic(r)
			}
		}
	}()

	_, err = vm.Run(script)
	if err != nil {
		return "", fmt.Errorf("otto: script execution failed: %w", err)
	}
	return result, nil
}

// SolveV2Challenge uses the original synchronous method to solve v2 challenges.
// It now also uses the unified shim generator.
func (e *OttoEngine) SolveV2Challenge(body string, pageURL *url.URL, scriptMatches [][]string, logger *log.Logger) (string, error) {
	vm := otto.New()

	// Generate the shim from our unified template.
	setupScript, err := GenerateShim(pageURL)
	if err != nil {
		return "", fmt.Errorf("otto: failed to generate JS shim: %w", err)
	}

	if _, err := vm.Run(setupScript); err != nil {
		return "", fmt.Errorf("otto: failed to set up DOM shim: %w", err)
	}

	for _, match := range scriptMatches {
		if len(match) > 1 {
			scriptContent := match[1]
			scriptContent = strings.ReplaceAll(scriptContent, `document.getElementById('challenge-form');`, "({})")
			if _, err := vm.Run(scriptContent); err != nil {
				logger.Printf("otto: warning, a script block failed to run: %v\n", err)
			}
		}
	}

	time.Sleep(4 * time.Second)

	answerObj, err := vm.Run(`document.getElementById('jschl-answer').value`)
	if err != nil || !answerObj.IsString() {
		return "", fmt.Errorf("otto: could not retrieve final answer from VM: %w", err)
	}

	return answerObj.String(), nil
}
