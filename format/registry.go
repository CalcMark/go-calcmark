package format

import (
	"path/filepath"
	"strings"
)

// Global formatter registry
var formatters = map[string]Formatter{
	"text": &TextFormatter{},
	"cm":   &CalcMarkFormatter{},
	"json": &JSONFormatter{},
	"html": &HTMLFormatter{},
	"md":   &MarkdownFormatter{},
}

// GetFormatter returns the appropriate formatter based on format name or filename extension.
// If format is specified, it takes precedence. Otherwise, the filename extension is used.
// Falls back to text formatter if no match is found.
func GetFormatter(format string, filename string) Formatter {
	// Explicit format takes precedence
	if format != "" {
		if f, ok := formatters[format]; ok {
			return f
		}
		// Unknown format falls back to default
		return formatters["text"]
	}

	// Auto-detect from extension
	if filename != "" {
		ext := strings.ToLower(filepath.Ext(filename))
		if ext != "" {
			for _, fmt := range formatters {
				for _, fmtExt := range fmt.Extensions() {
					if ext == fmtExt {
						return fmt
					}
				}
			}
		}
	}

	// Default to text formatter
	return formatters["text"]
}

// RegisterFormatter adds a custom formatter to the registry.
// This allows third-party formatters to be registered at runtime.
func RegisterFormatter(name string, formatter Formatter) {
	formatters[name] = formatter
}
