<script>
  // CalcMark Editor Component - Svelte 5 with server-side WASM
  import ValueTooltip from './ValueTooltip.svelte';
  import SyntaxHighlighter from './SyntaxHighlighter.svelte';
  import EvaluationResults from './EvaluationResults.svelte';
  import LineClassifications from './LineClassifications.svelte';
  import TokenizationDetails from './TokenizationDetails.svelte';

  const SAMPLE_DOCUMENT = `# CalcMark Demo - Mixing Markdown and Calculations

This is a demonstration of **CalcMark**, a language that blends *calculations* with _markdown_. Learn more at [CalcMark on GitHub](https://github.com).

> CalcMark uses "calculation by exclusion" - lines are greedily interpreted as calculations whenever possible.

## Key Features

- Seamless markdown integration
- Unicode support for variables
- Multiple currency formats

## Percentage Literals ✅

Percentages work! 20% means 0.20:
discount = 20%
tax_rate = 8.5%
growth = 150%
tip = $100 * 15%

Note: The ⭐ and ✅ emoji in this markdown are fine!

## Modulus vs Percentage

Ambiguous (will show hint):
remainder = 10 %3

Clear modulus:
mod_result = 10 % 3

## Unicode Support 🌏

Emoji variables work great:
🏠 = $1_500
🍕 = $800
💡 = $200
total_expenses = 🏠 + 🍕 + 💡

Chinese characters:
工资 = $5_000
奖金 = $500
总收入 = 工资 + 奖金

Arabic characters:
السعر = $100
الكمية = 5
المجموع = السعر * الكمية

Emoji with skin tone modifier:
👍🏻 = 10

## Currency & Quantities

Multiple currency symbols:
price1 = $100
price2 = €50
price3 = £75
price4 = ¥1000

ISO currency codes:
usd_amount = USD100
gbp_amount = GBP50
eur_amount = EUR75

Thousands separators:
big_number = $1,234,567.89
underscore = $1_000_000

## Error Example

This will show an error (undefined variable):
undefined_calc = missing_var + 100

## Comparisons

Boolean comparisons work:
is_affordable = total_expenses < 总收入
has_surplus = 总收入 > $4_000
exact_match = 工资 == $5_000`;

  // State using $state rune
  let input = $state(SAMPLE_DOCUMENT);
  let tokensByLine = $state({});
  let error = $state(null);
  let loading = $state(false);
  let lineClassifications = $state([]);
  let evaluationResults = $state([]);
  let diagnostics = $state({});
  let variableContext = $state({});

  // Tooltip state
  let tooltipVisible = $state(false);
  let tooltipText = $state('');
  let tooltipX = $state(0);
  let tooltipY = $state(0);

  // Debouncing
  let debounceTimer = $state(null);
  const DEBOUNCE_MS = 150;

  // Process input through server API
  async function processInput(text) {
    loading = true;
    error = null;

    try {
      const response = await fetch('/api/process', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ input: text })
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || 'Server error');
      }

      const result = await response.json();

      // Update state with results
      lineClassifications = result.classifications || [];
      tokensByLine = result.tokensByLine || {};
      variableContext = result.variableContext || {};
      console.log('Received variableContext from server:', JSON.stringify(variableContext));
      evaluationResults = result.evaluationResults || [];
      diagnostics = result.diagnostics || {};
    } catch (err) {
      error = err.message;
      console.error('Processing error:', err);
    } finally {
      loading = false;
    }
  }

  function handleInput() {
    // Clear existing timer
    if (debounceTimer) {
      clearTimeout(debounceTimer);
    }

    // Set new timer
    debounceTimer = setTimeout(() => {
      processInput(input);
      debounceTimer = null;
    }, DEBOUNCE_MS);
  }

  // Initial processing using $effect
  $effect(() => {
    // Process on mount
    processInput(input);
  });

  function handleMouseOver(event) {
    let target = event.target;

    while (target && target !== event.currentTarget) {
      if (target.classList && target.classList.contains('line')) {
        let text = '';

        if (target.dataset.diagnostic) {
          text = target.dataset.diagnostic;
        } else if (target.dataset.result) {
          text = `Result: ${target.dataset.result}`;
        }

        if (text) {
          tooltipText = text;
          tooltipVisible = true;

          const rect = target.getBoundingClientRect();
          tooltipX = rect.left + (rect.width / 2);
          tooltipY = rect.top - 10;
          return;
        }
      }
      target = target.parentElement;
    }
  }

  function handleMouseOut(event) {
    let target = event.target;

    while (target && target !== event.currentTarget) {
      if (target.classList && target.classList.contains('calculation-line')) {
        tooltipVisible = false;
        return;
      }
      target = target.parentElement;
    }
  }
</script>

<div class="container">
  <h1>CalcMark WASM Demo (Server-Side)</h1>

  {#if error}
    <div class="error">Error: {error}</div>
  {/if}

  <div class="editor-grid">
    <div class="demo-section">
      <h2>Raw Source (editable)</h2>
      <textarea
        class="source-input"
        bind:value={input}
        oninput={handleInput}
        spellcheck="false"
      ></textarea>
    </div>

    <div class="demo-section">
      <h2>Syntax Highlighted Output</h2>
      <div
        class="highlighter-wrapper"
        onmouseover={handleMouseOver}
        onmouseout={handleMouseOut}
      >
        <SyntaxHighlighter
          input={input}
          tokensByLine={tokensByLine}
          lineClassifications={lineClassifications}
          evaluationResults={evaluationResults}
          diagnostics={diagnostics}
          variableContext={variableContext}
        />
      </div>
    </div>
  </div>

  <ValueTooltip
    visible={tooltipVisible}
    text={tooltipText}
    x={tooltipX}
    y={tooltipY}
  />

  <div class="details-container">
    <div class="demo-section compact">
      <EvaluationResults evaluationResults={evaluationResults} />
    </div>

    <div class="demo-section compact">
      <LineClassifications lineClassifications={lineClassifications} />
    </div>

    <div class="demo-section compact">
      <TokenizationDetails tokensByLine={tokensByLine} />
    </div>
  </div>
</div>

<style>
  .container {
    max-width: 1800px;
    margin: 0 auto;
    padding: 20px;
    font-family: system-ui, -apple-system, sans-serif;
    height: 100vh;
    display: flex;
    flex-direction: column;
  }

  .editor-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 20px;
    margin-bottom: 20px;
    height: 500px;
    min-height: 400px;
  }

  .editor-grid .demo-section {
    height: 100%;
    display: flex;
    flex-direction: column;
  }

  @media (max-width: 1024px) {
    .editor-grid {
      grid-template-columns: 1fr;
      height: auto;
    }
  }

  h1 {
    margin-bottom: 30px;
    font-size: 28px;
    font-weight: 600;
    color: #1e293b;
  }

  h2 {
    margin: 0 0 12px 0;
    font-size: 16px;
    font-weight: 600;
    color: #475569;
  }

  .demo-section {
    margin-bottom: 20px;
    padding: 16px;
    background: #fff;
    border: 1px solid #e2e8f0;
    border-radius: 8px;
    box-shadow: 0 1px 3px rgba(0,0,0,0.1);
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .demo-section.compact {
    margin-bottom: 10px;
  }

  .editor-grid .demo-section {
    padding: 16px;
  }

  .details-container {
    width: 100%;
    display: flex;
    flex-direction: column;
    gap: 12px;
    margin-top: 20px;
  }

  .details-container .demo-section {
    width: 100%;
    margin-bottom: 0;
  }

  .highlighter-wrapper {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-height: 0;
    overflow: hidden;
  }

  .source-input {
    flex: 1;
    width: 100%;
    min-width: 0;
    margin: 0;
    padding: 16px;
    background: #f8fafc;
    border: 2px solid #cbd5e1;
    border-radius: 4px;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Helvetica', 'Arial', sans-serif;
    font-size: 15px;
    line-height: 1.5;
    color: #1e293b;
    resize: none;
    transition: border-color 0.2s;
    box-sizing: border-box;
  }

  .source-input:focus {
    outline: none;
    border-color: #0ea5e9;
    background: #ffffff;
  }

  .error {
    margin-bottom: 20px;
    padding: 16px;
    background: #fee;
    border: 1px solid #fcc;
    border-radius: 4px;
    color: #c00;
    font-size: 14px;
  }

  :global(.highlight-output .line) {
    min-height: 1.8em;
  }

  :global(.highlight-output .blank-line) {
    opacity: 0.3;
  }

  :global(.highlight-output .markdown-line) {
    color: #1e293b;
  }

  :global(.highlight-output .calculation-line) {
    background: #f0f9ff;
    padding: 4px 8px;
    margin: 8px 0;
    border-radius: 4px;
    border-left: 3px solid #0ea5e9;
    cursor: pointer;
    transition: background 0.15s ease;
  }

  :global(.highlight-output .calculation-line:hover) {
    background: #e0f2fe;
  }

  :global(.highlight-output .has-error) {
    text-decoration: underline wavy #dc2626;
    text-decoration-thickness: 2px;
    text-underline-offset: 2px;
  }

  :global(.highlight-output .has-warning) {
    text-decoration: underline wavy #f59e0b;
    text-decoration-thickness: 2px;
    text-underline-offset: 2px;
  }

  :global(.highlight-output .calculation-line + .calculation-line) {
    margin-top: 0;
    border-top-left-radius: 0;
    border-top-right-radius: 0;
  }

  :global(.highlight-output .calculation-line:has(+ .calculation-line)) {
    margin-bottom: 0;
    border-bottom-left-radius: 0;
    border-bottom-right-radius: 0;
  }
</style>
