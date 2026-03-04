package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/config"
	toml "github.com/pelletier/go-toml/v2"
)

// TestConfigShow_DefaultsOnly verifies that cm config outputs valid TOML
// containing all expected keys when no user config file is present.
func TestConfigShow_DefaultsOnly(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", "")
	config.Reload()

	stdout, _ := captureConfigOutput(t, func() error {
		return runConfigShow()
	})

	// Output must be parseable TOML
	var parsed map[string]any
	if err := toml.Unmarshal([]byte(stdout), &parsed); err != nil {
		t.Fatalf("output is not valid TOML: %v\nOutput:\n%s", err, stdout)
	}

	// Verify key sections/keys are present
	for _, key := range []string{"locale", "tui", "formatter"} {
		if _, ok := parsed[key]; !ok {
			t.Errorf("expected key %q in TOML output", key)
		}
	}

	// Should contain the header comment
	if !strings.Contains(stdout, "CalcMark") {
		t.Error("expected header comment containing 'CalcMark'")
	}
}

// TestConfigCreate_NewFile verifies that --create writes a config file
// with commented-out defaults and prints contents to stdout.
func TestConfigCreate_NewFile(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", "")

	stdout, stderr := captureConfigOutput(t, func() error {
		return runConfigCreate()
	})

	// Verify file was created
	expectedPath := filepath.Join(tmpHome, ".config", "calcmark", "config.toml")
	content, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("config file not created at %s: %v", expectedPath, err)
	}

	// File content should have commented-out values
	fileStr := string(content)
	if !strings.Contains(fileStr, "# locale") {
		t.Error("expected commented-out locale in created file")
	}
	if !strings.Contains(fileStr, "# color_mode") {
		t.Error("expected commented-out color_mode in created file")
	}

	// Section headers should NOT be commented out
	if !strings.Contains(fileStr, "[tui]") {
		t.Error("expected [tui] section header in created file")
	}
	if !strings.Contains(fileStr, "[formatter]") {
		t.Error("expected [formatter] section header in created file")
	}

	// Stdout should contain the file contents
	if !strings.Contains(stdout, "# locale") {
		t.Error("expected file contents on stdout")
	}

	// Stderr should contain confirmation
	if !strings.Contains(stderr, "Created") {
		t.Error("expected 'Created' confirmation on stderr")
	}
}

// TestConfigCreate_ExistingFile verifies that --create refuses to overwrite
// an existing config file.
func TestConfigCreate_ExistingFile(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", "")

	// Pre-create the config file
	configDir := filepath.Join(tmpHome, ".config", "calcmark")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	existingContent := "locale = \"fr-FR\"\n"
	configPath := filepath.Join(configDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("failed to write existing config: %v", err)
	}

	// runConfigCreate should return an error
	err := runConfigCreate()
	if err == nil {
		t.Fatal("expected error when config file already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' in error, got: %v", err)
	}

	// Verify existing file was NOT modified
	content, _ := os.ReadFile(configPath)
	if string(content) != existingContent {
		t.Error("existing config file was modified")
	}
}

// TestConfigCreate_ExistingDir verifies that --create works when the config
// directory already exists but the file does not.
func TestConfigCreate_ExistingDir(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", "")

	// Pre-create just the directory
	configDir := filepath.Join(tmpHome, ".config", "calcmark")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	_, _ = captureConfigOutput(t, func() error {
		return runConfigCreate()
	})

	// Verify file was created
	configPath := filepath.Join(configDir, "config.toml")
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("config file not created: %v", err)
	}
}

// TestConfigCreate_NoHome verifies that --create fails gracefully when
// $HOME cannot be determined.
func TestConfigCreate_NoHome(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	err := runConfigCreate()
	if err == nil {
		t.Fatal("expected error when $HOME is empty")
	}
}

// TestCommentOutDefaults verifies that commentOutDefaults correctly transforms
// embedded TOML into a user-facing template.
func TestCommentOutDefaults(t *testing.T) {
	input := `# CalcMark defaults
# Built-in settings

locale = "en-US"

[tui]
# Color mode
color_mode = "dark"

[tui.theme]
primary = ""
accent = ""
`
	got := commentOutDefaults(input)

	// Header should be replaced
	if !strings.Contains(got, "# CalcMark Configuration") {
		t.Error("expected custom header")
	}
	if strings.Contains(got, "# Built-in settings") {
		t.Error("original header comments should be stripped")
	}

	// Key=value lines should be commented out
	if !strings.Contains(got, "# locale") {
		t.Error("expected locale to be commented out")
	}
	if !strings.Contains(got, "# color_mode") {
		t.Error("expected color_mode to be commented out")
	}
	if !strings.Contains(got, "# primary") {
		t.Error("expected primary to be commented out")
	}

	// Section headers must stay uncommented
	if !strings.Contains(got, "\n[tui]\n") {
		t.Error("expected [tui] section header to remain uncommented")
	}
	if !strings.Contains(got, "\n[tui.theme]\n") {
		t.Error("expected [tui.theme] section header to remain uncommented")
	}

	// Descriptive comments should be preserved
	if !strings.Contains(got, "# Color mode") {
		t.Error("expected descriptive comment to be preserved")
	}
}

// TestConfigCheck_ValidConfig verifies --check exits cleanly with valid config.
func TestConfigCheck_ValidConfig(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", "")

	configDir := filepath.Join(tmpHome, ".config", "calcmark")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(`
locale = "de-DE"
[tui]
color_mode = "light"
[tui.theme]
primary = "#5B21B6"
[formatter]
default_format = "json"
`), 0644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := captureConfigOutputWithErr(t, runConfigCheck)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(stderr, "config ok") {
		t.Errorf("expected 'config ok' on stderr, got: %s", stderr)
	}
}

// TestConfigCheck_InvalidColor verifies --check catches invalid hex colors.
func TestConfigCheck_InvalidColor(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", "")

	configDir := filepath.Join(tmpHome, ".config", "calcmark")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(`
[tui.theme]
primary = "red"
`), 0644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := captureConfigOutputWithErr(t, runConfigCheck)
	if err == nil {
		t.Fatal("expected error for invalid hex color")
	}
	if !strings.Contains(stderr, "invalid hex color") {
		t.Errorf("expected 'invalid hex color' on stderr, got: %s", stderr)
	}
}

// TestConfigCheck_InvalidColorMode verifies --check catches invalid color_mode.
func TestConfigCheck_InvalidColorMode(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", "")

	configDir := filepath.Join(tmpHome, ".config", "calcmark")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(`
[tui]
color_mode = "system"
`), 0644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := captureConfigOutputWithErr(t, runConfigCheck)
	if err == nil {
		t.Fatal("expected error for invalid color_mode")
	}
	if !strings.Contains(stderr, "color_mode") {
		t.Errorf("expected 'color_mode' on stderr, got: %s", stderr)
	}
}

// TestConfigCheck_InvalidFormat verifies --check catches invalid default_format.
func TestConfigCheck_InvalidFormat(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", "")

	configDir := filepath.Join(tmpHome, ".config", "calcmark")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(`
[formatter]
default_format = "xml"
`), 0644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := captureConfigOutputWithErr(t, runConfigCheck)
	if err == nil {
		t.Fatal("expected error for invalid default_format")
	}
	if !strings.Contains(stderr, "default_format") {
		t.Errorf("expected 'default_format' on stderr, got: %s", stderr)
	}
}

// TestConfigCheck_TOMLSyntaxError verifies --check catches TOML syntax errors.
func TestConfigCheck_TOMLSyntaxError(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", "")

	configDir := filepath.Join(tmpHome, ".config", "calcmark")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte("primary = [\n"), 0644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := captureConfigOutputWithErr(t, runConfigCheck)
	if err == nil {
		t.Fatal("expected error for TOML syntax error")
	}
	if !strings.Contains(stderr, "TOML syntax error") {
		t.Errorf("expected 'TOML syntax error' on stderr, got: %s", stderr)
	}
}

// TestConfigCheck_NoConfigFiles verifies --check succeeds with no config files.
func TestConfigCheck_NoConfigFiles(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", "")

	_, stderr, err := captureConfigOutputWithErr(t, runConfigCheck)
	if err != nil {
		t.Fatalf("expected no error with no config files, got: %v", err)
	}
	if !strings.Contains(stderr, "config ok") {
		t.Errorf("expected 'config ok' on stderr, got: %s", stderr)
	}
}

// TestConfigCheck_MultipleErrors verifies --check reports all errors at once.
func TestConfigCheck_MultipleErrors(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", "")

	configDir := filepath.Join(tmpHome, ".config", "calcmark")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(`
[tui]
color_mode = "system"
[tui.theme]
primary = "red"
[formatter]
default_format = "xml"
`), 0644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := captureConfigOutputWithErr(t, runConfigCheck)
	if err == nil {
		t.Fatal("expected error for multiple validation failures")
	}
	// Should report all three errors
	if !strings.Contains(stderr, "color_mode") {
		t.Error("expected color_mode error on stderr")
	}
	if !strings.Contains(stderr, "primary") {
		t.Error("expected primary error on stderr")
	}
	if !strings.Contains(stderr, "default_format") {
		t.Error("expected default_format error on stderr")
	}
}

// captureConfigOutputWithErr captures stdout, stderr, and the returned error.
func captureConfigOutputWithErr(t *testing.T, fn func() error) (stdout, stderr string, fnErr error) {
	t.Helper()

	oldStdout := os.Stdout
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	os.Stdout = wOut

	oldStderr := os.Stderr
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stderr pipe: %v", err)
	}
	os.Stderr = wErr

	fnErr = fn()

	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var bufOut, bufErr bytes.Buffer
	io.Copy(&bufOut, rOut)
	io.Copy(&bufErr, rErr)

	return bufOut.String(), bufErr.String(), fnErr
}

// captureConfigOutput captures both stdout and stderr during execution of fn.
func captureConfigOutput(t *testing.T, fn func() error) (stdout, stderr string) {
	t.Helper()

	// Capture stdout
	oldStdout := os.Stdout
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	os.Stdout = wOut

	// Capture stderr
	oldStderr := os.Stderr
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stderr pipe: %v", err)
	}
	os.Stderr = wErr

	fnErr := fn()

	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var bufOut, bufErr bytes.Buffer
	io.Copy(&bufOut, rOut)
	io.Copy(&bufErr, rErr)

	if fnErr != nil {
		t.Fatalf("function returned error: %v", fnErr)
	}

	return bufOut.String(), bufErr.String()
}
