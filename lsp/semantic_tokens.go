package lsp

import (
	"strings"

	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/CalcMark/go-calcmark/spec/lexer"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Semantic token type indices (order matters — must match legend registration).
const (
	semVariable = iota
	semNumber
	semFunction
	semKeyword
	semOperator
	semComment
	semType // for units
	semString
)

// Semantic token modifier bit flags.
const (
	semModDeclaration = 1 << iota
	semModDocumentation
)

// semanticTokenTypes is the ordered list registered in the legend.
var semanticTokenTypes = []string{
	"variable",
	"number",
	"function",
	"keyword",
	"operator",
	"comment",
	"type",
	"string",
}

// semanticTokenModifiers is the ordered list registered in the legend.
var semanticTokenModifiers = []string{
	"declaration",
	"documentation",
}

// SemanticTokensLegend returns the legend to register during initialize.
func SemanticTokensLegend() protocol.SemanticTokensLegend {
	return protocol.SemanticTokensLegend{
		TokenTypes:     semanticTokenTypes,
		TokenModifiers: semanticTokenModifiers,
	}
}

// textDocumentSemanticTokensFull handles the textDocument/semanticTokens/full request.
func (s *Server) textDocumentSemanticTokensFull(_ *glsp.Context, params *protocol.SemanticTokensParams) (*protocol.SemanticTokens, error) {
	ds := s.getDocument(params.TextDocument.URI)
	if ds == nil {
		return nil, nil
	}

	snap := ds.getSnapshot()
	if snap == nil {
		return nil, nil
	}

	data := encodeSemanticTokens(snap)
	if len(data) == 0 {
		return nil, nil
	}

	return &protocol.SemanticTokens{
		Data: data,
	}, nil
}

// encodeSemanticTokens produces the LSP-encoded semantic token data from a snapshot.
// Uses the document's block structure to distinguish calc vs text lines.
func encodeSemanticTokens(snap *DocumentSnapshot) []protocol.UInteger {
	if snap.Document == nil {
		return nil
	}

	// Build a map of document line number (0-indexed) → block type.
	// CalcBlock lines get lexer-based tokens; TextBlock lines get comment tokens.
	lines := strings.Split(snap.Source, "\n")
	lineTypes := make([]document.BlockType, len(lines))

	// Walk blocks to classify lines.
	docLine := 0 // 0-indexed current line in the full document
	for _, node := range snap.Document.GetBlocks() {
		blockLines := node.Block.Source()
		bt := node.Block.Type()
		for range blockLines {
			if docLine < len(lineTypes) {
				lineTypes[docLine] = bt
			}
			docLine++
		}
	}

	var data []protocol.UInteger
	prevLine := 0
	prevStart := 0

	for lineIdx, lineText := range lines {
		if strings.TrimSpace(lineText) == "" {
			continue
		}

		if lineIdx < len(lineTypes) && lineTypes[lineIdx] == document.BlockText {
			// Markdown/text line → single comment token spanning the line
			runeLen := len([]rune(lineText))
			deltaLine := lineIdx - prevLine
			deltaStart := 0
			if deltaLine == 0 {
				deltaStart = 0 - prevStart
			}
			data = append(data,
				protocol.UInteger(deltaLine),
				protocol.UInteger(deltaStart),
				protocol.UInteger(runeLen),
				protocol.UInteger(semComment),
				protocol.UInteger(semModDocumentation),
			)
			prevLine = lineIdx
			prevStart = 0
			continue
		}

		// Calc line → tokenize with the lexer
		tokens := tokenizeLine(lineText)
		assignMods := classifyAssignmentLHS(tokens)
		for i, tok := range tokens {
			tokenType, tokenMod, ok := mapTokenType(tok)
			if !ok {
				continue
			}
			if extra, has := assignMods[i]; has {
				tokenMod |= extra
			}

			// tok.Column is 1-indexed; convert to 0-indexed rune position
			startChar := max(tok.Column-1, 0)
			length := len([]rune(tok.OriginalText))
			if length == 0 {
				length = len([]rune(tok.Value))
			}
			if length == 0 {
				continue
			}

			deltaLine := lineIdx - prevLine
			deltaStart := startChar
			if deltaLine == 0 {
				deltaStart = startChar - prevStart
			}

			data = append(data,
				protocol.UInteger(deltaLine),
				protocol.UInteger(deltaStart),
				protocol.UInteger(length),
				protocol.UInteger(tokenType),
				protocol.UInteger(tokenMod),
			)
			prevLine = lineIdx
			prevStart = startChar
		}
	}

	return data
}

// tokenizeLine lexes a single line, ignoring errors (best-effort for highlighting).
func tokenizeLine(line string) []lexer.Token {
	l := lexer.NewLexer(line)
	tokens, _ := l.Tokenize()
	// Filter out NEWLINE and EOF
	var result []lexer.Token
	for _, tok := range tokens {
		if tok.Type == lexer.NEWLINE || tok.Type == lexer.EOF {
			continue
		}
		result = append(result, tok)
	}
	return result
}

// mapTokenType maps a lexer token type to a semantic token type index and modifier bitmask.
// Returns (type, modifiers, ok). ok=false means skip this token.
func mapTokenType(tok lexer.Token) (int, int, bool) {
	switch tok.Type {
	// Variable/identifier
	case lexer.IDENTIFIER:
		return semVariable, 0, true

	// Numbers (all numeric forms)
	case lexer.NUMBER, lexer.NUMBER_PERCENT, lexer.NUMBER_K, lexer.NUMBER_M,
		lexer.NUMBER_B, lexer.NUMBER_T, lexer.NUMBER_SCI, lexer.FRACTION,
		lexer.BOOLEAN:
		return semNumber, 0, true

	// Currency
	case lexer.CURRENCY, lexer.CURRENCY_SYM, lexer.CURRENCY_CODE, lexer.QUANTITY:
		return semNumber, 0, true

	// Functions
	case lexer.FUNC_AVG, lexer.FUNC_SQRT, lexer.FUNC_SUM, lexer.FUNC_NUMBER,
		lexer.FUNC_AVERAGE_OF, lexer.FUNC_SQUARE_ROOT_OF, lexer.FUNC_SUM_OF:
		return semFunction, 0, true

	// Keywords
	case lexer.AS, lexer.AT, lexer.FROM, lexer.IN, lexer.OF, lexer.PER,
		lexer.OVER, lexer.WITH, lexer.NAPKIN, lexer.PRECISE, lexer.DOWNTIME,
		lexer.IF, lexer.THEN, lexer.ELSE, lexer.ELIF, lexer.END, lexer.FOR,
		lexer.WHILE, lexer.RETURN, lexer.BREAK, lexer.CONTINUE, lexer.LET, lexer.CONST,
		lexer.AND, lexer.OR, lexer.NOT:
		return semKeyword, 0, true

	// Date keywords
	case lexer.DATE_TODAY, lexer.DATE_TOMORROW, lexer.DATE_YESTERDAY,
		lexer.DATE_THIS_WEEK, lexer.DATE_THIS_MONTH, lexer.DATE_THIS_YEAR,
		lexer.DATE_NEXT_WEEK, lexer.DATE_NEXT_MONTH, lexer.DATE_NEXT_YEAR,
		lexer.DATE_LAST_WEEK, lexer.DATE_LAST_MONTH, lexer.DATE_LAST_YEAR,
		lexer.DATE_LITERAL, lexer.DURATION_LITERAL:
		return semKeyword, 0, true

	// Operators
	case lexer.PLUS, lexer.MINUS, lexer.MULTIPLY, lexer.DIVIDE,
		lexer.MODULUS, lexer.EXPONENT,
		lexer.GREATER_THAN, lexer.LESS_THAN, lexer.GREATER_EQUAL,
		lexer.LESS_EQUAL, lexer.EQUAL, lexer.NOT_EQUAL:
		return semOperator, 0, true

	// Assignment operator — the LHS identifier gets declaration modifier
	// but the = itself is an operator
	case lexer.ASSIGN:
		return semOperator, 0, true

	default:
		return 0, 0, false
	}
}

// classifyAssignmentLHS post-processes tokens to add declaration modifier
// to the variable on the left-hand side of an assignment.
// This is called on a per-line token list.
func classifyAssignmentLHS(tokens []lexer.Token) map[int]int {
	mods := make(map[int]int)
	// Pattern: IDENTIFIER followed by ASSIGN → declaration
	for i := 0; i+1 < len(tokens); i++ {
		if tokens[i].Type == lexer.IDENTIFIER && tokens[i+1].Type == lexer.ASSIGN {
			mods[i] = semModDeclaration
		}
	}
	return mods
}
