package display

import (
	"testing"
	"time"

	"golang.org/x/text/language"
)

func TestGetDateLocale(t *testing.T) {
	t.Run("English", func(t *testing.T) {
		loc := getDateLocale(language.AmericanEnglish)
		if loc.shortDays[0] != "Sun" {
			t.Errorf("expected Sun, got %s", loc.shortDays[0])
		}
		if len(loc.shortMonths) != 12 {
			t.Errorf("expected 12 months, got %d", len(loc.shortMonths))
		}
	})

	t.Run("German", func(t *testing.T) {
		loc := getDateLocale(language.German)
		if loc.shortDays[3] != "Mi." {
			t.Errorf("expected Mi., got %s", loc.shortDays[3])
		}
		if loc.shortMonths[0] != "Jan." {
			t.Errorf("expected Jan., got %s", loc.shortMonths[0])
		}
	})

	t.Run("French", func(t *testing.T) {
		loc := getDateLocale(language.French)
		if loc.shortDays[1] != "lun." {
			t.Errorf("expected lun., got %s", loc.shortDays[1])
		}
		if loc.shortMonths[0] != "janv." {
			t.Errorf("expected janv., got %s", loc.shortMonths[0])
		}
	})

	t.Run("unsupported locale falls back to English", func(t *testing.T) {
		loc := getDateLocale(language.Japanese)
		if loc.shortDays[0] != "Sun" {
			t.Errorf("expected English fallback Sun, got %s", loc.shortDays[0])
		}
	})
}

func TestFormatDate_Ordering(t *testing.T) {
	// Wed, Jan 12, 2025 — a known reference date
	ref := time.Date(2025, time.January, 12, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		tag      language.Tag
		expected string
	}{
		{"en-US: month-day-year", language.AmericanEnglish, "Sun, Jan 12, 2025"},
		{"de-DE: day-month-year", language.German, "So. 12. Jan. 2025"},
		{"fr-FR: day-month-year", language.French, "dim. 12 janv. 2025"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc := getDateLocale(tt.tag)
			got := formatDate(ref, loc)
			if got != tt.expected {
				t.Errorf("formatDate() = %q, want %q", got, tt.expected)
			}
		})
	}
}
