//go:build js || wasm

// Package store provides stubs for WASM builds where subprocess execution
// is unavailable. All operations return ErrUnavailable.
package store

import "errors"

// ErrCLINotFound indicates the backing CLI tool is not installed.
var ErrCLINotFound = errors.New("CLI tool not found")

// ErrNotAuthenticated indicates the user is not authenticated with the backend.
var ErrNotAuthenticated = errors.New("not authenticated")

// ErrUnavailable indicates remote store operations are not available in this build.
var ErrUnavailable = errors.New("remote store not available in browser")

// ShareResult holds the outcome of a Share operation.
type ShareResult struct {
	URL string
}

// OpenResult holds the outcome of an Open operation.
type OpenResult struct {
	Content  string
	Filename string
}

// Store defines the remote store interface (stub for WASM).
type Store interface {
	Name() string
	CheckAvailable() error
	CheckAuth() error
	Share(content, filename, description string, public bool) (ShareResult, error)
	Open(identifier string) (OpenResult, error)
}

// Executor abstracts command execution (stub for WASM).
type Executor interface {
	Run(name string, args ...string) (stdout, stderr []byte, err error)
	LookPath(name string) (string, error)
}
