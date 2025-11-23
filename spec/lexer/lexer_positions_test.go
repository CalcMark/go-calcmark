package lexer

import (
	"testing"
)

func TestTokenPositions4323(t *testing.T) {
	input := "average of 3, 4,323, 1003"
	lex := NewLexer(input)
	tokens, err := lex.Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Logf("Input: %q", input)
	t.Logf("Length: %d", len(input))
	t.Log("\nToken positions:")

	for i, token := range tokens {
		if token.Type == EOF {
			continue
		}

		// Extract what the position range actually contains in the source
		sourceText := ""
		if token.StartPos < len(input) && token.EndPos <= len(input) {
			sourceText = input[token.StartPos:token.EndPos]
		}

		t.Logf("Token %d: Type=%s, Value=%q, OriginalText=%q, Start=%d, End=%d, SourceRange=%q",
			i, token.Type, token.Value, token.OriginalText,
			token.StartPos, token.EndPos, sourceText)
	}

	// Visual map
	t.Log("\nVisual position map:")
	t.Logf("0123456789012345678901234567890")
	t.Logf("%s", input)

	// Show each token's range
	for i, token := range tokens {
		if token.Type == EOF {
			continue
		}

		visual := make([]byte, len(input))
		for j := range visual {
			visual[j] = ' '
		}

		for j := token.StartPos; j < token.EndPos && j < len(input); j++ {
			visual[j] = '^'
		}

		t.Logf("Token %d (%s): %s", i, token.Type, string(visual))
	}
}

func TestTokenPositionsSimple4323(t *testing.T) {
	input := "4,323"
	lex := NewLexer(input)
	tokens, err := lex.Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Logf("Input: %q", input)
	for i, token := range tokens {
		if token.Type == EOF {
			continue
		}

		sourceText := ""
		if token.StartPos < len(input) && token.EndPos <= len(input) {
			sourceText = input[token.StartPos:token.EndPos]
		}

		t.Logf("Token %d: Type=%s, Value=%q, OriginalText=%q, Start=%d, End=%d, SourceRange=%q",
			i, token.Type, token.Value, token.OriginalText,
			token.StartPos, token.EndPos, sourceText)
	}
}

func TestRenderSimulation(t *testing.T) {
	input := "average of 3, 4,323, 1003"
	lex := NewLexer(input)
	tokens, err := lex.Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Logf("Input: %q\n", input)

	var rendered string
	currentPos := 0

	for _, token := range tokens {
		if token.Type == EOF {
			continue
		}

		// Add any text before this token (whitespace, etc)
		if token.StartPos > currentPos {
			gap := input[currentPos:token.StartPos]
			rendered += gap
			t.Logf("Added gap %d-%d: %q", currentPos, token.StartPos, gap)
		}

		// Add the token's source text
		sourceText := input[token.StartPos:token.EndPos]
		rendered += sourceText
		t.Logf("Added token %s at %d-%d: %q", token.Type, token.StartPos, token.EndPos, sourceText)

		currentPos = token.EndPos
	}

	// Add any remaining text
	if currentPos < len(input) {
		remaining := input[currentPos:]
		rendered += remaining
		t.Logf("Added remaining %d-%d: %q", currentPos, len(input), remaining)
	}

	t.Logf("\nOriginal: %q", input)
	t.Logf("Rendered: %q", rendered)

	if rendered != input {
		t.Errorf("Rendered output doesn't match input!")
		t.Errorf("Expected: %q", input)
		t.Errorf("Got:      %q", rendered)

		// Character-by-character comparison
		minLen := min(len(rendered), len(input))

		for i := range minLen {
			if input[i] != rendered[i] {
				t.Errorf("First difference at position %d: expected %q, got %q",
					i, string(input[i]), string(rendered[i]))
				break
			}
		}

		if len(input) != len(rendered) {
			t.Errorf("Length mismatch: expected %d, got %d", len(input), len(rendered))
		}
	}
}

func TestMultilinePositions(t *testing.T) {
	input := "x = 5\ny = 10"
	lex := NewLexer(input)
	tokens, err := lex.Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Logf("Input: %q", input)
	t.Logf("Input bytes: %v", []byte(input))

	for i, token := range tokens {
		if token.Type == EOF {
			continue
		}

		sourceText := ""
		if token.StartPos < len(input) && token.EndPos <= len(input) {
			sourceText = input[token.StartPos:token.EndPos]
		}

		t.Logf("Token %d: Type=%s, Value=%q, Start=%d, End=%d, SourceRange=%q",
			i, token.Type, token.Value, token.StartPos, token.EndPos, sourceText)
	}

	// Test render simulation with newlines
	var rendered string
	currentPos := 0

	for _, token := range tokens {
		if token.Type == EOF {
			continue
		}

		// Add any text before this token
		if token.StartPos > currentPos {
			gap := input[currentPos:token.StartPos]
			rendered += gap
		}

		// Add the token's source text
		sourceText := input[token.StartPos:token.EndPos]
		rendered += sourceText

		currentPos = token.EndPos
	}

	// Add any remaining text
	if currentPos < len(input) {
		rendered += input[currentPos:]
	}

	if rendered != input {
		t.Errorf("Multiline render failed!")
		t.Errorf("Expected: %q", input)
		t.Errorf("Got:      %q", rendered)
	}
}
