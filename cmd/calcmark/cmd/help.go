package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/units"
	"github.com/spf13/cobra"
)

var helpCmd = &cobra.Command{
	Use:   "help [topic]",
	Short: "Display help for CalcMark topics",
	Long: `Display help for CalcMark topics.

Available topics:
  functions   List all CalcMark functions with descriptions and usage
  constants   List all built-in unit constants

Examples:
  cm help functions     Show all available functions
  cm help constants     Show all unit constants`,
	Run: func(cmd *cobra.Command, args []string) {
		// When run without subcommand, show available topics
		fmt.Println("CalcMark Help")
		fmt.Println()
		fmt.Println("Available topics:")
		fmt.Println("  functions   List all CalcMark functions")
		fmt.Println("  constants   List all built-in unit constants")
		fmt.Println()
		fmt.Println("Use \"cm help <topic>\" for more information.")
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

func init() {
	rootCmd.AddCommand(helpCmd)
	helpCmd.AddCommand(helpFunctionsCmd)
	helpCmd.AddCommand(helpConstantsCmd)
}

// printFunctions prints all functions grouped by category.
func printFunctions() {
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "CalcMark Functions")
	fmt.Fprintln(w)

	// Get functions by category
	byCategory := interpreter.GetFunctionsByCategory()

	// Define category order for consistent output
	categoryOrder := []string{"Math", "Conversion", "Network", "Storage", "Capacity"}

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
			fmt.Fprintln(w)
		}
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
