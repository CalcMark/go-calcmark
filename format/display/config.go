package display

import "errors"

// DisplayConfig holds locale-specific formatting preferences.
// It is a value type (~56 bytes) designed to be stack-allocatable.
type DisplayConfig struct {
	// DecimalSep is the decimal point character (e.g., "." for en-US, "," for de-DE).
	DecimalSep string

	// ThousandSep is the thousands grouping separator (e.g., "," for en-US, "." for de-DE).
	ThousandSep string
}

// DefaultConfig returns the en-US display configuration.
func DefaultConfig() DisplayConfig {
	return DisplayConfig{
		DecimalSep:  ".",
		ThousandSep: ",",
	}
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
