package editor

import (
	"io"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/cockroachdb/datadriven"
)

// footerObserver returns an observerV2 that renders the context footer,
// strips ANSI codes and trailing whitespace, and outputs the content
// (or "(empty)" if the footer is blank).
func footerObserver() optionV2 {
	return WithObserverV2("footer", func(out io.Writer, m tea.Model) error {
		model := m.(Model)
		results := model.GetLineResults()
		footer := model.renderContextFooter(80, results, model.contextFooterHeight(results))
		plain := stripAnsi(footer)

		// Trim trailing whitespace from each line and join
		lines := strings.Split(plain, "\n")
		var trimmed []string
		for _, line := range lines {
			trimmed = append(trimmed, strings.TrimRight(line, " "))
		}
		result := strings.Join(trimmed, "\n")
		result = strings.TrimRight(result, "\n")

		// If footer is empty (all whitespace), output "(empty)"
		if strings.TrimSpace(result) == "" {
			result = "(empty)"
		}

		_, err := out.Write([]byte(result + "\n"))
		return err
	})
}

// debugObserver returns the standard debug observerV2.
func debugObserver() optionV2 {
	return WithObserverV2("debug", func(out io.Writer, m tea.Model) error {
		_, err := out.Write([]byte(m.(Model).Debug()))
		return err
	})
}

// TestEditorCatwalkContextFooterRefs is a catwalk integration test that verifies
// the context footer in the rendered view shows variable references as the user
// navigates between lines. This tests the full pipeline:
// Document → Model → GetLineResults → getLineReferences → renderContextFooter
func TestEditorCatwalkContextFooterRefs(t *testing.T) {
	content := `rent = 1500
utils = 200
total = rent + utils
savings = total * 0.10
`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "context_footer_refs") {
			return
		}

		m := New(doc)
		m.width = 80
		m.height = 24
		m.previewMode = PreviewFull

		RunModelV2(t, path, m, debugObserver(), footerObserver())
	})
}

// TestEditorCatwalkContextFooterSelfRef verifies that self-references are
// filtered from the context footer. Reproduces the bug where
// "hundred_gig = throughput(hundred_gig)" showed hundred_gig in the footer
// because the function argument shares the variable name.
func TestEditorCatwalkContextFooterSelfRef(t *testing.T) {
	content := `hundred_gig = throughput(hundred_gig)
speed = throughput(gigabit)
total = hundred_gig + speed
`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "context_footer_self_ref") {
			return
		}

		m := New(doc)
		m.width = 80
		m.height = 24
		m.previewMode = PreviewFull

		RunModelV2(t, path, m, debugObserver(), footerObserver())
	})
}
