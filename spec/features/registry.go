// Package features provides a searchable catalog of CalcMark features
// for help systems, autocompletion, and documentation generation.
package features

import (
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/CalcMark/go-calcmark/spec/identifiers"
	"github.com/CalcMark/go-calcmark/spec/types"
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
	CategoryFrontmatter Category = "frontmatter"
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
	Name        string            // Primary name (e.g., "avg", "meter", "today")
	Category    Category          // Type of feature (function, unit, keyword, etc.)
	Subcategory string            // Grouping within category for help display (e.g., "Math", "Network", "Storage")
	Quantity    string            // Unit category for units (e.g., "Force", "Length"); empty for non-unit features
	Syntax      string            // Usage syntax (e.g., "avg(a, b, c)")
	Description string            // Human-readable description
	Aliases     []Alias           // Alternative names/spellings
	Synonyms    []string          // Runtime-dispatchable alternative names (e.g., "average", "mean" for "avg")
	Params      []types.ParamSpec // Parameter specifications for functions (empty for non-functions)
	Example     string            // Example usage (function-call form for help display)
	NLExample   string            // Natural language example for autosuggest (e.g., "100 MB/s over 1 day")
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
	r.features = append(r.features, getFractionFeatures()...)
	r.features = append(r.features, getFrontmatterFeatures()...)
	return r
}

var (
	defaultRegistry     *Registry
	defaultRegistryOnce sync.Once
)

// DefaultRegistry returns a shared, read-only Registry instance.
// Use this for production code; use NewRegistry() in tests that need a fresh instance.
func DefaultRegistry() *Registry {
	defaultRegistryOnce.Do(func() {
		defaultRegistry = NewRegistry()
	})
	return defaultRegistry
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

// GetByName returns the first feature with the given name, or nil if not found.
// Returns a copy to prevent mutation of the shared DefaultRegistry() singleton.
func (r *Registry) GetByName(name string) *Feature {
	for i := range r.features {
		if r.features[i].Name == name {
			f := r.features[i]
			return &f
		}
	}
	return nil
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

// NLTriggerKeywords returns the set of keywords that begin NL function syntax.
// These are extracted from parseable aliases: the first word of each alias name
// (before "...") is the trigger keyword. Both the document detector and the
// line classifier use this to identify NL function lines without hard-coding.
func (r *Registry) NLTriggerKeywords() []string {
	seen := make(map[string]bool)
	for _, f := range r.features {
		for _, a := range f.Aliases {
			if !a.Parseable {
				continue
			}
			// NL aliases use "..." as separator: "compound...by...over"
			if before, _, ok := strings.Cut(a.Name, "..."); ok {
				kw := strings.ToLower(before)
				if kw != "" {
					seen[kw] = true
				}
			}
		}
	}
	result := make([]string, 0, len(seen))
	for kw := range seen {
		result = append(result, kw)
	}
	slices.Sort(result)
	return result
}

// getFunctions returns built-in function features.
// Params are pulled from types.FunctionSpecs — single source of truth for parameter metadata.
func getFunctions() []Feature {
	features := []Feature{
		{
			Name:        "avg",
			Category:    CategoryFunction,
			Subcategory: "Math",
			Syntax:      "avg(value1, value2, ...)",
			Description: "Calculate the average of values",
			Aliases:     []Alias{{Name: "average", Parseable: false}, {Name: "mean", Parseable: false}, {Name: "average of", Parseable: true, Example: "average of 1, 2, 3"}},
			Synonyms:    []string{"average", "mean"},
			Example:     "avg(10, 20, 30) → 20",
			NLExample:   "average of 1 kg, 2 kg, 3 kg",
		},
		{
			Name:        "sum",
			Category:    CategoryFunction,
			Subcategory: "Math",
			Syntax:      "sum(value1, value2, ...)",
			Description: "Calculate the sum of values",
			Aliases:     []Alias{{Name: "sum of", Parseable: true, Example: "sum of $100, $200, $300"}, {Name: "total", Parseable: false}},
			Example:     "sum($100, $200, $300) → $600",
			NLExample:   "sum of total_a, total_b, total_c",
		},
		{
			Name:        "sqrt",
			Category:    CategoryFunction,
			Subcategory: "Math",
			Syntax:      "sqrt(value)",
			Description: "Calculate the square root",
			Aliases:     []Alias{{Name: "square root of", Parseable: true, Example: "square root of 16"}},
			Example:     "sqrt(16) → 4",
		},
		{
			Name:        "number",
			Category:    CategoryFunction,
			Subcategory: "Math",
			Syntax:      "number(value)",
			Description: "Strip the unit or currency from a typed value, returning a plain number. " +
				"Use when you need a dimensionless ratio from two typed values: " +
				"number($500) / number($1000) → 0.5. " +
				"Only wrap what's needed — if a value is already a plain number, don't wrap it again. " +
				"Which side you wrap matters: $100 / number($50) → $2.00 (currency), " +
				"number($100) / number($50) → 2 (plain number).",
			Aliases: nil,
			Example: "number($500) / number($1000) → 0.5",
		},
		{
			Name:        "accumulate",
			Category:    CategoryFunction,
			Subcategory: "Math",
			Syntax:      "accumulate(rate, duration)",
			Description: "Total from a rate over time: rate × duration. Use for bandwidth, request volume, or throughput planning.",
			Aliases:     nil,
			Example:     "accumulate(100 req/s, 1 hour) → 360000 req",
			NLExample:   "100 MB/s over 1 day",
		},
		{
			Name:        "convert_rate",
			Category:    CategoryFunction,
			Subcategory: "Conversion",
			Syntax:      "convert_rate(rate, time_unit)",
			Description: "Convert a rate to a different time unit. " +
				"Use the 'per' keyword as a natural-language synonym: " +
				"d per year is equivalent to convert_rate(d, year). " +
				"Works with literal rates (5 MB/s per minute) and variables holding rates (r per hour). " +
				"Supports all time units including sub-second: nanosecond (ns), microsecond (μs/us), millisecond (ms).",
			Aliases: []Alias{
				{Name: "per", Parseable: true, Example: "d per year"},
			},
			Example:   "convert_rate(1000 req/s, minute) → 60000 req/min",
			NLExample: "5 MB/s per minute",
		},
		{
			Name:        "capacity",
			Category:    CategoryFunction,
			Subcategory: "Capacity",
			Syntax:      "capacity(demand, capacity_per_unit, unit, buffer?)",
			Description: "How many units to handle a load: ceil(demand / capacity_per_unit). Optional buffer adds headroom.",
			Aliases:     []Alias{{Name: "requires", Parseable: false}},
			Example:     "capacity(10000 req/s, 500 req/s, server) → 20 servers",
			NLExample:   "10 TB at 2 TB per disk",
		},
		{
			Name:        "downtime",
			Category:    CategoryFunction,
			Subcategory: "Network",
			Syntax:      "downtime(availability, duration)",
			Description: "Downtime from an SLA: (1 - availability) × time_period. E.g., 99.9% over a month → 43 minutes.",
			Aliases:     nil,
			Example:     "downtime(99.9%, month) → 43.2 minutes",
			NLExample:   "99.9% downtime per month",
		},
		{
			Name:        "rtt",
			Category:    CategoryFunction,
			Subcategory: "Network",
			Syntax:      "rtt(scope)",
			Description: "Typical network round-trip latency: local ~0.5ms, regional ~10ms, continental ~50ms, global ~150ms.",
			Aliases:     []Alias{{Name: "round trip time", Parseable: false}},
			Example:     "rtt(regional) → 10 ms",
		},
		{
			Name:        "throughput",
			Category:    CategoryFunction,
			Subcategory: "Network",
			Syntax:      "throughput(network_type)",
			Description: "Network bandwidth by type: wifi ~6 MB/s, gigabit ~125 MB/s, ten_gig ~1.25 GB/s.",
			Aliases:     nil,
			Example:     "throughput(gigabit) → 125 MB/s",
		},
		{
			Name:        "transfer_time",
			Category:    CategoryFunction,
			Subcategory: "Network",
			Syntax:      "transfer_time(size, scope, network_type)",
			Description: "Time to transfer data: size / throughput(network_type) + rtt(scope).",
			Aliases:     []Alias{{Name: "transfer...across", Parseable: true, Example: "transfer 1 GB across regional gigabit"}},
			Example:     "transfer 1 GB across regional gigabit",
		},
		{
			Name:        "read",
			Category:    CategoryFunction,
			Subcategory: "Storage",
			Syntax:      "read(size, storage_type)",
			Description: "Sequential read time: size / read speed. NVMe ~3 GB/s, SSD ~500 MB/s, HDD ~100 MB/s.",
			Aliases:     []Alias{{Name: "read...from", Parseable: true, Example: "read 100 MB from ssd"}},
			Example:     "read 100 MB from ssd",
		},
		{
			Name:        "seek",
			Category:    CategoryFunction,
			Subcategory: "Storage",
			Syntax:      "seek(storage_type)",
			Description: "Random access latency: NVMe ~0.1ms, SSD ~0.1ms, HDD ~10ms.",
			Aliases:     nil,
			Example:     "seek(hdd) → 10 ms",
		},
		{
			Name:        "compress",
			Category:    CategoryFunction,
			Subcategory: "Storage",
			Syntax:      "compress(size, compression_type)",
			Description: "Compressed output size using typical ratios: gzip ~3x, lz4 ~2x, zstd ~3.5x, snappy ~1.5x.",
			Aliases:     []Alias{{Name: "compress...using", Parseable: true, Example: "compress 1 GB using gzip"}},
			Example:     "compress 1 GB using gzip",
		},
	}

	// Populate Params from types.FunctionSpecs — single source of truth
	for i := range features {
		if spec, ok := types.FunctionSpecs[features[i].Name]; ok {
			features[i].Params = spec.Params
		}
	}
	return features
}

// unitExamples provides realistic conversion examples for each unit quantity.
// These are more useful than generic "10 <symbol>" examples.
var unitExamples = map[string]string{
	// Acceleration
	"m/s^2":            "1 standard-gravity in m/s^2",
	"cm/s^2":           "100 cm/s^2",
	"ft/s^2":           "32.17 ft/s^2",
	"standard-gravity": "1 standard-gravity",
	// Area
	"acre":              "5 acres in hectares",
	"hectare":           "100 hectares in acres",
	"square meter":      "200 sq m",
	"square kilometer":  "1 sq km in acres",
	"square centimeter": "500 sq cm",
	"square foot":       "1500 sq ft in sq m",
	"square inch":       "144 sq in",
	"square yard":       "100 sq yd",
	"square mile":       "1 sq mi in sq km",
	// Energy
	"joule":       "1000 joules in kj",
	"kilojoule":   "500 kj in kcal",
	"calorie":     "100 calories in joules",
	"kilocalorie": "2000 kcal",
	"kwh":         "1 kwh in joules",
	// Force
	"newton":         "10 newtons in pound-force",
	"kilonewton":     "1 kilonewton in newtons",
	"dyne":           "100000 dynes in newtons",
	"kilogram-force": "1 kilogram-force in newtons",
	"pound-force":    "5 pound-force in newtons",
	"poundal":        "10 poundals in newtons",
	// Frequency
	"hertz":     "440 hertz",
	"kilohertz": "100 kilohertz in megahertz",
	"megahertz": "2.4 megahertz",
	"gigahertz": "2.4 gigahertz in megahertz",
	"terahertz": "1 terahertz in gigahertz",
	// Impulse
	"newton-second":      "110 pound-force-seconds in newton-seconds",
	"pound-force-second": "110 pound-force-seconds in newton-seconds",
	// Length
	"meter":         "100 meters in feet",
	"millimeter":    "10 mm in inches",
	"centimeter":    "30 cm in inches",
	"kilometer":     "5 km in miles",
	"inch":          "12 inches in cm",
	"foot":          "6 feet in meters",
	"yard":          "100 yards in meters",
	"mile":          "26.2 miles in km",
	"nautical mile": "1 nautical mile in km",
	// Mass
	"gram":       "500 grams in ounces",
	"milligram":  "200 mg",
	"kilogram":   "80 kg in pounds",
	"metric ton": "1 tonne in pounds",
	"ounce":      "8 oz in grams",
	"pound":      "150 pounds in kg",
	"troy ounce": "1 troy oz in grams",
	"troy pound": "1 troy lb in grams",
	"short ton":  "1 short ton in kg",
	"long ton":   "1 long ton in kg",
	// Power
	"watt":       "100 watts in horsepower",
	"kilowatt":   "1.5 kw in watts",
	"megawatt":   "1 mw in kilowatts",
	"horsepower": "300 hp in kilowatts",
	// Pressure
	"pascal":          "101325 pascals in atmospheres",
	"kilopascal":      "100 kpa in bar",
	"megapascal":      "1 mpa in psi",
	"bar":             "1 bar in psi",
	"millibar":        "1013 millibars in atmospheres",
	"atmosphere":      "1 atmosphere in psi",
	"torr":            "760 torr in atmospheres",
	"psi":             "14.7 psi in atmospheres",
	"inch of mercury": "30 inhg in atmospheres",
	// Speed
	"m/s":  "100 kph in mps",
	"km/h": "100 kph in mph",
	"mph":  "60 mph in kph",
	"knot": "30 knots in kph",
	// Temperature
	"celsius":    "100 celsius in fahrenheit",
	"fahrenheit": "72 fahrenheit in celsius",
	"kelvin":     "273 kelvin in celsius",
	// Volume
	"milliliter":           "500 ml in cups",
	"liter":                "2 liters in gallons",
	"teaspoon":             "3 tsp in tbsp",
	"tablespoon":           "2 tbsp in ml",
	"cup":                  "2 cups in ml",
	"fluid ounce":          "8 fl oz in ml",
	"pint":                 "1 pint in ml",
	"quart":                "1 quart in liters",
	"gallon":               "1 gallon in liters",
	"imperial gallon":      "1 imp gal in liters",
	"imperial quart":       "1 imp qt in ml",
	"imperial pint":        "1 imp pt in ml",
	"imperial cup":         "1 imp cup in ml",
	"imperial fluid ounce": "1 imp fl oz in ml",
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

		example := "10 " + unit.Symbol
		if ex, ok := unitExamples[unit.Canonical]; ok {
			example = ex
		}

		features = append(features, Feature{
			Name:        unit.Canonical,
			Category:    CategoryUnit,
			Quantity:    unit.Quantity,
			Syntax:      unit.Symbol,
			Description: unit.Description,
			Aliases:     aliases,
			Example:     example,
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
		{
			Name:        "ago",
			Category:    CategoryDate,
			Syntax:      "N units ago",
			Description: "Date/time in the past",
			Aliases:     nil,
			Example:     "2 weeks ago",
		},
		{
			Name:        "next weekday",
			Category:    CategoryDate,
			Syntax:      "next <weekday>",
			Description: "The soonest future occurrence of a weekday",
			Aliases:     []Alias{{Name: "this weekday", Parseable: true}, {Name: "last weekday", Parseable: true}},
			Example:     "next Friday",
		},
		{
			Name:        "next month name",
			Category:    CategoryDate,
			Syntax:      "next <month>",
			Description: "First day of the next occurrence of a named month",
			Aliases:     []Alias{{Name: "this month name", Parseable: true}, {Name: "last month name", Parseable: true}},
			Example:     "next April",
		},
		{
			Name:        "this week",
			Category:    CategoryDate,
			Syntax:      "this week",
			Description: "First day of the current week (Monday)",
			Aliases:     []Alias{{Name: "next week", Parseable: true}, {Name: "last week", Parseable: true}},
			Example:     "this week + 2 days",
		},
		{
			Name:        "this month",
			Category:    CategoryDate,
			Syntax:      "this month",
			Description: "First day of the current calendar month",
			Aliases:     []Alias{{Name: "next month", Parseable: true}, {Name: "last month", Parseable: true}},
			Example:     "this month + 14 days",
		},
		{
			Name:        "this year",
			Category:    CategoryDate,
			Syntax:      "this year",
			Description: "First day of the current calendar year",
			Aliases:     []Alias{{Name: "next year", Parseable: true}, {Name: "last year", Parseable: true}},
			Example:     "this year + 6 months",
		},
		{
			Name:        "this quarter",
			Category:    CategoryDate,
			Syntax:      "this quarter",
			Description: "First day of the current calendar quarter",
			Aliases:     []Alias{{Name: "next quarter", Parseable: true}, {Name: "last quarter", Parseable: true}},
			Example:     "this quarter + 30 days",
		},
		{
			Name:        "fiscal quarter",
			Category:    CategoryDate,
			Syntax:      "this fiscal quarter",
			Description: "First day of the current fiscal quarter (requires fiscal_year_starts frontmatter)",
			Aliases:     []Alias{{Name: "next fiscal quarter", Parseable: true}, {Name: "last fiscal quarter", Parseable: true}},
			Example:     "this fiscal quarter",
		},
		{
			Name:        "fiscal year",
			Category:    CategoryDate,
			Syntax:      "this fiscal year",
			Description: "First day of the current fiscal year (requires fiscal_year_starts frontmatter)",
			Aliases:     []Alias{{Name: "next fiscal year", Parseable: true}, {Name: "last fiscal year", Parseable: true}},
			Example:     "this fiscal year",
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
	features := []Feature{
		{
			Name:        "compound",
			Category:    CategoryFunction,
			Subcategory: "Growth",
			Syntax:      "compound(principal, rate, periods, period?)",
			Description: "Compound growth: principal × (1 + rate)^periods. Add period for monthly/quarterly compounding.",
			Aliases: []Alias{
				{Name: "compound...by...over", Parseable: true, Example: "compound $1000 by 5% over 10 years"},
				{Name: "compound...by...monthly...over", Parseable: true, Example: "compound $1000 by 5% monthly over 10 years"},
			},
			Example:   "compound(1000, 5%, 10 years, monthly) → 1647.01",
			NLExample: "compound $1000 by 5% monthly over 10 years",
		},
		{
			Name:        "grow",
			Category:    CategoryFunction,
			Subcategory: "Growth",
			Syntax:      "grow(amount, increment, periods)",
			Description: "Linear growth: amount + (increment × periods). Use for additive scaling like hiring or storage growth.",
			Aliases: []Alias{
				{Name: "grow...by...over", Parseable: true, Example: "grow 100 by 20 over 5 months"},
			},
			Example: "grow(100, 20 GB, 5) → 200 GB",
		},
		{
			Name:        "depreciate",
			Category:    CategoryFunction,
			Subcategory: "Growth",
			Syntax:      "depreciate(value, rate, periods, salvage?)",
			Description: "Declining balance depreciation: value × (1 - rate)^periods. Optional salvage sets a floor value.",
			Aliases: []Alias{
				{Name: "depreciate...by...over...to", Parseable: true, Example: "depreciate $50000 by 15% over 5 years to $5000"},
			},
			Example: "depreciate(10000, 20%, 5) → 3276.80",
		},
	}

	for i := range features {
		if spec, ok := types.FunctionSpecs[features[i].Name]; ok {
			features[i].Params = spec.Params
		}
	}
	return features
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
			Name:        "rate widening",
			Category:    CategoryKeyword,
			Syntax:      "number * rate, quantity * rate",
			Description: "When a rate appears on the right of * or /, its time denominator is dropped and the amount is used. Rate on the left stays a rate (scaling). This is asymmetric: operand order determines the result type.",
			Aliases:     nil,
			Example:     "3 * (2 posts/week) → 6 posts",
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
			Name:        "as precise",
			Category:    CategoryKeyword,
			Syntax:      "expression as precise",
			Description: "Show full float precision, skipping display rounding",
			Aliases:     []Alias{{Name: "precise", Parseable: false}},
			Example:     "1 second as hour as precise",
		},
		{
			Name:        "as % of",
			Category:    CategoryKeyword,
			Syntax:      "value as % of total",
			Description: "Compute the ratio of two same-type values as a percentage. Both operands must be the same type (e.g., both currency, both quantities). The inverse of '% of': 20% of $500 = $100, $100 as % of $500 = 20%.",
			Aliases:     nil,
			Example:     "$100 as % of $500 → 20%",
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
			Example:     "10 % 3 → 1, 50% → 50%, 100 + 20% → 120",
		},
	}
}

// getFrontmatterFeatures returns frontmatter directive features.
func getFrontmatterFeatures() []Feature {
	categories := strings.Join(units.Categories(), ", ")

	return []Feature{
		{
			Name:     "exchange",
			Category: CategoryFrontmatter,
			Syntax:   "exchange:\n  FROM_TO: rate",
			Description: "Define currency conversion rates. " +
				"Keys use FROM_TO format with 3-letter ISO 4217 codes (e.g., USD_EUR).",
			Example: "exchange:\n  USD_EUR: 0.92",
		},
		{
			Name:     "globals",
			Category: CategoryFrontmatter,
			Syntax:   "globals:\n  name: value",
			Description: "Define document-wide variables. " +
				"Values are CalcMark expressions evaluated before the document body.",
			Example: "globals:\n  tax_rate: 0.32\n  base_price: $100",
		},
		{
			Name:        "scale",
			Category:    CategoryFrontmatter,
			Syntax:      "scale: <factor>",
			Description: fmt.Sprintf("Multiply quantity results by a factor. Accepts a number or a map with factor and unit_categories. Currency scales only when Currency is listed in unit_categories. Valid categories: %s. Temperature excluded by default.", categories),
			Example:     "scale: 2",
		},
		{
			Name:        "convert_to",
			Category:    CategoryFrontmatter,
			Syntax:      "convert_to: <system>",
			Description: fmt.Sprintf("Convert quantity results to a measurement system (si or imperial). Accepts a system name or a map with system and unit_categories. Valid categories: %s.", categories),
			Example:     "convert_to: si",
		},
		{
			Name:     "measurement",
			Category: CategoryFrontmatter,
			Syntax:   "measurement:\n  volume: us|imperial\n  mass: standard|troy\n  ton: short|long|metric",
			Description: "Configure how ambiguous unit names are interpreted. " +
				"Each axis is independent. Only axes that differ from US Customary defaults need to be specified. " +
				"\"standard\" mass means avoirdupois (everyday weight: 1 oz = 28.35g). " +
				"\"troy\" mass is for precious metals (1 troy oz = 31.10g). " +
				"Optional strict: true/false controls whether formatter annotates bare ambiguous units in output.",
			Example: "measurement:\n  volume: imperial\n  mass: troy",
		},
		{
			Name:        "@scale",
			Category:    CategoryKeyword,
			Syntax:      "@scale",
			Description: "Reference the scale factor from frontmatter in expressions. Resolves to a number.",
			Example:     "per_loaf = total_cost / @scale",
		},
		{
			Name:        "@globals",
			Category:    CategoryKeyword,
			Syntax:      "@globals.name",
			Description: "Reference a named global variable from frontmatter in expressions. Resolves to the typed value of the global.",
			Example:     "tax = income * @globals.tax_rate",
		},
		{
			Name:        "{{variable}}",
			Category:    CategoryKeyword,
			Syntax:      "{{variable_name}}",
			Description: "Embed a calculated value in prose. After evaluation, {{var}} tags in text blocks are replaced with display-formatted values. Supports forward references — a summary at the top can reference results computed below.",
			Aliases:     []Alias{{Name: "template", Parseable: false}, {Name: "interpolation", Parseable: false}},
			Example:     "Total: {{revenue}}",
		},
	}
}

// getFractionFeatures returns fraction-related features.
func getFractionFeatures() []Feature {
	return []Feature{
		{
			Name:        "fraction literal",
			Category:    CategoryOperator,
			Syntax:      "1/3",
			Description: "Exact rational number. Write numerator/denominator without spaces to create a fraction.",
			Aliases:     []Alias{{Name: "fraction", Parseable: false}, {Name: "rational", Parseable: false}},
			Example:     "1/3",
		},
		{
			Name:        "mixed number",
			Category:    CategoryOperator,
			Syntax:      "11 3/8",
			Description: "Mixed number combining an integer and a fraction.",
			Aliases:     []Alias{{Name: "mixed fraction", Parseable: false}},
			Example:     "11 3/8 inch",
		},
		{
			Name:        "fraction arithmetic",
			Category:    CategoryOperator,
			Syntax:      "1/3 + 1/4",
			Description: "Exact rational arithmetic. Results are automatically simplified to lowest terms.",
			Example:     "1/3 + 1/3 + 1/3",
		},
	}
}
