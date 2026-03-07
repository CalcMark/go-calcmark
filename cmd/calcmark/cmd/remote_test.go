package cmd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchURL(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "x = 42\ny = x * 2\n")
		}))
		defer srv.Close()

		content, err := fetchURL(srv.URL + "/budget.cm")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if content != "x = 42\ny = x * 2\n" {
			t.Errorf("content = %q, want %q", content, "x = 42\ny = x * 2\n")
		}
	})

	t.Run("invalid_scheme", func(t *testing.T) {
		_, err := fetchURL("ftp://example.com/file.cm")
		if err == nil {
			t.Fatal("expected error for ftp scheme")
		}
		if !strings.Contains(err.Error(), "unsupported URL scheme") {
			t.Errorf("error = %q, want 'unsupported URL scheme'", err)
		}
	})

	t.Run("empty_scheme", func(t *testing.T) {
		_, err := fetchURL("example.com/file.cm")
		if err == nil {
			t.Fatal("expected error for missing scheme")
		}
	})

	t.Run("no_host", func(t *testing.T) {
		_, err := fetchURL("http://")
		if err == nil {
			t.Fatal("expected error for empty host")
		}
	})

	t.Run("http_404", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		}))
		defer srv.Close()

		_, err := fetchURL(srv.URL + "/missing.cm")
		if err == nil {
			t.Fatal("expected error for 404")
		}
		if !strings.Contains(err.Error(), "404") {
			t.Errorf("error = %q, want '404'", err)
		}
	})

	t.Run("http_500", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "internal error", http.StatusInternalServerError)
		}))
		defer srv.Close()

		_, err := fetchURL(srv.URL + "/fail.cm")
		if err == nil {
			t.Fatal("expected error for 500")
		}
		if !strings.Contains(err.Error(), "500") {
			t.Errorf("error = %q, want '500'", err)
		}
	})

	t.Run("empty_response", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Write nothing
		}))
		defer srv.Close()

		_, err := fetchURL(srv.URL + "/empty.cm")
		if err == nil {
			t.Fatal("expected error for empty response")
		}
		if !strings.Contains(err.Error(), "empty") {
			t.Errorf("error = %q, want 'empty'", err)
		}
	})

	t.Run("binary_content_rejected", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// PNG magic bytes
			w.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
		}))
		defer srv.Close()

		_, err := fetchURL(srv.URL + "/image.cm")
		if err == nil {
			t.Fatal("expected error for binary content")
		}
		if !strings.Contains(err.Error(), "not contain a valid text document") {
			t.Errorf("error = %q, want 'not contain a valid text document'", err)
		}
	})

	t.Run("size_limit_exceeded", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Write just over 1MB of text
			line := strings.Repeat("a", 1000) + "\n"
			for range 1050 {
				fmt.Fprint(w, line)
			}
		}))
		defer srv.Close()

		_, err := fetchURL(srv.URL + "/huge.cm")
		if err == nil {
			t.Fatal("expected error for oversized response")
		}
		if !strings.Contains(err.Error(), "1MB") {
			t.Errorf("error = %q, want '1MB'", err)
		}
	})
}

func TestRunRemoteFlags(t *testing.T) {
	t.Run("neither_flag", func(t *testing.T) {
		remoteGist = ""
		remoteHTTP = ""
		err := runRemote()
		if err == nil {
			t.Fatal("expected error when no flag specified")
		}
		if !strings.Contains(err.Error(), "specify --gist or --http") {
			t.Errorf("error = %q, want 'specify --gist or --http'", err)
		}
	})

	t.Run("both_flags", func(t *testing.T) {
		remoteGist = "abc123"
		remoteHTTP = "https://example.com/file.cm"
		err := runRemote()
		if err == nil {
			t.Fatal("expected error when both flags specified")
		}
		if !strings.Contains(err.Error(), "only one") {
			t.Errorf("error = %q, want 'only one'", err)
		}
		remoteGist = ""
		remoteHTTP = ""
	})
}
