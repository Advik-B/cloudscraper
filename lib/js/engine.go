package js

// Engine defines the interface for a JavaScript runtime.
type Engine interface {
	// Run executes a self-contained JavaScript script and returns the result from stdout.
	Run(script string) (string, error)
}

// Runtime represents the name of a supported JavaScript runtime.
type Runtime string

const (
	// Goja is the built-in Go-based interpreter. Recommended for standalone binaries.
	Goja Runtime = "goja"
	// Deprecated:  Use Goja instead. Otto will no longer be supported.
	Otto Runtime = "otto"
	// Node uses the external Node.js runtime.
	Node Runtime = "node"
	// Deno uses the external Deno runtime.
	Deno Runtime = "deno"
	// Bun uses the external Bun runtime.
	Bun Runtime = "bun"
)

// Custom allows using a custom JavaScript runtime by specifying its path.
// Warning: This is unsafe because you're executing arbitrary code.
func CustomRuntime(path string) Runtime {
	return Runtime(path)
}
