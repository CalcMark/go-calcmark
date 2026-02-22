package theme

import "github.com/charmbracelet/lipgloss"

// Semantic color palette with light/dark variants.
// Colors are named by their PURPOSE, not their appearance.
// Every color used in the TUI must be defined here as an AdaptiveColor.
// lipgloss resolves the Light/Dark slot at Render() time based on
// SetHasDarkBackground(), so styles built with these values adapt automatically.
//
// Theme: "Pearish" — warm stone tones with pear-green accents.
// Light: warm off-white backgrounds (#FAFAF9), brown-tinted text (#39322D)
// Dark: warm dark backgrounds (#211D19), warm light text (#F5F3F0)
var (
	// --- Core text colors ---

	Text = lipgloss.AdaptiveColor{
		Light: "#39322D",
		Dark:  "#F5F3F0",
	}

	TextMuted = lipgloss.AdaptiveColor{
		Light: "#6B7280",
		Dark:  "#9CA3AF",
	}

	TextBright = lipgloss.AdaptiveColor{
		Light: "#1F1A17",
		Dark:  "#FFFFFF",
	}

	// --- Brand / accent ---

	Primary = lipgloss.AdaptiveColor{
		Light: "#82B919",
		Dark:  "#A8C940",
	}

	Accent = lipgloss.AdaptiveColor{
		Light: "#65A30D",
		Dark:  "#BEF264",
	}

	// --- Semantic status colors ---

	Error = lipgloss.AdaptiveColor{
		Light: "#E84327",
		Dark:  "#F87171",
	}

	ErrorIcon = lipgloss.AdaptiveColor{
		Light: "#DC2626",
		Dark:  "#FCA5A5",
	}

	Warning = lipgloss.AdaptiveColor{
		Light: "#D97706",
		Dark:  "#F59E0B",
	}

	Success = lipgloss.AdaptiveColor{
		Light: "#82B919",
		Dark:  "#A8C940",
	}

	// --- Results and values ---

	Result = lipgloss.AdaptiveColor{
		Light: "#82B919",
		Dark:  "#A8C940",
	}

	ResultMuted = lipgloss.AdaptiveColor{
		Light: "#6B7280",
		Dark:  "#9CA3AF",
	}

	ResultArrow = lipgloss.AdaptiveColor{
		Light: "#9CA3AF",
		Dark:  "#6B7280",
	}

	// --- Headers ---

	Header = lipgloss.AdaptiveColor{
		Light: "#39322D",
		Dark:  "#F5F3F0",
	}

	// --- UI chrome ---

	Cursor = lipgloss.AdaptiveColor{
		Light: "#2563EB",
		Dark:  "#60A5FA",
	}

	CursorFg = lipgloss.AdaptiveColor{
		Light: "#FFFFFF",
		Dark:  "#211D19",
	}

	Selection = lipgloss.AdaptiveColor{
		Light: "#DBEAFE",
		Dark:  "#1E3A5F",
	}

	SelectionFg = lipgloss.AdaptiveColor{
		Light: "#39322D",
		Dark:  "#F5F3F0",
	}

	Hint = lipgloss.AdaptiveColor{
		Light: "#6B7280",
		Dark:  "#9CA3AF",
	}

	Command = lipgloss.AdaptiveColor{
		Light: "#9333EA",
		Dark:  "#A855F7",
	}

	// --- Separator / divider ---

	Separator = lipgloss.AdaptiveColor{
		Light: "#D1D5DB",
		Dark:  "#4B5563",
	}

	DividerFg = lipgloss.AdaptiveColor{
		Light: "#E5E7EB",
		Dark:  "#374151",
	}

	// --- Borders ---

	Border = lipgloss.AdaptiveColor{
		Light: "#D1D5DB",
		Dark:  "#4B5563",
	}

	// --- Pane backgrounds ---

	SourcePaneBg = lipgloss.AdaptiveColor{
		Light: "#FAFAF9",
		Dark:  "#211D19",
	}

	PreviewPaneBg = lipgloss.AdaptiveColor{
		Light: "#F9FAFB",
		Dark:  "#1A1714",
	}

	PaneBg = lipgloss.AdaptiveColor{
		Light: "#FAFAF9",
		Dark:  "#211D19",
	}

	// --- Status bar ---

	StatusBg = lipgloss.AdaptiveColor{
		Light: "#F3F4F6",
		Dark:  "#2A2520",
	}

	StatusFg = lipgloss.AdaptiveColor{
		Light: "#39322D",
		Dark:  "#F5F3F0",
	}

	// --- Context footer ---

	ContextFooterBg = lipgloss.AdaptiveColor{
		Light: "#F3F4F6",
		Dark:  "#2A2520",
	}

	// --- Popup / autocomplete ---

	PopupBg = lipgloss.AdaptiveColor{
		Light: "#FFFFFF",
		Dark:  "#2A2520",
	}

	PopupBorder = lipgloss.AdaptiveColor{
		Light: "#D1D5DB",
		Dark:  "#4B5563",
	}

	PopupSelectedBg = lipgloss.AdaptiveColor{
		Light: "#E8F5E9",
		Dark:  "#3A3330",
	}

	PopupSelectedFg = lipgloss.AdaptiveColor{
		Light: "#39322D",
		Dark:  "#F5F3F0",
	}

	// --- Overlay (help, export, command menu, file picker) ---

	OverlayBg = lipgloss.AdaptiveColor{
		Light: "#FFFFFF",
		Dark:  "#2A2520",
	}

	OverlayBorder = lipgloss.AdaptiveColor{
		Light: "#D1D5DB",
		Dark:  "#4B5563",
	}

	// --- Source pane syntax highlighting (block-level) ---

	SourceFrontmatter = lipgloss.AdaptiveColor{
		Light: "#9CA3AF",
		Dark:  "#6B7280",
	}

	SourceMarkdown = lipgloss.AdaptiveColor{
		Light: "#39322D",
		Dark:  "#F5F3F0",
	}

	SourceCalc = lipgloss.AdaptiveColor{
		Light: "#2563EB",
		Dark:  "#60A5FA",
	}

	// --- Editor-specific ---

	EditLineBg = lipgloss.AdaptiveColor{
		Light: "#F5F3F0",
		Dark:  "#2A2520",
	}

	EditLineFg = lipgloss.AdaptiveColor{
		Light: "#39322D",
		Dark:  "#F5F3F0",
	}

	CurrentLineBg = lipgloss.AdaptiveColor{
		Light: "#F5F3F0",
		Dark:  "#2A2520",
	}

	CurrentLineFg = lipgloss.AdaptiveColor{
		Light: "#39322D",
		Dark:  "#F5F3F0",
	}

	LineNumber = lipgloss.AdaptiveColor{
		Light: "#9CA3AF",
		Dark:  "#6B7280",
	}

	// --- Globals panel ---

	GlobalsVarName = lipgloss.AdaptiveColor{
		Light: "#82B919",
		Dark:  "#A8C940",
	}

	GlobalsExchange = lipgloss.AdaptiveColor{
		Light: "#D97706",
		Dark:  "#F59E0B",
	}

	GlobalsFocusBg = lipgloss.AdaptiveColor{
		Light: "#E5E7EB",
		Dark:  "#3A3330",
	}

	// --- Input / prompt ---

	PromptFg = lipgloss.AdaptiveColor{
		Light: "#39322D",
		Dark:  "#F5F3F0",
	}

	PromptBg = lipgloss.AdaptiveColor{
		Light: "#E5E7EB",
		Dark:  "#3A3330",
	}

	InputFg = lipgloss.AdaptiveColor{
		Light: "#39322D",
		Dark:  "#F5F3F0",
	}

	InputBg = lipgloss.AdaptiveColor{
		Light: "#FFFFFF",
		Dark:  "#211D19",
	}

	// --- Context footer detail colors ---

	FooterFuncName = lipgloss.AdaptiveColor{
		Light: "#2563EB",
		Dark:  "#60A5FA",
	}

	FooterParamHighlight = lipgloss.AdaptiveColor{
		Light: "#D97706",
		Dark:  "#FCD34D",
	}

	FooterVarRef = lipgloss.AdaptiveColor{
		Light: "#6B7280",
		Dark:  "#9CA3AF",
	}

	// --- Markdown preview (glamour) ---

	MdH1Bg = lipgloss.AdaptiveColor{
		Light: "#FEF3C7",
		Dark:  "#3D2800",
	}

	MdH2Bg = lipgloss.AdaptiveColor{
		Light: "#E8F5E9",
		Dark:  "#1A3D1A",
	}

	MdHeading = lipgloss.AdaptiveColor{
		Light: "#82B919",
		Dark:  "#A8C940",
	}

	MdLink = lipgloss.AdaptiveColor{
		Light: "#2563EB",
		Dark:  "#60A5FA",
	}

	MdQuote = lipgloss.AdaptiveColor{
		Light: "#6B7280",
		Dark:  "#9CA3AF",
	}

	MdCode = lipgloss.AdaptiveColor{
		Light: "#E84327",
		Dark:  "#F87171",
	}

	MdCodeBg = lipgloss.AdaptiveColor{
		Light: "#F3F4F6",
		Dark:  "#2A2520",
	}

	// --- File picker ---

	FilePickerDir = lipgloss.AdaptiveColor{
		Light: "#2563EB",
		Dark:  "#60A5FA",
	}

	FilePickerFile = lipgloss.AdaptiveColor{
		Light: "#39322D",
		Dark:  "#F5F3F0",
	}

	FilePickerSelected = lipgloss.AdaptiveColor{
		Light: "#82B919",
		Dark:  "#A8C940",
	}

	FilePickerCursor = lipgloss.AdaptiveColor{
		Light: "#9333EA",
		Dark:  "#A855F7",
	}

	// --- Mode indicator ---

	ModeIndicatorBg = lipgloss.AdaptiveColor{
		Light: "#82B919",
		Dark:  "#A8C940",
	}

	ModeIndicatorFg = lipgloss.AdaptiveColor{
		Light: "#FFFFFF",
		Dark:  "#211D19",
	}

	// --- Autosuggest inline hints (distinct from popup) ---

	AutosuggestText = lipgloss.AdaptiveColor{
		Light: "#6B7280",
		Dark:  "#9CA3AF",
	}

	AutosuggestSyntax = lipgloss.AdaptiveColor{
		Light: "#82B919",
		Dark:  "#A8C940",
	}

	AutosuggestSeparator = lipgloss.AdaptiveColor{
		Light: "#D1D5DB",
		Dark:  "#4B5563",
	}

	AutosuggestSelectedBg = lipgloss.AdaptiveColor{
		Light: "#E5E7EB",
		Dark:  "#3A3330",
	}

	// --- Overlay backdrop ---

	OverlayWhitespaceFg = lipgloss.AdaptiveColor{
		Light: "#E5E7EB",
		Dark:  "#2A2520",
	}

	// --- Calculation result errors ---

	CalcErrorFg = lipgloss.AdaptiveColor{
		Light: "#D97706",
		Dark:  "#F59E0B",
	}

	CalcBlockedFg = lipgloss.AdaptiveColor{
		Light: "#9CA3AF",
		Dark:  "#6B7280",
	}
)
