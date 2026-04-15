package lsp

import (
	"reflect"
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestDetectRegion(t *testing.T) {
	cases := []struct {
		name       string
		source     string
		wantOK     bool
		wantStart  int
		wantEnd    int
		wantKeys   map[int]string
	}{
		{
			name:      "well-formed single key",
			source:    "---\nconvert_to: si\n---\n# Body\n",
			wantOK:    true,
			wantStart: 0,
			wantEnd:   2,
			wantKeys:  map[int]string{1: "convert_to"},
		},
		{
			name:      "multi-key with nested map",
			source:    "---\nconvert_to: si\nglobals:\n  foo: 1\n---\n",
			wantOK:    true,
			wantStart: 0,
			wantEnd:   4,
			wantKeys:  map[int]string{1: "convert_to", 2: "globals"},
		},
		{
			name:   "no frontmatter",
			source: "# Just body\n",
			wantOK: false,
		},
		{
			name:   "missing closing fence",
			source: "---\nconvert_to: si\n",
			wantOK: false,
		},
		{
			name:      "empty body between fences",
			source:    "---\n---\n",
			wantOK:    true,
			wantStart: 0,
			wantEnd:   1,
			wantKeys:  map[int]string{},
		},
		{
			name:   "leading whitespace before fence is rejected",
			source: "  ---\nconvert_to: si\n---\n",
			wantOK: false,
		},
		{
			name:      "extra (non-registered) key included",
			source:    "---\ntitle: Hello\n---\n",
			wantOK:    true,
			wantStart: 0,
			wantEnd:   2,
			wantKeys:  map[int]string{1: "title"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			region, ok := DetectRegion(tc.source)
			if ok != tc.wantOK {
				t.Fatalf("DetectRegion ok = %v, want %v", ok, tc.wantOK)
			}
			if !ok {
				return
			}
			if region.StartLine != tc.wantStart {
				t.Errorf("StartLine = %d, want %d", region.StartLine, tc.wantStart)
			}
			if region.EndLine != tc.wantEnd {
				t.Errorf("EndLine = %d, want %d", region.EndLine, tc.wantEnd)
			}
			if !reflect.DeepEqual(region.KeyLines, tc.wantKeys) {
				t.Errorf("KeyLines = %v, want %v", region.KeyLines, tc.wantKeys)
			}
		})
	}
}

func TestClassifyCursor(t *testing.T) {
	// Region for "---\nconvert_to: si\n---\n# Body\n"
	simple, ok := DetectRegion("---\nconvert_to: si\n---\n# Body\n")
	if !ok {
		t.Fatal("setup: expected DetectRegion ok for simple source")
	}
	// Region for "---\nconvert_to: si\nglobals:\n  foo: bar\n---\n"
	nested, ok := DetectRegion("---\nconvert_to: si\nglobals:\n  foo: bar\n---\n")
	if !ok {
		t.Fatal("setup: expected DetectRegion ok for nested source")
	}
	// Region for "---\n\n---\n" — blank line between fences
	blank, ok := DetectRegion("---\n\n---\n")
	if !ok {
		t.Fatal("setup: expected DetectRegion ok for blank-body source")
	}

	cases := []struct {
		name     string
		region   FrontmatterRegion
		pos      protocol.Position
		wantIn   bool
		wantPos  CursorPosition
		wantKey  string
	}{
		{
			name:    "outside region (after end fence)",
			region:  simple,
			pos:     protocol.Position{Line: 3, Character: 0},
			wantIn:  false,
			wantPos: CursorOutside,
			wantKey: "",
		},
		{
			name:    "on opening fence",
			region:  simple,
			pos:     protocol.Position{Line: 0, Character: 0},
			wantIn:  true,
			wantPos: CursorFence,
		},
		{
			name:    "on closing fence",
			region:  simple,
			pos:     protocol.Position{Line: 2, Character: 0},
			wantIn:  true,
			wantPos: CursorFence,
		},
		{
			name:    "key position at column 0",
			region:  simple,
			pos:     protocol.Position{Line: 1, Character: 0},
			wantIn:  true,
			wantPos: CursorKey,
			wantKey: "convert_to",
		},
		{
			name:    "value position past colon",
			region:  simple,
			pos:     protocol.Position{Line: 1, Character: 12},
			wantIn:  true,
			wantPos: CursorValue,
			wantKey: "convert_to",
		},
		{
			name:    "exactly on the colon counts as key",
			region:  simple,
			pos:     protocol.Position{Line: 1, Character: 10},
			wantIn:  true,
			wantPos: CursorKey,
			wantKey: "convert_to",
		},
		{
			name:    "blank line between fences → key, no name",
			region:  blank,
			pos:     protocol.Position{Line: 1, Character: 0},
			wantIn:  true,
			wantPos: CursorKey,
			wantKey: "",
		},
		{
			name:    "indented continuation line → value, parent key",
			region:  nested,
			pos:     protocol.Position{Line: 3, Character: 4},
			wantIn:  true,
			wantPos: CursorValue,
			wantKey: "globals",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := ClassifyCursor(tc.region, tc.pos)
			if ctx.InRegion != tc.wantIn {
				t.Errorf("InRegion = %v, want %v", ctx.InRegion, tc.wantIn)
			}
			if ctx.Position != tc.wantPos {
				t.Errorf("Position = %q, want %q", ctx.Position, tc.wantPos)
			}
			if ctx.Key != tc.wantKey {
				t.Errorf("Key = %q, want %q", ctx.Key, tc.wantKey)
			}
		})
	}
}
