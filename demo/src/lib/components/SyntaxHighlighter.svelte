<script>
  import CalcMarkBlock from './CalcMarkBlock.svelte';

  // Props using Svelte 5 pattern (still use let for props)
  let { input = '', tokensByLine = {}, lineClassifications = [], evaluationResults = [], diagnostics = {}, variableContext = {} } = $props();

  // Transform flat data into line-based structure using $derived
  const structuredLines = $derived.by(() => {
    if (!input) return [];

    const lines = input.split('\n');
    const result = [];

    // Build evaluation result map by original line number
    const evalResultsByLine = {};
    for (const result of evaluationResults) {
      if (result.originalLine) {
        evalResultsByLine[result.originalLine] = result;
      }
    }

    // Build diagnostic map by line number with proper structure
    const diagnosticsByLine = {};
    for (const lineNum in diagnostics) {
      const lineDiags = diagnostics[lineNum];
      if (lineDiags && lineDiags.Diagnostics) {
        diagnosticsByLine[lineNum] = lineDiags.Diagnostics.map(d => {
          // Fix diagnostic range line numbers to match the actual document line
          const fixedRange = d.range ? {
            start: { ...d.range.start, line: parseInt(lineNum) },
            end: { ...d.range.end, line: parseInt(lineNum) }
          } : d.range;

          return {
            severity: d.severity || d.Severity,
            message: d.message || d.Message,
            range: fixedRange
          };
        });
      }
    }

    // Create line objects
    for (let i = 0; i < lines.length; i++) {
      const lineNumber = i + 1;
      const lineContent = lines[i];
      const classification = lineClassifications[i];
      const lineType = classification ? classification.lineType : 'MARKDOWN';

      result.push({
        lineNumber,
        lineType,
        content: lineContent,
        tokens: tokensByLine[lineNumber] || [], // Pass only this line's tokens (positions are line-relative)
        diagnostics: diagnosticsByLine[lineNumber] || [],
        evaluationResult: evalResultsByLine[lineNumber] || null,
        lineText: lineContent // Pass line content for token rendering
      });
    }

    return result;
  });
</script>

<div class="highlight-output">
  <CalcMarkBlock lines={structuredLines} variableContext={variableContext} />
</div>

<style>
  .highlight-output {
    flex: 1;
    margin: 0;
    padding: 20px;
    background: #ffffff;
    border: 1px solid #e2e8f0;
    border-radius: 4px;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Helvetica', 'Arial', sans-serif;
    font-size: 16px;
    line-height: 1.8;
    overflow-y: auto;
    overflow-x: auto;
  }
</style>
