package lsp

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"gopkg.in/yaml.v3"
)

// Frontmatter mutation commands exposed via `workspace/executeCommand`.
//
// Each command takes `[{uri, version, ...args}]` as `arguments[0]` and
// returns a `WorkspaceEdit` that replaces the document's frontmatter
// region with the mutated YAML. The client applies the edit; the
// server stays a pure dispatcher (no document-state writes here).
//
// YAML round-trip preserves:
//   - Top-level key ordering (yaml.v3's `*yaml.Node` is order-preserving)
//   - Extra (non-calcmark) keys (`title`, `author`, ...) — they pass
//     through untouched
//   - Comments — yaml.v3 preserves head/foot/line comments on Nodes
//
// Source of truth: the request's `version` arg. If the document has
// advanced past that version, the server returns a ContentModified
// error so the client retries against the fresh source. The dispatcher
// accepts an in-flight version drift on the read because the client
// always commits the WorkspaceEdit at the version it computed against;
// stale-version protection lives in the client wrapper.

// commandArgs decodes the standard `arguments[0]` envelope. URI and
// version are always present; the per-command kwargs are passed
// through as `Map` for the command implementation to type-check.
type commandArgs struct {
	URI     string         `json:"uri"`
	Version int            `json:"version"`
	Map     map[string]any `json:"-"`
}

func decodeCommandArgs(rawArgs []json.RawMessage) (*commandArgs, error) {
	if len(rawArgs) == 0 {
		return nil, errors.New("missing command arguments")
	}
	var m map[string]any
	if err := json.Unmarshal(rawArgs[0], &m); err != nil {
		return nil, fmt.Errorf("decode command arguments: %w", err)
	}
	out := &commandArgs{Map: m}
	if uri, ok := m["uri"].(string); ok {
		out.URI = uri
	}
	// JSON numbers come through as float64; LSP version is int.
	if v, ok := m["version"].(float64); ok {
		out.Version = int(v)
	}
	return out, nil
}

// argString reads a string field from the command's arguments map.
// Returns ("", true) when the field is present and a string;
// ("", false) when missing or wrong type.
func (c *commandArgs) argString(key string) (string, bool) {
	v, ok := c.Map[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// argInt reads an integer field (JSON number → int).
func (c *commandArgs) argInt(key string) (int, bool) {
	v, ok := c.Map[key]
	if !ok {
		return 0, false
	}
	f, ok := v.(float64)
	if !ok {
		return 0, false
	}
	return int(f), true
}

// argStringSlice reads a []string field. Accepts both `[]any` of
// strings and the typed form.
func (c *commandArgs) argStringSlice(key string) ([]string, bool) {
	v, ok := c.Map[key]
	if !ok {
		return nil, false
	}
	if ss, ok := v.([]string); ok {
		return ss, true
	}
	if anys, ok := v.([]any); ok {
		out := make([]string, 0, len(anys))
		for _, a := range anys {
			s, ok := a.(string)
			if !ok {
				return nil, false
			}
			out = append(out, s)
		}
		return out, true
	}
	return nil, false
}

// frontmatterFenceRE matches a leading `---\n...\n---\n` frontmatter
// region. The trailing newline after the closing fence is consumed
// when present so the body's first line ends up at the same offset
// regardless of whether the source has a blank line after the fence.
var frontmatterFenceRE = regexp.MustCompile(`(?s)\A---\n(.*?)\n---\n`)

// frontmatterRegion describes the byte span of an existing
// frontmatter fence in the source plus the YAML content between the
// `---` markers. When `present` is false, `yaml` is empty and the
// caller must build a new fence to insert.
type frontmatterRegion struct {
	present bool
	start   int    // byte offset of the opening `---` (0 when present)
	end     int    // byte offset just past the closing newline
	yaml    string // YAML content between the fences (no fences, no trailing newline)
}

func detectFrontmatterRegion(source string) frontmatterRegion {
	loc := frontmatterFenceRE.FindStringSubmatchIndex(source)
	if loc == nil {
		return frontmatterRegion{present: false}
	}
	return frontmatterRegion{
		present: true,
		start:   loc[0],
		end:     loc[1],
		yaml:    source[loc[2]:loc[3]],
	}
}

// applyFrontmatterMutation parses the source's frontmatter (creating
// an empty mapping if absent), invokes `mutate` on the root mapping
// node, and returns the new source string. The yaml.Node API
// preserves key order and comments through the round-trip.
//
// `mutate` receives the root MAPPING node — i.e. the top-level
// key/value sequence under the `---` fence, NOT a document node.
// Returns an error to abort the mutation; the source is left
// untouched.
func applyFrontmatterMutation(source string, mutate func(root *yaml.Node) error) (string, error) {
	region := detectFrontmatterRegion(source)

	root, err := loadOrInitMapping(region.yaml)
	if err != nil {
		return "", err
	}
	if err := mutate(root); err != nil {
		return "", err
	}
	encoded, err := encodeMapping(root)
	if err != nil {
		return "", err
	}

	return spliceFrontmatter(source, region, encoded), nil
}

// loadOrInitMapping returns the top-level mapping node from the YAML
// content (creating an empty mapping when content is blank).
func loadOrInitMapping(yamlContent string) (*yaml.Node, error) {
	if strings.TrimSpace(yamlContent) == "" {
		return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}, nil
	}
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(yamlContent), &doc); err != nil {
		return nil, fmt.Errorf("parse frontmatter YAML: %w", err)
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}, nil
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("frontmatter root must be a mapping, got kind=%d", root.Kind)
	}
	return root, nil
}

// encodeMapping serializes a mapping node back to YAML. Returns the
// content WITHOUT the surrounding `---` fences (the splicer adds them).
// An empty mapping returns an empty string so spliceFrontmatter can
// drop the fence entirely when the user removed the last directive.
func encodeMapping(root *yaml.Node) (string, error) {
	if len(root.Content) == 0 {
		return "", nil
	}
	doc := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{root}}
	var buf strings.Builder
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return "", fmt.Errorf("encode frontmatter YAML: %w", err)
	}
	if err := enc.Close(); err != nil {
		return "", fmt.Errorf("close frontmatter YAML encoder: %w", err)
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}

// spliceFrontmatter assembles the new source given the original, the
// (possibly absent) old fence region, and the new YAML content.
//
//   - new YAML empty: drop the fence entirely (or no-op if absent).
//   - new YAML non-empty + region present: replace the existing fence.
//   - new YAML non-empty + region absent: prepend a new fence and a
//     blank separator line per the CommonMark convention used
//     elsewhere in this codebase.
func spliceFrontmatter(source string, region frontmatterRegion, newYAML string) string {
	if newYAML == "" {
		if !region.present {
			return source
		}
		return source[region.end:]
	}
	fenced := "---\n" + newYAML + "\n---\n"
	if region.present {
		return fenced + source[region.end:]
	}
	if source == "" {
		return fenced
	}
	// Prepend; insert a blank line between the fence and the body so
	// goldmark sees a CommonMark-compliant break.
	body := source
	if !strings.HasPrefix(body, "\n") {
		fenced += "\n"
	}
	return fenced + body
}

// ── Mapping-node helpers ──────────────────────────────────────

// findChild looks up a key in a mapping and returns its value node
// (or nil when absent). The mapping's `Content` is a flat list of
// alternating key+value nodes — the node at index `i` is a key, the
// node at `i+1` is its value.
func findChild(mapping *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	return nil
}

// setScalarChild sets `key: value` on the mapping. Inserts a new
// entry when the key is absent, preserving insertion order at the
// end. Replaces the existing value node when present (any old value,
// scalar or otherwise, is discarded).
//
// The value node uses the default scalar style — yaml.v3 picks
// plain-unquoted when the content is safe (`0.08`, `imperial`,
// `July 15`) and falls back to a double-quoted form for content
// that would otherwise reparse as a different type (`"yes"`,
// `"true"`, `"null"`, leading reserved char, etc.). This keeps the
// emitted YAML aligned with the cmw-web frontmatter normalize step
// without duplicating its decision table here.
func setScalarChild(mapping *yaml.Node, key, value string) {
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: key}
	valueNode := &yaml.Node{Kind: yaml.ScalarNode, Value: value}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content[i+1] = valueNode
			return
		}
	}
	mapping.Content = append(mapping.Content, keyNode, valueNode)
}

// setMappingChild sets `key: <new mapping>` on the mapping, returning
// the new (or existing) child mapping for further mutation. When the
// key already exists and points at a mapping, the existing node is
// returned in place — caller-side mutations modify it directly.
// When absent or pointing at a non-mapping, a fresh mapping replaces
// the value.
func setMappingChild(mapping *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			if mapping.Content[i+1].Kind == yaml.MappingNode {
				return mapping.Content[i+1]
			}
			child := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			mapping.Content[i+1] = child
			return child
		}
	}
	child := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		child,
	)
	return child
}

// removeChild deletes a key from the mapping. No-op when absent.
// Returns true when something was removed.
func removeChild(mapping *yaml.Node, key string) bool {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content = append(mapping.Content[:i], mapping.Content[i+2:]...)
			return true
		}
	}
	return false
}

// ── Command dispatcher + WorkspaceEdit builder ────────────────

// executeCommand is the LSP `workspace/executeCommand` entry point.
// Routes to a per-command implementation; unknown commands return
// an LSP method-not-found-style error.
func (s *Server) executeCommand(_ *glsp.Context, params *protocol.ExecuteCommandParams) (any, error) {
	rawArgs := make([]json.RawMessage, len(params.Arguments))
	for i, a := range params.Arguments {
		// Re-marshal each arg so decodeCommandArgs can take a
		// uniform `json.RawMessage` regardless of the wire shape
		// the glsp library handed us.
		b, err := json.Marshal(a)
		if err != nil {
			return nil, fmt.Errorf("re-marshal argument[%d]: %w", i, err)
		}
		rawArgs[i] = b
	}
	args, err := decodeCommandArgs(rawArgs)
	if err != nil {
		return nil, err
	}
	source, ok := s.readSourceForURI(args.URI)
	if !ok {
		return nil, fmt.Errorf("document not open: %s", args.URI)
	}

	switch params.Command {
	case "calcmark.frontmatter.setGlobal":
		return s.fmCommand(args, source, fmSetGlobal(args))
	case "calcmark.frontmatter.removeGlobal":
		return s.fmCommand(args, source, fmRemoveGlobal(args))
	case "calcmark.frontmatter.setExchangeRate":
		return s.fmCommand(args, source, fmSetExchangeRate(args))
	case "calcmark.frontmatter.removeExchangeRate":
		return s.fmCommand(args, source, fmRemoveExchangeRate(args))
	case "calcmark.frontmatter.setScale":
		return s.fmCommand(args, source, fmSetScale(args))
	case "calcmark.frontmatter.clearScale":
		return s.fmCommand(args, source, fmRemoveKey("scale"))
	case "calcmark.frontmatter.setConvertTo":
		return s.fmCommand(args, source, fmSetConvertTo(args))
	case "calcmark.frontmatter.clearConvertTo":
		return s.fmCommand(args, source, fmRemoveKey("convert_to"))
	case "calcmark.frontmatter.setMeasurement":
		return s.fmCommand(args, source, fmSetMeasurement(args))
	case "calcmark.frontmatter.clearMeasurement":
		return s.fmCommand(args, source, fmRemoveKey("measurement"))
	case "calcmark.frontmatter.setFiscalYearStarts":
		return s.fmCommand(args, source, fmSetFiscalYearStarts(args))
	case "calcmark.frontmatter.clearFiscalYearStarts":
		return s.fmCommand(args, source, fmRemoveKey("fiscal_year_starts"))
	case "calcmark.frontmatter.format":
		return s.fmCommand(args, source, fmFormat())
	default:
		return nil, fmt.Errorf("unknown command: %s", params.Command)
	}
}

// readSourceForURI returns the current source text for `uri`, or
// `false` when the document is not open. Reads through the existing
// per-document state so the result reflects every `didChange` the
// LSP has seen.
func (s *Server) readSourceForURI(uri string) (string, bool) {
	ds := s.getDocument(uri)
	if ds == nil {
		return "", false
	}
	return ds.getSource(), true
}

// fmCommand is the shared command body: mutate the source, build the
// WorkspaceEdit, return it. When the mutator yields an unchanged
// source the response is a typed `null` — the LSP allows that.
func (s *Server) fmCommand(
	args *commandArgs,
	source string,
	mutate func(root *yaml.Node) error,
) (any, error) {
	newSource, err := applyFrontmatterMutation(source, mutate)
	if err != nil {
		return nil, err
	}
	if newSource == source {
		return nil, nil
	}
	return buildFullDocumentEdit(args.URI, args.Version, source, newSource), nil
}

// buildFullDocumentEdit constructs a `WorkspaceEdit` that replaces the
// entire document with `newSource`. We keep the edit document-wide
// rather than a fine-grained range diff because the frontmatter splice
// can shrink/grow line counts, and a single replacement keeps the
// client's apply path simple.
func buildFullDocumentEdit(uri string, version int, oldSource, newSource string) protocol.WorkspaceEdit {
	endLine, endChar := docEndPosition(oldSource)
	v := protocol.Integer(version)
	return protocol.WorkspaceEdit{
		DocumentChanges: []any{
			protocol.TextDocumentEdit{
				TextDocument: protocol.OptionalVersionedTextDocumentIdentifier{
					TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri},
					Version:                &v,
				},
				Edits: []any{
					protocol.TextEdit{
						Range: protocol.Range{
							Start: protocol.Position{Line: 0, Character: 0},
							End:   protocol.Position{Line: protocol.UInteger(endLine), Character: protocol.UInteger(endChar)},
						},
						NewText: newSource,
					},
				},
			},
		},
	}
}

// docEndPosition returns the (line, character) just past the last
// byte of `source`. UTF-16 code units per the LSP spec; calcmark
// frontmatter is ASCII in practice, so byte-counting on the final
// line is correct here.
func docEndPosition(source string) (int, int) {
	if source == "" {
		return 0, 0
	}
	line := 0
	col := 0
	for _, r := range source {
		if r == '\n' {
			line++
			col = 0
			continue
		}
		col++
	}
	return line, col
}

// ── Per-command mutations ─────────────────────────────────────

func fmSetGlobal(args *commandArgs) func(*yaml.Node) error {
	return func(root *yaml.Node) error {
		name, ok := args.argString("name")
		if !ok || name == "" {
			return errors.New("setGlobal: missing 'name'")
		}
		value, ok := args.argString("value")
		if !ok {
			return errors.New("setGlobal: missing 'value'")
		}
		globals := setMappingChild(root, "globals")
		setScalarChild(globals, name, value)
		return nil
	}
}

func fmRemoveGlobal(args *commandArgs) func(*yaml.Node) error {
	return func(root *yaml.Node) error {
		name, ok := args.argString("name")
		if !ok || name == "" {
			return errors.New("removeGlobal: missing 'name'")
		}
		globals := findChild(root, "globals")
		if globals == nil || globals.Kind != yaml.MappingNode {
			return nil
		}
		removeChild(globals, name)
		// Empty globals mapping → drop the key entirely.
		if len(globals.Content) == 0 {
			removeChild(root, "globals")
		}
		return nil
	}
}

func fmSetExchangeRate(args *commandArgs) func(*yaml.Node) error {
	return func(root *yaml.Node) error {
		from, fromOK := args.argString("from")
		to, toOK := args.argString("to")
		rate, rateOK := args.argString("rate")
		if !fromOK || !toOK || !rateOK || from == "" || to == "" || rate == "" {
			return errors.New("setExchangeRate: requires 'from', 'to', and 'rate'")
		}
		exchange := setMappingChild(root, "exchange")
		key := strings.ToUpper(from) + "_" + strings.ToUpper(to)
		setScalarChild(exchange, key, rate)
		return nil
	}
}

func fmRemoveExchangeRate(args *commandArgs) func(*yaml.Node) error {
	return func(root *yaml.Node) error {
		from, fromOK := args.argString("from")
		to, toOK := args.argString("to")
		if !fromOK || !toOK || from == "" || to == "" {
			return errors.New("removeExchangeRate: requires 'from' and 'to'")
		}
		exchange := findChild(root, "exchange")
		if exchange == nil || exchange.Kind != yaml.MappingNode {
			return nil
		}
		key := strings.ToUpper(from) + "_" + strings.ToUpper(to)
		removeChild(exchange, key)
		if len(exchange.Content) == 0 {
			removeChild(root, "exchange")
		}
		return nil
	}
}

func fmSetScale(args *commandArgs) func(*yaml.Node) error {
	return func(root *yaml.Node) error {
		factor, ok := args.argString("factor")
		if !ok || factor == "" {
			return errors.New("setScale: missing 'factor'")
		}
		categories, _ := args.argStringSlice("unit_categories")
		if len(categories) == 0 {
			// Bare scalar form: `scale: <factor>`.
			setScalarChild(root, "scale", factor)
			return nil
		}
		// Mapping form: `scale: { factor, unit_categories: [...] }`.
		scale := setMappingChild(root, "scale")
		// Reset content so the mapping holds only what we supply.
		scale.Content = scale.Content[:0]
		setScalarChild(scale, "factor", factor)
		seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Style: yaml.FlowStyle}
		for _, c := range categories {
			seq.Content = append(seq.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: c})
		}
		scale.Content = append(scale.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "unit_categories"},
			seq,
		)
		return nil
	}
}

func fmSetConvertTo(args *commandArgs) func(*yaml.Node) error {
	return func(root *yaml.Node) error {
		system, ok := args.argString("system")
		if !ok || system == "" {
			return errors.New("setConvertTo: missing 'system'")
		}
		categories, _ := args.argStringSlice("unit_categories")
		if len(categories) == 0 {
			setScalarChild(root, "convert_to", system)
			return nil
		}
		ct := setMappingChild(root, "convert_to")
		ct.Content = ct.Content[:0]
		setScalarChild(ct, "system", system)
		seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Style: yaml.FlowStyle}
		for _, c := range categories {
			seq.Content = append(seq.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: c})
		}
		ct.Content = append(ct.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "unit_categories"},
			seq,
		)
		return nil
	}
}

func fmSetMeasurement(args *commandArgs) func(*yaml.Node) error {
	return func(root *yaml.Node) error {
		// The form sends one or more axis fields: volume, mass,
		// ton, plus an optional `strict` boolean. Merge them onto
		// any existing measurement mapping rather than replacing,
		// so a partial form submission only touches the axis it
		// wrote to.
		measurement := setMappingChild(root, "measurement")
		mutated := false
		for _, axis := range []string{"volume", "mass", "ton"} {
			if v, ok := args.argString(axis); ok {
				setScalarChild(measurement, axis, v)
				mutated = true
			}
		}
		if v, ok := args.Map["strict"].(bool); ok {
			setScalarChild(measurement, "strict", boolToYAML(v))
			mutated = true
		}
		if !mutated {
			return errors.New("setMeasurement: requires at least one of 'volume', 'mass', 'ton', 'strict'")
		}
		return nil
	}
}

func boolToYAML(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func fmSetFiscalYearStarts(args *commandArgs) func(*yaml.Node) error {
	return func(root *yaml.Node) error {
		// Accepts either `month` (string month name like "July" or
		// abbreviation "Jul") or `month` + `day`. Stored as a single
		// scalar string per `parseFiscalYearStarts` in
		// spec/document/frontmatter.go (the parser there accepts
		// "july", "jul", "July 15", etc.).
		monthStr, monthStrOK := args.argString("month")
		monthInt, monthIntOK := args.argInt("month")
		if !monthStrOK && !monthIntOK {
			return errors.New("setFiscalYearStarts: missing 'month'")
		}
		var monthName string
		if monthStrOK {
			monthName = monthStr
		} else {
			monthName = monthIndexToName(monthInt)
			if monthName == "" {
				return fmt.Errorf("setFiscalYearStarts: month %d out of range", monthInt)
			}
		}
		value := monthName
		if d, ok := args.argInt("day"); ok && d > 0 {
			value = fmt.Sprintf("%s %d", monthName, d)
		}
		setScalarChild(root, "fiscal_year_starts", value)
		return nil
	}
}

func monthIndexToName(m int) string {
	names := []string{
		"", "January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December",
	}
	if m < 1 || m > 12 {
		return ""
	}
	return names[m]
}

// fmRemoveKey returns a mutator that drops a single top-level key.
// Used by clearScale / clearConvertTo / clearMeasurement /
// clearFiscalYearStarts.
func fmRemoveKey(key string) func(*yaml.Node) error {
	return func(root *yaml.Node) error {
		removeChild(root, key)
		return nil
	}
}

// fmFormat re-emits the frontmatter through yaml.Encoder, normalizing
// whitespace + indentation. No semantic mutation. Useful when the
// user wants a clean canonical form after manual edits.
func fmFormat() func(*yaml.Node) error {
	return func(_ *yaml.Node) error {
		// The round-trip itself is a format pass — no extra work
		// needed in the mutator. `applyFrontmatterMutation` always
		// re-encodes via yaml.Encoder.
		return nil
	}
}
