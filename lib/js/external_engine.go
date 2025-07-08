package js

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// ExternalEngine uses an external command-line JS runtime (node, deno, bun).
type ExternalEngine struct {
	Command string
}

// NewExternalEngine creates a new engine that shells out to an external command.
func NewExternalEngine(command string) (*ExternalEngine, error) {
	switch command {
	case "node", "deno", "bun":
		// Supported runtimes
	default:
		return nil, fmt.Errorf("unsupported or unsafe external JS runtime: '%s'", command)
	}

	if _, err := exec.LookPath(command); err != nil {
		return nil, fmt.Errorf("javascript runtime '%s' not found in PATH: %w", command, err)
	}
	return &ExternalEngine{Command: command}, nil
}

// Run executes a script. For 'node', it uses a jsdom bootstrap script for a full browser environment.
func (e *ExternalEngine) Run(script string) (string, error) {
	var cmd *exec.Cmd

	// --- CHANGE: Use JSDOM for the Node.js runtime ---
	if e.Command == "node" {
		// This script initializes jsdom, injects the challenge script, and listens for the result.
		// It expects the challenge script to be passed as the first argument.
		bootstrapScript := `
			const { JSDOM } = require('jsdom');
			const scriptToRun = process.argv[1];

			if (!scriptToRun) {
				console.error("No script provided to jsdom bootstrap.");
				process.exit(1);
			}

			const dom = new JSDOM('<body></body>', {
				url: "https://nowsecure.nl/", // A placeholder, the actual URL is set within the script via location obj
				runScripts: "dangerously",   // We must allow the script to execute
				pretendToBeVisual: true,     // Helps with things like requestAnimationFrame
			});
			
			const window = dom.window;
			const document = window.document;

			// Redirect console.log to stdout to capture the answer
			window.console.log = (data) => {
				process.stdout.write(String(data));
				// Give a moment for stdout to flush before exiting
				setTimeout(() => process.exit(0), 50);
			};
			
			// Handle uncaught exceptions
			window.addEventListener('error', (event) => {
  				console.error('Script Error:', event.error);
				process.exit(1);
			});

			try {
				// The provided script will define location, navigator, etc.
				// and then run the challenge logic.
				window.eval(scriptToRun);
			} catch (e) {
				console.error('Eval Error:', e);
				process.exit(1);
			}

			// Timeout if the script doesn't call console.log
			setTimeout(() => {
				// console.error("JSDOM execution timed out.");
				process.exit(0); // Exit gracefully so we can get partial results if any
			}, 10000);
		`
		// Pass our full script (shim + challenge) as an argument to the node bootstrap script.
		cmd = exec.Command(e.Command, "-e", bootstrapScript, "--", script)

	} else {
		// Other runtimes (deno, bun) will use the previous stdin method.
		cmd = exec.Command(e.Command)
		cmd.Stdin = strings.NewReader(script)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// jsdom with exit code 0 is a success, even if it looks like an error to Go.
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 0 {
			// It's a success.
		} else {
			return "", fmt.Errorf("external js runtime '%s' failed: %w. Stderr: %s", e.Command, err, stderr.String())
		}
	}

	return strings.TrimSpace(stdout.String()), nil
}
