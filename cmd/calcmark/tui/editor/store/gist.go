//go:build !js && !wasm

package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// maxContentSize is the maximum allowed content size for fetched gists (1MB).
// Matches the file loading limit in SECURITY.md.
const maxContentSize = 1 * 1024 * 1024

// GistStore implements Store for GitHub Gists via the gh CLI.
type GistStore struct {
	exec Executor
}

// NewGistStore creates a GistStore with the given command executor.
func NewGistStore(exec Executor) *GistStore {
	return &GistStore{exec: exec}
}

// Name returns the display name for this store.
func (g *GistStore) Name() string {
	return "GitHub Gist"
}

// CheckAvailable verifies that gh CLI is installed and on PATH.
func (g *GistStore) CheckAvailable() error {
	_, err := g.exec.LookPath("gh")
	if err != nil {
		return fmt.Errorf("%w: gh CLI not found (install: https://cli.github.com)", ErrCLINotFound)
	}
	return nil
}

// CheckAuth verifies the user is authenticated with GitHub via gh.
func (g *GistStore) CheckAuth() error {
	_, stderr, err := g.exec.Run("gh", "auth", "status")
	if err != nil {
		msg := strings.TrimSpace(string(stderr))
		if msg == "" {
			msg = "not logged in"
		}
		return fmt.Errorf("%w: %s", ErrNotAuthenticated, msg)
	}
	return nil
}

// Share creates a new GitHub Gist from the given content.
// The content is written to a temp file which is passed to gh gist create.
// This avoids shell injection: all arguments are passed as separate exec args.
func (g *GistStore) Share(content, filename, description string, public bool) (ShareResult, error) {
	// Write content to a temp file for gh to read
	tmpDir, err := os.MkdirTemp("", "calcmark-share-*")
	if err != nil {
		return ShareResult{}, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, filename)
	if err := os.WriteFile(tmpFile, []byte(content), 0600); err != nil {
		return ShareResult{}, fmt.Errorf("failed to write temp file: %w", err)
	}

	// Build args -- never use shell string concatenation
	args := []string{"gist", "create"}
	if public {
		args = append(args, "--public")
	}
	if description != "" {
		args = append(args, "-d", description)
	}
	args = append(args, tmpFile)

	stdout, stderr, err := g.exec.Run("gh", args...)
	if err != nil {
		msg := strings.TrimSpace(string(stderr))
		if msg == "" {
			msg = err.Error()
		}
		return ShareResult{}, fmt.Errorf("gist create failed: %s", msg)
	}

	url := strings.TrimSpace(string(stdout))
	if url == "" {
		return ShareResult{}, fmt.Errorf("gist create returned empty URL")
	}

	return ShareResult{URL: url}, nil
}

// gistFile represents a file entry in the gh gist JSON output.
type gistFile struct {
	Filename string `json:"filename"`
}

// gistJSON represents the JSON output from gh gist view --json files.
type gistJSON struct {
	Files []gistFile `json:"files"`
}

// Open fetches content from a GitHub Gist by URL or ID.
// For multi-file gists, it picks the first .cm file, or the first file if
// no .cm file exists.
func (g *GistStore) Open(identifier string) (OpenResult, error) {
	// Inspect gist to get the file list
	filename, err := g.selectGistFile(identifier)
	if err != nil {
		return OpenResult{}, err
	}

	// Fetch the selected file's content
	args := []string{"gist", "view", identifier, "-r"}
	if filename != "" {
		args = append(args, "--filename", filename)
	}

	stdout, stderr, err := g.exec.Run("gh", args...)
	if err != nil {
		msg := strings.TrimSpace(string(stderr))
		if msg == "" {
			msg = err.Error()
		}
		return OpenResult{}, fmt.Errorf("gist view failed: %s", msg)
	}

	content := string(stdout)

	// Enforce size limit
	if len(content) > maxContentSize {
		return OpenResult{}, fmt.Errorf("gist content exceeds 1MB limit (%d bytes)", len(content))
	}

	return OpenResult{
		Content:  content,
		Filename: filename,
	}, nil
}

// selectGistFile inspects a gist's file list and picks the best file to open.
// Priority: first .cm file > first file.
func (g *GistStore) selectGistFile(identifier string) (string, error) {
	stdout, stderr, err := g.exec.Run("gh", "gist", "view", identifier, "--json", "files")
	if err != nil {
		msg := strings.TrimSpace(string(stderr))
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("gist inspect failed: %s", msg)
	}

	var gist gistJSON
	if err := json.Unmarshal(stdout, &gist); err != nil {
		return "", fmt.Errorf("failed to parse gist metadata: %w", err)
	}

	if len(gist.Files) == 0 {
		return "", fmt.Errorf("gist contains no files")
	}

	// Single file: use it directly (no --filename needed for single-file gists)
	if len(gist.Files) == 1 {
		return gist.Files[0].Filename, nil
	}

	// Multiple files: prefer first .cm file
	for _, f := range gist.Files {
		if strings.HasSuffix(f.Filename, ".cm") {
			return f.Filename, nil
		}
	}

	// No .cm file: fall back to first file
	return gist.Files[0].Filename, nil
}
