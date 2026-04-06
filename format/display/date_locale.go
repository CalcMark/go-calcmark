package display

import (
	"fmt"
	"time"

	"golang.org/x/text/language"
)

// dateLocale holds locale-specific abbreviated day/month names and a format
// function that encodes the locale's conventional date element ordering.
type dateLocale struct {
	shortDays   [7]string  // Sun=0 .. Sat=6 (matches time.Weekday)
	shortMonths [12]string // Jan=0 .. Dec=11 (month-1 index)
	format      func(weekday, month string, day, year int) string
}

// dateLocales maps language base tags to locale-specific date data.
// Keyed by language.Base.String() (e.g., "en", "de", "fr").
var dateLocales = map[string]dateLocale{
	"en": {
		shortDays:   [7]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"},
		shortMonths: [12]string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"},
		format: func(weekday, month string, day, year int) string {
			return fmt.Sprintf("%s, %s %d, %d", weekday, month, day, year)
		},
	},
	"de": {
		shortDays:   [7]string{"So.", "Mo.", "Di.", "Mi.", "Do.", "Fr.", "Sa."},
		shortMonths: [12]string{"Jan.", "Feb.", "März", "Apr.", "Mai", "Juni", "Juli", "Aug.", "Sep.", "Okt.", "Nov.", "Dez."},
		format: func(weekday, month string, day, year int) string {
			return fmt.Sprintf("%s %d. %s %d", weekday, day, month, year)
		},
	},
	"fr": {
		shortDays:   [7]string{"dim.", "lun.", "mar.", "mer.", "jeu.", "ven.", "sam."},
		shortMonths: [12]string{"janv.", "févr.", "mars", "avr.", "mai", "juin", "juil.", "août", "sept.", "oct.", "nov.", "déc."},
		format: func(weekday, month string, day, year int) string {
			return fmt.Sprintf("%s %d %s %d", weekday, day, month, year)
		},
	},
}

// getDateLocale returns the dateLocale for the given language tag.
// Falls back to English if the language is not in the table.
func getDateLocale(tag language.Tag) dateLocale {
	base, _ := tag.Base()
	if loc, ok := dateLocales[base.String()]; ok {
		return loc
	}
	return dateLocales["en"]
}

// formatDate formats a time.Time using the locale's short date format.
func formatDate(t time.Time, loc dateLocale) string {
	weekday := loc.shortDays[t.Weekday()]
	month := loc.shortMonths[t.Month()-1]
	return loc.format(weekday, month, t.Day(), t.Year())
}
