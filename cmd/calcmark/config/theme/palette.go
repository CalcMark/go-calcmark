package theme

import "github.com/charmbracelet/lipgloss"

// Semantic color palette with light/dark variants.
// Colors are named by their PURPOSE, not their appearance.
var (
	// Core text colors
	Text = lipgloss.AdaptiveColor{
		Light: "#1a1a1a",
		Dark:  "#e5e5e5",
	}

	TextMuted = lipgloss.AdaptiveColor{
		Light: "#666666",
		Dark:  "#888888",
	}

	// Results and values
	Result = lipgloss.AdaptiveColor{
		Light: "#0969da",
		Dark:  "#58a6ff",
	}

	ResultMuted = lipgloss.AdaptiveColor{
		Light: "#57606a",
		Dark:  "#7d8590",
	}

	// Errors and warnings (amber, not red)
	Error = lipgloss.AdaptiveColor{
		Light: "#b35900",
		Dark:  "#f0a020",
	}

	// Headers
	Header = lipgloss.AdaptiveColor{
		Light: "#1a1a1a",
		Dark:  "#ffffff",
	}

	// UI elements
	Cursor = lipgloss.AdaptiveColor{
		Light: "#f6f8fa",
		Dark:  "#262626",
	}

	Selection = lipgloss.AdaptiveColor{
		Light: "#ddf4ff",
		Dark:  "#1f3a5f",
	}

	// Hints and help
	Hint = lipgloss.AdaptiveColor{
		Light: "#6e7781",
		Dark:  "#6e7781",
	}

	// Commands
	Command = lipgloss.AdaptiveColor{
		Light: "#8250df",
		Dark:  "#a371f7",
	}

	// Success states
	Success = lipgloss.AdaptiveColor{
		Light: "#1a7f37",
		Dark:  "#3fb950",
	}

	// Borders and backgrounds
	Border = lipgloss.AdaptiveColor{
		Light: "#d1d5db",
		Dark:  "#3d3d3d",
	}

	PaneBg = lipgloss.AdaptiveColor{
		Light: "#ffffff",
		Dark:  "#1a1a1a",
	}

	StatusBg = lipgloss.AdaptiveColor{
		Light: "#f3f4f6",
		Dark:  "#262626",
	}

	StatusFg = lipgloss.AdaptiveColor{
		Light: "#374151",
		Dark:  "#d1d5d9",
	}
)
