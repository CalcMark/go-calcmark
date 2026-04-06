package display

import (
	"testing"
	"time"

	"github.com/CalcMark/go-calcmark/spec/types"
	"golang.org/x/text/language"
)

func TestGetDateFormat(t *testing.T) {
	t.Run("exact match en-US", func(t *testing.T) {
		df := getDateFormat(language.AmericanEnglish)
		if df.locale != "en_US" {
			t.Errorf("expected en_US locale, got %s", df.locale)
		}
	})

	t.Run("exact match de-DE", func(t *testing.T) {
		df := getDateFormat(language.German)
		if df.locale != "de_DE" {
			t.Errorf("expected de_DE locale, got %s", df.locale)
		}
	})

	t.Run("language-only fallback fr -> fr_FR", func(t *testing.T) {
		tag := language.French
		df := getDateFormat(tag)
		if df.locale != "fr_FR" {
			t.Errorf("expected fr_FR locale for French, got %s", df.locale)
		}
	})

	t.Run("unsupported locale falls back to en-US", func(t *testing.T) {
		tag, _ := language.Parse("hi-IN")
		df := getDateFormat(tag)
		if df.locale != "en_US" {
			t.Errorf("expected en_US fallback, got %s", df.locale)
		}
	})

	t.Run("monday-supported locale not in dateFormats falls back gracefully", func(t *testing.T) {
		// hu-HU is supported by monday but not in our dateFormats map
		tag, _ := language.Parse("hu-HU")
		df := getDateFormat(tag)
		if df.locale != "en_US" {
			t.Errorf("expected en_US fallback for hu-HU, got %s", df.locale)
		}
	})
}

// TestFormatDate_AllLocales verifies every locale entry in dateFormats produces
// the expected output for a known reference date (Thu, Dec 25, 2025).
// These expected values are the actual output from monday — if a layout string
// is wrong, the assertion catches it.
func TestFormatDate_AllLocales(t *testing.T) {
	ref := time.Date(2025, time.December, 25, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		locale   string
		expected string
	}{
		{"en-US", "Thu, Dec 25, 2025"},
		{"en-GB", "Thu, 25 Dec 2025"},
		{"de-DE", "Do. 25. Dez. 2025"},
		{"fr-FR", "jeu. 25 déc. 2025"},
		{"es-ES", "jue. 25 dic. 2025"},
		{"it-IT", "gio 25 dic 2025"},
		{"pt-BR", "qui, 25 dez. 2025"},
		{"pt-PT", "qui, 25 Dez. 2025"},
		{"nl-NL", "do 25 dec. 2025"},
		{"da-DK", "tor. 25. dec. 2025"},
		{"sv-SE", "tors 25 dec. 2025"},
		{"nb-NO", "to. 25. des. 2025"},
		{"fi-FI", "to 25. joulu. 2025"},
		{"pl-PL", "Czw, 25 gru 2025"},
		{"ru-RU", "Чт, 25 дек. 2025"},
		{"uk-UA", "Чт, 25 гру. 2025"},
		{"tr-TR", "Per, 25 Ara 2025"},
		{"ja-JP", "2025年12月25日(木)"},
		{"zh-CN", "2025年12月25日 四"},
		{"zh-TW", "2025年12月25日 四"},
		{"ko-KR", "2025년 12월 25일 (목)"},
	}

	for _, tt := range tests {
		t.Run(tt.locale, func(t *testing.T) {
			tag, err := language.Parse(tt.locale)
			if err != nil {
				t.Fatalf("language.Parse(%q): %v", tt.locale, err)
			}
			df := getDateFormat(tag)
			got := formatDate(ref, df)
			if got != tt.expected {
				t.Errorf("formatDate(%s) = %q, want %q", tt.locale, got, tt.expected)
			}
		})
	}
}

// TestFormatDate_EndToEnd_EnGB verifies en-GB flows correctly through
// the config -> formatter -> FormatDate pipeline.
func TestFormatDate_EndToEnd_EnGB(t *testing.T) {
	cfg, err := NewConfig("en-GB")
	if err != nil {
		t.Fatalf("NewConfig(en-GB): %v", err)
	}
	f := NewFormatter(cfg)

	d, _ := types.NewDate(2025, 12, 25)
	got := f.FormatDate(d)
	if got != "Thu, 25 Dec 2025" {
		t.Errorf("FormatDate(en-GB) = %q, want %q", got, "Thu, 25 Dec 2025")
	}
}
