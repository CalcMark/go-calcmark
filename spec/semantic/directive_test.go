package semantic

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/ast"
)

// mockFrontmatter implements FrontmatterInfo for testing.
type mockFrontmatter struct {
	hasScale   bool
	hasGlobals bool
	globals    map[string]bool
	globalKeys []string
}

func (m *mockFrontmatter) HasScale() bool             { return m.hasScale }
func (m *mockFrontmatter) HasGlobals() bool           { return m.hasGlobals }
func (m *mockFrontmatter) HasGlobal(name string) bool { return m.globals[name] }
func (m *mockFrontmatter) GlobalKeys() []string       { return m.globalKeys }

func TestCheckDirectiveRef_ScaleValid(t *testing.T) {
	c := NewChecker()
	c.SetFrontmatter(&mockFrontmatter{hasScale: true})

	nodes := []ast.Node{
		&ast.Assignment{
			Name:  "x",
			Value: &ast.DirectiveRef{Directive: "scale"},
		},
	}
	diags := c.Check(nodes)
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics, got %d: %v", len(diags), diags)
	}
}

func TestCheckDirectiveRef_ScaleNoFrontmatter(t *testing.T) {
	c := NewChecker()
	// No frontmatter set

	nodes := []ast.Node{
		&ast.Expression{
			Expr: &ast.DirectiveRef{Directive: "scale"},
		},
	}
	diags := c.Check(nodes)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
	if diags[0].Code != DiagMissingFrontmatter {
		t.Errorf("expected code %q, got %q", DiagMissingFrontmatter, diags[0].Code)
	}
	if !strings.Contains(diags[0].Message, "@scale requires") {
		t.Errorf("unexpected message: %s", diags[0].Message)
	}
}

func TestCheckDirectiveRef_GlobalsValid(t *testing.T) {
	c := NewChecker()
	c.SetFrontmatter(&mockFrontmatter{
		hasGlobals: true,
		globals:    map[string]bool{"tax_rate": true},
		globalKeys: []string{"tax_rate"},
	})

	nodes := []ast.Node{
		&ast.Assignment{
			Name:  "x",
			Value: &ast.DirectiveRef{Directive: "globals", Field: "tax_rate"},
		},
	}
	diags := c.Check(nodes)
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics, got %d: %v", len(diags), diags)
	}
}

func TestCheckDirectiveRef_GlobalsUndefined(t *testing.T) {
	c := NewChecker()
	c.SetFrontmatter(&mockFrontmatter{
		hasGlobals: true,
		globals:    map[string]bool{"budget": true, "tax_rate": true},
		globalKeys: []string{"budget", "tax_rate"},
	})

	nodes := []ast.Node{
		&ast.Expression{
			Expr: &ast.DirectiveRef{Directive: "globals", Field: "nonexistent"},
		},
	}
	diags := c.Check(nodes)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
	if diags[0].Code != DiagUndefinedGlobal {
		t.Errorf("expected code %q, got %q", DiagUndefinedGlobal, diags[0].Code)
	}
	if !strings.Contains(diags[0].Message, "nonexistent") {
		t.Errorf("expected message to mention 'nonexistent': %s", diags[0].Message)
	}
	if !strings.Contains(diags[0].Message, "budget") || !strings.Contains(diags[0].Message, "tax_rate") {
		t.Errorf("expected message to list defined globals: %s", diags[0].Message)
	}
}

func TestCheckDirectiveRef_GlobalsNoFrontmatter(t *testing.T) {
	c := NewChecker()

	nodes := []ast.Node{
		&ast.Expression{
			Expr: &ast.DirectiveRef{Directive: "globals", Field: "tax_rate"},
		},
	}
	diags := c.Check(nodes)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
	if diags[0].Code != DiagMissingFrontmatter {
		t.Errorf("expected code %q, got %q", DiagMissingFrontmatter, diags[0].Code)
	}
}

func TestCheckDirectiveRef_InvalidDirective(t *testing.T) {
	tests := []string{"exchange", "convert_to", "foo"}
	for _, directive := range tests {
		t.Run(directive, func(t *testing.T) {
			c := NewChecker()
			nodes := []ast.Node{
				&ast.Expression{
					Expr: &ast.DirectiveRef{Directive: directive},
				},
			}
			diags := c.Check(nodes)
			if len(diags) != 1 {
				t.Fatalf("expected 1 diagnostic, got %d", len(diags))
			}
			if diags[0].Code != DiagInvalidDirective {
				t.Errorf("expected code %q, got %q", DiagInvalidDirective, diags[0].Code)
			}
			if !strings.Contains(diags[0].Message, "not a supported directive") {
				t.Errorf("unexpected message: %s", diags[0].Message)
			}
		})
	}
}
