package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/features"
	"github.com/CalcMark/go-calcmark/spec/units"
	"github.com/spf13/cobra"
)

var (
	helpShowFunctions   bool
	helpShowConstants   bool
	helpShowFrontmatter bool
)

var helpCmd = &cobra.Command{
	Use:   "help [topic]",
	Short: "Show CalcMark reference information",
	Long: `Display CalcMark functions, constants, and frontmatter directives.

Topics:
  functions    All built-in functions with descriptions and usage patterns
  constants    All unit constants grouped by quantity type
  frontmatter  All frontmatter directives with valid options and examples

Flags can filter output: --functions, --constants, --frontmatter
With no topic or flags, all sections are shown.`,
	// Accept arbitrary args so Cobra doesn't error on unknown subcommand names.
	// This lets `cm help eval` still work via Cobra's default help routing.
	Args:               cobra.ArbitraryArgs,
	DisableFlagParsing: false,
	Run: func(cmd *cobra.Command, args []string) {
		// If args provided but not a known topic, delegate to Cobra's
		// default help behavior (show help for that subcommand).
		if len(args) > 0 {
			// Check if arg is a known root subcommand
			sub, _, err := rootCmd.Find(args)
			if err == nil && sub != rootCmd {
				sub.Help()
				return
			}
		}

		showFuncs := helpShowFunctions
		showConsts := helpShowConstants
		showFM := helpShowFrontmatter

		// If no flags set, show all
		if !showFuncs && !showConsts && !showFM {
			showFuncs, showConsts, showFM = true, true, true
		}

		if showFuncs {
			printFunctions()
		}
		if showConsts {
			if showFuncs {
				fmt.Println()
			}
			printConstants()
		}
		if showFM {
			if showFuncs || showConsts {
				fmt.Println()
			}
			printFrontmatter()
		}
	},
}

var helpFunctionsCmd = &cobra.Command{
	Use:   "functions",
	Short: "List all CalcMark functions",
	Long:  `List all CalcMark functions with descriptions, synonyms, and usage patterns.`,
	Run: func(cmd *cobra.Command, args []string) {
		printFunctions()
	},
}

var helpConstantsCmd = &cobra.Command{
	Use:   "constants",
	Short: "List all built-in constants",
	Long:  `List all built-in unit constants with descriptions and aliases.`,
	Run: func(cmd *cobra.Command, args []string) {
		printConstants()
	},
}

var helpFrontmatterCmd = &cobra.Command{
	Use:   "frontmatter",
	Short: "List all frontmatter directives",
	Long:  `List all frontmatter directives with valid options, sub-keys, defaults, and YAML examples.`,
	Run: func(cmd *cobra.Command, args []string) {
		printFrontmatter()
	},
}

func init() {
	helpCmd.Flags().BoolVar(&helpShowFunctions, "functions", false, "Show functions only")
	helpCmd.Flags().BoolVar(&helpShowConstants, "constants", false, "Show constants only")
	helpCmd.Flags().BoolVar(&helpShowFrontmatter, "frontmatter", false, "Show frontmatter directives only")

	helpCmd.AddCommand(helpFunctionsCmd)
	helpCmd.AddCommand(helpConstantsCmd)
	helpCmd.AddCommand(helpFrontmatterCmd)

	rootCmd.SetHelpCommand(helpCmd)
}

// printFunctions prints all functions grouped by category.
func printFunctions() {
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "CalcMark Functions")
	fmt.Fprintln(w)

	// Build NL alias lookup from the spec/features registry
	nlAliases := buildNLAliasMap()

	// Get functions by category
	byCategory := interpreter.GetFunctionsByCategory()

	// Derive category order from registered functions (no hardcoded list to drift)
	categoryOrder := interpreter.GetCategoryOrder()

	for _, category := range categoryOrder {
		functions, ok := byCategory[category]
		if !ok || len(functions) == 0 {
			continue
		}

		fmt.Fprintf(w, "%s\n", category)
		fmt.Fprintln(w, strings.Repeat("-", len(category)))

		for _, fn := range functions {
			// Print function name with synonyms
			name := fn.Name
			if len(fn.Synonyms) > 0 {
				name = fmt.Sprintf("%s (%s)", fn.Name, strings.Join(fn.Synonyms, ", "))
			}
			fmt.Fprintf(w, "  %s\n", name)
			fmt.Fprintf(w, "    %s\n", fn.Description)
			fmt.Fprintf(w, "    Usage: %s\n", fn.Signature)

			// Print NL syntax if available
			if aliases, ok := nlAliases[fn.Name]; ok {
				for _, nl := range aliases {
					fmt.Fprintf(w, "    NL:    %s\n", nl)
				}
			}
			fmt.Fprintln(w)
		}
	}
}

// buildNLAliasMap returns NL usage examples for each function name.
// Combines parseable aliases from the spec/features registry with
// keyword-based alternatives (over, at...per, per).
func buildNLAliasMap() map[string][]string {
	registry := features.NewRegistry()
	result := make(map[string][]string)

	// NL examples for functions with parseable aliases in the registry
	for _, f := range registry.ByCategory(features.CategoryFunction) {
		for _, alias := range f.Aliases {
			if alias.Parseable {
				result[f.Name] = append(result[f.Name], nlExample(f.Name, alias.Name))
			}
		}
	}

	// Keyword-based NL forms not represented as aliases
	result["accumulate"] = append(result["accumulate"], "100 MB/s over 1 day")
	result["convert_rate"] = append(result["convert_rate"], "1000 req/s per minute")
	result["capacity"] = append(result["capacity"], "10 TB at 2 TB per disk")

	return result
}

// nlExample returns a concrete NL usage example for a function alias.
func nlExample(funcName, alias string) string {
	switch alias {
	case "average of":
		return "average of 10, 20, 30"
	case "square root of":
		return "square root of 16"
	case "read...from":
		return "read 100 MB from ssd"
	case "compress...using":
		return "compress 1 GB using gzip"
	case "transfer...across":
		return "transfer 1 GB across regional gigabit"
	default:
		return alias
	}
}

// printConstants prints all unit constants grouped by quantity type.
func printConstants() {
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "CalcMark Unit Constants")
	fmt.Fprintln(w)

	// Collect unique units (dedupe by Canonical name)
	seen := make(map[string]bool)
	var unitList []units.UnitMapping
	for _, u := range units.StandardUnits {
		if !seen[u.Canonical] {
			unitList = append(unitList, u)
			seen[u.Canonical] = true
		}
	}

	// Sort by Quantity then Canonical name
	sort.Slice(unitList, func(i, j int) bool {
		if unitList[i].Quantity != unitList[j].Quantity {
			return unitList[i].Quantity < unitList[j].Quantity
		}
		return unitList[i].Canonical < unitList[j].Canonical
	})

	// Group by Quantity
	byQuantity := make(map[string][]units.UnitMapping)
	for _, u := range unitList {
		byQuantity[u.Quantity] = append(byQuantity[u.Quantity], u)
	}

	// Get sorted quantity names
	quantities := make([]string, 0, len(byQuantity))
	for q := range byQuantity {
		quantities = append(quantities, q)
	}
	sort.Strings(quantities)

	for _, quantity := range quantities {
		unitGroup := byQuantity[quantity]

		fmt.Fprintf(w, "%s\n", quantity)
		fmt.Fprintln(w, strings.Repeat("-", len(quantity)))

		for _, u := range unitGroup {
			// Print canonical name with symbol
			name := fmt.Sprintf("%s (%s)", u.Canonical, u.Symbol)
			fmt.Fprintf(w, "  %s\n", name)
			fmt.Fprintf(w, "    %s\n", u.Description)

			// Print aliases (excluding canonical and symbol which are already shown)
			aliases := filterAliases(u.Aliases, u.Canonical, u.Symbol)
			if len(aliases) > 0 {
				fmt.Fprintf(w, "    Aliases: %s\n", strings.Join(aliases, ", "))
			}
			fmt.Fprintln(w)
		}
	}
}

// printFrontmatter prints all frontmatter directives with verbose YAML examples.
func printFrontmatter() {
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "CalcMark Frontmatter Directives")
	fmt.Fprintln(w)

	categories := units.Categories()

	// exchange
	fmt.Fprintln(w, "exchange")
	fmt.Fprintln(w, "--------")
	fmt.Fprintln(w, "  Define currency conversion rates.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "    exchange:")
	fmt.Fprintln(w, "      USD_EUR: 0.92")
	fmt.Fprintln(w, "      EUR_GBP: 0.86")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Keys: FROM_TO format, 3-letter ISO 4217 codes")
	fmt.Fprintln(w, "  Values: positive numbers (exchange rate)")
	fmt.Fprintln(w)

	// globals
	fmt.Fprintln(w, "globals")
	fmt.Fprintln(w, "-------")
	fmt.Fprintln(w, "  Define document-wide variables.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "    globals:")
	fmt.Fprintln(w, "      tax_rate: 0.32")
	fmt.Fprintln(w, "      base_price: $100")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Values are CalcMark expressions evaluated before the document body.")
	fmt.Fprintln(w, "  Reference in expressions with @globals.name (e.g., @globals.tax_rate).")
	fmt.Fprintln(w)

	// scale
	fmt.Fprintln(w, "scale")
	fmt.Fprintln(w, "-----")
	fmt.Fprintln(w, "  Multiply quantity results by a factor. Requires unit_categories.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Simple form:")
	fmt.Fprintln(w, "    scale: 2")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Map form (filter by category):")
	fmt.Fprintln(w, "    scale:")
	fmt.Fprintln(w, "      factor: 4")
	fmt.Fprintln(w, "      unit_categories: [Length, Mass]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Options:")
	fmt.Fprintln(w, "    factor           Number (required). Multiplier for quantities.")
	fmt.Fprintln(w, "    unit_categories  List. Categories to scale (required for scaling to occur).")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Valid categories:")
	for _, cat := range categories {
		fmt.Fprintf(w, "    - %s\n", cat)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Scaling is explicit: nothing scales without unit_categories.")
	fmt.Fprintln(w, "  Use [All] to scale every category.")
	fmt.Fprintln(w, "  Custom scales units not in the standard library (e.g., bananas, eggs).")
	fmt.Fprintln(w, "  Each category only scales when explicitly listed in unit_categories.")
	fmt.Fprintln(w, "  Reference the scale factor in expressions with @scale.")
	fmt.Fprintln(w)

	// convert_to
	fmt.Fprintln(w, "convert_to")
	fmt.Fprintln(w, "----------")
	fmt.Fprintln(w, "  Convert quantity results to a measurement system.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Simple form:")
	fmt.Fprintln(w, "    convert_to: si")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Map form (filter by category):")
	fmt.Fprintln(w, "    convert_to:")
	fmt.Fprintln(w, "      system: imperial")
	fmt.Fprintln(w, "      unit_categories: [Length]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Options:")
	fmt.Fprintln(w, "    system           si or imperial (required).")
	fmt.Fprintln(w, "    unit_categories  List (optional). Limit conversion to these categories.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Valid categories:")
	for _, cat := range categories {
		fmt.Fprintf(w, "    - %s\n", cat)
	}
	fmt.Fprintln(w)
}

// filterAliases returns aliases excluding the canonical name and symbol.
func filterAliases(aliases []string, canonical, symbol string) []string {
	result := make([]string, 0, len(aliases))
	for _, a := range aliases {
		if a != canonical && a != symbol {
			result = append(result, a)
		}
	}
	return result
}
