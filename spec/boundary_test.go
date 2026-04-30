package spec_test

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

// TestSpecNeverImportsImpl verifies the architectural boundary:
// spec/ packages must never depend on impl/ packages.
// This prevents the language specification from coupling to its implementation.
func TestSpecNeverImportsImpl(t *testing.T) {
	// Use the module path, not relative path, since test cwd may vary.
	cmd := exec.Command("go", "list", "-json", "github.com/CalcMark/go-calcmark/v2/spec/...")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list failed: %v", err)
	}

	decoder := json.NewDecoder(strings.NewReader(string(out)))
	for decoder.More() {
		var pkg struct {
			ImportPath string
			Imports    []string
		}
		if err := decoder.Decode(&pkg); err != nil {
			t.Fatalf("decode error: %v", err)
		}

		for _, imp := range pkg.Imports {
			if strings.HasPrefix(imp, "github.com/CalcMark/go-calcmark/v2/impl/") {
				t.Errorf("spec package %s imports impl package %s — this violates the spec/impl boundary", pkg.ImportPath, imp)
			}
		}
	}
}
