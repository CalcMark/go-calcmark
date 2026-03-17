// Command docgen generates site/data/features.json from the features registry.
// It is invoked by `task generate-docs` and runs before Hugo builds the site.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/CalcMark/go-calcmark/spec/features"
)

// JSONFeature is the JSON representation of a single feature.
type JSONFeature struct {
	Name        string      `json:"name"`
	Category    string      `json:"category"`
	Subcategory string      `json:"subcategory,omitempty"` // Grouping within category (e.g., "Math", "Network")
	Quantity    string      `json:"quantity,omitempty"`     // Unit category (e.g., "Force", "Length"); empty for non-unit features
	Syntax      string      `json:"syntax"`
	Description string      `json:"description"`
	Aliases     []JSONAlias `json:"aliases,omitempty"`
	Example     string      `json:"example"`
	Anchor      string      `json:"anchor"`
}

// JSONAlias is the JSON representation of a feature alias.
type JSONAlias struct {
	Name      string `json:"name"`
	Parseable bool   `json:"parseable"`
}

// JSONOutput is the top-level JSON structure grouped by category.
type JSONOutput map[string][]JSONFeature

// kebabCase converts a feature name to a URL-safe anchor ID.
var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

func kebabCase(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = nonAlphanumeric.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func run() error {
	reg := features.NewRegistry()

	output := make(JSONOutput)
	for _, cat := range reg.Categories() {
		feats := reg.ByCategory(cat)
		jsonFeats := make([]JSONFeature, 0, len(feats))
		for _, f := range feats {
			jf := JSONFeature{
				Name:        f.Name,
				Category:    string(f.Category),
				Subcategory: f.Subcategory,
				Quantity:    f.Quantity,
				Syntax:      f.Syntax,
				Description: f.Description,
				Example:     f.Example,
				Anchor:      kebabCase(f.Name),
			}
			for _, a := range f.Aliases {
				jf.Aliases = append(jf.Aliases, JSONAlias{
					Name:      a.Name,
					Parseable: a.Parseable,
				})
			}
			jsonFeats = append(jsonFeats, jf)
		}
		output[string(cat)] = jsonFeats
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}

	outPath := filepath.Join("site", "data", "features.json")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}

	fmt.Printf("Generated %s (%d bytes, %d categories)\n", outPath, len(data), len(output))
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "docgen: %v\n", err)
		os.Exit(1)
	}
}
