package js

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

const (
	// Maximum script size to prevent DoS attacks (5MB)
	maxExternalScriptSize = 5 * 1024 * 1024
)

// ExternalEngine uses an external command-line JS runtime (node, deno, bun).
type ExternalEngine struct {
	Command string
}

// NewExternalEngine creates a new engine that shells out to an external command.
func NewExternalEngine(command string) (*ExternalEngine, error) {
	// Security: Only allow known, safe commands to be executed to prevent command injection.
	switch command {
	case "node", "deno", "bun":
		// This is a supported and expected runtime.
	default:
		return nil, fmt.Errorf("unsupported or unsafe external JS runtime: '%s'", command)
	}

	// Check if the command exists in the system's PATH.
	if _, err := exec.LookPath(command); err != nil {
		return nil, fmt.Errorf("javascript runtime '%s' not found in PATH: %w", command, err)
	}
	return &ExternalEngine{Command: command}, nil
}

// Run executes a script by piping it to the external runtime's stdin.
func (e *ExternalEngine) Run(script string) (string, error) {
	// Security: Check script size to prevent DoS attacks
	if len(script) > maxExternalScriptSize {
		return "", fmt.Errorf("script size exceeds maximum allowed size (%d bytes)", maxExternalScriptSize)
	}

	// Security: The `e.Command` field is sanitized in the constructor (NewExternalEngine),
	// making this call safe from command injection.
	cmd := exec.Command(e.Command)
	cmd.Stdin = strings.NewReader(script)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("external js runtime '%s' failed with exit error: %w. Stderr: %s", e.Command, err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}
