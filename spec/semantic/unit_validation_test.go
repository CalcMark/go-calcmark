package semantic_test

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/semantic"
)

// TestUnitCompatibility_IncompatibleUnits tests that incompatible units produce errors
// USER REQUIREMENT: "10 meters + 5 kg" must produce error
func TestUnitCompatibility_IncompatibleUnits(t *testing.T) {
	tests := []struct {
		name  string
		unit1 string
		unit2 string
	}{
		{"meters + kg", "meters", "kg"},
		{"pounds + feet", "lb", "feet"},
		{"liters + grams", "liters", "grams"},
		{"kilometers + pounds", "km", "lbs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := semantic.NewChecker()

			binOp := &ast.BinaryOp{
				Operator: "+",
				Left: &ast.QuantityLiteral{
					Value: "10",
					Unit:  tt.unit1,
					Range: &ast.Range{},
				},
				Right: &ast.QuantityLiteral{
					Value: "5",
					Unit:  tt.unit2,
					Range: &ast.Range{},
				},
				Range: &ast.Range{},
			}

			diagnostics := checker.Check([]ast.Node{binOp})

			if len(diagnostics) == 0 {
				t.Errorf("Expected error for incompatible units %s + %s", tt.unit1, tt.unit2)
			}

			if len(diagnostics) > 0 {
				d := diagnostics[0]
				if d.Code != semantic.DiagIncompatibleUnits {
					t.Errorf("Expected code %s, got %s", semantic.DiagIncompatibleUnits, d.Code)
				}
				if d.Severity != semantic.Error {
					t.Errorf("Expected ERROR severity, got %s", d.Severity)
				}
				// USER REQUIREMENT: Detailed message explaining incompatibility
				if d.Detailed == "" {
					t.Error("Expected detailed message about incompatible units")
				}
			}
		})
	}
}

// TestUnitCompatibility_CompatibleUnits tests that compatible units don't produce errors
// USER REQUIREMENT: First-unit-wins rule
func TestUnitCompatibility_CompatibleUnits(t *testing.T) {
	tests := []struct {
		name  string
		unit1 string
		unit2 string
	}{
		{"meters + feet", "meters", "feet"},      // Both length
		{"kg + pounds", "kg", "lb"},              // Both mass
		{"liters + gallons", "liters", "gallon"}, // Both volume
		{"kilometers + miles", "km", "miles"},    // Both length
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := semantic.NewChecker()

			binOp := &ast.BinaryOp{
				Operator: "+",
				Left: &ast.QuantityLiteral{
					Value: "10",
					Unit:  tt.unit1,
					Range: &ast.Range{},
				},
				Right: &ast.QuantityLiteral{
					Value: "5",
					Unit:  tt.unit2,
					Range: &ast.Range{},
				},
				Range: &ast.Range{},
			}

			diagnostics := checker.Check([]ast.Node{binOp})

			// Should have no errors for compatible units
			for _, d := range diagnostics {
				if d.Code == semantic.DiagIncompatibleUnits {
					t.Errorf("Expected no compatibility error for %s + %s, got: %s",
						tt.unit1, tt.unit2, d.Detailed)
				}
			}
		})
	}
}

// TestCurrencyValidation_EnhancedDiagnostics tests enhanced currency diagnostics
// USER REQUIREMENT: Short + detailed + link messages
func TestCurrencyValidation_EnhancedDiagnostics(t *testing.T) {
	// Test that XXX is invalid
	valid, diag := semantic.ValidateCurrencyCodeWithDiagnostic("XXX")
	if valid {
		t.Error("Expected XXX to be invalid currency")
	}

	if diag == nil {
		t.Error("Expected diagnostic for invalid currency")
		return
	} else if diag.Severity != semantic.Warning {
		t.Errorf("Expected warning, got %v", diag.Severity)
	}

	// USER REQUIREMENT: Short message
	if diag.Message == "" {
		t.Error("Expected non-empty short message")
	}
	// USER REQUIREMENT: Detailed message with explanation
	if diag.Detailed == "" {
		t.Error("Expected detailed message")
	}

	if !contains(diag.Detailed, "XXX is not a known currency") {
		t.Error("Expected detailed message to mention XXX is not known currency")
	}

	if !contains(diag.Detailed, "user-defined unit") {
		t.Error("Expected detailed message to mention user-defined unit option")
	}

	// USER REQUIREMENT: Documentation link
	if diag.Link == "" {
		t.Error("Expected documentation link")
	}

	if !contains(diag.Link, "currency") {
		t.Errorf("Expected link to reference currency, got: %s", diag.Link)
	}

	// Should be warning not error
	if diag.Severity != semantic.Warning {
		t.Errorf("Expected WARNING severity, got %s", diag.Severity)
	}
}

// TestCurrencyValidation_ValidCodes tests that valid currencies don't produce warnings
func TestCurrencyValidation_ValidCodes(t *testing.T) {
	codes := []string{"USD", "EUR", "GBP", "JPY", "$", "€", "£", "¥"}

	for _, code := range codes {
		t.Run(code, func(t *testing.T) {
			valid, diag := semantic.ValidateCurrencyCodeWithDiagnostic(code)

			if !valid {
				t.Errorf("Expected %s to be valid currency", code)
			}

			if diag != nil {
				t.Errorf("Expected no diagnostic for valid currency %s, got: %s", code, diag.Message)
			}
		})
	}
}

// TestQuantityType tests the quantity type classification system
func TestQuantityType(t *testing.T) {
	tests := []struct {
		unit         string
		expectedType semantic.QuantityType
	}{
		{"meters", semantic.QuantityLength},
		{"feet", semantic.QuantityLength},
		{"km", semantic.QuantityLength},
		{"kg", semantic.QuantityMass},
		{"pounds", semantic.QuantityMass},
		{"lb", semantic.QuantityMass},
		{"liters", semantic.QuantityVolume},
		{"gallons", semantic.QuantityVolume},
		{"day", semantic.QuantityTime},
		{"hours", semantic.QuantityTime},
	}

	for _, tt := range tests {
		t.Run(tt.unit, func(t *testing.T) {
			got := semantic.GetQuantityType(tt.unit)
			if got != tt.expectedType {
				t.Errorf("GetQuantityType(%s) = %s, want %s", tt.unit, got, tt.expectedType)
			}
		})
	}
}

// TestDataSizeBaseMixing tests that mixing binary and decimal data units produces hints
func TestDataSizeBaseMixing(t *testing.T) {
	tests := []struct {
		name      string
		unit1     string
		unit2     string
		wantHint  bool
		hintParts []string // Parts expected in the detailed message
	}{
		{
			name:      "GB + Mbps (binary + decimal)",
			unit1:     "GB",
			unit2:     "Mbps",
			wantHint:  true,
			hintParts: []string{"binary", "decimal", "1024", "1000"},
		},
		{
			name:      "MB + Gbps",
			unit1:     "MB",
			unit2:     "Gbps",
			wantHint:  true,
			hintParts: []string{"binary", "decimal"},
		},
		{
			name:      "GiB + Kbps",
			unit1:     "GiB",
			unit2:     "Kbps",
			wantHint:  true,
			hintParts: []string{"binary", "decimal"},
		},
		{
			name:     "GB + MB (both binary)",
			unit1:    "GB",
			unit2:    "MB",
			wantHint: false,
		},
		{
			name:     "GiB + MiB (both binary)",
			unit1:    "GiB",
			unit2:    "MiB",
			wantHint: false,
		},
		{
			name:     "Mbps + Gbps (both decimal)",
			unit1:    "Mbps",
			unit2:    "Gbps",
			wantHint: false,
		},
		{
			name:     "meters + feet (not data units)",
			unit1:    "meters",
			unit2:    "feet",
			wantHint: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := semantic.NewChecker()

			binOp := &ast.BinaryOp{
				Operator: "+",
				Left: &ast.QuantityLiteral{
					Value: "10",
					Unit:  tt.unit1,
					Range: &ast.Range{},
				},
				Right: &ast.QuantityLiteral{
					Value: "5",
					Unit:  tt.unit2,
					Range: &ast.Range{},
				},
				Range: &ast.Range{},
			}

			diagnostics := checker.Check([]ast.Node{binOp})

			// Find the mixed base hint
			var mixedBaseHint *semantic.Diagnostic
			for i := range diagnostics {
				if diagnostics[i].Code == semantic.DiagMixedBaseUnits {
					mixedBaseHint = &diagnostics[i]
					break
				}
			}

			if tt.wantHint {
				if mixedBaseHint == nil {
					t.Errorf("Expected hint for mixing %s + %s, got none", tt.unit1, tt.unit2)
					return
				}

				// Check severity is Hint
				if mixedBaseHint.Severity != semantic.Hint {
					t.Errorf("Expected HINT severity, got %s", mixedBaseHint.Severity)
				}

				// Check detailed message contains expected parts
				for _, part := range tt.hintParts {
					if !contains(mixedBaseHint.Detailed, part) {
						t.Errorf("Expected detailed message to contain %q, got: %s", part, mixedBaseHint.Detailed)
					}
				}

				t.Logf("✓ %s → HINT: %s", tt.name, mixedBaseHint.Message)
			} else {
				if mixedBaseHint != nil {
					t.Errorf("Expected no hint for %s + %s, got: %s", tt.unit1, tt.unit2, mixedBaseHint.Message)
				}
			}
		})
	}
}

// TestDataSizeBaseClassification tests the GetDataSizeBase function
func TestDataSizeBaseClassification(t *testing.T) {
	tests := []struct {
		unit     string
		expected semantic.DataSizeBase
	}{
		// Binary (IEC) units
		{"KiB", semantic.DataSizeBaseBinary},
		{"MiB", semantic.DataSizeBaseBinary},
		{"GiB", semantic.DataSizeBaseBinary},
		{"TiB", semantic.DataSizeBaseBinary},

		// Ambiguous units treated as binary
		{"KB", semantic.DataSizeBaseBinary},
		{"MB", semantic.DataSizeBaseBinary},
		{"GB", semantic.DataSizeBaseBinary},
		{"TB", semantic.DataSizeBaseBinary},

		// Decimal/network units
		{"Kbps", semantic.DataSizeBaseDecimal},
		{"Mbps", semantic.DataSizeBaseDecimal},
		{"Gbps", semantic.DataSizeBaseDecimal},
		{"Kbit", semantic.DataSizeBaseDecimal},
		{"Mbit", semantic.DataSizeBaseDecimal},

		// Base units (no base distinction)
		{"bit", semantic.DataSizeBaseNone},
		{"byte", semantic.DataSizeBaseNone},

		// Non-data units
		{"meters", semantic.DataSizeBaseNone},
		{"kg", semantic.DataSizeBaseNone},
	}

	for _, tt := range tests {
		t.Run(tt.unit, func(t *testing.T) {
			got := semantic.GetDataSizeBase(tt.unit)
			if got != tt.expected {
				t.Errorf("GetDataSizeBase(%q) = %v, want %v", tt.unit, got, tt.expected)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
