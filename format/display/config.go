package display

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

// maxLocaleLen is the maximum allowed length for a locale string.
// Prevents pathological input from reaching language.Parse().
const maxLocaleLen = 64

// DisplayConfig holds locale-specific formatting preferences.
// It is a value type designed to be stack-allocatable.
type DisplayConfig struct {
	// Tag is the parsed BCP 47 language tag.
	Tag language.Tag

	// DecimalSep is the decimal point character (e.g., "." for en-US, "," for de-DE).
	DecimalSep string

	// ThousandSep is the thousands grouping separator (e.g., "," for en-US, "." for de-DE).
	ThousandSep string

	// UnicodeFractions enables Unicode Number Forms (e.g., ½, ⅓, ¾) for fraction display.
	// Only used in TUI; JSON and CLI output always uses ASCII fractions.
	UnicodeFractions bool
}

// DefaultConfig returns the en-US display configuration.
func DefaultConfig() DisplayConfig {
	return DisplayConfig{
		Tag:         language.AmericanEnglish,
		DecimalSep:  ".",
		ThousandSep: ",",
	}
}

// NewConfig creates a DisplayConfig from a locale string (e.g., "en-US", "de-DE").
// Returns an error if the locale is invalid. Use DefaultConfig() for en-US defaults.
func NewConfig(locale string) (DisplayConfig, error) {
	// Security: length bound before any parsing
	if len(locale) > maxLocaleLen {
		return DisplayConfig{}, fmt.Errorf("display: locale string too long (%d bytes, max %d)", len(locale), maxLocaleLen)
	}

	// Security: ASCII-only validation before language.Parse()
	for i := range len(locale) {
		if locale[i] > 127 {
			return DisplayConfig{}, fmt.Errorf("display: locale must be ASCII-only, got non-ASCII byte at position %d", i)
		}
	}

	tag, err := language.Parse(locale)
	if err != nil {
		return DisplayConfig{}, fmt.Errorf("display: invalid locale %q: %w", locale, err)
	}

	decimalSep, thousandSep := extractSeparators(tag)

	cfg := DisplayConfig{
		Tag:         tag,
		DecimalSep:  decimalSep,
		ThousandSep: thousandSep,
	}

	if err := cfg.Validate(); err != nil {
		return DisplayConfig{}, err
	}

	return cfg, nil
}

// extractSeparators discovers locale-specific separators by formatting a probe value.
// Uses the public golang.org/x/text API to format 1234.5 and extract separators
// from known digit positions. This avoids depending on internal packages.
func extractSeparators(tag language.Tag) (decimalSep, thousandSep string) {
	p := message.NewPrinter(tag)
	formatted := p.Sprintf("%v", number.Decimal(1234.5))
	// Expected patterns:
	//   en-US: "1,234.5"
	//   de-DE: "1.234,5"
	//   fr-FR: "1\u202F234,5"  (narrow no-break space)

	// Extract thousand separator between '1' and '2'
	idx1 := strings.IndexByte(formatted, '1')
	idx2 := strings.IndexByte(formatted, '2')
	if idx1 >= 0 && idx2 > idx1+1 {
		thousandSep = formatted[idx1+1 : idx2]
	}

	// Extract decimal separator between '4' and '5'
	idx4 := strings.IndexByte(formatted, '4')
	idx5 := strings.LastIndexByte(formatted, '5')
	if idx4 >= 0 && idx5 > idx4+1 {
		decimalSep = formatted[idx4+1 : idx5]
	}

	// Fallbacks for edge cases
	if decimalSep == "" {
		decimalSep = "."
	}
	if thousandSep == "" {
		thousandSep = ","
	}

	return
}

// Validate checks that the DisplayConfig has valid, non-empty separators.
func (c DisplayConfig) Validate() error {
	if c.DecimalSep == "" {
		return errors.New("display: DecimalSep must not be empty")
	}
	if c.ThousandSep == "" {
		return errors.New("display: ThousandSep must not be empty")
	}
	if c.DecimalSep == c.ThousandSep {
		return errors.New("display: DecimalSep and ThousandSep must differ")
	}
	return nil
}
