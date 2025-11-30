package editor

import (
	"io"
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cockroachdb/datadriven"
	"github.com/knz/catwalk"
	"github.com/muesli/termenv"
)

func init() {
	// Force ASCII color profile for consistent test output across environments
	lipgloss.SetColorProfile(termenv.Ascii)
}

// TestEditorCatwalk runs data-driven tests for the editor model.
// Test files are in testdata/ directory.
// Run with -rewrite flag to regenerate expected output:
//
//	go test ./cmd/calcmark/tui/editor/... -args -rewrite
func TestEditorCatwalk(t *testing.T) {
	// Create a simple test document
	content := `# Header
x = 10
y = 20
z = 30`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		// Skip compression subdirectory (handled by separate test)
		if strings.HasPrefix(path, "testdata/compression/") {
			return
		}

		m := New(doc)
		m.width = 80
		m.height = 24
		m.previewMode = PreviewFull

		catwalk.RunModel(t, path, m,
			catwalk.WithObserver("debug", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).Debug()))
				return err
			}),
			catwalk.WithObserver("lines", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).DebugLines()))
				return err
			}),
		)
	})
}

// TestEditorCatwalkCompression runs tests with compression.cm-like content
// that causes wrapping at narrow widths.
func TestEditorCatwalkCompression(t *testing.T) {
	content := `# Compression Function - compress()

# Compressed size estimates for different compression types
gzip_compressed = compress(1 GB, gzip)
lz4_compressed = compress(100 MB, lz4)
zstd_compressed = compress(500 MB, zstd)
bzip2_compressed = compress(1000 MB, bzip2)
snappy_compressed = compress(300 MB, snappy)
no_compression = compress(200 MB, none)

# Use in calculations
storage_savings = 10 GB - compress(10 GB, gzip)
compressed_transfer = transfer_time(compress(1 GB, lz4), global, gigabit)`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata/compression", func(t *testing.T, path string) {
		m := New(doc)
		m.width = 80 // Narrower width to test wrapping
		m.height = 24
		m.previewMode = PreviewFull

		catwalk.RunModel(t, path, m,
			catwalk.WithObserver("debug", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).Debug()))
				return err
			}),
			catwalk.WithObserver("lines", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).DebugLines()))
				return err
			}),
		)
	})
}
