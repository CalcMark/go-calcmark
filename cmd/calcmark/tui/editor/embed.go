package editor

import _ "embed"

// DefaultFrontmatter is the template inserted by Ctrl+F.
// It lives in a .cm file so it can be validated by the spec layer in tests.
//
//go:embed default_frontmatter.cm
var DefaultFrontmatter string
