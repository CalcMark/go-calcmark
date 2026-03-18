package document

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/classifier"
)

// testResolver is a simple IdentifierResolver for parity tests.
// It reports all names in the set as defined.
type testResolver struct {
	vars map[string]bool
}

func (r *testResolver) Has(name string) bool { return r.vars[name] }

// TestNLFunctionDetectorClassifierParity verifies that the document detector
// and the line classifier agree on whether NL function lines are calculations.
// Both paths must classify the same lines identically — any disagreement
// means a new NL function was added to one but not the other.
//
// After R2 (registry-driven refactor), both paths derive NL keywords from
// the feature registry, making this test the safety net against future drift.
func TestNLFunctionDetectorClassifierParity(t *testing.T) {
	detector := NewDetector()

	// Build a resolver with variables that the NL lines reference.
	env := &testResolver{vars: map[string]bool{
		"data":      true,
		"price":     true,
		"servers":   true,
		"car_value": true,
	}}

	tests := []struct {
		name       string
		line       string
		detector   bool // expected detector.IsCalculation result
		classifier bool // expected classifier.ClassifyLine == Calculation
	}{
		// --- System NL functions with LITERAL arguments ---
		// Detector handles these; classifier will after R2.
		{name: "read literal", line: "read 100 MB from ssd", detector: true, classifier: true},
		{name: "compress literal", line: "compress 1 GB using gzip", detector: true, classifier: true},
		{name: "transfer literal", line: "transfer 1 GB across regional gigabit", detector: true, classifier: true},

		// --- System NL functions with VARIABLE arguments ---
		{name: "read variable", line: "read data from ssd", detector: true, classifier: true},
		{name: "compress variable", line: "compress data using gzip", detector: true, classifier: true},
		{name: "transfer variable", line: "transfer data across regional gigabit", detector: true, classifier: true},

		// --- Growth NL functions with LITERAL arguments ---
		{name: "compound literal", line: "compound $1000 by 5% over 10 years", detector: true, classifier: true},
		{name: "grow literal", line: "grow 100 by 20 over 5 months", detector: true, classifier: true},
		{name: "depreciate literal", line: "depreciate $50000 by 15% over 5 years", detector: true, classifier: true},

		// --- Growth NL functions with VARIABLE arguments ---
		// Detector fix (R1) makes these pass in the detector.
		{name: "compound variable", line: "compound price by 5% over 10 years", detector: true, classifier: true},
		{name: "grow variable", line: "grow servers by 20 over 5 months", detector: true, classifier: true},
		{name: "depreciate variable", line: "depreciate car_value by 15% over 5 years", detector: true, classifier: true},

		// --- Prose that should NOT be classified as calculations ---
		{name: "prose compound", line: "Compound interest is calculated annually", detector: false, classifier: false},
		{name: "prose grow", line: "Grow your business with us", detector: false, classifier: false},
		{name: "prose read", line: "Read more about this topic", detector: false, classifier: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Detector path (stateless)
			detectorResult, err := detector.IsCalculation(tt.line)
			if err != nil {
				t.Fatalf("detector.IsCalculation(%q) error: %v", tt.line, err)
			}
			if detectorResult != tt.detector {
				t.Errorf("detector.IsCalculation(%q) = %v, want %v", tt.line, detectorResult, tt.detector)
			}

			// Classifier path (stateful, has environment)
			classifierResult, err := classifier.ClassifyLine(tt.line, env)
			if err != nil {
				t.Fatalf("classifier.ClassifyLine(%q) error: %v", tt.line, err)
			}
			classifierIsCalc := classifierResult == classifier.Calculation
			if classifierIsCalc != tt.classifier {
				t.Errorf("classifier.ClassifyLine(%q) = %v, want %v", tt.line, classifierIsCalc, tt.classifier)
			}

			// If both expected to be the same, also assert parity
			if tt.detector == tt.classifier && detectorResult != classifierIsCalc {
				t.Errorf("PARITY MISMATCH for %q:\n  detector = %v\n  classifier = %v (%v)",
					tt.line, detectorResult, classifierIsCalc, classifierResult)
			}
		})
	}
}
