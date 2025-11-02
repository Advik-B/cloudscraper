package js

import (
	_ "embed"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/dop251/goja"
)

// Create a Simulated Browser Environment (DOM Shim)
//
//go:embed setup.js
var setupScript string

// GojaEngine uses the embedded goja interpreter.
type GojaEngine struct{}

// NewGojaEngine creates a new engine that uses the built-in goja interpreter.
func NewGojaEngine() *GojaEngine {
	return &GojaEngine{}
}

// Run executes a script in goja. It captures output by overriding console.log.
func (e *GojaEngine) Run(script string) (string, error) {
	vm := goja.New()
	var result string

	// Setup safe console.log capturing
	console := vm.NewObject()
	console.Set("log", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			result = call.Arguments[0].String()
		}
		return goja.Undefined()
	})
	vm.Set("console", console)

	// === Hardened Execution ===
	const maxExecutionTime = 3 * time.Second
	done := make(chan struct{})
	var execErr error

	// Run the script in a goroutine with timeout
	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				execErr = fmt.Errorf("goja: panic during execution: %v", r)
			}
		}()
		
		_, execErr = vm.RunString(script)
	}()

	// Wait for completion or timeout
	timer := time.NewTimer(maxExecutionTime)
	defer timer.Stop()
	
	select {
	case <-done:
		if execErr != nil {
			return "", fmt.Errorf("goja: script execution failed: %w", execErr)
		}
		return result, nil
	case <-timer.C:
		vm.Interrupt("execution timeout")
		<-done // Wait for goroutine to finish
		return "", fmt.Errorf("goja: script execution timed out after %v", maxExecutionTime)
	}
}

// SolveV2Challenge uses the original synchronous method to solve v2 challenges,
// as goja does not support asynchronous operations like setTimeout without additional setup.
func (e *GojaEngine) SolveV2Challenge(body, domain string, scriptMatches [][]string, logger *log.Logger) (string, error) {
	vm := goja.New()

	// Security: Running setup script in VM.
	if _, err := vm.RunString(setupScript); err != nil {
		return "", fmt.Errorf("goja: failed to set up DOM shim: %w", err)
	}

	// Execute all extracted Cloudflare scripts in the same VM context.
	for _, match := range scriptMatches {
		if len(match) > 1 {
			scriptContent := match[1]
			scriptContent = strings.ReplaceAll(scriptContent, `document.getElementById('challenge-form');`, "({})")
			// Security: This executes JavaScript from the Cloudflare challenge page.
			// The goja VM is sandboxed, but this is an inherent risk of the library's function.
			if _, err := vm.RunString(scriptContent); err != nil {
				logger.Printf("goja: warning, a script block failed to run: %v\n", err)
			}
		}
	}

	// Wait for the script's internal timeouts to complete.
	time.Sleep(4 * time.Second)

	// Get the final answer from the 'jschl_answer' field in the dummy document.
	// Security: This executes a small, controlled script to retrieve a value.
	answerVal, err := vm.RunString(`document.getElementById('jschl-answer').value`)
	if err != nil {
		return "", fmt.Errorf("goja: could not retrieve final answer from VM: %w", err)
	}

	answer := answerVal.String()
	if answer == "" || answer == "undefined" {
		return "", fmt.Errorf("goja: answer value is empty or undefined")
	}

	return answer, nil
}
