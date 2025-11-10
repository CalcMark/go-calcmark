package syntax_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CalcMark/go-calcmark/syntax"
)

// This example demonstrates how the CalcMark server can serve the embedded spec
func TestExampleHTTPEndpoint(t *testing.T) {
	// Create a handler that serves the embedded spec
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(syntax.SyntaxHighlighterSpecBytes())
	})

	// Create a test request
	req := httptest.NewRequest("GET", "/syntax", nil)
	rec := httptest.NewRecorder()

	// Serve the request
	handler.ServeHTTP(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	if contentType := rec.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Verify the body contains the spec
	body := rec.Body.String()
	if len(body) == 0 {
		t.Error("Response body is empty")
	}

	// Verify it contains expected CalcMark content
	if !contains(body, "calcmark") {
		t.Error("Response does not contain 'calcmark'")
	}

	if !contains(body, "version") {
		t.Error("Response does not contain 'version'")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
