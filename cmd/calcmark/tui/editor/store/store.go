//go:build !js && !wasm

// Package store defines the interface for remote document storage backends.
// Each backend delegates authentication and network operations to an external
// CLI tool (e.g., gh for GitHub Gists), so CalcMark never handles credentials.
package store

import "errors"

// ErrCLINotFound indicates the backing CLI tool is not installed.
var ErrCLINotFound = errors.New("CLI tool not found")

// ErrNotAuthenticated indicates the user is not authenticated with the backend.
var ErrNotAuthenticated = errors.New("not authenticated")

// ShareResult holds the outcome of a Share operation.
type ShareResult struct {
	URL string // Shareable URL for the created resource
}

// OpenResult holds the outcome of an Open operation.
type OpenResult struct {
	Content  string // Document content fetched from the remote resource
	Filename string // Original filename from the remote resource
}

// Store defines the operations a remote store backend must support.
// Implementations delegate to external CLI tools for auth and network access.
type Store interface {
	// Name returns the display name (e.g., "GitHub Gist").
	Name() string

	// CheckAvailable verifies the backing CLI tool is installed and on PATH.
	CheckAvailable() error

	// CheckAuth verifies the user is authenticated with the backend.
	CheckAuth() error

	// Share creates a new remote resource from the given content.
	// Returns the shareable URL on success.
	Share(content, filename, description string, public bool) (ShareResult, error)

	// Open fetches content from a remote resource identified by a URL or ID.
	Open(identifier string) (OpenResult, error)
}

// Executor abstracts command execution for testability.
// Production code uses RealExecutor; tests inject a mock.
type Executor interface {
	// Run executes a command and returns captured stdout, stderr, and any error.
	Run(name string, args ...string) (stdout, stderr []byte, err error)

	// LookPath searches for an executable on PATH.
	LookPath(name string) (string, error)
}
