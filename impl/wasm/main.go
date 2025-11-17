// Package main provides WASM bindings for CalcMark library
//
// Architecture Notes:
// - All functions follow the Go WASM signature: func(js.Value, []js.Value) interface{}
// - Return values use map[string]interface{} to create JS objects with consistent error handling
// - JSON serialization is used to pass complex data structures to JavaScript
// - A global context persists across evaluation calls to maintain variable state
package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/CalcMark/go-calcmark/spec/classifier"
	"github.com/CalcMark/go-calcmark/impl/evaluator"
	"github.com/CalcMark/go-calcmark/spec/lexer"
	"github.com/CalcMark/go-calcmark/spec/parser"
	"github.com/CalcMark/go-calcmark/spec/validator"
)

// ==============================================================================
// Global State
// ==============================================================================

// globalContext maintains variable bindings across evaluation calls.
// This allows applications to evaluate calculations line-by-line while preserving
// previous variable assignments. Use resetContext() to clear this state.
var globalContext = evaluator.NewContext()

// ==============================================================================
// Type Definitions for JavaScript Interop
// ==============================================================================

// TokenInfo represents a token with position information for JavaScript.
// Position fields use byte offsets compatible with JavaScript string indexing.
type TokenInfo struct {
	Type         string `json:"type"`         // Token type as string (e.g., "NUMBER", "IDENTIFIER")
	Value        string `json:"value"`        // Parsed/normalized value
	OriginalText string `json:"originalText"` // Exact text from source
	Start        int    `json:"start"`        // Byte offset of token start
	End          int    `json:"end"`          // Byte offset of token end (exclusive)
	Line         int    `json:"line"`         // 1-indexed line number
}

// ClassificationResult represents line classification with context.
// Used by classifyLines to provide both the classification and the original line.
type ClassificationResult struct {
	LineType string `json:"lineType"` // "CALCULATION", "MARKDOWN", or "BLANK"
	Line     string `json:"line"`     // The original line text
	Index    int    `json:"index"`    // 0-indexed position in input array
}

// ==============================================================================
// Error Handling Helpers
// ==============================================================================

// errorResponse creates a standardized error response for JavaScript.
// All WASM functions return this format when validation or processing fails.
func errorResponse(errorMsg string, fields ...string) map[string]interface{} {
	result := map[string]interface{}{"error": errorMsg}
	for _, field := range fields {
		result[field] = nil
	}
	return result
}

// successResponse creates a standardized success response with JSON-serialized data.
// The 'field' parameter becomes the key in the returned JS object.
func successResponse(field string, data interface{}) map[string]interface{} {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return errorResponse(err.Error(), field)
	}
	return map[string]interface{}{
		field:   string(jsonBytes),
		"error": nil,
	}
}

// ==============================================================================
// WASM Function: tokenize
// ==============================================================================

// tokenize exposes lexer.Tokenize to JavaScript.
//
// Why this exists: JavaScript needs token positions for syntax highlighting.
// The lexer provides byte-offset positions that match JS string indexing.
//
// Usage: calcmark.tokenize(sourceCode: string)
// Returns: {tokens: string (JSON array), error: string|null}
func tokenize(this js.Value, args []js.Value) interface{} {
	if len(args) != 1 {
		return errorResponse("Expected 1 argument: sourceCode (string)", "tokens")
	}

	source := args[0].String()
	tokens, err := lexer.Tokenize(source)
	if err != nil {
		return errorResponse(err.Error(), "tokens")
	}

	// Convert internal Token type to TokenInfo for JavaScript.
	// Why: Go's lexer.Token contains internal types (TokenType enum) that don't
	// serialize cleanly to JSON. TokenInfo provides clean string-based types.
	tokenInfos := make([]TokenInfo, 0, len(tokens))
	for _, token := range tokens {
		tokenInfos = append(tokenInfos, TokenInfo{
			Type:         token.Type.String(), // Convert enum to string for JS
			Value:        token.Value,
			OriginalText: token.OriginalText,
			Start:        token.StartPos, // Byte offsets match JS string indexing
			End:          token.EndPos,
			Line:         token.Line,
		})
	}

	return successResponse("tokens", tokenInfos)
}

// ==============================================================================
// WASM Function: parse
// ==============================================================================

// parse exposes parser.Parse to JavaScript.
//
// Why this exists: Provides AST access for tools that need to analyze code structure
// beyond just tokenization (e.g., IDE features, code analysis tools).
//
// Usage: calcmark.parse(sourceCode: string)
// Returns: {ast: string (JSON), error: string|null}
func parse(this js.Value, args []js.Value) interface{} {
	if len(args) != 1 {
		return errorResponse("Expected 1 argument: sourceCode (string)", "ast")
	}

	source := args[0].String()
	nodes, err := parser.Parse(source)
	if err != nil {
		return errorResponse(err.Error(), "ast")
	}

	return successResponse("ast", nodes)
}

// ==============================================================================
// WASM Function: evaluate
// ==============================================================================

// evaluate exposes evaluator.Evaluate to JavaScript.
//
// Why context management matters: CalcMark evaluates line-by-line. To preserve
// variable definitions across calls (e.g., line 1: "x = 5", line 2: "y = x + 1"),
// we maintain a global context. Fresh contexts are used for isolated evaluation.
//
// Usage: calcmark.evaluate(sourceCode: string, useGlobalContext?: boolean)
// Returns: {results: string (JSON array), error: string|null}
func evaluate(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return errorResponse("Expected at least 1 argument: sourceCode (string)", "results")
	}

	source := args[0].String()

	// Default to using global context for stateful evaluation
	useGlobalCtx := true
	if len(args) > 1 {
		useGlobalCtx = args[1].Bool()
	}

	// Choose context: global for persistent state, fresh for isolation
	ctx := evaluator.NewContext()
	if useGlobalCtx {
		ctx = globalContext
	}

	results, err := evaluator.Evaluate(source, ctx)
	if err != nil {
		return errorResponse(err.Error(), "results")
	}

	return successResponse("results", results)
}

// ==============================================================================
// WASM Function: validate
// ==============================================================================

// validate exposes validator.ValidateDocument to JavaScript.
//
// Critical design decision: Always uses a FRESH context, never globalContext.
// Why: Validation should detect undefined variables in the document being validated,
// not rely on variables defined in previous unrelated evaluations.
//
// Special handling: Uses Diagnostic.ToMap() to convert Go enums (severity, code)
// to human-readable strings. Standard json.Marshal would serialize enums as integers,
// making tooltips show "2" instead of "Hint".
//
// Usage: calcmark.validate(sourceCode: string)
// Returns: {diagnostics: string (JSON object), error: string|null}
//
// Response structure:
// {
//   "diagnostics": {
//     "3": {  // Line number (1-indexed)
//       "Diagnostics": [
//         {
//           "severity": "error",     // Not 0, 1, 2 - human readable!
//           "code": "undefined_variable",
//           "message": "Undefined variable: foo",
//           "range": { "start": {...}, "end": {...} }
//         }
//       ]
//     }
//   },
//   "error": null
// }
func validate(this js.Value, args []js.Value) interface{} {
	if len(args) != 1 {
		return errorResponse("Expected 1 argument: sourceCode (string)", "diagnostics")
	}

	source := args[0].String()

	// CRITICAL: Use fresh context to detect undefined variables in THIS document.
	// Using globalContext would incorrectly mark variables as "defined" if they
	// were set in a previous, unrelated evaluation.
	freshContext := evaluator.NewContext()
	result := validator.ValidateDocument(source, freshContext)

	// Transform diagnostics to use string enums instead of integer constants.
	// Why: Go's json.Marshal serializes enums as integers by default.
	// Diagnostic.ToMap() calls .String() methods to get human-readable values.
	resultMap := make(map[int]interface{})
	for lineNum, lineResult := range result {
		diagnosticsArray := make([]map[string]interface{}, 0, len(lineResult.Diagnostics))
		for _, diag := range lineResult.Diagnostics {
			// ToMap converts: Severity (int) -> "error"/"warning"/"hint" (string)
			diagnosticsArray = append(diagnosticsArray, diag.ToMap())
		}
		resultMap[lineNum] = map[string]interface{}{
			"Diagnostics": diagnosticsArray,
		}
	}

	return successResponse("diagnostics", resultMap)
}

// ==============================================================================
// WASM Function: classifyLine
// ==============================================================================

// classifyLine exposes classifier.ClassifyLine to JavaScript.
//
// Why context matters: Classification depends on variable definitions.
// Example: "total" is CALCULATION if 'total' is defined, MARKDOWN otherwise.
// Uses globalContext to check current variable state.
//
// Usage: calcmark.classifyLine(line: string)
// Returns: {lineType: string, error: string|null}
func classifyLine(this js.Value, args []js.Value) interface{} {
	if len(args) != 1 {
		return errorResponse("Expected 1 argument: line (string)", "lineType")
	}

	line := args[0].String()
	lineType := classifier.ClassifyLine(line, globalContext)

	return map[string]interface{}{
		"lineType": lineType.String(),
		"error":    nil,
	}
}

// ==============================================================================
// WASM Function: classifyLines
// ==============================================================================

// classifyLines classifies multiple lines with progressive context tracking.
//
// Why this exists: Efficiently classifies entire documents while maintaining
// context state. Each CALCULATION line updates the context for subsequent lines.
//
// Example:
//   Line 1: "x = 5"        -> CALCULATION (x now defined)
//   Line 2: "x"            -> CALCULATION (x is defined from line 1)
//   Line 3: "y"            -> MARKDOWN (y is undefined)
//
// Critical: Uses a FRESH context, not globalContext, so each document is
// classified independently without pollution from previous calls.
//
// Usage: calcmark.classifyLines(lines: string[])
// Returns: {classifications: string (JSON array), error: string|null}
func classifyLines(this js.Value, args []js.Value) interface{} {
	if len(args) != 1 {
		return errorResponse("Expected 1 argument: lines (array of strings)", "classifications")
	}

	jsArray := args[0]
	length := jsArray.Length()
	results := make([]ClassificationResult, 0, length)

	// Use fresh context: each document classification is independent
	ctx := evaluator.NewContext()

	for i := 0; i < length; i++ {
		line := jsArray.Index(i).String()
		lineType := classifier.ClassifyLine(line, ctx)

		results = append(results, ClassificationResult{
			LineType: lineType.String(),
			Line:     line,
			Index:    i,
		})

		// Update context if calculation: makes subsequent line classification context-aware
		if lineType == classifier.Calculation {
			// Evaluate to update context (ignore errors - classification shouldn't fail)
			_, _ = evaluator.Evaluate(line, ctx)
		}
	}

	return successResponse("classifications", results)
}

// ==============================================================================
// WASM Function: evaluateDocument
// ==============================================================================

// EvaluationResultWithLine extends evaluation result with line number tracking.
// Used by evaluateDocument to map results back to their original line numbers.
type EvaluationResultWithLine struct {
	Value        interface{} `json:"Value"`        // The computed value
	Symbol       string      `json:"Symbol"`       // Currency symbol if applicable
	SourceFormat string      `json:"SourceFormat"` // Original formatting
	OriginalLine int         `json:"OriginalLine"` // 1-indexed line number in source document
}

// evaluateDocument exposes evaluator.EvaluateDocument to JavaScript.
//
// Why this exists: The evaluate() function expects pure calculation input and fails
// on markdown content (# headers, text, etc.). This function properly handles mixed
// documents by using the evaluator.EvaluateDocument function which classifies lines
// first, then evaluates only CALCULATION lines.
//
// This is a thin wrapper around evaluator.EvaluateDocument - see that function for
// implementation details.
//
// Context behavior:
//   - If useGlobalContext=true, uses globalContext (persistent across calls)
//   - If useGlobalContext=false, uses fresh context (isolated evaluation)
//
// Usage: calcmark.evaluateDocument(sourceCode: string, useGlobalContext?: boolean)
// Returns: {results: string (JSON array of EvaluationResultWithLine), error: string|null}
//
// Example:
//   Input:
//     # Title
//     x = 5
//     Some text
//     y = x + 10
//
//   Output:
//     [
//       {Value: 5, OriginalLine: 2},
//       {Value: 15, OriginalLine: 4}
//     ]
func evaluateDocument(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return errorResponse("Expected at least 1 argument: sourceCode (string)", "results")
	}

	source := args[0].String()

	// Default to using global context for stateful evaluation
	useGlobalCtx := true
	if len(args) > 1 {
		useGlobalCtx = args[1].Bool()
	}

	// Choose context: global for persistent state, fresh for isolation
	ctx := evaluator.NewContext()
	if useGlobalCtx {
		ctx = globalContext
	}

	// Call the evaluator package function
	evalResults, err := evaluator.EvaluateDocument(source, ctx)
	if err != nil {
		return errorResponse(err.Error(), "results")
	}

	// Convert evaluator.EvaluationResult to EvaluationResultWithLine for JavaScript
	results := make([]EvaluationResultWithLine, 0, len(evalResults))

	for _, evalResult := range evalResults {
		resultWithLine := EvaluationResultWithLine{
			OriginalLine: evalResult.OriginalLine,
		}

		// Extract fields from types.Type interface
		// The evaluator returns types.Number, types.Currency, or types.Boolean
		switch v := evalResult.Value.(type) {
		case interface{ GetValue() interface{} }:
			resultWithLine.Value = v.GetValue()
		default:
			// Fallback for types that don't implement GetValue
			resultWithLine.Value = v
		}

		// Check for Symbol (currency types)
		if symbolType, ok := evalResult.Value.(interface{ GetSymbol() string }); ok {
			resultWithLine.Symbol = symbolType.GetSymbol()
		}

		// Check for SourceFormat
		if sourceType, ok := evalResult.Value.(interface{ GetSourceFormat() string }); ok {
			resultWithLine.SourceFormat = sourceType.GetSourceFormat()
		}

		results = append(results, resultWithLine)
	}

	return successResponse("results", results)
}

// ==============================================================================
// WASM Function: resetContext
// ==============================================================================

// resetContext clears the global evaluation context.
//
// Why this exists: Allows users to start fresh without reloading the page.
// Example use case: User wants to clear all variable definitions and start over.
//
// Usage: calcmark.resetContext()
// Returns: void
func resetContext(this js.Value, args []js.Value) interface{} {
	globalContext = evaluator.NewContext()
	return nil
}

// ==============================================================================
// WASM Function: getVersion
// ==============================================================================

// getVersion returns the CalcMark library version.
//
// Why this exists: Allows JavaScript to display version info and detect
// compatibility issues if API changes in future versions.
//
// Usage: calcmark.getVersion()
// Returns: string (version number)
func getVersion(this js.Value, args []js.Value) interface{} {
	return "0.1.1"
}

// ==============================================================================
// Main Entry Point
// ==============================================================================

func main() {
	// WASM programs must block forever - if main() exits, the module unloads.
	// We use an unbuffered channel that never receives to keep the program alive.
	done := make(chan struct{})

	// Register all functions on window.calcmark object
	js.Global().Set("calcmark", map[string]interface{}{
		"tokenize":         js.FuncOf(tokenize),
		"parse":            js.FuncOf(parse),
		"evaluate":         js.FuncOf(evaluate),
		"evaluateDocument": js.FuncOf(evaluateDocument),
		"validate":         js.FuncOf(validate),
		"classifyLine":     js.FuncOf(classifyLine),
		"classifyLines":    js.FuncOf(classifyLines),
		"resetContext":     js.FuncOf(resetContext),
		"getVersion":       js.FuncOf(getVersion),
	})

	// Block forever to keep WASM module loaded
	<-done
}
