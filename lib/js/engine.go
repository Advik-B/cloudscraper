package js

// Engine defines the interface for a JavaScript runtime.
type Engine interface {
	// Run executes a self-contained JavaScript script and returns the result from stdout.
	Run(script string) (string, error)
}

// Runtime represents the name of a supported JavaScript runtime.
type Runtime string

const (
	// Goja is the built-in Go-based interpreter (default, recommended).
	Goja Runtime = "goja"
	// Otto is the legacy built-in Go-based interpreter (deprecated, use Goja instead).
	// For backwards compatibility, Otto requests are redirected to Goja with a warning.
	Otto Runtime = "otto"
	// Node uses the external Node.js runtime.
	Node Runtime = "node"
	// Deno uses the external Deno runtime.
	Deno Runtime = "deno"
	// Bun uses the external Bun runtime.
	Bun Runtime = "bun"
)
