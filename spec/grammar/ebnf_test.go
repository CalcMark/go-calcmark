package grammar

import (
	"strings"
	"testing"
)

func TestGenerateEBNF_W3CFormat(t *testing.T) {
	ebnf := GenerateEBNF("0.1.1")

	// Verify W3C EBNF format (not ISO EBNF)
	if strings.Contains(ebnf, "(* ") {
		t.Error("EBNF should use /* */ comments, not (* *) comments")
	}

	// Should use ::= not =
	if strings.Contains(ebnf, "Document        = ") {
		t.Error("EBNF should use ::= operator, not = operator")
	}

	// Should have ::= operator
	if !strings.Contains(ebnf, "::=") {
		t.Error("EBNF should use ::= operator for production rules")
	}

	// Verify key productions exist
	requiredProductions := []string{
		"Document",
		"Statement",
		"Expression",
		"Number",
		"Currency",
		"Identifier",
	}

	for _, prod := range requiredProductions {
		if !strings.Contains(ebnf, prod) {
			t.Errorf("EBNF should contain production for %s", prod)
		}
	}
}

func TestGenerateEBNF_BasicStructure(t *testing.T) {
	ebnf := GenerateEBNF("0.1.1")

	// Should contain version
	if !strings.Contains(ebnf, "0.1.1") {
		t.Error("EBNF should contain version number")
	}

	// Should contain grammar description
	if !strings.Contains(ebnf, "CalcMark") {
		t.Error("EBNF should mention CalcMark")
	}
}

func TestGenerateEBNF_IntrospectedElements(t *testing.T) {
	ebnf := GenerateEBNF("0.1.1")

	// Should introspect boolean keywords
	if !strings.Contains(ebnf, "true") || !strings.Contains(ebnf, "false") {
		t.Error("EBNF should include introspected boolean keywords")
	}

	// Should introspect function names
	if !strings.Contains(ebnf, "avg") || !strings.Contains(ebnf, "sqrt") {
		t.Error("EBNF should include introspected function names")
	}

	// Should introspect emoji ranges
	if !strings.Contains(ebnf, "Emoji") {
		t.Error("EBNF should include emoji range information")
	}

	// Should introspect diagnostic examples (in Title Case format)
	if !strings.Contains(ebnf, "Blank Line Isolation") || !strings.Contains(ebnf, "Unsupported Emoji In Calc") {
		t.Error("EBNF should include introspected diagnostic examples")
	}
}

func TestGenerateEBNF_RailroadDiagramCompatibility(t *testing.T) {
	ebnf := GenerateEBNF("0.1.1")

	// W3C EBNF format requirements for https://www.bottlecaps.de/rr/ui

	// Should use ::= for production rules
	if !strings.Contains(ebnf, "Document        ::=") {
		t.Error("Production rules must use ::= operator")
	}

	// Should use /* */ for comments (not (* *))
	if strings.Contains(ebnf, "(*") {
		t.Error("Comments must use /* */ syntax, not (* *)")
	}

	// Should have W3C-style quantifiers
	if !strings.Contains(ebnf, "Statement*") {
		t.Error("Should use * quantifier for zero-or-more")
	}

	if !strings.Contains(ebnf, "Digit+") {
		t.Error("Should use + quantifier for one-or-more")
	}

	if !strings.Contains(ebnf, "Percentage?") {
		t.Error("Should use ? quantifier for optional")
	}

	// Should use grouping with parentheses
	if !strings.Contains(ebnf, "( ") || !strings.Contains(ebnf, " )?") {
		t.Error("Should use parentheses for grouping")
	}

	// Should use character classes for emoji ranges
	if !strings.Contains(ebnf, "[#x") {
		t.Error("Should use [#xHHHH-#xHHHH] syntax for character ranges")
	}

	// Should use [A-Z] syntax for character ranges
	if !strings.Contains(ebnf, "[A-Z]") {
		t.Error("Should use [A-Z] syntax for ASCII letter ranges")
	}
}
