package semantic_test

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/semantic"
)

// TestDateValidation_February30 tests that February 30 produces an error
// USER REQUIREMENT: Feb 30 must produce semantic error
func TestDateValidation_February30(t *testing.T) {
	checker := semantic.NewChecker()

	dateLiteral := &ast.DateLiteral{
		Month: "February",
		Day:   "30",
		Year:  strPtr("2025"),
		Range: &ast.Range{},
	}

	diagnostics := checker.Check([]ast.Node{dateLiteral})

	if len(diagnostics) != 1 {
		t.Fatalf("Expected 1 diagnostic for Feb 30, got %d", len(diagnostics))
	}

	d := diagnostics[0]
	if d.Code != semantic.DiagInvalidDate {
		t.Errorf("Expected code %s, got %s", semantic.DiagInvalidDate, d.Code)
	}

	if d.Severity != semantic.Error {
		t.Errorf("Expected ERROR severity, got %s", d.Severity)
	}

	// Check for enhanced diagnostics (USER REQUIREMENT)
	if d.Message == "" {
		t.Error("Expected non-empty short message")
	}
	if d.Detailed == "" {
		t.Error("Expected non-empty detailed message")
	}

	// Detailed message should mention February only has 28 days
	if d.Detailed != "February only has 28 days in 2025 (not a leap year)" {
		t.Errorf("Unexpected detailed message: %s", d.Detailed)
	}
}

// TestDateValidation_LeapYear tests leap year handling
// USER REQUIREMENT: Feb 29 should be valid in leap years
func TestDateValidation_LeapYear(t *testing.T) {
	tests := []struct {
		name      string
		year      string
		expectErr bool
		detailed  string
	}{
		{"2024 leap year", "2024", false, ""},
		{"2025 not leap", "2025", true, "February only has 28 days in 2025 (not a leap year)"},
		{"2000 leap year", "2000", false, ""},                                                  // Divisible by 400
		{"1900 not leap", "1900", true, "February only has 28 days in 1900 (not a leap year)"}, // Divisible by 100 but not 400
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := semantic.NewChecker()

			dateLiteral := &ast.DateLiteral{
				Month: "February",
				Day:   "29",
				Year:  &tt.year,
				Range: &ast.Range{},
			}

			diagnostics := checker.Check([]ast.Node{dateLiteral})

			if tt.expectErr {
				if len(diagnostics) == 0 {
					t.Errorf("Expected error for Feb 29 %s, got none", tt.year)
				} else if diagnostics[0].Detailed != tt.detailed {
					t.Errorf("Expected detailed: %s, got: %s", tt.detailed, diagnostics[0].Detailed)
				}
			} else {
				if len(diagnostics) > 0 {
					t.Errorf("Expected no error for Feb 29 %s, got: %s", tt.year, diagnostics[0].Message)
				}
			}
		})
	}
}

// TestDateValidation_InvalidDays tests all months with invalid days
func TestDateValidation_InvalidDays(t *testing.T) {
	tests := []struct {
		month      string
		invalidDay string
		maxDays    int
	}{
		{"January", "32", 31},
		{"February", "30", 28},
		{"March", "32", 31},
		{"April", "31", 30},
		{"May", "32", 31},
		{"June", "31", 30},
		{"July", "32", 31},
		{"August", "32", 31},
		{"September", "31", 30},
		{"October", "32", 31},
		{"November", "31", 30},
		{"December", "32", 31},
	}

	for _, tt := range tests {
		t.Run(tt.month+" "+tt.invalidDay, func(t *testing.T) {
			checker := semantic.NewChecker()

			dateLiteral := &ast.DateLiteral{
				Month: tt.month,
				Day:   tt.invalidDay,
				Year:  strPtr("2025"),
				Range: &ast.Range{},
			}

			diagnostics := checker.Check([]ast.Node{dateLiteral})

			if len(diagnostics) == 0 {
				t.Errorf("Expected error for %s %s", tt.month, tt.invalidDay)
			}
		})
	}
}

// TestDateValidation_ValidDates tests that valid dates don't produce errors
func TestDateValidation_ValidDates(t *testing.T) {
	tests := []struct {
		month string
		day   string
		year  string
	}{
		{"January", "1", "2025"},
		{"January", "31", "2025"},
		{"February", "28", "2025"},
		{"February", "29", "2024"}, // Leap year
		{"April", "30", "2025"},
		{"December", "25", "2025"},
		{"December", "31", "2025"},
	}

	for _, tt := range tests {
		t.Run(tt.month+" "+tt.day+" "+tt.year, func(t *testing.T) {
			checker := semantic.NewChecker()

			dateLiteral := &ast.DateLiteral{
				Month: tt.month,
				Day:   tt.day,
				Year:  &tt.year,
				Range: &ast.Range{},
			}

			diagnostics := checker.Check([]ast.Node{dateLiteral})

			if len(diagnostics) > 0 {
				t.Errorf("Expected no error for valid date %s %s %s, got: %s",
					tt.month, tt.day, tt.year, diagnostics[0].Detailed)
			}
		})
	}
}

// TestDateValidation_YearRange tests year validation
func TestDateValidation_YearRange(t *testing.T) {
	tests := []struct {
		year      string
		expectErr bool
	}{
		{"1899", true},  // Too early
		{"1900", false}, // Valid
		{"2025", false}, // Valid
		{"2100", false}, // Valid
		{"2101", true},  // Too late
	}

	for _, tt := range tests {
		t.Run(tt.year, func(t *testing.T) {
			checker := semantic.NewChecker()

			dateLiteral := &ast.DateLiteral{
				Month: "January",
				Day:   "1",
				Year:  &tt.year,
				Range: &ast.Range{},
			}

			diagnostics := checker.Check([]ast.Node{dateLiteral})

			if tt.expectErr && len(diagnostics) == 0 {
				t.Errorf("Expected error for year %s", tt.year)
			}
			if !tt.expectErr && len(diagnostics) > 0 {
				t.Errorf("Expected no error for year %s, got: %s", tt.year, diagnostics[0].Message)
			}
		})
	}
}

// Helper function
func strPtr(s string) *string {
	return &s
}
