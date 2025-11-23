package lexer

// tokenizeHelper is a test helper function that wraps the lexer API for easier testing.
// It creates a new lexer and calls Tokenize(), returning the tokens.
func tokenizeHelper(input string) ([]Token, error) {
	lex := NewLexer(input)
	return lex.Tokenize()
}
