package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/CalcMark/go-calcmark/syntax"
)

const version = "0.1.1"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "spec":
		handleSpec()
	case "generate":
		handleGenerate()
	case "version", "-v", "--version":
		fmt.Printf("calcmark version %s\n", version)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func handleSpec() {
	fmt.Print(syntax.SyntaxHighlighterSpec)
}

func handleGenerate() {
	spec := buildSpec()

	// Marshal with indentation for readability
	jsonData, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	// Write to both spec/ (documentation) and syntax/ (for embedding)
	paths := []string{
		"spec/SYNTAX_HIGHLIGHTER_SPEC.json",
		"syntax/SYNTAX_HIGHLIGHTER_SPEC.json",
	}

	for _, outputPath := range paths {
		if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to %s: %v\n", outputPath, err)
			os.Exit(1)
		}
		fmt.Printf("✓ Generated %s\n", outputPath)
	}

	fmt.Printf("Version: %s\n", version)
}

func buildSpec() map[string]interface{} {
	return map[string]interface{}{
		"$comment":       "IMPORTANT: This file is a first-class deliverable that MUST stay synchronized with the Go implementation. It is validated by tests in lexer/syntax_spec_test.go on every test run. This file can be served via HTTP endpoint (/syntax) or bundled in TypeScript clients for syntax highlighting.",
		"version":        version,
		"description":    "CalcMark Language Syntax Specification for Syntax Highlighters",
		"language":       "calcmark",
		"fileExtensions": []string{".cm", ".calcmark"},

		"tokens": map[string]interface{}{
			"keywords": map[string]interface{}{
				"description": "Reserved keywords that cannot be used as identifiers",
				"logicalOperators": map[string]interface{}{
					"description":     "Logical operators (Go spec compliant)",
					"caseInsensitive": true,
					"tokens":          []string{"and", "or", "not"},
				},
				"controlFlow": map[string]interface{}{
					"description":     "Reserved for future control flow (not yet implemented)",
					"caseInsensitive": true,
					"tokens":          []string{"if", "then", "else", "elif", "end", "for", "in", "while", "return", "break", "continue", "let", "const"},
				},
				"functions": map[string]interface{}{
					"description":     "Built-in function names (canonical forms)",
					"caseInsensitive": true,
					"canonical": map[string]interface{}{
						"avg": map[string]interface{}{
							"aliases":     []string{"average of"},
							"description": "Calculate average of numbers",
							"syntax":      []string{"avg(x, y, ...)", "average of x, y, ..."},
						},
						"sqrt": map[string]interface{}{
							"aliases":     []string{"square root of"},
							"description": "Calculate square root",
							"syntax":      []string{"sqrt(x)", "square root of x"},
						},
					},
				},
				"multiTokenFunctions": map[string]interface{}{
					"description": "Multi-word function names that are combined during tokenization",
					"tokens": []map[string]interface{}{
						{
							"pattern":            "average of",
							"canonical":          "avg",
							"requiresWhitespace": true,
							"caseInsensitive":    true,
						},
						{
							"pattern":            "square root of",
							"canonical":          "sqrt",
							"requiresWhitespace": true,
							"caseInsensitive":    true,
						},
					},
				},
			},

			"operators": map[string]interface{}{
				"arithmetic": map[string]interface{}{
					"description": "Arithmetic operators",
					"tokens": []map[string]interface{}{
						{"symbol": "+", "name": "plus", "description": "Addition"},
						{"symbol": "-", "name": "minus", "description": "Subtraction or unary negation"},
						{"symbol": "*", "name": "multiply", "description": "Multiplication", "aliases": []string{"×", "x", "X"}},
						{"symbol": "/", "name": "divide", "description": "Division"},
						{"symbol": "%", "name": "modulus", "description": "Modulus/remainder"},
						{"symbol": "^", "name": "exponent", "description": "Exponentiation", "aliases": []string{"**"}},
					},
				},
				"comparison": map[string]interface{}{
					"description": "Comparison operators",
					"tokens": []map[string]interface{}{
						{"symbol": ">", "name": "greater_than", "description": "Greater than"},
						{"symbol": "<", "name": "less_than", "description": "Less than"},
						{"symbol": ">=", "name": "greater_equal", "description": "Greater than or equal"},
						{"symbol": "<=", "name": "less_equal", "description": "Less than or equal"},
						{"symbol": "==", "name": "equal", "description": "Equal"},
						{"symbol": "!=", "name": "not_equal", "description": "Not equal"},
					},
				},
				"assignment": map[string]interface{}{
					"description": "Assignment operator",
					"tokens": []map[string]interface{}{
						{"symbol": "=", "name": "assign", "description": "Variable assignment"},
					},
				},
			},

			"literals": map[string]interface{}{
				"number": map[string]interface{}{
					"description":         "Numeric literals",
					"pattern":             "\\d+(\\.\\d+)?",
					"thousandsSeparators": []string{",", "_"},
					"examples":            []string{"42", "3.14", "1,000", "1_000_000"},
					"notes":               "Commas and underscores are allowed as thousand separators and are stripped during tokenization",
				},
				"currency": map[string]interface{}{
					"description":         "Currency literals",
					"pattern":             "\\$\\d+(\\.\\d+)?",
					"thousandsSeparators": []string{",", "_"},
					"examples":            []string{"$100", "$1,000.50", "$1_000"},
					"notes":               "Currently only $ symbol supported, commas/underscores allowed",
				},
				"boolean": map[string]interface{}{
					"description":     "Boolean literals",
					"caseInsensitive": true,
					"trueValues":      []string{"true", "yes", "t", "y"},
					"falseValues":     []string{"false", "no", "f", "n"},
					"examples":        []string{"true", "false", "yes", "no", "t", "f", "y", "n"},
					"notes":           "Boolean keywords are normalized to lowercase 'true' or 'false' during tokenization",
				},
			},

			"identifiers": map[string]interface{}{
				"description":       "Variable names",
				"pattern":           "[^\\s\\d\\+\\-\\*×\\/=$><! %^(),][^\\s\\+\\-\\*×\\/=$><! %^(),]*",
				"unicodeSupport":    true,
				"allowedCharacters": "Any Unicode character except whitespace and reserved operators",
				"notAllowed":        []string{"Spaces", "Reserved keywords", "Reserved operators"},
				"examples":          []string{"salary", "my_budget", "weeks_in_year", "給料", "💰", "résumé"},
				"notes":             "BREAKING CHANGE: Spaces are NOT allowed in identifiers (use underscores instead). Unicode and emoji fully supported.",
			},

			"punctuation": map[string]interface{}{
				"description": "Punctuation and delimiters",
				"tokens": []map[string]interface{}{
					{"symbol": "(", "name": "lparen", "description": "Left parenthesis"},
					{"symbol": ")", "name": "rparen", "description": "Right parenthesis"},
					{"symbol": ",", "name": "comma", "description": "Comma (function argument separator)"},
				},
			},

			"markdown": map[string]interface{}{
				"description": "Markdown elements that CalcMark preserves",
				"prefixes": []map[string]interface{}{
					{"pattern": "^#+\\s", "description": "Headers (#, ##, ###, etc.)"},
					{"pattern": "^>\\s", "description": "Blockquotes"},
					{"pattern": "^[-*]\\s", "description": "Unordered lists"},
					{"pattern": "^\\d+\\.\\s", "description": "Ordered lists"},
				},
				"notes": "Lines starting with markdown prefixes are classified as markdown, not calculations",
			},
		},

		"classification": map[string]interface{}{
			"description": "How lines are classified",
			"lineTypes": map[string]interface{}{
				"blank": map[string]interface{}{
					"description": "Empty lines or whitespace-only",
					"pattern":     "^\\s*$",
				},
				"markdown": map[string]interface{}{
					"description": "Lines that are pure markdown",
					"criteria": []string{
						"Starts with markdown prefix (# > - * digit.)",
						"Contains trailing text after calculation",
						"Contains unknown variables (context-aware)",
						"Contains malformed expressions",
						"Contains URLs or natural language",
					},
				},
				"calculation": map[string]interface{}{
					"description": "Lines containing CalcMark calculations",
					"criteria": []string{
						"Literal values (number, currency, boolean)",
						"Variable assignments (x = 5)",
						"Arithmetic expressions (3 + 5)",
						"Comparison expressions (x > 5)",
						"Logical expressions (x > 5 and y < 10)",
						"Function calls (avg(1, 2, 3) or average of 1, 2, 3)",
						"Variable references (if variable is defined in context)",
					},
				},
			},
			"contextAware": map[string]interface{}{
				"description": "Classification depends on execution context",
				"notes":       "A line like 'unknown_var' is markdown if the variable is undefined, but a calculation if it's defined in the context",
			},
		},

		"syntaxRules": map[string]interface{}{
			"whitespace": map[string]interface{}{
				"significance": "mostly insignificant",
				"notes": []string{
					"Spaces around operators are optional (3+5 or 3 + 5)",
					"Spaces separate tokens in multi-token functions (average of)",
					"Tabs and spaces are equivalent",
					"Newlines separate statements",
				},
			},
			"caseInsensitivity": map[string]interface{}{
				"keywords":    true,
				"booleans":    true,
				"functions":   true,
				"identifiers": false,
				"notes":       "Identifiers are case-sensitive, but keywords and functions are not",
			},
			"precedence": map[string]interface{}{
				"description": "Operator precedence (not yet fully implemented in parser)",
				"order": []map[string]interface{}{
					{"level": 1, "operators": []string{"^", "**"}, "description": "Exponentiation (highest)"},
					{"level": 2, "operators": []string{"*", "×", "x", "X", "/", "%"}, "description": "Multiplication, division, modulus"},
					{"level": 3, "operators": []string{"+", "-"}, "description": "Addition, subtraction"},
					{"level": 4, "operators": []string{">", "<", ">=", "<=", "==", "!="}, "description": "Comparison"},
					{"level": 5, "operators": []string{"not"}, "description": "Logical NOT"},
					{"level": 6, "operators": []string{"and"}, "description": "Logical AND"},
					{"level": 7, "operators": []string{"or"}, "description": "Logical OR (lowest)"},
				},
				"notes": "Precedence follows Go language specification for logical operators",
			},
		},

		"breakingChanges": map[string]interface{}{
			"v1.0.0": []map[string]interface{}{
				{
					"change":    "Spaces no longer allowed in identifiers",
					"before":    "my budget = 1000",
					"after":     "my_budget = 1000",
					"rationale": "Required to support multi-token function names like 'average of' and 'square root of'",
					"migration": "Replace spaces in identifier names with underscores",
				},
			},
		},

		"implementationNotes": map[string]interface{}{
			"tokenization": map[string]interface{}{
				"multiTokenCombining": "After initial tokenization, sequences like 'IDENTIFIER(average) + IDENTIFIER(of)' are post-processed and combined into FUNC_AVERAGE_OF",
				"thousandSeparators":  "Commas and underscores within numbers are only consumed if followed by a digit, otherwise they become separate tokens",
				"multiplyAliases":     "The letter 'x' or 'X' is treated as multiply operator when preceded by a number and followed by whitespace or digit",
			},
			"validation": map[string]interface{}{
				"diagnosticCodes": true,
				"severityLevels":  []string{"error", "warning", "info", "hint"},
				"notes":           "Validator runs in parallel with evaluator and provides detailed diagnostic codes",
			},
		},

		"examples": map[string]interface{}{
			"simple": []string{
				"salary = $50000",
				"bonus = $5000",
				"total = salary + bonus",
			},
			"functions": []string{
				"avg(1, 2, 3, 4, 5)",
				"average of 10, 20, 30",
				"sqrt(16)",
				"square root of 25",
			},
			"logical": []string{
				"x > 5 and y < 10",
				"salary > $50000 or bonus > $10000",
				"not (x == 0)",
			},
			"mixed": []string{
				"# Budget Calculator",
				"",
				"monthly_income = $5000",
				"monthly_expenses = $3500",
				"",
				"Has surplus:",
				"surplus = monthly_income > monthly_expenses",
			},
		},

		"futureFeatures": map[string]interface{}{
			"description": "Reserved but not yet implemented",
			"features": []map[string]interface{}{
				{
					"name":     "Conditionals",
					"keywords": []string{"if", "then", "else", "elif", "end"},
					"example":  "if sales > $100000 then tax = 20% else tax = 15% end",
				},
				{
					"name":     "Loops",
					"keywords": []string{"for", "in", "while"},
					"example":  "for month in months do total = total + month end",
				},
				{
					"name":     "Additional functions",
					"examples": []string{"min()", "max()", "sum()", "round()"},
				},
			},
		},
	}
}

func printUsage() {
	fmt.Printf(`calcmark - CalcMark Language Tools (v%s)

Usage:
  calcmark <command>

Commands:
  spec       Output the syntax highlighter specification (JSON)
  generate   Generate the syntax highlighter spec file (for development)
  version    Show version information
  help       Show this help message

Examples:
  calcmark spec                    # Print syntax spec to stdout
  calcmark spec > syntax.json      # Save spec to file
  calcmark generate                # Regenerate spec/SYNTAX_HIGHLIGHTER_SPEC.json
  calcmark version                 # Show version

Documentation:
  https://github.com/CalcMark/go-calcmark
`, version)
}
