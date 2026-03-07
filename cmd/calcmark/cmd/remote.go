package cmd

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/filecheck"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/editor/store"
	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/spf13/cobra"
)

var (
	remoteGist string
	remoteHTTP string
)

var remoteCmd = &cobra.Command{
	Use:   "remote",
	Short: "Open a remote CalcMark document",
	Long: `Fetch a CalcMark document from a remote source and open it in the editor.

Use --gist to open a GitHub Gist (requires gh CLI):
  cm remote --gist abc123def
  cm remote --gist https://gist.github.com/user/abc123

The --gist flag accepts any identifier that gh gist view supports: a gist
ID, a gist URL, or a URL to any user's public gist. For multi-file gists,
the first .cm file is opened (or the first file if no .cm file exists).

Use --http to open a public URL via HTTP GET:
  cm remote --http https://example.com/budget.cm
  cm remote --http https://raw.githubusercontent.com/CalcMark/go-calcmark/refs/heads/main/site/content/docs/examples/napkin-math.md

Exactly one of --gist or --http must be provided.
Remote content is validated before opening (binary files and content
larger than 1MB are rejected).`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRemote()
	},
}

func init() {
	remoteCmd.Flags().StringVar(&remoteGist, "gist", "",
		"GitHub Gist URL or ID — any identifier gh gist view accepts (requires gh CLI)")
	remoteCmd.Flags().StringVar(&remoteHTTP, "http", "",
		"Public HTTP(S) URL to fetch via GET")
	rootCmd.AddCommand(remoteCmd)
}

func runRemote() error {
	hasGist := remoteGist != ""
	hasHTTP := remoteHTTP != ""

	if !hasGist && !hasHTTP {
		return fmt.Errorf("specify --gist or --http")
	}
	if hasGist && hasHTTP {
		return fmt.Errorf("specify only one of --gist or --http")
	}

	var content string
	var err error

	if hasGist {
		content, err = fetchGist(remoteGist)
	} else {
		content, err = fetchURL(remoteHTTP)
	}
	if err != nil {
		return err
	}

	doc, parseErr := document.NewDocument(content)
	if parseErr != nil {
		return fmt.Errorf("parse error: %w", parseErr)
	}
	app := tui.NewEditorApp(doc, "")
	app.SetFormatter(localeFormatter())
	runTUIApp(app)
	return nil
}

// fetchGist retrieves a gist via the gh CLI using the existing GistStore.
// The identifier can be anything gh gist view accepts: a gist ID, a full
// gist URL, or any user's public gist URL. For multi-file gists, GistStore
// selects the first .cm file or falls back to the first file.
func fetchGist(identifier string) (string, error) {
	gist := store.NewGistStore(store.RealExecutor{})

	if err := gist.CheckAvailable(); err != nil {
		return "", fmt.Errorf("gh CLI not available: %w", err)
	}

	if err := gist.CheckAuth(); err != nil {
		return "", fmt.Errorf("not authenticated (run 'gh auth login'): %w", err)
	}

	result, err := gist.Open(identifier)
	if err != nil {
		return "", fmt.Errorf("gist open: %w", err)
	}

	return result.Content, nil
}

// maxRemoteSize is the maximum allowed content size for remote fetches (1MB).
// The +1 byte lets io.ReadAll detect overflow without buffering unlimited data.
const maxRemoteSize = 1*1024*1024 + 1

// fetchURL retrieves content from a public HTTP(S) URL.
// Security: validates scheme, enforces 1MB size limit on the body stream,
// rejects binary content via filecheck.ValidateContent, and validates
// redirect targets stay on HTTP(S).
func fetchURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("unsupported scheme %q (only http and https)", u.Scheme)
	}
	if u.Host == "" {
		return "", fmt.Errorf("URL has no host")
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 15 * time.Second,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return fmt.Errorf("redirect to non-HTTP scheme %q", req.URL.Scheme)
			}
			return nil
		},
	}

	resp, err := client.Get(rawURL)
	if err != nil {
		return "", fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxRemoteSize)))
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}
	if len(data) >= maxRemoteSize {
		return "", fmt.Errorf("response exceeds 1MB limit")
	}

	if err := filecheck.ValidateContent(data); err != nil {
		return "", fmt.Errorf("content validation failed: %w", err)
	}

	if len(data) == 0 {
		return "", fmt.Errorf("remote document is empty")
	}

	return string(data), nil
}
