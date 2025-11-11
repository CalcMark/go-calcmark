<script>
  import CalcToken from './CalcToken.svelte';
  import { runeToUtf16Position } from '$lib/utils/unicode';

  let {
    lineNumber,
    tokens = [],
    diagnostics = [],
    evaluationResult = null,
    lineText = '',
    variableContext = {}
  } = $props();

  // Format evaluation result for tooltip
  function formatResultValue(result) {
    if (!result) return null;

    if (result.Symbol) {
      return result.SourceFormat || `${result.Symbol}${result.Value}`;
    } else if (typeof result.Value === 'boolean') {
      return result.Value ? 'true' : 'false';
    } else if (result.Value !== undefined) {
      return result.SourceFormat || result.Value;
    }
    return null;
  }

  // Build diagnostic messages for the line (only line-level diagnostics without range)
  const lineDiagnosticMessage = $derived(() => {
    const lineLevelDiags = diagnostics.filter(d => !d.range);
    if (lineLevelDiags.length === 0) return null;
    return lineLevelDiags.map(d => {
      const severity = d.severity.charAt(0).toUpperCase() + d.severity.slice(1);
      return `${severity}: ${d.message}`;
    }).join('\n');
  });

  const resultText = $derived(formatResultValue(evaluationResult));
  const tooltipText = $derived(resultText || lineDiagnosticMessage());

  // Group tokens and whitespace for rendering
  // Tokens are line-relative (positions start at 0 for this line)
  // Token positions are in RUNES (Go's unit), need conversion to UTF-16 for JavaScript
  function buildLineSegments() {
    if (tokens.length === 0 || !lineText) {
      return [{ type: 'text', content: lineText }];
    }

    const segments = [];
    let currentUtf16Pos = 0;

    // Filter and sort tokens (positions are already line-relative, in runes)
    const lineTokens = tokens
      .filter(t => t.type !== 'NEWLINE' && t.type !== 'EOF')
      .sort((a, b) => a.start - b.start);

    for (const token of lineTokens) {
      // Convert token rune positions to UTF-16 positions
      const tokenUtf16Start = runeToUtf16Position(lineText, token.start);
      const tokenUtf16End = runeToUtf16Position(lineText, token.end);

      // Add whitespace before token
      if (tokenUtf16Start > currentUtf16Pos) {
        const whitespace = lineText.substring(currentUtf16Pos, tokenUtf16Start);
        segments.push({ type: 'text', content: whitespace });
      }

      // Get diagnostics for this token (diagnostic columns are 1-indexed, in runes)
      const tokenDiagnostics = diagnostics.filter(d => {
        if (!d.range) return false;
        const tokenColumn = token.start + 1; // Convert to 1-indexed
        const tokenEndColumn = token.end + 1;
        return tokenColumn <= d.range.end.column && tokenEndColumn >= d.range.start.column;
      });

      segments.push({
        type: 'token',
        token: token,
        diagnostics: tokenDiagnostics
      });

      currentUtf16Pos = tokenUtf16End;
    }

    // Add any trailing text
    if (currentUtf16Pos < lineText.length) {
      const trailing = lineText.substring(currentUtf16Pos);
      segments.push({ type: 'text', content: trailing });
    }

    return segments;
  }

  // React to changes in tokens, diagnostics, or lineText
  const segments = $derived.by(() => {
    // Access dependencies to track them
    tokens; diagnostics; lineText;
    return buildLineSegments();
  });
</script>

<div class="calculation-line">
  {#if segments.length > 0}
    {#each segments as segment}
      {#if segment.type === 'token'}
        <CalcToken token={segment.token} diagnostics={segment.diagnostics} variableContext={variableContext} />
      {:else}
        {segment.content}
      {/if}
    {/each}
  {:else}
    {sourceText}
  {/if}
</div>

<style>
  .calculation-line {
    background: #f0f9ff;
    padding: 4px 8px;
    margin: 8px 0;
    border-radius: 4px;
    border-left: 3px solid #0ea5e9;
    cursor: pointer;
    transition: background 0.15s ease;
    min-height: 1.8em;
  }

  .calculation-line:hover {
    background: #e0f2fe;
  }
</style>
