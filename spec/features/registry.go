// Package features provides a searchable catalog of CalcMark features
// for help systems, autocompletion, and documentation generation.
package features

import (
	"slices"
	"strings"

	"github.com/CalcMark/go-calcmark/spec/identifiers"
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
	CategoryGrowth      Category = "growth"
)

// Alias represents an alternative name for a feature, with a flag indicating
// whether it can be used as input syntax (Parseable) or is search-only.
type Alias struct {
	Name      string // Alternative name (e.g., "average of", "round trip time")
	Parseable bool   // true if this alias works as input syntax, false if search-only
	Example   string // Concrete NL example for autosuggest (e.g., "average of 1, 2, 3")
}

// Feature represents a single CalcMark feature that users can discover.
type Feature struct {
	Name        string   // Primary name (e.g., "avg", "meter", "today")
	Category    Category // Type of feature
	Syntax      string   // Usage syntax (e.g., "avg(a, b, c)")
	Description string   // Human-readable description
	Aliases     []Alias  // Alternative names/spellings
	Example     string   // Example usage (function-call form for help display)
	NLExample   string   // Natural language example for autosuggest (e.g., "100 MB/s over 1 day")
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
		if strings.HasPrefix(strings.ToLower(alias.Name), query) {
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
	r.features = append(r.features, getGrowthFeatures()...)
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
			Aliases:     []Alias{{Name: "average", Parseable: false}, {Name: "mean", Parseable: false}, {Name: "average of", Parseable: true, Example: "average of 1, 2, 3"}},
			Example:     "avg(10, 20, 30) → 20",
		},
		{
			Name:        "sqrt",
			Category:    CategoryFunction,
			Syntax:      "sqrt(n)",
			Description: "Calculate the square root",
			Aliases:     []Alias{{Name: "square root of", Parseable: true, Example: "square root of 16"}},
			Example:     "sqrt(16) → 4",
		},
		{
			Name:        "accumulate",
			Category:    CategoryFunction,
			Syntax:      "accumulate(rate, time)",
			Description: "Calculate total from a rate over time",
			Aliases:     nil,
			Example:     "accumulate(100 req/s, 1 hour) → 360000 req",
			NLExample:   "100 MB/s over 1 day",
		},
		{
			Name:        "convert_rate",
			Category:    CategoryFunction,
			Syntax:      "convert_rate(rate, unit)",
			Description: "Convert a rate to a different time unit",
			Aliases:     nil,
			Example:     "convert_rate(1000 req/s, minute) → 60000 req/min",
			NLExample:   "5 MB/s per minute",
		},
		{
			Name:        "capacity",
			Category:    CategoryFunction,
			Syntax:      "capacity(demand, capacity_per_unit, unit) or capacity(demand, capacity_per_unit, unit, buffer)",
			Description: "Calculate how many units needed for a given load",
			Aliases:     []Alias{{Name: "requires", Parseable: false}},
			Example:     "capacity(10000 req/s, 500 req/s, server) → 20 servers",
			NLExample:   "10 TB at 2 TB per disk",
		},
		{
			Name:        "downtime",
			Category:    CategoryFunction,
			Syntax:      "downtime(availability, period)",
			Description: "Calculate downtime from availability percentage",
			Aliases:     nil,
			Example:     "downtime(99.9%, month) → 43.2 minutes",
			NLExample:   "99.9% downtime per month",
		},
		{
			Name:        "rtt",
			Category:    CategoryFunction,
			Syntax:      "rtt(scope)",
			Description: "Network round-trip time for a scope",
			Aliases:     []Alias{{Name: "round trip time", Parseable: false}},
			Example:     "rtt(regional) → 10 ms",
		},
		{
			Name:        "throughput",
			Category:    CategoryFunction,
			Syntax:      "throughput(network_type)",
			Description: "Network bandwidth for a connection type",
			Aliases:     nil,
			Example:     "throughput(gigabit) → 125 MB/s",
		},
		{
			Name:        "transfer_time",
			Category:    CategoryFunction,
			Syntax:      "transfer_time(size, scope, network)",
			Description: "Time to transfer data over a network",
			Aliases:     []Alias{{Name: "transfer...across", Parseable: true, Example: "transfer 1 GB across regional gigabit"}},
			Example:     "transfer 1 GB across regional gigabit",
		},
		{
			Name:        "read",
			Category:    CategoryFunction,
			Syntax:      "read(size, storage_type)",
			Description: "Time to read data from storage",
			Aliases:     []Alias{{Name: "read...from", Parseable: true, Example: "read 100 MB from ssd"}},
			Example:     "read 100 MB from ssd",
		},
		{
			Name:        "seek",
			Category:    CategoryFunction,
			Syntax:      "seek(storage_type)",
			Description: "Access latency for storage type",
			Aliases:     nil,
			Example:     "seek(hdd) → 10 ms",
		},
		{
			Name:        "compress",
			Category:    CategoryFunction,
			Syntax:      "compress(size, algorithm)",
			Description: "Estimate compressed data size",
			Aliases:     []Alias{{Name: "compress...using", Parseable: true, Example: "compress 1 GB using gzip"}},
			Example:     "compress 1 GB using gzip",
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

		aliases := make([]Alias, len(unit.Aliases))
		for i, a := range unit.Aliases {
			aliases[i] = Alias{Name: a, Parseable: true}
		}
		features = append(features, Feature{
			Name:        unit.Canonical,
			Category:    CategoryUnit,
			Syntax:      unit.Symbol,
			Description: unit.Description,
			Aliases:     aliases,
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
			Aliases:     nil,
			Example:     "today + 7 days",
		},
		{
			Name:        "tomorrow",
			Category:    CategoryDate,
			Syntax:      "tomorrow",
			Description: "Tomorrow's date",
			Aliases:     nil,
			Example:     "tomorrow + 1 week",
		},
		{
			Name:        "yesterday",
			Category:    CategoryDate,
			Syntax:      "yesterday",
			Description: "Yesterday's date",
			Aliases:     nil,
			Example:     "yesterday - 3 days",
		},
		{
			Name:        "days",
			Category:    CategoryDate,
			Syntax:      "N days",
			Description: "Duration in days",
			Aliases:     []Alias{{Name: "day", Parseable: true}},
			Example:     "today + 30 days",
		},
		{
			Name:        "weeks",
			Category:    CategoryDate,
			Syntax:      "N weeks",
			Description: "Duration in weeks",
			Aliases:     []Alias{{Name: "week", Parseable: true}},
			Example:     "2 weeks from today",
		},
		{
			Name:        "months",
			Category:    CategoryDate,
			Syntax:      "N months",
			Description: "Duration in months",
			Aliases:     []Alias{{Name: "month", Parseable: true}},
			Example:     "Dec 25 + 1 month",
		},
		{
			Name:        "years",
			Category:    CategoryDate,
			Syntax:      "N years",
			Description: "Duration in years",
			Aliases:     []Alias{{Name: "year", Parseable: true}, {Name: "yr", Parseable: true}, {Name: "yrs", Parseable: true}},
			Example:     "today + 1 year",
		},
		{
			Name:        "from",
			Category:    CategoryDate,
			Syntax:      "N units from date",
			Description: "Calculate date offset",
			Aliases:     nil,
			Example:     "7 days from Dec 25",
		},
	}
}

// networkScopeMeta holds presentation metadata for network scope identifiers.
type networkScopeMeta struct {
	Description string
	Example     string
}

// networkTypeMeta holds presentation metadata for network type identifiers.
type networkTypeMeta struct {
	Description string
	Example     string
}

var networkScopes = map[string]networkScopeMeta{
	"local":       {Description: "Same datacenter latency (~0.5ms)", Example: "rtt(local) → 0.5 ms"},
	"regional":    {Description: "Same region latency (~10ms)", Example: "rtt(regional) → 10 ms"},
	"continental": {Description: "Cross-continent latency (~50ms)", Example: "rtt(continental) → 50 ms"},
	"global":      {Description: "Global latency (~150ms)", Example: "rtt(global) → 150 ms"},
}

var networkTypes = map[string]networkTypeMeta{
	"gigabit":     {Description: "1 Gbps network (~125 MB/s)", Example: "throughput(gigabit) → 125 MB/s"},
	"ten_gig":     {Description: "10 Gbps network (~1.25 GB/s)", Example: "throughput(ten_gig) → 1250 MB/s"},
	"hundred_gig": {Description: "100 Gbps network (~12.5 GB/s)", Example: "throughput(hundred_gig) → 12500 MB/s"},
	"wifi":        {Description: "Typical WiFi (~12.5 MB/s)", Example: "throughput(wifi) → 12.5 MB/s"},
	"four_g":      {Description: "4G mobile network (~2.5 MB/s)", Example: "throughput(four_g) → 2.5 MB/s"},
	"five_g":      {Description: "5G mobile network (~50 MB/s)", Example: "throughput(five_g) → 50 MB/s"},
}

// getNetworkFeatures returns network-related features derived from canonical identifiers.
func getNetworkFeatures() []Feature {
	var features []Feature

	// RTT scopes — iterate canonical slice to preserve ordering
	for _, name := range identifiers.NetworkScopes {
		meta := networkScopes[name]
		features = append(features, Feature{
			Name:        name,
			Category:    CategoryNetwork,
			Syntax:      "rtt(" + name + ")",
			Description: meta.Description,
			Example:     meta.Example,
		})
	}

	// Throughput types
	for _, name := range identifiers.NetworkTypes {
		meta := networkTypes[name]
		features = append(features, Feature{
			Name:        name,
			Category:    CategoryNetwork,
			Syntax:      "throughput(" + name + ")",
			Description: meta.Description,
			Example:     meta.Example,
		})
	}

	return features
}

// storageMeta holds presentation metadata for storage type identifiers.
type storageMeta struct {
	Description string
	Example     string
}

var storageTypes = map[string]storageMeta{
	"ssd":      {Description: "SATA SSD (~550 MB/s, 0.1ms seek)", Example: "read(1 GB, ssd)"},
	"nvme":     {Description: "NVMe SSD (~3.5 GB/s, 0.01ms seek)", Example: "read(1 GB, nvme)"},
	"pcie_ssd": {Description: "PCIe Gen4 SSD (~7 GB/s, 0.01ms seek)", Example: "read(1 GB, pcie_ssd)"},
	"hdd":      {Description: "7200 RPM HDD (~150 MB/s, 10ms seek)", Example: "seek(hdd) → 10 ms"},
}

// getStorageFeatures returns storage-related features derived from canonical identifiers.
func getStorageFeatures() []Feature {
	var features []Feature

	for _, name := range identifiers.StorageTypes {
		meta := storageTypes[name]

		// Build aliases from the canonical StorageAliases map
		var aliases []Alias
		for alias, canonical := range identifiers.StorageAliases {
			if canonical == name {
				aliases = append(aliases, Alias{Name: alias, Parseable: true})
			}
		}

		features = append(features, Feature{
			Name:        name,
			Category:    CategoryStorage,
			Syntax:      "read(size, " + name + ") or seek(" + name + ")",
			Description: meta.Description,
			Aliases:     aliases,
			Example:     meta.Example,
		})
	}

	return features
}

// compressionMeta holds presentation metadata for compression type identifiers.
type compressionMeta struct {
	Description string
	Example     string
}

var compressionTypes = map[string]compressionMeta{
	"gzip":   {Description: "Gzip compression (~3:1 ratio)", Example: "compress(1 GB, gzip) → 333 MB"},
	"zstd":   {Description: "Zstandard compression (~3.5:1 ratio)", Example: "compress(1 GB, zstd) → 286 MB"},
	"lz4":    {Description: "LZ4 fast compression (~2:1 ratio)", Example: "compress(1 GB, lz4) → 500 MB"},
	"snappy": {Description: "Snappy fast compression (~2.5:1 ratio)", Example: "compress(1 GB, snappy) → 400 MB"},
	"bzip2":  {Description: "Bzip2 compression (~4:1 ratio, slow)", Example: "compress(1 GB, bzip2) → 250 MB"},
	"none":   {Description: "No compression (1:1 ratio)", Example: "compress(1 GB, none) → 1 GB"},
}

// getCompressionFeatures returns compression-related features derived from canonical identifiers.
func getCompressionFeatures() []Feature {
	var features []Feature

	for _, name := range identifiers.CompressionTypes {
		meta := compressionTypes[name]
		features = append(features, Feature{
			Name:        name,
			Category:    CategoryCompression,
			Syntax:      "compress(size, " + name + ")",
			Description: meta.Description,
			Example:     meta.Example,
		})
	}

	return features
}

// getGrowthFeatures returns growth and depreciation function features.
func getGrowthFeatures() []Feature {
	return []Feature{
		{
			Name:        "compound",
			Category:    CategoryFunction,
			Syntax:      "compound(principal, rate, periods)",
			Description: "Calculate compound growth over time periods",
			Aliases: []Alias{
				{Name: "compound...by...over", Parseable: true, Example: "compound $1000 by 5% over 10 years"},
			},
			Example: "compound(1000, 5%, 10) → 1628.89",
		},
		{
			Name:        "grow",
			Category:    CategoryFunction,
			Syntax:      "grow(amount, increment, periods)",
			Description: "Calculate linear growth by adding a fixed amount each period",
			Aliases: []Alias{
				{Name: "grow...by...over", Parseable: true, Example: "grow 100 by 20 over 5 months"},
			},
			Example: "grow(100, 20 GB, 5) → 200 GB",
		},
		{
			Name:        "depreciate",
			Category:    CategoryFunction,
			Syntax:      "depreciate(value, rate, periods, salvage?)",
			Description: "Calculate declining balance depreciation over time",
			Aliases: []Alias{
				{Name: "depreciate...by...over...to", Parseable: true, Example: "depreciate $50000 by 15% over 5 years to $5000"},
			},
			Example: "depreciate(10000, 20%, 5) → 3276.80",
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
			Aliases:     nil,
			Example:     "100 cm in inches → 39.37 inches",
		},
		{
			Name:        "as",
			Category:    CategoryKeyword,
			Syntax:      "value as unit",
			Description: "Convert to a different unit (alias for 'in')",
			Aliases:     nil,
			Example:     "1 mile as km → 1.609 km",
		},
		{
			Name:        "of",
			Category:    CategoryKeyword,
			Syntax:      "X% of value",
			Description: "Calculate percentage of a value",
			Aliases:     nil,
			Example:     "15% of 200 → 30",
		},
		{
			Name:        "per",
			Category:    CategoryKeyword,
			Syntax:      "value per unit",
			Description: "Create a rate",
			Aliases:     nil,
			Example:     "1000 requests per second",
		},
		{
			Name:        "over",
			Category:    CategoryKeyword,
			Syntax:      "rate over duration",
			Description: "Accumulate a rate over time",
			Aliases:     nil,
			Example:     "100 MB/s over 1 day",
		},
		{
			Name:        "as napkin",
			Category:    CategoryKeyword,
			Syntax:      "expression as napkin",
			Description: "Round to 2 significant figures for estimates",
			Aliases:     []Alias{{Name: "napkin", Parseable: false}},
			Example:     "432000 MB as napkin → ~400 GB",
		},
		{
			Name:        "at",
			Category:    CategoryKeyword,
			Syntax:      "demand at capacity per unit [with N% buffer]",
			Description: "Capacity planning: calculate how many units needed",
			Aliases:     nil,
			Example:     "10 TB at 2 TB per disk → 5 disks",
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
			Aliases:     []Alias{{Name: "plus", Parseable: false}, {Name: "add", Parseable: false}},
			Example:     "10 + 5 → 15",
		},
		{
			Name:        "-",
			Category:    CategoryOperator,
			Syntax:      "a - b",
			Description: "Subtraction",
			Aliases:     []Alias{{Name: "minus", Parseable: false}, {Name: "subtract", Parseable: false}},
			Example:     "10 - 5 → 5",
		},
		{
			Name:        "*",
			Category:    CategoryOperator,
			Syntax:      "a * b",
			Description: "Multiplication",
			Aliases:     []Alias{{Name: "times", Parseable: false}, {Name: "multiply", Parseable: false}},
			Example:     "10 * 5 → 50",
		},
		{
			Name:        "/",
			Category:    CategoryOperator,
			Syntax:      "a / b",
			Description: "Division",
			Aliases:     []Alias{{Name: "divide", Parseable: false}, {Name: "divided by", Parseable: false}},
			Example:     "10 / 5 → 2",
		},
		{
			Name:        "^",
			Category:    CategoryOperator,
			Syntax:      "a ^ b",
			Description: "Exponentiation",
			Aliases:     []Alias{{Name: "power", Parseable: false}, {Name: "to the power of", Parseable: false}},
			Example:     "2 ^ 10 → 1024",
		},
		{
			Name:        "%",
			Category:    CategoryOperator,
			Syntax:      "a % b or N%",
			Description: "Modulo or percentage",
			Aliases:     []Alias{{Name: "mod", Parseable: false}, {Name: "percent", Parseable: false}},
			Example:     "10 % 3 → 1, 50% → 0.5",
		},
	}
}
