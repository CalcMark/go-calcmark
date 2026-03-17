; CalcMark syntax highlighting queries for tree-sitter
; Used by Zed and other tree-sitter-aware editors

; Headings
(heading_marker) @keyword
(heading_content) @title

; Assignments — LHS variable gets declaration highlight
(assignment
  name: (identifier) @variable.declaration)

; Function calls
(function_call
  name: (function_name) @function)

; NL function syntax
(nl_function_name) @function

; Numbers and literals
(number) @number
(percentage) @number
(fraction) @number
(currency_literal) @number
(boolean) @constant.builtin

; Identifiers (variables)
(identifier) @variable

; Operators
(binary_expression
  operator: _ @operator)

; Keywords
"in" @keyword
"as" @keyword
"and" @keyword
"or" @keyword
"not" @keyword

; Punctuation
"=" @operator
"(" @punctuation.bracket
")" @punctuation.bracket
"," @punctuation.delimiter

; Text/markdown lines (fallback)
(text_line) @comment
