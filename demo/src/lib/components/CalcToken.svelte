<script>
  import Tooltip from './Tooltip.svelte';

  let { token, diagnostics = [], variableContext = {} } = $props();

  // Token type to CSS class mapping
  const tokenTypeToClass = {
    'NUMBER': 'token-number',
    'CURRENCY': 'token-currency',
    'QUANTITY': 'token-currency', // Use same styling as CURRENCY
    'BOOLEAN': 'token-boolean',
    'IDENTIFIER': 'token-identifier',
    'PLUS': 'token-operator',
    'MINUS': 'token-operator',
    'MULTIPLY': 'token-operator',
    'DIVIDE': 'token-operator',
    'MODULUS': 'token-operator',
    'EXPONENT': 'token-operator',
    'ASSIGN': 'token-operator',
    'GREATER_THAN': 'token-operator',
    'LESS_THAN': 'token-operator',
    'GREATER_EQUAL': 'token-operator',
    'LESS_EQUAL': 'token-operator',
    'EQUAL': 'token-operator',
    'NOT_EQUAL': 'token-operator',
    'AND': 'token-keyword',
    'OR': 'token-keyword',
    'NOT': 'token-keyword',
    'LPAREN': 'token-punctuation',
    'RPAREN': 'token-punctuation',
    'COMMA': 'token-punctuation',
    'FUNC_AVG': 'token-function',
    'FUNC_SQRT': 'token-function',
    'FUNC_AVERAGE_OF': 'token-function',
    'FUNC_SQUARE_ROOT_OF': 'token-function',
  };

  const cssClass = $derived(tokenTypeToClass[token.type] || 'token-default');
  const hasError = $derived(diagnostics.some(d => d.severity === 'error'));
  const hasWarning = $derived(!hasError && diagnostics.some(d => d.severity === 'warning'));

  const diagnosticMessage = $derived(diagnostics.length > 0
    ? diagnostics.map(d => {
        const severity = d.severity.charAt(0).toUpperCase() + d.severity.slice(1);
        return `${severity}: ${d.message}`;
      }).join('\n')
    : null);

  // For IDENTIFIER tokens, look up the value in the variable context
  const identifierValue = $derived.by(() => {
    if (token.type === 'IDENTIFIER' && token.value && variableContext[token.value]) {
      const result = variableContext[token.value];

      // The result structure from WASM is: {Value: {Value, Symbol, SourceFormat}, Symbol, SourceFormat, OriginalLine}
      // The inner Value object has the actual data
      const innerValue = result.Value;

      // Format the result for display
      if (innerValue.SourceFormat) {
        // Use the SourceFormat if available (e.g., "20%", "$800", "$1,234,567.89")
        return innerValue.SourceFormat;
      } else if (innerValue.Symbol) {
        // Currency without SourceFormat: Symbol + Value
        return `${innerValue.Symbol}${innerValue.Value}`;
      } else if (typeof innerValue.Value === 'boolean') {
        // Boolean
        return innerValue.Value ? 'true' : 'false';
      } else if (innerValue.Value !== undefined) {
        // Number: just show the value
        return String(innerValue.Value);
      }
    }
    return null;
  });

  // Build tooltip: prioritize identifier value, then diagnostics
  const tooltipMessage = $derived(identifierValue || diagnosticMessage);

  // Debug logging
  $effect(() => {
    if (token.type === 'IDENTIFIER' && token.value) {
      console.log(`Token: "${token.value}" | Has in context: ${token.value in variableContext} | Context keys: [${Object.keys(variableContext).join(', ')}] | IdentifierValue: "${identifierValue}" | TooltipMessage: "${tooltipMessage}"`);
    }
  });

  // Always use originalText to preserve user intent
  // The user wrote "$1,500" so we display "$1,500", not "1500$"
  const displayValue = $derived(token.originalText || token.value);
</script>

<Tooltip message={tooltipMessage}>
  <span
    class={cssClass}
    class:has-error={hasError}
    class:has-warning={hasWarning}
  >
    {displayValue}
  </span>
</Tooltip>

<style>
  /* Token styling */
  .token-number {
    background: #dbeafe;
    color: #1e40af;
    padding: 2px 6px;
    border-radius: 4px;
    font-weight: 500;
  }

  .token-currency {
    background: #ccfbf1;
    color: #0f766e;
    padding: 2px 6px;
    border-radius: 4px;
    font-weight: 500;
  }

  .token-boolean {
    background: #f3e8ff;
    color: #7e22ce;
    padding: 2px 6px;
    border-radius: 4px;
    font-weight: 500;
  }

  .token-identifier {
    background: #f1f5f9;
    color: #334155;
    padding: 2px 6px;
    border-radius: 4px;
    font-weight: 500;
  }

  .token-operator {
    color: #374151;
    font-weight: 500;
  }

  .token-keyword {
    background: #fed7aa;
    color: #c2410c;
    padding: 2px 6px;
    border-radius: 4px;
    font-weight: 500;
  }

  .token-function {
    background: #ede9fe;
    color: #6b21a8;
    padding: 2px 6px;
    border-radius: 4px;
    font-weight: 500;
  }

  .token-punctuation {
    background: #f1f5f9;
    color: #475569;
    padding: 2px 6px;
    border-radius: 4px;
    font-weight: 500;
  }

  /* Diagnostic indicators */
  .has-error {
    border-bottom: 2px dashed #dc2626;
    background: rgba(220, 38, 38, 0.1) !important;
  }

  .has-warning {
    border-bottom: 2px dashed #f59e0b;
    background: rgba(245, 158, 11, 0.1) !important;
  }
</style>
