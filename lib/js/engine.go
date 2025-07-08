package js

import "net/url"

// Engine defines the interface for a JavaScript runtime.
type Engine interface {
	// Run executes a self-contained JavaScript script and returns the result from stdout.
	// The pageURL is provided for context, especially for external engines like Node/JSDOM.
	Run(script string, pageURL *url.URL, htmlBody string) (string, error)
}

// Runtime represents the name of a supported JavaScript runtime.
type Runtime string

const (
	// Goja is the built-in Go-based interpreter. Recommended for standalone binaries.
	Goja Runtime = "goja"
	// Otto is a DEPRECATED alias for Goja. It will be removed in a future version.
	Otto Runtime = "otto"
	// Node uses the external Node.js runtime.
	Node Runtime = "node"
	// Deno uses the external Deno runtime.
	Deno Runtime = "deno"
	// Bun uses the external Bun runtime.
	Bun Runtime = "bun"
)
