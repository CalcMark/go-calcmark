package document

// FrontmatterKeyType describes the value shape a registered CalcMark
// frontmatter key accepts. It exists so tooling (LSP completion, hover,
// semantic diagnostics) can describe and validate keys uniformly without
// special-casing each one.
type FrontmatterKeyType int

const (
	// FrontmatterKeyMapStringDecimal: map of string -> decimal.Decimal.
	// Used by `exchange` (e.g., "USD_EUR" -> 0.92).
	FrontmatterKeyMapStringDecimal FrontmatterKeyType = iota

	// FrontmatterKeyMapStringString: map of string -> string.
	// Used by `globals` (variable name -> CalcMark expression source).
	FrontmatterKeyMapStringString

	// FrontmatterKeyEnumString: a string drawn from a fixed set of values.
	// EnumValues lists the accepted values in canonical order.
	FrontmatterKeyEnumString

	// FrontmatterKeyStruct: a structured value (typically a YAML map with
	// known sub-keys). Sub-key shape is documented in the registered key's
	// Doc string and enforced by the key's parser.
	FrontmatterKeyStruct
)

// String returns a stable, human-readable label for the key type.
// Useful for diagnostics and tooling output.
func (t FrontmatterKeyType) String() string {
	switch t {
	case FrontmatterKeyMapStringDecimal:
		return "MapStringDecimal"
	case FrontmatterKeyMapStringString:
		return "MapStringString"
	case FrontmatterKeyEnumString:
		return "EnumString"
	case FrontmatterKeyStruct:
		return "Struct"
	default:
		return "Unknown"
	}
}

// RegisteredKey describes a single CalcMark-specific frontmatter key.
// All CalcMark grammar keys (those that ParseFrontmatter recognizes and
// validates beyond YAML decoding) are listed in the Registry.
type RegisteredKey struct {
	// Name is the YAML key as it appears in document frontmatter.
	// Comparison is case-sensitive.
	Name string

	// Type describes the accepted value shape.
	Type FrontmatterKeyType

	// Doc is a 1-2 sentence description of what the key configures.
	// It is the source of truth for hover text in tooling (LSP).
	Doc string

	// EnumValues lists accepted values when Type == FrontmatterKeyEnumString.
	// Empty for all other types. Order is canonical (preferred form first).
	EnumValues []string
}

// Registry lists every CalcMark-grammar frontmatter key recognized by
// ParseFrontmatter. Non-registered top-level keys are preserved verbatim
// in Frontmatter.Extra and carry no CalcMark semantics.
//
// Complexity note: lookup helpers walk this slice linearly. With ~6
// entries this is faster than a map and keeps the data trivially
// inspectable. If the registry grows past ~50 entries, consider an
// auxiliary map index.
var Registry = []RegisteredKey{
	{
		Name: "exchange",
		Type: FrontmatterKeyMapStringDecimal,
		Doc:  "Currency exchange rates as 'FROM_TO' -> rate (e.g., 'USD_EUR': 0.92 means 1 USD = 0.92 EUR). Used by currency conversion expressions in the document.",
	},
	{
		Name: "globals",
		Type: FrontmatterKeyMapStringString,
		Doc:  "User-defined variables as name -> CalcMark expression string. Each value is parsed as a CalcMark expression and made available throughout the document.",
	},
	{
		Name: "scale",
		Type: FrontmatterKeyStruct,
		Doc:  "Document-level scale transform. Either a positive number (factor only) or a map with 'factor' and optional 'unit_categories' to restrict which categories scale.",
	},
	{
		Name:       "convert_to",
		Type:       FrontmatterKeyEnumString,
		Doc:        "Document-level unit system conversion. Either the string 'si' or 'imperial', or a map with 'system' and optional 'unit_categories' to restrict which categories convert.",
		EnumValues: []string{"si", "imperial"},
	},
	{
		Name: "measurement",
		Type: FrontmatterKeyStruct,
		Doc:  "Disambiguates ambiguous unit names. Map with axis keys 'volume' (us|imperial), 'mass' (standard|troy), 'ton' (short|long|metric), and optional 'strict' boolean controlling formatter annotation.",
	},
	{
		Name: "fiscal_year_starts",
		Type: FrontmatterKeyStruct,
		Doc:  "Anchors fiscal-period expressions (FQ1, FY26, 'this fiscal quarter') to a calendar start. String value naming a month, optionally with a day (e.g., 'July', 'October 1').",
	},
	{
		Name:       "calendar_year_offset",
		Type:       FrontmatterKeyEnumString,
		Doc:        "Selects which calendar year a fiscal-year label refers to. 'before' (default) — FY label = year FY ends in (Australian government year, US tax year, most companies). 'after' — FY label = year FY starts in (some companies). Has no effect when fiscal_year_starts is January.",
		EnumValues: []string{"before", "after"},
	},
}

// IsRegisteredKey reports whether name is a CalcMark-grammar frontmatter key.
// Comparison is case-sensitive. Runs in O(|Registry|).
func IsRegisteredKey(name string) bool {
	for i := range Registry {
		if Registry[i].Name == name {
			return true
		}
	}
	return false
}

// LookupKey returns the RegisteredKey with the given name, if any.
// Comparison is case-sensitive. Runs in O(|Registry|).
func LookupKey(name string) (RegisteredKey, bool) {
	for i := range Registry {
		if Registry[i].Name == name {
			return Registry[i], true
		}
	}
	return RegisteredKey{}, false
}
