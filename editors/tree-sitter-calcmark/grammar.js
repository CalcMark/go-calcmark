/// <reference types="tree-sitter-cli/dsl" />
// @ts-check

// Tree-sitter grammar for CalcMark.
//
// This is a conservative grammar for syntax highlighting only.
// Context-dependent classification (calc vs markdown) is handled by the
// LSP's semantic tokens — this grammar provides a baseline.

module.exports = grammar({
  name: "calcmark",

  extras: ($) => [/[ \t]/],

  rules: {
    document: ($) => repeat(choice($.heading, $.assignment, $.expression_line, $.blank_line, $.text_line)),

    // Markdown headings: # ... ## ... ### ...
    heading: ($) => seq($.heading_marker, optional($.heading_content), $._newline),

    heading_marker: (_) => /#{1,6}\s/,

    heading_content: (_) => /[^\n]+/,

    // Variable assignment: identifier = expression
    assignment: ($) =>
      seq(
        field("name", $.identifier),
        "=",
        field("value", $._expression),
        $._newline,
      ),

    // Expression on its own line (no assignment)
    expression_line: ($) => seq($._expression, $._newline),

    _expression: ($) =>
      choice(
        $.binary_expression,
        $.unary_expression,
        $.function_call,
        $.nl_function,
        $.unit_conversion,
        $.parenthesized_expression,
        $.currency_literal,
        $.number,
        $.percentage,
        $.fraction,
        $.quantity,
        $.boolean,
        $.identifier,
      ),

    binary_expression: ($) =>
      choice(
        ...[["+", 1], ["-", 1], ["*", 2], ["/", 2], ["%", 2], ["^", 3]].map(
          ([op, prec]) =>
            prec.left(
              /** @type {number} */ (prec),
              seq(field("left", $._expression), field("operator", /** @type {string} */ (op)), field("right", $._expression)),
            ),
        ),
        // Comparison operators
        ...[">=", "<=", "==", "!=", ">", "<"].map((op) =>
          prec.left(0, seq(field("left", $._expression), field("operator", op), field("right", $._expression))),
        ),
        // Logical operators
        prec.left(-1, seq(field("left", $._expression), field("operator", "and"), field("right", $._expression))),
        prec.left(-2, seq(field("left", $._expression), field("operator", "or"), field("right", $._expression))),
      ),

    unary_expression: ($) =>
      choice(
        prec(4, seq("-", $._expression)),
        prec(4, seq("not", $._expression)),
      ),

    function_call: ($) =>
      seq(
        field("name", $.function_name),
        "(",
        optional(seq($._expression, repeat(seq(",", $._expression)))),
        ")",
      ),

    // Natural language function syntax: "average of x, y", "square root of x"
    nl_function: ($) =>
      prec(5, choice(
        seq(alias(/[Aa]verage\s+[Oo]f/, $.nl_function_name), $._expression, repeat(seq(",", $._expression))),
        seq(alias(/[Ss]quare\s+[Rr]oot\s+[Oo]f/, $.nl_function_name), $._expression),
        seq(alias(/[Ss]um\s+[Oo]f/, $.nl_function_name), $._expression, repeat(seq(",", $._expression))),
      )),

    // Unit conversion: expr in unit
    unit_conversion: ($) =>
      prec.left(-3, seq(
        field("value", $._expression),
        field("keyword", choice("in", "as")),
        field("unit", $.identifier),
      )),

    parenthesized_expression: ($) => seq("(", $._expression, ")"),

    // Function names (canonical + synonyms)
    function_name: (_) => /[a-zA-Z_][a-zA-Z0-9_]*/,

    // Currency: $100, $1,000.50
    currency_literal: (_) => /\$[\d][[\d,_]*[\d]]?(\.\d+)?/,

    // Numbers: 42, 3.14, 1,000, 1_000, 1.2e10, 12k, 1.2M, 5B, 2.5T
    number: (_) => /\d[\d,_]*(\.\d+)?([eE][+-]?\d+)?[kKmMbBtT]?/,

    // Percentages: 5%, 10.5%
    percentage: (_) => /\d[\d,_]*(\.\d+)?%/,

    // Fractions: 1/3, 7/8
    fraction: (_) => /\d+\/\d+/,

    // Quantity: number followed by unit identifier (e.g., "5 kg", "10 meters")
    quantity: ($) => seq($.number, $.identifier),

    boolean: (_) => choice("true", "false"),

    // Identifiers: Unicode-aware variable names
    identifier: (_) => /[a-zA-Z_\u00C0-\u024F\u0370-\u03FF\u4E00-\u9FFF\uAC00-\uD7AF][\w\u00C0-\u024F\u0370-\u03FF\u4E00-\u9FFF\uAC00-\uD7AF]*/,

    // Blank line
    blank_line: (_) => /\n/,

    // Fallback: any line that doesn't match above patterns (treated as markdown text)
    text_line: (_) => seq(/[^\n]+/, /\n/),

    _newline: (_) => /\n/,
  },
});
