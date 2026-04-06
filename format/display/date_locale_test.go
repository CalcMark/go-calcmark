package display

import (
	"testing"
	"time"

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
}

func TestFormatDate_Locales(t *testing.T) {
	// Thu, Dec 25, 2025 — a known reference date
	ref := time.Date(2025, time.December, 25, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		tag      language.Tag
		expected string
	}{
		{"en-US", language.AmericanEnglish, "Thu, Dec 25, 2025"},
		{"en-GB", language.BritishEnglish, "Thu, 25 Dec 2025"},
		{"de-DE", language.German, "Do. 25. Dez. 2025"},
		{"fr-FR", language.French, "jeu. 25 déc. 2025"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			df := getDateFormat(tt.tag)
			got := formatDate(ref, df)
			if got != tt.expected {
				t.Errorf("formatDate() = %q, want %q", got, tt.expected)
			}
		})
	}
}
