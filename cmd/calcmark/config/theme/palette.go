package theme

import "github.com/charmbracelet/lipgloss"

// Semantic color palette with light/dark variants.
// Colors are named by their PURPOSE, not their appearance.
// Every color used in the TUI must be defined here as an AdaptiveColor.
// lipgloss resolves the Light/Dark slot at Render() time based on
// SetHasDarkBackground(), so styles built with these values adapt automatically.
var (
	// --- Core text colors ---

	Text = lipgloss.AdaptiveColor{
		Light: "#1a1a1a",
		Dark:  "#e5e5e5",
	}

	TextMuted = lipgloss.AdaptiveColor{
		Light: "#666666",
		Dark:  "#888888",
	}

	TextBright = lipgloss.AdaptiveColor{
		Light: "#000000",
		Dark:  "#ffffff",
	}

	// --- Brand / accent ---

	Primary = lipgloss.AdaptiveColor{
		Light: "#6639b6",
		Dark:  "#7D56F4",
	}

	Accent = lipgloss.AdaptiveColor{
		Light: "#6f3fd6",
		Dark:  "#874BFD",
	}

	// --- Semantic status colors ---

	Error = lipgloss.AdaptiveColor{
		Light: "#cc3333",
		Dark:  "#FF5555",
	}

	ErrorIcon = lipgloss.AdaptiveColor{
		Light: "#cc0000",
		Dark:  "#FF4444",
	}

	Warning = lipgloss.AdaptiveColor{
		Light: "#b35900",
		Dark:  "#FFAA00",
	}

	Success = lipgloss.AdaptiveColor{
		Light: "#1a7f37",
		Dark:  "#4ECDC4",
	}

	// --- Results and values ---

	Result = lipgloss.AdaptiveColor{
		Light: "#0969da",
		Dark:  "#4ECDC4",
	}

	ResultMuted = lipgloss.AdaptiveColor{
		Light: "#57606a",
		Dark:  "#BBBBBB",
	}

	ResultArrow = lipgloss.AdaptiveColor{
		Light: "#57606a",
		Dark:  "#888888",
	}

	// --- Headers ---

	Header = lipgloss.AdaptiveColor{
		Light: "#1a1a1a",
		Dark:  "#ffffff",
	}

	// --- UI chrome ---

	Cursor = lipgloss.AdaptiveColor{
		Light: "#6639b6",
		Dark:  "#7D56F4",
	}

	CursorFg = lipgloss.AdaptiveColor{
		Light: "#ffffff",
		Dark:  "#000000",
	}

	Selection = lipgloss.AdaptiveColor{
		Light: "#ddf4ff",
		Dark:  "#1f3a5f",
	}

	SelectionFg = lipgloss.AdaptiveColor{
		Light: "#1a1a1a",
		Dark:  "#ffffff",
	}

	Hint = lipgloss.AdaptiveColor{
		Light: "#6e7781",
		Dark:  "#6e7781",
	}

	Command = lipgloss.AdaptiveColor{
		Light: "#8250df",
		Dark:  "#a371f7",
	}

	// --- Separator / divider ---

	Separator = lipgloss.AdaptiveColor{
		Light: "#d1d5db",
		Dark:  "#555555",
	}

	DividerFg = lipgloss.AdaptiveColor{
		Light: "#c0c0c0",
		Dark:  "#444444",
	}

	// --- Borders ---

	Border = lipgloss.AdaptiveColor{
		Light: "#d1d5db",
		Dark:  "#3d3d3d",
	}

	// --- Pane backgrounds ---

	SourcePaneBg = lipgloss.AdaptiveColor{
		Light: "#fafafa",
		Dark:  "#1C1C1C",
	}

	PreviewPaneBg = lipgloss.AdaptiveColor{
		Light: "#f5f5f5",
		Dark:  "#1a1a1a",
	}

	PaneBg = lipgloss.AdaptiveColor{
		Light: "#ffffff",
		Dark:  "#1a1a1a",
	}

	// --- Status bar ---

	StatusBg = lipgloss.AdaptiveColor{
		Light: "#e8e8e8",
		Dark:  "#2A2A2A",
	}

	StatusFg = lipgloss.AdaptiveColor{
		Light: "#374151",
		Dark:  "#FFFFFF",
	}

	// --- Context footer ---

	ContextFooterBg = lipgloss.AdaptiveColor{
		Light: "#f0f0f0",
		Dark:  "#252525",
	}

	// --- Popup / autocomplete ---

	PopupBg = lipgloss.AdaptiveColor{
		Light: "#ffffff",
		Dark:  "#1E1E1E",
	}

	PopupBorder = lipgloss.AdaptiveColor{
		Light: "#c0c0c0",
		Dark:  "#5C5C5C",
	}

	PopupSelectedBg = lipgloss.AdaptiveColor{
		Light: "#dbeafe",
		Dark:  "#4A90D9",
	}

	PopupSelectedFg = lipgloss.AdaptiveColor{
		Light: "#1a1a1a",
		Dark:  "#FFFFFF",
	}

	// --- Overlay (help, export, command menu, file picker) ---

	OverlayBg = lipgloss.AdaptiveColor{
		Light: "#ffffff",
		Dark:  "#1E1E1E",
	}

	OverlayBorder = lipgloss.AdaptiveColor{
		Light: "#c0c0c0",
		Dark:  "#5C5C5C",
	}

	// --- Source pane syntax highlighting (block-level) ---

	SourceFrontmatter = lipgloss.AdaptiveColor{
		Light: "#6e7781",
		Dark:  "#7d8590",
	}

	SourceMarkdown = lipgloss.AdaptiveColor{
		Light: "#1a1a1a",
		Dark:  "#e5e5e5",
	}

	SourceCalc = lipgloss.AdaptiveColor{
		Light: "#0969da",
		Dark:  "#79c0ff",
	}

	// --- Editor-specific ---

	EditLineBg = lipgloss.AdaptiveColor{
		Light: "#f0f0f0",
		Dark:  "#2E2E2E",
	}

	EditLineFg = lipgloss.AdaptiveColor{
		Light: "#1a1a1a",
		Dark:  "#FFFFFF",
	}

	CurrentLineBg = lipgloss.AdaptiveColor{
		Light: "#f5f5f5",
		Dark:  "#262626",
	}

	CurrentLineFg = lipgloss.AdaptiveColor{
		Light: "#1a1a1a",
		Dark:  "#FFFFFF",
	}

	LineNumber = lipgloss.AdaptiveColor{
		Light: "#999999",
		Dark:  "#666666",
	}

	// --- Globals panel ---

	GlobalsVarName = lipgloss.AdaptiveColor{
		Light: "#0969da",
		Dark:  "#4ECDC4",
	}

	GlobalsExchange = lipgloss.AdaptiveColor{
		Light: "#b35900",
		Dark:  "#FFD93D",
	}

	GlobalsFocusBg = lipgloss.AdaptiveColor{
		Light: "#e8e8e8",
		Dark:  "#333333",
	}

	// --- Input / prompt ---

	PromptFg = lipgloss.AdaptiveColor{
		Light: "#1a1a1a",
		Dark:  "#FFFFFF",
	}

	PromptBg = lipgloss.AdaptiveColor{
		Light: "#e0e0e0",
		Dark:  "#333333",
	}

	InputFg = lipgloss.AdaptiveColor{
		Light: "#1a1a1a",
		Dark:  "#FFFFFF",
	}

	InputBg = lipgloss.AdaptiveColor{
		Light: "#ffffff",
		Dark:  "#1A1A1A",
	}

	// --- Context footer detail colors ---

	FooterFuncName = lipgloss.AdaptiveColor{
		Light: "#0969da",
		Dark:  "#58a6ff",
	}

	FooterParamHighlight = lipgloss.AdaptiveColor{
		Light: "#b35900",
		Dark:  "#FFD93D",
	}

	FooterVarRef = lipgloss.AdaptiveColor{
		Light: "#666666",
		Dark:  "#888888",
	}

	// --- Markdown preview (glamour) ---

	MdH1Bg = lipgloss.AdaptiveColor{
		Light: "#fff3e0",
		Dark:  "#3d2800",
	}

	MdH2Bg = lipgloss.AdaptiveColor{
		Light: "#e8f5e9",
		Dark:  "#1a3d1a",
	}

	MdHeading = lipgloss.AdaptiveColor{
		Light: "#b35900",
		Dark:  "#FF9900",
	}

	MdLink = lipgloss.AdaptiveColor{
		Light: "#0969da",
		Dark:  "#22AA22",
	}

	MdQuote = lipgloss.AdaptiveColor{
		Light: "#666666",
		Dark:  "#888888",
	}

	MdCode = lipgloss.AdaptiveColor{
		Light: "#c7254e",
		Dark:  "#33CC33",
	}

	MdCodeBg = lipgloss.AdaptiveColor{
		Light: "#f5f2f0",
		Dark:  "#333333",
	}

	// --- File picker ---

	FilePickerDir = lipgloss.AdaptiveColor{
		Light: "#0969da",
		Dark:  "#58a6ff",
	}

	FilePickerFile = lipgloss.AdaptiveColor{
		Light: "#1a1a1a",
		Dark:  "#e5e5e5",
	}

	FilePickerSelected = lipgloss.AdaptiveColor{
		Light: "#6639b6",
		Dark:  "#a371f7",
	}

	FilePickerCursor = lipgloss.AdaptiveColor{
		Light: "#8250df",
		Dark:  "#d2a8ff",
	}

	// --- Mode indicator ---

	ModeIndicatorBg = lipgloss.AdaptiveColor{
		Light: "#0969da",
		Dark:  "#4ECDC4",
	}

	ModeIndicatorFg = lipgloss.AdaptiveColor{
		Light: "#ffffff",
		Dark:  "#000000",
	}
)
