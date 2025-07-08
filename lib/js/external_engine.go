package js

import (
	"bytes"
	"fmt"
	"net/url"
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

// Run executes a script. For 'node', it uses the embedded jsdom bundle.
func (e *ExternalEngine) Run(script string, pageURL *url.URL, htmlBody string) (string, error) {
	var cmd *exec.Cmd

	if e.Command == "node" {
		bootstrapPath, err := GetNodeBundlePath()
		if err != nil {
			return "", fmt.Errorf("could not prepare embedded node environment: %w", err)
		}
		if pageURL == nil {
			return "", fmt.Errorf("node engine requires a pageURL, but got nil")
		}
		// Pass bootstrap path, URL, and the script itself as arguments
		cmd = exec.Command(e.Command, bootstrapPath, pageURL.String(), script)
		// Pipe the full HTML of the challenge page to the bootstrap script's stdin
		cmd.Stdin = strings.NewReader(htmlBody)
	} else {
		// Other engines still get the full script via stdin
		fullPayload := script
		if !strings.Contains(script, "global.location") { // Basic check if shim is missing
			shim, _ := GenerateShim(pageURL)
			fullPayload = shim + script
		}
		cmd = exec.Command(e.Command)
		cmd.Stdin = strings.NewReader(fullPayload)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("external js runtime '%s' failed: %w. Stderr: %s", e.Command, err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}
