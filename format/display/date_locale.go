package display

import (
	"time"

	"github.com/goodsign/monday"
	"golang.org/x/text/language"
)

// dateFormat pairs a Go time layout with a monday locale identifier.
// The layout controls element ordering (month-day vs day-month),
// and monday handles translating day/month names to the target language.
type dateFormat struct {
	layout string
	locale monday.Locale
}

// dateFormats maps BCP 47 locale strings to their date format configuration.
// Keys use the underscore format that monday expects (e.g., "en_US").
// Looked up by tagToMondayLocale, which handles fallback.
var dateFormats = map[string]dateFormat{
	// Month-day ordering
	"en_US": {layout: "Mon, Jan 2, 2006", locale: monday.LocaleEnUS},
	"en_GB": {layout: "Mon, 2 Jan 2006", locale: monday.LocaleEnGB},

	// Day-month ordering (most European languages)
	"de_DE": {layout: "Mon. 2. Jan. 2006", locale: monday.LocaleDeDE},
	"fr_FR": {layout: "Mon. 2 Jan. 2006", locale: monday.LocaleFrFR},
	"es_ES": {layout: "Mon. 2 Jan. 2006", locale: monday.LocaleEsES},
	"it_IT": {layout: "Mon 2 Jan 2006", locale: monday.LocaleItIT},
	"pt_BR": {layout: "Mon, 2 Jan. 2006", locale: monday.LocalePtBR},
	"pt_PT": {layout: "Mon, 2 Jan. 2006", locale: monday.LocalePtPT},
	"nl_NL": {layout: "Mon 2 Jan. 2006", locale: monday.LocaleNlNL},
	"da_DK": {layout: "Mon. 2. Jan. 2006", locale: monday.LocaleDaDK},
	"sv_SE": {layout: "Mon 2 Jan. 2006", locale: monday.LocaleSvSE},
	"nb_NO": {layout: "Mon. 2. Jan. 2006", locale: monday.LocaleNbNO},
	"fi_FI": {layout: "Mon 2. Jan. 2006", locale: monday.LocaleFiFI},
	"pl_PL": {layout: "Mon, 2 Jan 2006", locale: monday.LocalePlPL},
	"ru_RU": {layout: "Mon, 2 Jan. 2006", locale: monday.LocaleRuRU},
	"uk_UA": {layout: "Mon, 2 Jan. 2006", locale: monday.LocaleUkUA},
	"tr_TR": {layout: "Mon, 2 Jan 2006", locale: monday.LocaleTrTR},
	"ja_JP": {layout: "2006年1月2日(Mon)", locale: monday.LocaleJaJP},
	"zh_CN": {layout: "2006年1月2日 Mon", locale: monday.LocaleZhCN},
	"zh_TW": {layout: "2006年1月2日 Mon", locale: monday.LocaleZhTW},
	"ko_KR": {layout: "2006년 1월 2일 (Mon)", locale: monday.LocaleKoKR},
}

// defaultDateFormat is used when the user's locale has no specific entry.
var defaultDateFormat = dateFormats["en_US"]

// getDateFormat returns the date format configuration for the given language tag.
// Falls back to en-US if the locale is not in the table.
func getDateFormat(tag language.Tag) dateFormat {
	// Try exact match first: "en_US", "de_DE"
	key := tagToMondayKey(tag)
	if fmt, ok := dateFormats[key]; ok {
		return fmt
	}

	// Try language-only fallback: "fr" -> "fr_FR", "de" -> "de_DE"
	base, _ := tag.Base()
	for k, fmt := range dateFormats {
		if len(k) >= 2 && k[:2] == base.String() {
			return fmt
		}
	}

	return defaultDateFormat
}

// tagToMondayKey converts a BCP 47 language.Tag to a monday-style locale key.
// e.g., "en-US" -> "en_US", "de-DE" -> "de_DE"
func tagToMondayKey(tag language.Tag) string {
	base, _ := tag.Base()
	region, _ := tag.Region()
	if region.String() == "ZZ" {
		return base.String()
	}
	return base.String() + "_" + region.String()
}

// formatDate formats a time.Time using locale-aware short date format.
func formatDate(t time.Time, df dateFormat) string {
	return monday.Format(t, df.layout, df.locale)
}
