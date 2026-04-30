package display

import (
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/types"
	"github.com/shopspring/decimal"
)

// NoBreakSpace is U+00A0, used by golang.org/x/text for fr-FR thousand separator.
const NoBreakSpace = "\u00a0"

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name        string
		locale      string
		wantDecimal string
		wantThous   string
		wantErr     bool
	}{
		{"en-US", "en-US", ".", ",", false},
		{"de-DE", "de-DE", ",", ".", false},
		{"fr-FR", "fr-FR", ",", NoBreakSpace, false},
		{"invalid locale", "xx-INVALID-TOOLONG-BLAH", "", "", true},
		{"empty locale", "", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := NewConfig(tt.locale)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewConfig(%q) error = %v, wantErr %v", tt.locale, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if cfg.DecimalSep != tt.wantDecimal {
				t.Errorf("DecimalSep = %q, want %q", cfg.DecimalSep, tt.wantDecimal)
			}
			if cfg.ThousandSep != tt.wantThous {
				t.Errorf("ThousandSep = %q, want %q", cfg.ThousandSep, tt.wantThous)
			}
		})
	}
}

func TestNewConfigSecurity(t *testing.T) {
	t.Run("rejects too-long locale", func(t *testing.T) {
		long := make([]byte, maxLocaleLen+1)
		for i := range long {
			long[i] = 'a'
		}
		_, err := NewConfig(string(long))
		if err == nil {
			t.Error("expected error for locale exceeding max length")
		}
	})

	t.Run("rejects non-ASCII locale", func(t *testing.T) {
		_, err := NewConfig("en-US\xff")
		if err == nil {
			t.Error("expected error for non-ASCII locale")
		}
	})
}

func TestNewConfigRoundTrip(t *testing.T) {
	// DefaultConfig() and NewConfig("en-US") must produce identical output
	def := DefaultConfig()
	parsed, err := NewConfig("en-US")
	if err != nil {
		t.Fatalf("NewConfig(en-US): %v", err)
	}

	if def.DecimalSep != parsed.DecimalSep {
		t.Errorf("DecimalSep mismatch: default=%q, parsed=%q", def.DecimalSep, parsed.DecimalSep)
	}
	if def.ThousandSep != parsed.ThousandSep {
		t.Errorf("ThousandSep mismatch: default=%q, parsed=%q", def.ThousandSep, parsed.ThousandSep)
	}

	// Verify identical formatting output
	f1 := NewFormatter(def)
	f2 := NewFormatter(parsed)

	testValues := []types.Type{
		types.NewNumber(decimal.NewFromInt(100000)),
		types.NewNumber(decimal.NewFromFloat(42.50)),
		types.NewCurrency(decimal.NewFromInt(1500), "$"),
		types.NewQuantity(decimal.NewFromInt(1000), "m"),
	}

	for _, v := range testValues {
		r1 := f1.Format(v)
		r2 := f2.Format(v)
		if r1 != r2 {
			t.Errorf("output mismatch for %v: default=%q, parsed=%q", v, r1, r2)
		}
	}
}

func TestFormatterLocaleNumbers(t *testing.T) {
	tests := []struct {
		name   string
		locale string
		value  string
		want   string
	}{
		// en-US
		{"en-US small int", "en-US", "42", "42"},
		{"en-US small decimal", "en-US", "3.14", "3.14"},
		{"en-US 1K", "en-US", "1000", "1K"},
		{"en-US 1.5K", "en-US", "1500", "1.5K"},
		{"en-US 1M", "en-US", "1000000", "1M"},
		{"en-US 1.5M", "en-US", "1500000", "1.5M"},
		{"en-US negative", "en-US", "-5000", "-5K"},
		{"en-US zero", "en-US", "0", "0"},

		// de-DE: comma as decimal, dot as thousand
		{"de-DE small int", "de-DE", "42", "42"},
		{"de-DE small decimal", "de-DE", "3.14", "3,14"},
		{"de-DE 1K", "de-DE", "1000", "1K"},
		{"de-DE 1.5K", "de-DE", "1500", "1,5K"},
		{"de-DE 1M", "de-DE", "1000000", "1M"},
		{"de-DE 1.5M", "de-DE", "1500000", "1,5M"},
		{"de-DE negative", "de-DE", "-5000", "-5K"},
		{"de-DE zero", "de-DE", "0", "0"},

		// fr-FR: comma as decimal, NBSP as thousand
		{"fr-FR small int", "fr-FR", "42", "42"},
		{"fr-FR small decimal", "fr-FR", "3.14", "3,14"},
		{"fr-FR 1K", "fr-FR", "1000", "1K"},
		{"fr-FR 1.5K", "fr-FR", "1500", "1,5K"},
		{"fr-FR 1M", "fr-FR", "1000000", "1M"},
		{"fr-FR negative", "fr-FR", "-5000", "-5K"},
		{"fr-FR zero", "fr-FR", "0", "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := NewConfig(tt.locale)
			if err != nil {
				t.Fatalf("NewConfig(%q): %v", tt.locale, err)
			}
			f := NewFormatter(cfg)
			value, _ := decimal.NewFromString(tt.value)
			got := f.FormatNumber(value)
			if got != tt.want {
				t.Errorf("FormatNumber(%s) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestFormatterLocaleCurrency(t *testing.T) {
	tests := []struct {
		name   string
		locale string
		value  string
		code   string
		want   string
	}{
		// en-US
		{"en-US small", "en-US", "42.50", "$", "$42.50"},
		{"en-US mid", "en-US", "1500", "$", "$1,500.00"},
		{"en-US large", "en-US", "15000", "$", "$15K"},
		{"en-US JPY", "en-US", "100", "JPY", "¥100"},

		// de-DE: comma decimal, dot thousand
		{"de-DE small", "de-DE", "42.50", "$", "$42,50"},
		{"de-DE mid", "de-DE", "1500", "$", "$1.500,00"},
		{"de-DE large", "de-DE", "15000", "$", "$15K"},
		{"de-DE JPY", "de-DE", "100", "JPY", "¥100"},

		// fr-FR: comma decimal, NBSP thousand
		{"fr-FR small", "fr-FR", "42.50", "$", "$42,50"},
		{"fr-FR mid", "fr-FR", "1500", "$", "$1" + NoBreakSpace + "500,00"},
		{"fr-FR large", "fr-FR", "15000", "$", "$15K"},

		// Negative values (prefix-symbol currencies)
		{"en-US negative small", "en-US", "-50.00", "$", "-$50.00"},
		{"de-DE negative small", "de-DE", "-50.00", "$", "-$50,00"},
		{"en-US negative mid", "en-US", "-1500", "$", "-$1,500.00"},
		{"de-DE negative mid", "de-DE", "-1500", "$", "-$1.500,00"},

		// Postfix-code currencies (no symbol mapping — stay as ISO code with space)
		// CNY
		{"en-US CNY small", "en-US", "42.50", "CNY", "CNY 42.50"},
		{"de-DE CNY small", "de-DE", "42.50", "CNY", "CNY 42,50"},
		{"fr-FR CNY small", "fr-FR", "42.50", "CNY", "CNY 42,50"},
		{"en-US CNY mid", "en-US", "1500", "CNY", "CNY 1,500.00"},
		{"de-DE CNY mid", "de-DE", "1500", "CNY", "CNY 1.500,00"},
		{"fr-FR CNY mid", "fr-FR", "1500", "CNY", "CNY 1" + NoBreakSpace + "500,00"},
		{"en-US CNY large", "en-US", "15000", "CNY", "CNY 15K"},
		{"de-DE CNY large", "de-DE", "15000", "CNY", "CNY 15K"},

		// VND (zero-decimal currency)
		{"en-US VND mid", "en-US", "5000", "VND", "VND 5,000"},
		{"de-DE VND mid", "de-DE", "5000", "VND", "VND 5.000"},

		// KRW (zero-decimal currency)
		{"en-US KRW mid", "en-US", "5000", "KRW", "KRW 5,000"},
		{"de-DE KRW mid", "de-DE", "5000", "KRW", "KRW 5.000"},

		// Negative postfix-code currencies
		{"en-US CNY negative mid", "en-US", "-1500", "CNY", "-CNY 1,500.00"},
		{"de-DE CNY negative mid", "de-DE", "-1500", "CNY", "-CNY 1.500,00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := NewConfig(tt.locale)
			if err != nil {
				t.Fatalf("NewConfig(%q): %v", tt.locale, err)
			}
			f := NewFormatter(cfg)
			value, _ := decimal.NewFromString(tt.value)
			c := types.NewCurrency(value, tt.code)
			got := f.FormatCurrency(c)
			if got != tt.want {
				// Use %q to make non-ASCII chars like U+00A0 visible
				t.Errorf("FormatCurrency(%s %s) = %q, want %q", tt.code, tt.value, got, tt.want)
			}
		})
	}
}

func TestFormatterLocaleQuantity(t *testing.T) {
	tests := []struct {
		name   string
		locale string
		value  string
		unit   string
		want   string
	}{
		// Known units: normalized
		{"en-US 1 km", "en-US", "1000", "m", "1 km"},
		{"de-DE 1 km", "de-DE", "1000", "m", "1 km"},
		{"en-US 50.5 kg", "en-US", "50.5", "kg", "50.5 kg"},
		{"de-DE 50.5 kg", "de-DE", "50.5", "kg", "50,5 kg"},
		{"fr-FR 50.5 kg", "fr-FR", "50.5", "kg", "50,5 kg"},

		// Arbitrary units: K/M/B/T
		{"en-US 100K users", "en-US", "100000", "users", "100K users"},
		{"de-DE 100K users", "de-DE", "100000", "users", "100K users"},

		// Napkin estimates: locale decimal in small values
		{"en-US ~400 GB", "en-US", "400", "GB", "400 GB"},
		{"de-DE ~400 GB", "de-DE", "400", "GB", "400 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := NewConfig(tt.locale)
			if err != nil {
				t.Fatalf("NewConfig(%q): %v", tt.locale, err)
			}
			f := NewFormatter(cfg)
			value, _ := decimal.NewFromString(tt.value)
			q := types.NewQuantity(value, tt.unit)
			got := f.FormatQuantity(q)
			if got != tt.want {
				t.Errorf("FormatQuantity(%s %s) = %q, want %q", tt.value, tt.unit, got, tt.want)
			}
		})
	}
}

func TestFormatterLocaleRate(t *testing.T) {
	tests := []struct {
		name    string
		locale  string
		value   string
		unit    string
		perUnit string
		want    string
	}{
		{"en-US 1 km/h", "en-US", "1000", "m", "hour", "1 km/h"},
		{"de-DE 1 km/h", "de-DE", "1000", "m", "hour", "1 km/h"},
		{"en-US 100K users/day", "en-US", "100000", "users", "day", "100K users/day"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := NewConfig(tt.locale)
			if err != nil {
				t.Fatalf("NewConfig(%q): %v", tt.locale, err)
			}
			f := NewFormatter(cfg)
			value, _ := decimal.NewFromString(tt.value)
			r := types.NewRate(&types.Quantity{Value: value, Unit: tt.unit}, tt.perUnit)
			got := f.FormatRate(r)
			if got != tt.want {
				t.Errorf("FormatRate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInsertGroupSeparators(t *testing.T) {
	f := DefaultFormatter()

	tests := []struct {
		input string
		want  string
	}{
		{"999", "999"},
		{"1000", "1,000"},
		{"12345", "12,345"},
		{"123456", "123,456"},
		{"1234567", "1,234,567"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := f.insertGroupSeparators(tt.input)
			if got != tt.want {
				t.Errorf("insertGroupSeparators(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestInsertGroupSeparatorsLocale(t *testing.T) {
	deCfg, _ := NewConfig("de-DE")
	deF := NewFormatter(deCfg)

	tests := []struct {
		input string
		want  string
	}{
		{"999", "999"},
		{"1000", "1.000"},
		{"1234567", "1.234.567"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := deF.insertGroupSeparators(tt.input)
			if got != tt.want {
				t.Errorf("insertGroupSeparators(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestInsertGroupSeparatorsPathological(t *testing.T) {
	f := DefaultFormatter()

	// Strings exceeding maxSeparatorInputLen should be returned as-is
	long := make([]byte, maxSeparatorInputLen+1)
	for i := range long {
		long[i] = '1'
	}
	got := f.insertGroupSeparators(string(long))
	if got != string(long) {
		t.Error("pathological string should be returned unchanged")
	}
}

func TestFormatterLocaleDate(t *testing.T) {
	// Dec 25, 2025 = Thursday
	d, err := types.NewDate(2025, 12, 25)
	if err != nil {
		t.Fatalf("NewDate: %v", err)
	}

	tests := []struct {
		name   string
		locale string
		want   string
	}{
		{"en-US", "en-US", "Thu, Dec 25, 2025"},
		{"de-DE", "de-DE", "Do. 25. Dez. 2025"},
		{"fr-FR", "fr-FR", "jeu. 25 déc. 2025"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := NewConfig(tt.locale)
			if err != nil {
				t.Fatalf("NewConfig(%q): %v", tt.locale, err)
			}
			f := NewFormatter(cfg)
			got := f.FormatDate(d)
			if got != tt.want {
				t.Errorf("FormatDate() = %q, want %q", got, tt.want)
			}
		})
	}
}
