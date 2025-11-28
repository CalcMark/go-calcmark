// Package features provides a searchable catalog of CalcMark features
// for help systems, autocompletion, and documentation generation.
package features

import (
	"slices"
	"strings"

	"github.com/CalcMark/go-calcmark/spec/units"
)

// Category represents a type of CalcMark feature.
type Category string

const (
	CategoryFunction    Category = "function"
	CategoryUnit        Category = "unit"
	CategoryKeyword     Category = "keyword"
	CategoryOperator    Category = "operator"
	CategoryDate        Category = "date"
	CategoryNetwork     Category = "network"
	CategoryStorage     Category = "storage"
	CategoryCompression Category = "compression"
)

// Feature represents a single CalcMark feature that users can discover.
type Feature struct {
	Name        string   // Primary name (e.g., "avg", "meter", "today")
	Category    Category // Type of feature
	Syntax      string   // Usage syntax (e.g., "avg(a, b, c)")
	Description string   // Human-readable description
	Aliases     []string // Alternative names/spellings
	Example     string   // Example usage
}

// Match checks if a query matches this feature (case-insensitive prefix match).
func (f Feature) Match(query string) bool {
	if query == "" {
		return false
	}
	query = strings.ToLower(query)
	if strings.HasPrefix(strings.ToLower(f.Name), query) {
		return true
	}
	for _, alias := range f.Aliases {
		if strings.HasPrefix(strings.ToLower(alias), query) {
			return true
		}
	}
	return false
}

// Registry holds all discoverable CalcMark features.
type Registry struct {
	features []Feature
}

// NewRegistry creates a registry with all CalcMark features.
func NewRegistry() *Registry {
	r := &Registry{}
	r.features = append(r.features, getFunctions()...)
	r.features = append(r.features, getUnits()...)
	r.features = append(r.features, getDateFeatures()...)
	r.features = append(r.features, getNetworkFeatures()...)
	r.features = append(r.features, getStorageFeatures()...)
	r.features = append(r.features, getCompressionFeatures()...)
	r.features = append(r.features, getKeywords()...)
	r.features = append(r.features, getOperators()...)
	return r
}

// Search finds features matching a query string (prefix match on name or aliases).
func (r *Registry) Search(query string) []Feature {
	if query == "" {
		return nil
	}
	var matches []Feature
	for _, f := range r.features {
		if f.Match(query) {
			matches = append(matches, f)
		}
	}
	// Sort by name for consistent ordering
	slices.SortFunc(matches, func(a, b Feature) int {
		return strings.Compare(a.Name, b.Name)
	})
	return matches
}

// ByCategory returns all features of a specific category.
func (r *Registry) ByCategory(cat Category) []Feature {
	var matches []Feature
	for _, f := range r.features {
		if f.Category == cat {
			matches = append(matches, f)
		}
	}
	slices.SortFunc(matches, func(a, b Feature) int {
		return strings.Compare(a.Name, b.Name)
	})
	return matches
}

// All returns all features.
func (r *Registry) All() []Feature {
	result := make([]Feature, len(r.features))
	copy(result, r.features)
	slices.SortFunc(result, func(a, b Feature) int {
		return strings.Compare(a.Name, b.Name)
	})
	return result
}

// Categories returns all available category names.
func (r *Registry) Categories() []Category {
	seen := make(map[Category]bool)
	for _, f := range r.features {
		seen[f.Category] = true
	}
	cats := make([]Category, 0, len(seen))
	for c := range seen {
		cats = append(cats, c)
	}
	slices.Sort(cats)
	return cats
}

// getFunctions returns built-in function features.
func getFunctions() []Feature {
	return []Feature{
		{
			Name:        "avg",
			Category:    CategoryFunction,
			Syntax:      "avg(a, b, c, ...)",
			Description: "Calculate the average of numbers",
			Aliases:     []string{"average", "average of"},
			Example:     "avg(10, 20, 30) → 20",
		},
		{
			Name:        "sqrt",
			Category:    CategoryFunction,
			Syntax:      "sqrt(n)",
			Description: "Calculate the square root",
			Aliases:     []string{"square root of"},
			Example:     "sqrt(16) → 4",
		},
		{
			Name:        "accumulate",
			Category:    CategoryFunction,
			Syntax:      "accumulate(rate, time)",
			Description: "Calculate total from a rate over time",
			Aliases:     []string{},
			Example:     "accumulate(100 req/s, 1 hour) → 360000 req",
		},
		{
			Name:        "convert_rate",
			Category:    CategoryFunction,
			Syntax:      "convert_rate(rate, unit)",
			Description: "Convert a rate to a different time unit",
			Aliases:     []string{},
			Example:     "convert_rate(1000 req/s, minute) → 60000 req/min",
		},
		{
			Name:        "requires",
			Category:    CategoryFunction,
			Syntax:      "requires(load, capacity) or requires(load, capacity, buffer)",
			Description: "Calculate how many units needed for a given load",
			Aliases:     []string{"capacity"},
			Example:     "requires(10000 req/s, 500 req/s) → 20",
		},
		{
			Name:        "downtime",
			Category:    CategoryFunction,
			Syntax:      "downtime(availability, period)",
			Description: "Calculate downtime from availability percentage",
			Aliases:     []string{},
			Example:     "downtime(99.9%, month) → 43.2 minutes",
		},
		{
			Name:        "rtt",
			Category:    CategoryFunction,
			Syntax:      "rtt(scope)",
			Description: "Network round-trip time for a scope",
			Aliases:     []string{"round trip time"},
			Example:     "rtt(regional) → 10 ms",
		},
		{
			Name:        "throughput",
			Category:    CategoryFunction,
			Syntax:      "throughput(network_type)",
			Description: "Network bandwidth for a connection type",
			Aliases:     []string{},
			Example:     "throughput(gigabit) → 125 MB/s",
		},
		{
			Name:        "transfer_time",
			Category:    CategoryFunction,
			Syntax:      "transfer_time(size, scope, network)",
			Description: "Time to transfer data over a network",
			Aliases:     []string{},
			Example:     "transfer_time(1 GB, regional, gigabit)",
		},
		{
			Name:        "read",
			Category:    CategoryFunction,
			Syntax:      "read(size, storage_type)",
			Description: "Time to read data from storage",
			Aliases:     []string{},
			Example:     "read(100 MB, ssd) → 0.18 s",
		},
		{
			Name:        "seek",
			Category:    CategoryFunction,
			Syntax:      "seek(storage_type)",
			Description: "Access latency for storage type",
			Aliases:     []string{},
			Example:     "seek(hdd) → 10 ms",
		},
		{
			Name:        "compress",
			Category:    CategoryFunction,
			Syntax:      "compress(size, algorithm)",
			Description: "Estimate compressed data size",
			Aliases:     []string{},
			Example:     "compress(1 GB, gzip) → 333 MB",
		},
	}
}

// getUnits returns unit features from the canonical units registry.
func getUnits() []Feature {
	// Deduplicate by canonical name
	seen := make(map[string]bool)
	var features []Feature

	for _, unit := range units.StandardUnits {
		if seen[unit.Canonical] {
			continue
		}
		seen[unit.Canonical] = true

		features = append(features, Feature{
			Name:        unit.Canonical,
			Category:    CategoryUnit,
			Syntax:      unit.Symbol,
			Description: unit.Description,
			Aliases:     unit.Aliases,
			Example:     "10 " + unit.Symbol,
		})
	}
	return features
}

// getDateFeatures returns date-related features.
func getDateFeatures() []Feature {
	return []Feature{
		{
			Name:        "today",
			Category:    CategoryDate,
			Syntax:      "today",
			Description: "Current date",
			Aliases:     []string{},
			Example:     "today + 7 days",
		},
		{
			Name:        "tomorrow",
			Category:    CategoryDate,
			Syntax:      "tomorrow",
			Description: "Tomorrow's date",
			Aliases:     []string{},
			Example:     "tomorrow + 1 week",
		},
		{
			Name:        "yesterday",
			Category:    CategoryDate,
			Syntax:      "yesterday",
			Description: "Yesterday's date",
			Aliases:     []string{},
			Example:     "yesterday - 3 days",
		},
		{
			Name:        "days",
			Category:    CategoryDate,
			Syntax:      "N days",
			Description: "Duration in days",
			Aliases:     []string{"day"},
			Example:     "today + 30 days",
		},
		{
			Name:        "weeks",
			Category:    CategoryDate,
			Syntax:      "N weeks",
			Description: "Duration in weeks",
			Aliases:     []string{"week"},
			Example:     "2 weeks from today",
		},
		{
			Name:        "months",
			Category:    CategoryDate,
			Syntax:      "N months",
			Description: "Duration in months",
			Aliases:     []string{"month"},
			Example:     "Dec 25 + 1 month",
		},
		{
			Name:        "years",
			Category:    CategoryDate,
			Syntax:      "N years",
			Description: "Duration in years",
			Aliases:     []string{"year", "yr", "yrs"},
			Example:     "today + 1 year",
		},
		{
			Name:        "from",
			Category:    CategoryDate,
			Syntax:      "N units from date",
			Description: "Calculate date offset",
			Aliases:     []string{},
			Example:     "7 days from Dec 25",
		},
	}
}

// getNetworkFeatures returns network-related features.
func getNetworkFeatures() []Feature {
	return []Feature{
		// RTT scopes
		{
			Name:        "local",
			Category:    CategoryNetwork,
			Syntax:      "rtt(local)",
			Description: "Same datacenter latency (~0.5ms)",
			Aliases:     []string{},
			Example:     "rtt(local) → 0.5 ms",
		},
		{
			Name:        "regional",
			Category:    CategoryNetwork,
			Syntax:      "rtt(regional)",
			Description: "Same region latency (~10ms)",
			Aliases:     []string{},
			Example:     "rtt(regional) → 10 ms",
		},
		{
			Name:        "continental",
			Category:    CategoryNetwork,
			Syntax:      "rtt(continental)",
			Description: "Cross-continent latency (~50ms)",
			Aliases:     []string{},
			Example:     "rtt(continental) → 50 ms",
		},
		{
			Name:        "global",
			Category:    CategoryNetwork,
			Syntax:      "rtt(global)",
			Description: "Global latency (~150ms)",
			Aliases:     []string{},
			Example:     "rtt(global) → 150 ms",
		},
		// Throughput types
		{
			Name:        "gigabit",
			Category:    CategoryNetwork,
			Syntax:      "throughput(gigabit)",
			Description: "1 Gbps network (~125 MB/s)",
			Aliases:     []string{},
			Example:     "throughput(gigabit) → 125 MB/s",
		},
		{
			Name:        "ten_gig",
			Category:    CategoryNetwork,
			Syntax:      "throughput(ten_gig)",
			Description: "10 Gbps network (~1.25 GB/s)",
			Aliases:     []string{},
			Example:     "throughput(ten_gig) → 1250 MB/s",
		},
		{
			Name:        "hundred_gig",
			Category:    CategoryNetwork,
			Syntax:      "throughput(hundred_gig)",
			Description: "100 Gbps network (~12.5 GB/s)",
			Aliases:     []string{},
			Example:     "throughput(hundred_gig) → 12500 MB/s",
		},
		{
			Name:        "wifi",
			Category:    CategoryNetwork,
			Syntax:      "throughput(wifi)",
			Description: "Typical WiFi (~12.5 MB/s)",
			Aliases:     []string{},
			Example:     "throughput(wifi) → 12.5 MB/s",
		},
		{
			Name:        "four_g",
			Category:    CategoryNetwork,
			Syntax:      "throughput(four_g)",
			Description: "4G mobile network (~2.5 MB/s)",
			Aliases:     []string{},
			Example:     "throughput(four_g) → 2.5 MB/s",
		},
		{
			Name:        "five_g",
			Category:    CategoryNetwork,
			Syntax:      "throughput(five_g)",
			Description: "5G mobile network (~50 MB/s)",
			Aliases:     []string{},
			Example:     "throughput(five_g) → 50 MB/s",
		},
	}
}

// getStorageFeatures returns storage-related features.
func getStorageFeatures() []Feature {
	return []Feature{
		{
			Name:        "ssd",
			Category:    CategoryStorage,
			Syntax:      "read(size, ssd) or seek(ssd)",
			Description: "SATA SSD (~550 MB/s, 0.1ms seek)",
			Aliases:     []string{"sata_ssd"},
			Example:     "read(1 GB, ssd)",
		},
		{
			Name:        "nvme",
			Category:    CategoryStorage,
			Syntax:      "read(size, nvme) or seek(nvme)",
			Description: "NVMe SSD (~3.5 GB/s, 0.01ms seek)",
			Aliases:     []string{},
			Example:     "read(1 GB, nvme)",
		},
		{
			Name:        "pcie_ssd",
			Category:    CategoryStorage,
			Syntax:      "read(size, pcie_ssd) or seek(pcie_ssd)",
			Description: "PCIe Gen4 SSD (~7 GB/s, 0.01ms seek)",
			Aliases:     []string{},
			Example:     "read(1 GB, pcie_ssd)",
		},
		{
			Name:        "hdd",
			Category:    CategoryStorage,
			Syntax:      "read(size, hdd) or seek(hdd)",
			Description: "7200 RPM HDD (~150 MB/s, 10ms seek)",
			Aliases:     []string{},
			Example:     "seek(hdd) → 10 ms",
		},
	}
}

// getCompressionFeatures returns compression-related features.
func getCompressionFeatures() []Feature {
	return []Feature{
		{
			Name:        "gzip",
			Category:    CategoryCompression,
			Syntax:      "compress(size, gzip)",
			Description: "Gzip compression (~3:1 ratio)",
			Aliases:     []string{},
			Example:     "compress(1 GB, gzip) → 333 MB",
		},
		{
			Name:        "lz4",
			Category:    CategoryCompression,
			Syntax:      "compress(size, lz4)",
			Description: "LZ4 fast compression (~2:1 ratio)",
			Aliases:     []string{},
			Example:     "compress(1 GB, lz4) → 500 MB",
		},
		{
			Name:        "zstd",
			Category:    CategoryCompression,
			Syntax:      "compress(size, zstd)",
			Description: "Zstandard compression (~3.5:1 ratio)",
			Aliases:     []string{},
			Example:     "compress(1 GB, zstd) → 286 MB",
		},
		{
			Name:        "bzip2",
			Category:    CategoryCompression,
			Syntax:      "compress(size, bzip2)",
			Description: "Bzip2 compression (~4:1 ratio, slow)",
			Aliases:     []string{},
			Example:     "compress(1 GB, bzip2) → 250 MB",
		},
		{
			Name:        "snappy",
			Category:    CategoryCompression,
			Syntax:      "compress(size, snappy)",
			Description: "Snappy fast compression (~2.5:1 ratio)",
			Aliases:     []string{},
			Example:     "compress(1 GB, snappy) → 400 MB",
		},
	}
}

// getKeywords returns language keywords.
func getKeywords() []Feature {
	return []Feature{
		{
			Name:        "in",
			Category:    CategoryKeyword,
			Syntax:      "value in unit",
			Description: "Convert to a different unit",
			Aliases:     []string{},
			Example:     "100 cm in inches → 39.37 inches",
		},
		{
			Name:        "as",
			Category:    CategoryKeyword,
			Syntax:      "value as unit",
			Description: "Convert to a different unit (alias for 'in')",
			Aliases:     []string{},
			Example:     "1 mile as km → 1.609 km",
		},
		{
			Name:        "of",
			Category:    CategoryKeyword,
			Syntax:      "X% of value",
			Description: "Calculate percentage of a value",
			Aliases:     []string{},
			Example:     "15% of 200 → 30",
		},
		{
			Name:        "per",
			Category:    CategoryKeyword,
			Syntax:      "value per unit",
			Description: "Create a rate",
			Aliases:     []string{},
			Example:     "1000 requests per second",
		},
	}
}

// getOperators returns operator features.
func getOperators() []Feature {
	return []Feature{
		{
			Name:        "+",
			Category:    CategoryOperator,
			Syntax:      "a + b",
			Description: "Addition",
			Aliases:     []string{"plus", "add"},
			Example:     "10 + 5 → 15",
		},
		{
			Name:        "-",
			Category:    CategoryOperator,
			Syntax:      "a - b",
			Description: "Subtraction",
			Aliases:     []string{"minus", "subtract"},
			Example:     "10 - 5 → 5",
		},
		{
			Name:        "*",
			Category:    CategoryOperator,
			Syntax:      "a * b",
			Description: "Multiplication",
			Aliases:     []string{"times", "multiply"},
			Example:     "10 * 5 → 50",
		},
		{
			Name:        "/",
			Category:    CategoryOperator,
			Syntax:      "a / b",
			Description: "Division",
			Aliases:     []string{"divide", "divided by"},
			Example:     "10 / 5 → 2",
		},
		{
			Name:        "^",
			Category:    CategoryOperator,
			Syntax:      "a ^ b",
			Description: "Exponentiation",
			Aliases:     []string{"power", "to the power of"},
			Example:     "2 ^ 10 → 1024",
		},
		{
			Name:        "%",
			Category:    CategoryOperator,
			Syntax:      "a % b or N%",
			Description: "Modulo or percentage",
			Aliases:     []string{"mod", "percent"},
			Example:     "10 % 3 → 1, 50% → 0.5",
		},
	}
}
