package calcmark

import (
	"strings"
	"testing"
)

// Test basic evaluation
func TestEvalSimple(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "simple addition",
			input: "1 + 1",
			want:  "2",
		},
		{
			name:  "multiplication",
			input: "5 * 3",
			want:  "15",
		},
		{
			name:  "number with multiplier",
			input: "1.2k",
			want:  "1200",
		},
		{
			name:  "currency",
			input: "$100",
			want:  "$100.00", // Currency preserves the symbol for display
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Eval(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Eval() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && result.Value != nil {
				got := result.Value.String()
				if got != tt.want {
					t.Errorf("Eval() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

// Test session with variable persistence
func TestSession(t *testing.T) {
	session := NewSession()

	// Set variable
	result, err := session.Eval("x = 10")
	if err != nil {
		t.Fatalf("session.Eval('x = 10') error = %v", err)
	}
	if result.Value.String() != "10" {
		t.Errorf("Expected 10, got %v", result.Value)
	}

	// Use variable
	result, err = session.Eval("x + 5")
	if err != nil {
		t.Fatalf("session.Eval('x + 5') error = %v", err)
	}
	if result.Value.String() != "15" {
		t.Errorf("Expected 15, got %v", result.Value)
	}

	// Reset session
	session.Reset()

	// Variable should be gone
	result, err = session.Eval("x")
	if err == nil && len(result.Diagnostics) == 0 {
		t.Error("Expected undefined variable error after reset")
	}
}

// Test multi-line evaluation
func TestEvalMultiLine(t *testing.T) {
	input := `x = 10
y = 20
x + y`

	result, err := Eval(input)
	if err != nil {
		t.Fatalf("Eval(multiline) error = %v", err)
	}

	// Should have 3 values
	if len(result.AllValues) != 3 {
		t.Errorf("Expected 3 values, got %d", len(result.AllValues))
	}

	// Last value should be 30
	if result.Value.String() != "30" {
		t.Errorf("Expected final value 30, got %v", result.Value)
	}
}

// Test diagnostics
func TestEvalDiagnostics(t *testing.T) {
	// Undefined variable should produce ERROR diagnostic and block evaluation
	result, _ := Eval("undefined_var")

	// Should have ERROR diagnostic (blocks evaluation)
	if result == nil {
		t.Fatal("Expected result with diagnostics, got nil")
	}

	if len(result.Diagnostics) == 0 {
		t.Error("Expected ERROR diagnostic for undefined variable")
	}

	hasError := false
	for _, d := range result.Diagnostics {
		if d.Severity == Error {
			hasError = true
			break
		}
	}

	if !hasError {
		t.Error("Expected ERROR diagnostic (not warning) for undefined variable")
	}

	// Should NOT have evaluated (blocked by semantic error)
	if result.Value != nil {
		t.Error("Expected no value when semantic errors block evaluation")
	}
}

// Test error handling
func TestEvalErrors(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldBlock bool // Should block interpretation
	}{
		{
			name:        "division by zero",
			input:       "10 / 0",
			shouldBlock: false, // Warning, not error
		},
		{
			name:        "invalid syntax",
			input:       "1 +",
			shouldBlock: true, // Parse error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Eval(tt.input)

			if tt.shouldBlock {
				// Should either return error or have ERROR diagnostics
				if err == nil && !hasErrorDiagnostic(result) {
					t.Error("Expected blocking error")
				}
			}
		})
	}
}

// Helper function
func hasErrorDiagnostic(result *Result) bool {
	if result == nil {
		return false
	}
	for _, d := range result.Diagnostics {
		if d.Severity == Error {
			return true
		}
	}
	return false
}

// Test currency conversion with frontmatter exchange rates
func TestCurrencyConversion(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		want           string
		wantErr        bool
		wantErrContain string // substring that must appear in error message
	}{
		{
			name: "USD to EUR",
			input: `---
exchange:
  USD_EUR: 0.92
---
100 USD in EUR`,
			want: "€92.00",
		},
		{
			name: "EUR to GBP",
			input: `---
exchange:
  EUR_GBP: 0.86
---
50 EUR in GBP`,
			want: "£43.00",
		},
		{
			name: "using dollar symbol",
			input: `---
exchange:
  USD_EUR: 0.92
---
$200 in EUR`,
			want: "€184.00",
		},
		{
			name: "no exchange rate defined",
			input: `---
exchange:
  USD_EUR: 0.92
---
100 USD in GBP`,
			wantErr:        true,
			wantErrContain: "no exchange rate defined for USD",
		},
		{
			name: "same currency no-op with code",
			input: `---
exchange:
  USD_EUR: 0.92
---
100 USD in USD`,
			want: "USD100.00", // Input uses code, output preserves code
		},
		{
			name: "same currency no-op with symbol",
			input: `---
exchange:
  USD_EUR: 0.92
---
$100 in USD`,
			want: "$100.00", // Input uses symbol, output preserves symbol
		},
		{
			name: "variable with currency conversion",
			input: `---
exchange:
  USD_EUR: 0.92
---
price = $1000
price in EUR`,
			want: "€920.00",
		},
		{
			name:           "no frontmatter at all",
			input:          "100 USD in EUR",
			wantErr:        true,
			wantErrContain: "no exchange rate defined for USD",
		},
		{
			name: "wrong pair defined",
			input: `---
exchange:
  EUR_JPY: 130.50
---
100 USD in EUR`,
			wantErr:        true,
			wantErrContain: "USD",
		},
		{
			name: "reverse direction not auto-computed",
			input: `---
exchange:
  USD_EUR: 0.92
---
100 EUR in USD`,
			wantErr:        true,
			wantErrContain: "no exchange rate defined for EUR",
		},
		{
			name: "frontmatter exchange then convert",
			input: `---
exchange:
  EUR_GBP: 0.86
  USD_EUR: 0.92
---
100 USD in EUR`,
			want: "€92.00",
		},
		{
			name: "multiple sequential conversions",
			input: `---
exchange:
  USD_EUR: 0.92
  EUR_GBP: 0.86
---
price_eur = 100 USD in EUR
price_gbp = 50 EUR in GBP`,
			want: "£43.00", // last value
		},
		{
			name: "lowercase exchange key normalizes",
			input: `---
exchange:
  usd_eur: 0.92
---
100 USD in EUR`,
			want: "€92.00",
		},
		{
			name: "error message suggests correct underscore format",
			input: `---
exchange:
  EUR_GBP: 0.86
---
100 USD in EUR`,
			wantErr:        true,
			wantErrContain: "USD_EUR", // must use underscore, not slash
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Eval(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got result: %v", result.Value)
					return
				}
				if tt.wantErrContain != "" && !strings.Contains(err.Error(), tt.wantErrContain) {
					t.Errorf("error %q should contain %q", err.Error(), tt.wantErrContain)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if result.Value == nil {
				t.Error("expected value, got nil")
				return
			}
			got := result.Value.String()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// Test frontmatter parsing errors
func TestFrontmatterErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name: "unclosed frontmatter",
			input: `---
exchange:
  USD_EUR: 0.92
x = 10`,
			wantErr: "missing closing '---' delimiter",
		},
		{
			name: "invalid exchange key format",
			input: `---
exchange:
  USDEUR: 0.92
---
x = 10`,
			wantErr: "expected format 'FROM_TO'",
		},
		{
			name: "slash separator instead of underscore",
			input: `---
exchange:
  EUR/GBP: 0.86
---
x = 10`,
			wantErr: "expected format 'FROM_TO'",
		},
		{
			name: "dot separator instead of underscore",
			input: `---
exchange:
  EUR.GBP: 0.86
---
x = 10`,
			wantErr: "expected format 'FROM_TO'",
		},
		{
			name: "too many underscores in exchange key",
			input: `---
exchange:
  USD_EUR_GBP: 0.5
---
x = 10`,
			wantErr: "expected format 'FROM_TO'",
		},
		{
			name: "NaN exchange rate",
			input: `---
exchange:
  USD_EUR: .nan
---
x = 10`,
			wantErr: "not a finite number",
		},
		{
			name: "Inf exchange rate",
			input: `---
exchange:
  USD_EUR: .inf
---
x = 10`,
			wantErr: "not a finite number",
		},
		{
			name: "negative exchange rate",
			input: `---
exchange:
  USD_EUR: -0.5
---
x = 10`,
			wantErr: "must be positive",
		},
		{
			name: "zero exchange rate",
			input: `---
exchange:
  USD_EUR: 0
---
x = 10`,
			wantErr: "must be positive",
		},
		{
			name: "short currency code (2 letters)",
			input: `---
exchange:
  US_EUR: 0.92
---
x = 10`,
			wantErr: "must be exactly 3 letters",
		},
		{
			name: "long currency code (4 letters)",
			input: `---
exchange:
  USDD_EUR: 0.92
---
x = 10`,
			wantErr: "must be exactly 3 letters",
		},
		{
			name: "numeric currency code",
			input: `---
exchange:
  123_EUR: 0.92
---
x = 10`,
			wantErr: "must be exactly 3 letters",
		},
		{
			name: "mixed alphanumeric currency code",
			input: `---
exchange:
  U1D_EUR: 0.92
---
x = 10`,
			wantErr: "must be exactly 3 letters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Eval(tt.input)
			if err == nil {
				t.Error("expected error")
				return
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q should contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// Test globals in frontmatter
func TestGlobalsInFrontmatter(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name: "simple number global",
			input: `---
globals:
  tax_rate: 0.32
---
100 * @globals.tax_rate`,
			want: "32",
		},
		{
			name: "currency global",
			input: `---
globals:
  base_price: $100
---
@globals.base_price * 2`,
			want: "$200.00",
		},
		{
			name: "quantity global",
			input: `---
globals:
  distance: 42 km
---
@globals.distance * 2`,
			want: "84 km",
		},
		{
			name: "duration global",
			input: `---
globals:
  sprint_length: 2 weeks
---
@globals.sprint_length * 3`,
			want: "6 week", // Duration uses canonical singular unit
		},
		{
			name: "globals with exchange rates",
			input: `---
exchange:
  USD_EUR: 0.92
globals:
  base_price: $100
---
@globals.base_price in EUR`,
			want: "€92.00",
		},
		{
			name: "multiple globals",
			input: `---
globals:
  price: $50
  quantity: 3
---
total = @globals.price * @globals.quantity
total`,
			want: "$150.00",
		},
		{
			name: "rate global",
			input: `---
globals:
  bandwidth: 100 MB/s
---
@globals.bandwidth over 10 seconds`,
			want: "1000 MB",
		},
		{
			name: "expression not allowed in global",
			input: `---
globals:
  sum: 1 + 1
---
@globals.sum`,
			wantErr: true,
		},
		{
			name: "unknown frontmatter key rejected",
			input: `---
my_var: 42
---
my_var`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Eval(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got result: %v", result.Value)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if result.Value == nil {
				t.Error("expected value, got nil")
				return
			}
			got := result.Value.String()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
