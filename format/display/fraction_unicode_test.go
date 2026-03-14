package display

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/types"
)

func TestFormatFractionUnicode(t *testing.T) {
	tests := []struct {
		name  string
		num   int64
		denom int64
		want  string
	}{
		// Simple fractions with Unicode equivalents
		{"1/2", 1, 2, "½"},
		{"1/3", 1, 3, "⅓"},
		{"2/3", 2, 3, "⅔"},
		{"1/4", 1, 4, "¼"},
		{"3/4", 3, 4, "¾"},
		{"1/8", 1, 8, "⅛"},
		{"3/8", 3, 8, "⅜"},
		{"5/8", 5, 8, "⅝"},
		{"7/8", 7, 8, "⅞"},

		// Mixed numbers with Unicode
		{"7/3 → 2⅓", 7, 3, "2⅓"},
		{"3/2 → 1½", 3, 2, "1½"},
		{"11/8 → 1⅜", 11, 8, "1⅜"},

		// Fractions without Unicode → ASCII fallback
		{"7/12 no unicode", 7, 12, "7/12"},
		{"5/11 no unicode", 5, 11, "5/11"},

		// Mixed number without Unicode fraction part → ASCII fallback
		{"19/12 → 1 7/12", 19, 12, "1 7/12"},

		// Integer result
		{"6/3 → 2", 6, 3, "2"},
		{"0/1 → 0", 0, 1, "0"},

		// Negative
		{"-1/3", -1, 3, "-⅓"},
		{"-7/3 → -2⅓", -7, 3, "-2⅓"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := types.NewFraction(tt.num, tt.denom)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := FormatFractionUnicode(f)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatFractionUnicodeWithUnit(t *testing.T) {
	f, _ := types.NewFraction(1, 3)
	f.Unit = "cup"
	got := FormatFractionUnicode(f)
	if got != "⅓ cup" {
		t.Errorf("got %q, want \"⅓ cup\"", got)
	}
}

func TestFormatFractionUnicodeNapkin(t *testing.T) {
	f, _ := types.NewFraction(2, 3)
	f.IsNapkin = true
	got := FormatFractionUnicode(f)
	if got != "~⅔" {
		t.Errorf("got %q, want \"~⅔\"", got)
	}
}

func TestFormatterUnicodeFlagRouting(t *testing.T) {
	f, _ := types.NewFraction(1, 3)

	// ASCII mode (default, for JSON/CLI)
	asciiFormatter := DefaultFormatter()
	got := asciiFormatter.Format(f)
	if got != "1/3" {
		t.Errorf("ASCII mode: got %q, want \"1/3\"", got)
	}

	// Unicode mode (TUI)
	cfg := DefaultConfig()
	cfg.UnicodeFractions = true
	unicodeFormatter := NewFormatter(cfg)
	got = unicodeFormatter.Format(f)
	if got != "⅓" {
		t.Errorf("Unicode mode: got %q, want \"⅓\"", got)
	}
}
