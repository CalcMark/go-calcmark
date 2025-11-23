package parser_test

import (
	"testing"
	"time"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/lexer"
	"github.com/CalcMark/go-calcmark/spec/parser"
)

// TestTokenization verifies the lexer produces tokens correctly
func TestTokenization(t *testing.T) {
	input := "42\n"

	lex := lexer.NewLexer(input)
	tokens, err := lex.Tokenize()
	if err != nil {
		t.Fatalf("Tokenization failed: %v", err)
	}

	t.Logf("Input: %q", input)
	t.Logf("Tokens (%d total):", len(tokens))
	for i, tok := range tokens {
		t.Logf("  [%d] Type=%v Value=%q", i, tok.Type, tok.Value)
	}

	// Verify we have EOF
	if len(tokens) == 0 {
		t.Fatal("No tokens produced")
	}

	lastTok := tokens[len(tokens)-1]
	if lastTok.Type != lexer.EOF {
		t.Errorf("Last token should be EOF, got %v", lastTok.Type)
	}
}

// TestParserDebug tests parser with detailed logging
func TestParserDebug(t *testing.T) {
	input := "42\n"

	// First verify tokenization
	lex := lexer.NewLexer(input)
	tokens, err := lex.Tokenize()
	if err != nil {
		t.Fatalf("Tokenization failed: %v", err)
	}

	t.Logf("Tokens:")
	for i, tok := range tokens {
		t.Logf("  [%d] %v = %q", i, tok.Type, tok.Value)
	}

	// Now try parsing with timeout via goroutine
	done := make(chan bool, 1)
	var nodes []ast.Node
	var parseErr error

	go func() {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Parser panicked: %v", r)
			}
			done <- true
		}()

		p := parser.NewRecursiveDescentParser(input)
		nodes, parseErr = p.Parse()
	}()

	// Wait for completion or timeout
	select {
	case <-done:
		if parseErr != nil {
			t.Errorf("Parse error: %v", parseErr)
		} else {
			t.Logf("Success! Nodes: %d", len(nodes))
			for i, n := range nodes {
				t.Logf("  [%d] %v", i, n)
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Parser hung - infinite loop detected")
	}
}
