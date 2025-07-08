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

// Run executes a script. For 'node', it uses the embedded jsdom bundle.
func (e *ExternalEngine) Run(script string) (string, error) {
	var cmd *exec.Cmd

	// --- CHANGE: Use the embedded and extracted JSDOM bundle for Node.js ---
	if e.Command == "node" {
		bootstrapPath, err := GetNodeBundlePath()
		if err != nil {
			return "", fmt.Errorf("could not prepare embedded node environment: %w", err)
		}

		cmd = exec.Command(e.Command, bootstrapPath)
		// We pipe the full script (shim + challenge) to the bootstrap script's stdin.
		cmd.Stdin = strings.NewReader(script)
	} else {
		// Other runtimes (deno, bun) will continue to use the direct stdin method.
		cmd = exec.Command(e.Command)
		cmd.Stdin = strings.NewReader(script)
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
