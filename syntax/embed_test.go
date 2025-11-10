package syntax

import (
	"encoding/json"
	"testing"
)

func TestSyntaxHighlighterSpecEmbedded(t *testing.T) {
	// Verify the spec is not empty
	if SyntaxHighlighterSpec == "" {
		t.Fatal("SyntaxHighlighterSpec is empty - embedding failed")
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(SyntaxHighlighterSpec), &parsed); err != nil {
		t.Fatalf("Embedded spec is not valid JSON: %v", err)
	}

	// Verify required fields
	if version, ok := parsed["version"].(string); !ok || version == "" {
		t.Error("Embedded spec missing version field")
	}

	if language, ok := parsed["language"].(string); !ok || language != "calcmark" {
		t.Error("Embedded spec missing or incorrect language field")
	}
}

func TestSyntaxHighlighterSpecBytes(t *testing.T) {
	bytes := SyntaxHighlighterSpecBytes()

	if len(bytes) == 0 {
		t.Fatal("SyntaxHighlighterSpecBytes() returned empty slice")
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(bytes, &parsed); err != nil {
		t.Fatalf("SyntaxHighlighterSpecBytes() is not valid JSON: %v", err)
	}
}
