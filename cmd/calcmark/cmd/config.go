package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/config"
	toml "github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
)

var configCreate bool

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Print or create CalcMark configuration",
	Long: `Print the current effective configuration as TOML.

With --create, write a starter config file to the XDG config path
with all values commented out and descriptive comments included.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if configCreate {
			return runConfigCreate()
		}
		return runConfigShow()
	},
}

func init() {
	configCmd.Flags().BoolVar(&configCreate, "create", false,
		"Create a starter config file at ~/.config/calcmark/config.toml")
	rootCmd.AddCommand(configCmd)
}

// runConfigShow prints the fully-resolved effective configuration as TOML.
func runConfigShow() error {
	cfg := config.Get()
	bytes, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	fmt.Println("# CalcMark effective configuration")
	fmt.Println()
	os.Stdout.Write(bytes)
	return nil
}

// runConfigCreate writes a starter config file with all values commented out.
func runConfigCreate() error {
	path, err := config.XDGConfigPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	// Atomic create — fails if file already exists (O_EXCL)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("config file already exists: %s", path)
		}
		return fmt.Errorf("create config file: %w", err)
	}
	defer f.Close()

	content := commentOutDefaults(config.DefaultsTOML())
	if _, err := f.WriteString(content); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Created %s\n", path)
	fmt.Print(content)
	return nil
}

// commentOutDefaults transforms the embedded defaults.toml into a user-facing
// template with all key = value lines commented out while preserving section
// headers and descriptive comments.
func commentOutDefaults(defaults string) string {
	var b strings.Builder
	b.WriteString("# CalcMark Configuration\n")
	b.WriteString("# Uncomment and modify values below to customize.\n")
	b.WriteString("# See: https://calcmark.dev/docs/configuration/\n")

	lines := strings.Split(defaults, "\n")
	pastHeader := false
	for _, line := range lines {
		// Skip the original header block (comment lines before first non-comment content)
		if !pastHeader {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				pastHeader = true
				b.WriteString("\n")
			}
			continue
		}

		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "":
			b.WriteString("\n")
		case strings.HasPrefix(trimmed, "#"):
			// Keep existing descriptive comments
			b.WriteString(line)
			b.WriteByte('\n')
		case strings.HasPrefix(trimmed, "["):
			// Keep section headers (must be uncommented for TOML structure)
			b.WriteByte('\n')
			b.WriteString(line)
			b.WriteByte('\n')
		default:
			// Comment out key = value lines
			b.WriteString("# ")
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	return b.String()
}
