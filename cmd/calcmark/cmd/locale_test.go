package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/config"
	"github.com/CalcMark/go-calcmark/format"
	"github.com/CalcMark/go-calcmark/format/display"
	implDoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/spec/document"
)

func TestLocaleFormatter_Default(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	if _, err := config.Reload(); err != nil {
		t.Fatalf("Reload() error: %v", err)
	}

	f := localeFormatter()
	cfg := f.Config()

	if cfg.DecimalSep != "." {
		t.Errorf("expected en-US decimal '.', got %q", cfg.DecimalSep)
	}
	if cfg.ThousandSep != "," {
		t.Errorf("expected en-US thousand ',', got %q", cfg.ThousandSep)
	}
}

func TestLocaleFormatter_FromConfig(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	configDir := filepath.Join(tmpHome, ".config", "calcmark")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	userConfig := `locale = "de-DE"
`
	configPath := filepath.Join(configDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(userConfig), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if _, err := config.Reload(); err != nil {
		t.Fatalf("Reload() error: %v", err)
	}

	f := localeFormatter()
	cfg := f.Config()

	if cfg.DecimalSep != "," {
		t.Errorf("expected de-DE decimal ',', got %q", cfg.DecimalSep)
	}
	if cfg.ThousandSep != "." {
		t.Errorf("expected de-DE thousand '.', got %q", cfg.ThousandSep)
	}
}

func TestLocaleFormatter_InvalidFallsBackToEnUS(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	if _, err := config.Reload(); err != nil {
		t.Fatalf("Reload() error: %v", err)
	}

	// Simulate CLI flag with invalid locale
	config.Get().Locale = "xx-INVALID-TOOLONG-BLAH"

	f := localeFormatter()
	cfg := f.Config()

	// Should fall back to en-US
	if cfg.DecimalSep != "." {
		t.Errorf("expected fallback decimal '.', got %q", cfg.DecimalSep)
	}
}

func TestEvalWithLocale_EndToEnd(t *testing.T) {
	tests := []struct {
		name   string
		locale string
		input  string
		want   string
	}{
		{
			name:   "en-US currency",
			locale: "en-US",
			input:  "$1500",
			want:   "$1,500.00",
		},
		{
			name:   "de-DE currency",
			locale: "de-DE",
			input:  "$1500",
			want:   "$1.500,00",
		},
		{
			name:   "de-DE decimal number",
			locale: "de-DE",
			input:  "3.14",
			want:   "3,14",
		},
		{
			name:   "en-US decimal number",
			locale: "en-US",
			input:  "3.14",
			want:   "3.14",
		},
		{
			name:   "de-DE date",
			locale: "de-DE",
			input:  "d = Dec 25 2025",
			want:   "Do. 25. Dez. 2025",
		},
		{
			name:   "en-US date",
			locale: "en-US",
			input:  "d = Dec 25 2025",
			want:   "Thu, Dec 25, 2025",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create formatter for locale
			dcfg, err := display.NewConfig(tt.locale)
			if err != nil {
				t.Fatalf("NewConfig(%q): %v", tt.locale, err)
			}
			df := display.NewFormatter(dcfg)

			// Parse and evaluate
			doc, err := document.NewDocument(tt.input)
			if err != nil {
				t.Fatalf("NewDocument: %v", err)
			}

			eval := implDoc.NewEvaluator()
			if err := eval.Evaluate(doc); err != nil {
				t.Fatalf("Evaluate: %v", err)
			}

			// Format with text formatter
			formatter := format.GetFormatter("text", "")
			var buf bytes.Buffer
			opts := format.Options{
				DisplayFormatter: df,
			}
			if err := formatter.Format(&buf, doc, opts); err != nil {
				t.Fatalf("Format: %v", err)
			}

			got := strings.TrimSpace(buf.String())
			if got != tt.want {
				t.Errorf("locale=%s, input=%q: got %q, want %q", tt.locale, tt.input, got, tt.want)
			}
		})
	}
}
