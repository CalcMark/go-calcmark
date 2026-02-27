//go:build !js && !wasm

package store

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// mockExecutor records calls and returns canned responses.
type mockExecutor struct {
	// calls records each invocation as "name arg1 arg2 ...".
	calls []string

	// results maps a command prefix to its canned response.
	// The key is matched against the joined command string.
	results map[string]mockResult
}

type mockResult struct {
	stdout []byte
	stderr []byte
	err    error
}

func (m *mockExecutor) Run(name string, args ...string) ([]byte, []byte, error) {
	call := name + " " + strings.Join(args, " ")
	m.calls = append(m.calls, call)

	// Try exact match first, then prefix match
	if r, ok := m.results[call]; ok {
		return r.stdout, r.stderr, r.err
	}
	for prefix, r := range m.results {
		if strings.HasPrefix(call, prefix) {
			return r.stdout, r.stderr, r.err
		}
	}
	return nil, nil, errors.New("unexpected command: " + call)
}

func (m *mockExecutor) LookPath(name string) (string, error) {
	key := "lookpath:" + name
	if r, ok := m.results[key]; ok {
		return string(r.stdout), r.err
	}
	return "", errors.New("not found: " + name)
}

func TestGistStore_Name(t *testing.T) {
	g := NewGistStore(&mockExecutor{})
	if got := g.Name(); got != "GitHub Gist" {
		t.Errorf("Name() = %q, want %q", got, "GitHub Gist")
	}
}

func TestGistStore_CheckAvailable(t *testing.T) {
	tests := []struct {
		name    string
		results map[string]mockResult
		wantErr error
	}{
		{
			name: "gh found on PATH",
			results: map[string]mockResult{
				"lookpath:gh": {stdout: []byte("/usr/bin/gh")},
			},
			wantErr: nil,
		},
		{
			name:    "gh not found",
			results: map[string]mockResult{},
			wantErr: ErrCLINotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := &mockExecutor{results: tt.results}
			g := NewGistStore(exec)
			err := g.CheckAvailable()
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("CheckAvailable() unexpected error: %v", err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("CheckAvailable() error = %v, want %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestGistStore_CheckAuth(t *testing.T) {
	tests := []struct {
		name    string
		results map[string]mockResult
		wantErr error
	}{
		{
			name: "authenticated",
			results: map[string]mockResult{
				"gh auth status": {stdout: []byte("Logged in")},
			},
			wantErr: nil,
		},
		{
			name: "not authenticated with stderr",
			results: map[string]mockResult{
				"gh auth status": {
					stderr: []byte("You are not logged in"),
					err:    errors.New("exit status 1"),
				},
			},
			wantErr: ErrNotAuthenticated,
		},
		{
			name: "not authenticated without stderr",
			results: map[string]mockResult{
				"gh auth status": {
					err: errors.New("exit status 1"),
				},
			},
			wantErr: ErrNotAuthenticated,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := &mockExecutor{results: tt.results}
			g := NewGistStore(exec)
			err := g.CheckAuth()
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("CheckAuth() unexpected error: %v", err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("CheckAuth() error = %v, want %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestGistStore_Share(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		filename    string
		description string
		public      bool
		results     map[string]mockResult
		wantURL     string
		wantErr     bool
		wantArgs    []string // partial match on call args
	}{
		{
			name:        "public gist with description",
			content:     "# Test\n1 + 1 =\n",
			filename:    "test.cm",
			description: "My calculation",
			public:      true,
			results: map[string]mockResult{
				"gh gist create": {stdout: []byte("https://gist.github.com/abc123\n")},
			},
			wantURL:  "https://gist.github.com/abc123",
			wantArgs: []string{"--public", "-d", "My calculation"},
		},
		{
			name:     "secret gist without description",
			content:  "# Test\n",
			filename: "untitled.cm",
			public:   false,
			results: map[string]mockResult{
				"gh gist create": {stdout: []byte("https://gist.github.com/def456\n")},
			},
			wantURL: "https://gist.github.com/def456",
		},
		{
			name:     "gh gist create fails",
			content:  "content",
			filename: "test.cm",
			public:   false,
			results: map[string]mockResult{
				"gh gist create": {
					stderr: []byte("HTTP 403: forbidden"),
					err:    errors.New("exit status 1"),
				},
			},
			wantErr: true,
		},
		{
			name:     "gh returns empty URL",
			content:  "content",
			filename: "test.cm",
			public:   false,
			results: map[string]mockResult{
				"gh gist create": {stdout: []byte("  \n")},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := &mockExecutor{results: tt.results}
			g := NewGistStore(exec)
			result, err := g.Share(tt.content, tt.filename, tt.description, tt.public)
			if tt.wantErr {
				if err == nil {
					t.Fatal("Share() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Share() unexpected error: %v", err)
			}
			if result.URL != tt.wantURL {
				t.Errorf("Share() URL = %q, want %q", result.URL, tt.wantURL)
			}
			// Check that expected args were passed
			if len(exec.calls) > 0 && len(tt.wantArgs) > 0 {
				call := exec.calls[0]
				for _, arg := range tt.wantArgs {
					if !strings.Contains(call, arg) {
						t.Errorf("Share() call %q missing expected arg %q", call, arg)
					}
				}
			}
		})
	}
}

func TestGistStore_Open(t *testing.T) {
	singleFileJSON := mustJSON(t, gistJSON{
		Files: []gistFile{{Filename: "calc.cm"}},
	})
	multiFileJSON := mustJSON(t, gistJSON{
		Files: []gistFile{
			{Filename: "readme.md"},
			{Filename: "budget.cm"},
			{Filename: "notes.txt"},
		},
	})
	multiFileNoCM := mustJSON(t, gistJSON{
		Files: []gistFile{
			{Filename: "data.txt"},
			{Filename: "notes.md"},
		},
	})
	emptyFilesJSON := mustJSON(t, gistJSON{Files: []gistFile{}})

	tests := []struct {
		name         string
		identifier   string
		results      map[string]mockResult
		wantContent  string
		wantFilename string
		wantErr      bool
		wantErrMsg   string
	}{
		{
			name:       "single file gist",
			identifier: "abc123",
			results: map[string]mockResult{
				"gh gist view abc123 --json files": {stdout: singleFileJSON},
				"gh gist view abc123 -r":           {stdout: []byte("# Budget\n100 + 200 =\n")},
			},
			wantContent:  "# Budget\n100 + 200 =\n",
			wantFilename: "calc.cm",
		},
		{
			name:       "multi-file gist picks .cm file",
			identifier: "def456",
			results: map[string]mockResult{
				"gh gist view def456 --json files":              {stdout: multiFileJSON},
				"gh gist view def456 -r --filename budget.cm":   {stdout: []byte("# Budget\n")},
			},
			wantContent:  "# Budget\n",
			wantFilename: "budget.cm",
		},
		{
			name:       "multi-file gist with no .cm picks first",
			identifier: "ghi789",
			results: map[string]mockResult{
				"gh gist view ghi789 --json files":              {stdout: multiFileNoCM},
				"gh gist view ghi789 -r --filename data.txt":    {stdout: []byte("some data")},
			},
			wantContent:  "some data",
			wantFilename: "data.txt",
		},
		{
			name:       "gist with no files",
			identifier: "empty1",
			results: map[string]mockResult{
				"gh gist view empty1 --json files": {stdout: emptyFilesJSON},
			},
			wantErr:    true,
			wantErrMsg: "no files",
		},
		{
			name:       "inspect fails",
			identifier: "bad1",
			results: map[string]mockResult{
				"gh gist view bad1 --json files": {
					stderr: []byte("Not Found"),
					err:    errors.New("exit status 1"),
				},
			},
			wantErr:    true,
			wantErrMsg: "gist inspect failed",
		},
		{
			name:       "fetch content fails",
			identifier: "fail2",
			results: map[string]mockResult{
				"gh gist view fail2 --json files": {stdout: singleFileJSON},
				"gh gist view fail2 -r":           {stderr: []byte("timeout"), err: errors.New("exit status 1")},
			},
			wantErr:    true,
			wantErrMsg: "gist view failed",
		},
		{
			name:       "content exceeds size limit",
			identifier: "big1",
			results: map[string]mockResult{
				"gh gist view big1 --json files": {stdout: singleFileJSON},
				"gh gist view big1 -r":           {stdout: make([]byte, maxContentSize+1)},
			},
			wantErr:    true,
			wantErrMsg: "exceeds 1MB",
		},
		{
			name:       "invalid JSON from inspect",
			identifier: "badjson",
			results: map[string]mockResult{
				"gh gist view badjson --json files": {stdout: []byte("{invalid json}")},
			},
			wantErr:    true,
			wantErrMsg: "parse gist metadata",
		},
		{
			name:       "full gist URL as identifier",
			identifier: "https://gist.github.com/user/abc123",
			results: map[string]mockResult{
				"gh gist view https://gist.github.com/user/abc123 --json files": {stdout: singleFileJSON},
				"gh gist view https://gist.github.com/user/abc123 -r":           {stdout: []byte("content")},
			},
			wantContent:  "content",
			wantFilename: "calc.cm",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := &mockExecutor{results: tt.results}
			g := NewGistStore(exec)
			result, err := g.Open(tt.identifier)
			if tt.wantErr {
				if err == nil {
					t.Fatal("Open() expected error, got nil")
				}
				if tt.wantErrMsg != "" && !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("Open() error = %q, want to contain %q", err.Error(), tt.wantErrMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("Open() unexpected error: %v", err)
			}
			if result.Content != tt.wantContent {
				t.Errorf("Open() Content = %q, want %q", result.Content, tt.wantContent)
			}
			if result.Filename != tt.wantFilename {
				t.Errorf("Open() Filename = %q, want %q", result.Filename, tt.wantFilename)
			}
		})
	}
}

func TestGistStore_SelectGistFile(t *testing.T) {
	tests := []struct {
		name         string
		files        []gistFile
		wantFilename string
		wantErr      bool
	}{
		{
			name:         "single file",
			files:        []gistFile{{Filename: "main.go"}},
			wantFilename: "main.go",
		},
		{
			name: "multiple files with .cm",
			files: []gistFile{
				{Filename: "readme.md"},
				{Filename: "budget.cm"},
			},
			wantFilename: "budget.cm",
		},
		{
			name: "multiple files first .cm wins",
			files: []gistFile{
				{Filename: "first.cm"},
				{Filename: "second.cm"},
			},
			wantFilename: "first.cm",
		},
		{
			name: "multiple files no .cm falls back to first",
			files: []gistFile{
				{Filename: "data.txt"},
				{Filename: "notes.md"},
			},
			wantFilename: "data.txt",
		},
		{
			name:    "empty file list",
			files:   []gistFile{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBytes := mustJSON(t, gistJSON{Files: tt.files})
			exec := &mockExecutor{
				results: map[string]mockResult{
					"gh gist view test-id --json files": {stdout: jsonBytes},
				},
			}
			g := NewGistStore(exec)
			filename, err := g.selectGistFile("test-id")
			if tt.wantErr {
				if err == nil {
					t.Fatal("selectGistFile() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("selectGistFile() unexpected error: %v", err)
			}
			if filename != tt.wantFilename {
				t.Errorf("selectGistFile() = %q, want %q", filename, tt.wantFilename)
			}
		})
	}
}

// mustJSON marshals v to JSON bytes, failing the test on error.
func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	return b
}
