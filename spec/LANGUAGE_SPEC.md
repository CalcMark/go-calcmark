# CalcMark Language Specification

The canonical language reference lives on the CalcMark website:

**https://calcmark.org/docs/language-reference/**

Source: [`site/content/docs/language-reference.md`](../site/content/docs/language-reference.md)

This file previously contained a standalone spec, but it drifted from reality. The site's language reference is now the single source of truth — it's actively maintained, rendered with live code examples, and validated by the site build pipeline (`task site:build`).

## For Contributors

The **code** in `spec/` (types, units, lexer, parser, features) defines what CalcMark *does*. The site's language reference *describes* that behavior. When adding features:

1. Implement in `spec/` and `impl/`
2. Add golden tests in `testdata/`
3. Update `site/content/docs/language-reference.md`

Do not maintain a separate spec document here.
