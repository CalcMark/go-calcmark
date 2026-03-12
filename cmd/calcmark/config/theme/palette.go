package theme

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
)

// ac is a shorthand to create an adaptive color from light/dark hex strings.
func ac(light, dark string) compat.AdaptiveColor {
	return compat.AdaptiveColor{Light: lipgloss.Color(light), Dark: lipgloss.Color(dark)}
}

// Semantic color palette with light/dark variants.
// Colors are named by their PURPOSE, not their appearance.
// Every color used in the TUI must be defined here as an AdaptiveColor.
// lipgloss resolves the Light/Dark slot at Render() time based on
// compat.HasDarkBackground, so styles built with these values adapt automatically.
//
// Theme: "Pearish" — warm stone tones with pear-green accents.
// Light: warm off-white backgrounds (#FAFAF9), brown-tinted text (#39322D)
// Dark: warm dark backgrounds (#211D19), warm light text (#F5F3F0)
var (
	// --- Core text colors ---

	Text = ac(
		"#39322D",
		"#F5F3F0",
	)

	TextMuted = ac(
		"#6B7280",
		"#9CA3AF",
	)

	TextBright = ac(
		"#1F1A17",
		"#FFFFFF",
	)

	// --- Brand / accent ---

	Primary = ac(
		"#82B919",
		"#A8C940",
	)

	Accent = ac(
		"#65A30D",
		"#BEF264",
	)

	// --- Semantic status colors ---

	Error = ac(
		"#E84327",
		"#F87171",
	)

	ErrorIcon = ac(
		"#DC2626",
		"#FCA5A5",
	)

	Warning = ac(
		"#D97706",
		"#F59E0B",
	)

	Success = ac(
		"#82B919",
		"#A8C940",
	)

	// --- Results and values ---

	Result = ac(
		"#82B919",
		"#A8C940",
	)

	ResultMuted = ac(
		"#6B7280",
		"#9CA3AF",
	)

	ResultArrow = ac(
		"#9CA3AF",
		"#6B7280",
	)

	// --- Headers ---

	Header = ac(
		"#39322D",
		"#F5F3F0",
	)

	// --- UI chrome ---

	Cursor = ac(
		"#2563EB",
		"#60A5FA",
	)

	CursorFg = ac(
		"#FFFFFF",
		"#211D19",
	)

	Selection = ac(
		"#DBEAFE",
		"#1E3A5F",
	)

	SelectionFg = ac(
		"#39322D",
		"#F5F3F0",
	)

	Hint = ac(
		"#6B7280",
		"#9CA3AF",
	)

	Command = ac(
		"#9333EA",
		"#A855F7",
	)

	// --- Separator / divider ---

	Separator = ac(
		"#D1D5DB",
		"#4B5563",
	)

	DividerFg = ac(
		"#E5E7EB",
		"#374151",
	)

	// --- Borders ---

	Border = ac(
		"#D1D5DB",
		"#4B5563",
	)

	// --- Pane backgrounds ---

	SourcePaneBg = ac(
		"#FAFAF9",
		"#211D19",
	)

	PreviewPaneBg = ac(
		"#F9FAFB",
		"#1A1714",
	)

	// ReadingCursorBg — visible highlight for cursor line in Reading mode.
	// Noticeably lighter than PreviewPaneBg so the line clearly stands out.
	ReadingCursorBg = ac(
		"#E2E8F0",
		"#3D3530",
	)

	PaneBg = ac(
		"#FAFAF9",
		"#211D19",
	)

	// --- Status bar ---

	StatusBg = ac(
		"#F3F4F6",
		"#2A2520",
	)

	StatusFg = ac(
		"#39322D",
		"#F5F3F0",
	)

	// --- Context footer ---

	ContextFooterBg = ac(
		"#F3F4F6",
		"#2A2520",
	)

	// --- Popup / autocomplete ---

	PopupBg = ac(
		"#FFFFFF",
		"#2A2520",
	)

	PopupBorder = ac(
		"#D1D5DB",
		"#4B5563",
	)

	PopupSelectedBg = ac(
		"#E8F5E9",
		"#3A3330",
	)

	PopupSelectedFg = ac(
		"#39322D",
		"#F5F3F0",
	)

	// --- Overlay (help, export, command menu, file picker) ---

	OverlayBg = ac(
		"#FFFFFF",
		"#2A2520",
	)

	OverlayBorder = ac(
		"#D1D5DB",
		"#4B5563",
	)

	// --- Source pane syntax highlighting (block-level) ---

	SourceFrontmatter = ac(
		"#6B7280",
		"#B0B8C4",
	)

	SourceMarkdown = ac(
		"#39322D",
		"#F5F3F0",
	)

	SourceCalc = ac(
		"#2563EB",
		"#60A5FA",
	)

	// --- Editor-specific ---

	EditLineBg = ac(
		"#F5F3F0",
		"#2A2520",
	)

	EditLineFg = ac(
		"#39322D",
		"#F5F3F0",
	)

	CurrentLineBg = ac(
		"#F5F3F0",
		"#2A2520",
	)

	CurrentLineFg = ac(
		"#39322D",
		"#F5F3F0",
	)

	LineNumber = ac(
		"#9CA3AF",
		"#6B7280",
	)

	// --- Globals panel ---

	GlobalsVarName = ac(
		"#82B919",
		"#A8C940",
	)

	GlobalsExchange = ac(
		"#D97706",
		"#F59E0B",
	)

	GlobalsFocusBg = ac(
		"#E5E7EB",
		"#3A3330",
	)

	// --- Input / prompt ---

	PromptFg = ac(
		"#39322D",
		"#F5F3F0",
	)

	PromptBg = ac(
		"#E5E7EB",
		"#3A3330",
	)

	InputFg = ac(
		"#39322D",
		"#F5F3F0",
	)

	InputBg = ac(
		"#FFFFFF",
		"#211D19",
	)

	// --- Context footer detail colors ---

	FooterFuncName = ac(
		"#2563EB",
		"#60A5FA",
	)

	FooterParamHighlight = ac(
		"#D97706",
		"#FCD34D",
	)

	FooterVarRef = ac(
		"#6B7280",
		"#9CA3AF",
	)

	// --- Markdown preview (glamour) ---

	MdH1Bg = ac(
		"#FEF3C7",
		"#3D2800",
	)

	MdH2Bg = ac(
		"#E8F5E9",
		"#1A3D1A",
	)

	MdHeading = ac(
		"#82B919",
		"#A8C940",
	)

	MdLink = ac(
		"#2563EB",
		"#60A5FA",
	)

	MdQuote = ac(
		"#6B7280",
		"#9CA3AF",
	)

	MdCode = ac(
		"#E84327",
		"#F87171",
	)

	MdCodeBg = ac(
		"#F3F4F6",
		"#2A2520",
	)

	// --- File picker ---

	FilePickerDir = ac(
		"#2563EB",
		"#60A5FA",
	)

	FilePickerFile = ac(
		"#39322D",
		"#F5F3F0",
	)

	FilePickerSelected = ac(
		"#82B919",
		"#A8C940",
	)

	FilePickerCursor = ac(
		"#9333EA",
		"#A855F7",
	)

	// --- Mode indicator ---

	ModeIndicatorBg = ac(
		"#82B919",
		"#A8C940",
	)

	ModeIndicatorFg = ac(
		"#FFFFFF",
		"#211D19",
	)

	// --- Autosuggest inline hints (distinct from popup) ---

	AutosuggestText = ac(
		"#6B7280",
		"#9CA3AF",
	)

	AutosuggestSyntax = ac(
		"#82B919",
		"#A8C940",
	)

	AutosuggestSeparator = ac(
		"#D1D5DB",
		"#4B5563",
	)

	AutosuggestSelectedBg = ac(
		"#E5E7EB",
		"#3A3330",
	)

	// --- Overlay backdrop ---

	OverlayWhitespaceFg = ac(
		"#E5E7EB",
		"#2A2520",
	)

	// --- Calculation result errors ---

	CalcErrorFg = ac(
		"#D97706",
		"#F59E0B",
	)

	CalcBlockedFg = ac(
		"#9CA3AF",
		"#6B7280",
	)

	ScaleIndicator = ac(
		"#D97706",
		"#F59E0B",
	)

	ConvertIndicator = ac(
		"#0891B2", // teal-600: distinct from amber scale, readable on light bg
		"#22D3EE", // teal-300: bright on dark bg, clear next to amber
	)
)
