package lsp

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/lexer"
)

// --- Token type mapping tests ---

func TestMapTokenType_Identifiers(t *testing.T) {
	tok := lexer.Token{Type: lexer.IDENTIFIER, Value: "price"}
	tokenType, _, ok := mapTokenType(tok)
	if !ok {
		t.Fatal("expected identifier to be mapped")
	}
	if tokenType != semVariable {
		t.Errorf("expected semVariable (%d), got %d", semVariable, tokenType)
	}
}

func TestMapTokenType_Numbers(t *testing.T) {
	numTypes := []lexer.TokenType{
		lexer.NUMBER, lexer.NUMBER_PERCENT, lexer.NUMBER_K,
		lexer.NUMBER_M, lexer.NUMBER_B, lexer.NUMBER_T,
		lexer.NUMBER_SCI, lexer.FRACTION, lexer.BOOLEAN,
		lexer.CURRENCY, lexer.CURRENCY_SYM, lexer.CURRENCY_CODE, lexer.QUANTITY,
	}

	for _, tt := range numTypes {
		tok := lexer.Token{Type: tt, Value: "42"}
		tokenType, _, ok := mapTokenType(tok)
		if !ok {
			t.Errorf("expected %s to be mapped", tt)
			continue
		}
		if tokenType != semNumber {
			t.Errorf("expected semNumber (%d) for %s, got %d", semNumber, tt, tokenType)
		}
	}
}

func TestMapTokenType_Functions(t *testing.T) {
	funcTypes := []lexer.TokenType{
		lexer.FUNC_AVG, lexer.FUNC_SQRT, lexer.FUNC_SUM,
		lexer.FUNC_AVERAGE_OF, lexer.FUNC_SQUARE_ROOT_OF, lexer.FUNC_SUM_OF,
	}

	for _, tt := range funcTypes {
		tok := lexer.Token{Type: tt, Value: "avg"}
		tokenType, _, ok := mapTokenType(tok)
		if !ok {
			t.Errorf("expected %s to be mapped", tt)
			continue
		}
		if tokenType != semFunction {
			t.Errorf("expected semFunction (%d) for %s, got %d", semFunction, tt, tokenType)
		}
	}
}

func TestMapTokenType_Keywords(t *testing.T) {
	kwTypes := []lexer.TokenType{
		lexer.AS, lexer.IN, lexer.OF, lexer.PER,
		lexer.AND, lexer.OR, lexer.NOT,
		lexer.DATE_TODAY, lexer.DATE_TOMORROW,
	}

	for _, tt := range kwTypes {
		tok := lexer.Token{Type: tt, Value: "as"}
		tokenType, _, ok := mapTokenType(tok)
		if !ok {
			t.Errorf("expected %s to be mapped", tt)
			continue
		}
		if tokenType != semKeyword {
			t.Errorf("expected semKeyword (%d) for %s, got %d", semKeyword, tt, tokenType)
		}
	}
}

func TestMapTokenType_Operators(t *testing.T) {
	opTypes := []lexer.TokenType{
		lexer.PLUS, lexer.MINUS, lexer.MULTIPLY, lexer.DIVIDE,
		lexer.ASSIGN, lexer.GREATER_THAN, lexer.EQUAL,
	}

	for _, tt := range opTypes {
		tok := lexer.Token{Type: tt, Value: "+"}
		tokenType, _, ok := mapTokenType(tok)
		if !ok {
			t.Errorf("expected %s to be mapped", tt)
			continue
		}
		if tokenType != semOperator {
			t.Errorf("expected semOperator (%d) for %s, got %d", semOperator, tt, tokenType)
		}
	}
}

func TestMapTokenType_SkipsUnknown(t *testing.T) {
	// Grouping tokens (parens, commas) should be skipped
	tok := lexer.Token{Type: lexer.LPAREN, Value: "("}
	_, _, ok := mapTokenType(tok)
	if ok {
		t.Error("expected LPAREN to be skipped")
	}
}

// --- Assignment LHS classification ---

func TestClassifyAssignmentLHS(t *testing.T) {
	tokens := []lexer.Token{
		{Type: lexer.IDENTIFIER, Value: "price"},
		{Type: lexer.ASSIGN, Value: "="},
		{Type: lexer.NUMBER, Value: "100"},
	}

	mods := classifyAssignmentLHS(tokens)
	if mods[0] != semModDeclaration {
		t.Errorf("expected declaration modifier on LHS identifier, got %d", mods[0])
	}
	if _, has := mods[1]; has {
		t.Error("did not expect modifier on ASSIGN token")
	}
	if _, has := mods[2]; has {
		t.Error("did not expect modifier on NUMBER token")
	}
}

func TestClassifyAssignmentLHS_NoAssignment(t *testing.T) {
	tokens := []lexer.Token{
		{Type: lexer.IDENTIFIER, Value: "price"},
		{Type: lexer.PLUS, Value: "+"},
		{Type: lexer.NUMBER, Value: "1"},
	}

	mods := classifyAssignmentLHS(tokens)
	if len(mods) != 0 {
		t.Errorf("expected no modifiers for non-assignment, got %v", mods)
	}
}

// --- Tokenize line tests ---

func TestTokenizeLine(t *testing.T) {
	tokens := tokenizeLine("price = 100")
	if len(tokens) == 0 {
		t.Fatal("expected tokens from calc line")
	}

	// Should not contain NEWLINE or EOF
	for _, tok := range tokens {
		if tok.Type == lexer.NEWLINE || tok.Type == lexer.EOF {
			t.Errorf("unexpected %s token in result", tok.Type)
		}
	}
}

func TestTokenizeLine_Empty(t *testing.T) {
	tokens := tokenizeLine("")
	if len(tokens) != 0 {
		t.Errorf("expected no tokens for empty line, got %d", len(tokens))
	}
}

// --- Semantic token encoding integration test ---

func TestEncodeSemanticTokens_SimpleAssignment(t *testing.T) {
	s := NewServer()
	snap := s.evaluate("price = 100")

	data := encodeSemanticTokens(snap)
	if len(data) == 0 {
		t.Fatal("expected semantic token data for assignment")
	}

	// Data is encoded as groups of 5: deltaLine, deltaStart, length, tokenType, tokenModifiers
	if len(data)%5 != 0 {
		t.Fatalf("data length %d is not divisible by 5", len(data))
	}

	// First token should be "price" (identifier with declaration modifier)
	// deltaLine=0, deltaStart=0, length=5, type=variable, mods=declaration
	if data[0] != 0 { // deltaLine
		t.Errorf("first token deltaLine = %d, want 0", data[0])
	}
	if data[2] != 5 { // length of "price"
		t.Errorf("first token length = %d, want 5", data[2])
	}
	if data[3] != uint32(semVariable) {
		t.Errorf("first token type = %d, want semVariable (%d)", data[3], semVariable)
	}
	if data[4] != uint32(semModDeclaration) {
		t.Errorf("first token modifiers = %d, want semModDeclaration (%d)", data[4], semModDeclaration)
	}
}

func TestEncodeSemanticTokens_MarkdownLine(t *testing.T) {
	s := NewServer()
	snap := s.evaluate("# Total Cost\nprice = 100")

	data := encodeSemanticTokens(snap)
	if len(data) == 0 {
		t.Fatal("expected semantic token data")
	}

	// First token should be the markdown heading (comment with documentation modifier)
	if data[3] != uint32(semComment) {
		t.Errorf("markdown line token type = %d, want semComment (%d)", data[3], semComment)
	}
	if data[4] != uint32(semModDocumentation) {
		t.Errorf("markdown line token modifiers = %d, want semModDocumentation (%d)", data[4], semModDocumentation)
	}
}

func TestEncodeSemanticTokens_MultiLine(t *testing.T) {
	s := NewServer()
	snap := s.evaluate("a = 1\nb = 2")

	data := encodeSemanticTokens(snap)
	if len(data) < 10 { // At least 2 tokens
		t.Fatalf("expected tokens for two lines, got %d values", len(data))
	}

	// Verify delta encoding: second line's tokens should have deltaLine > 0
	// Find the first token on line 1 (deltaLine will be 1 from line 0)
	foundLine1 := false
	for i := 0; i < len(data); i += 5 {
		if data[i] > 0 { // deltaLine > 0 means we moved to next line
			foundLine1 = true
			break
		}
	}
	if !foundLine1 {
		t.Error("expected tokens on multiple lines with deltaLine > 0")
	}
}

func TestEncodeSemanticTokens_NilDocument(t *testing.T) {
	snap := &DocumentSnapshot{Source: "bad input"}
	data := encodeSemanticTokens(snap)
	if len(data) != 0 {
		t.Errorf("expected no data for nil document, got %d values", len(data))
	}
}

// --- Legend tests ---

func TestSemanticTokensLegend(t *testing.T) {
	legend := SemanticTokensLegend()
	if len(legend.TokenTypes) == 0 {
		t.Fatal("expected token types in legend")
	}
	if len(legend.TokenModifiers) == 0 {
		t.Fatal("expected token modifiers in legend")
	}

	// Verify the indices match our constants
	if legend.TokenTypes[semVariable] != "variable" {
		t.Errorf("token type at semVariable = %q, want 'variable'", legend.TokenTypes[semVariable])
	}
	if legend.TokenTypes[semFunction] != "function" {
		t.Errorf("token type at semFunction = %q, want 'function'", legend.TokenTypes[semFunction])
	}
	if legend.TokenTypes[semComment] != "comment" {
		t.Errorf("token type at semComment = %q, want 'comment'", legend.TokenTypes[semComment])
	}
	if legend.TokenModifiers[0] != "declaration" {
		t.Errorf("first modifier = %q, want 'declaration'", legend.TokenModifiers[0])
	}
}
