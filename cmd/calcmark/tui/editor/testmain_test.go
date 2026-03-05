package editor

import (
	"os"
	"testing"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/config"
)

// TestMain ensures all editor tests run with default config (dark mode),
// isolated from any user config on the host machine.
func TestMain(m *testing.M) {
	// Isolate from user config so golden files are reproducible.
	tmpHome, err := os.MkdirTemp("", "calcmark-editor-test-*")
	if err != nil {
		panic("failed to create temp home: " + err.Error())
	}
	defer os.RemoveAll(tmpHome)
	os.Setenv("HOME", tmpHome)

	// Reload config with isolated HOME (overrides any init() Load calls).
	config.Reload()

	os.Exit(m.Run())
}
