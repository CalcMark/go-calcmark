<script>
  let { tokensByLine = {} } = $props();

  // Flatten tokens for display
  const allTokens = $derived(Object.entries(tokensByLine).flatMap(([lineNum, tokens]) =>
    tokens.map(token => ({ ...token, lineNum: parseInt(lineNum) }))
  ));
</script>

<details>
  <summary><strong>Tokenization Details</strong> ({allTokens.length} tokens)</summary>
  {#if allTokens.length > 0}
    <table class="tokens-table">
      <thead>
        <tr>
          <th>Type</th>
          <th>Value</th>
          <th>Line</th>
          <th>Position</th>
        </tr>
      </thead>
      <tbody>
        {#each allTokens as token}
          <tr>
            <td class="token-type">
              <span class="type-badge">{token.type}</span>
            </td>
            <td class="token-value">{token.value || '(empty)'}</td>
            <td class="token-line">{token.lineNum}</td>
            <td class="token-position">{token.start}-{token.end}</td>
          </tr>
        {/each}
      </tbody>
    </table>
  {:else}
    <p class="no-data">No tokens yet.</p>
  {/if}
</details>

<style>
  details {
    margin: 0;
  }

  summary {
    cursor: pointer;
    padding: 12px 16px;
    background: #f8fafc;
    border: 1px solid #e2e8f0;
    border-radius: 4px;
    user-select: none;
  }

  summary:hover {
    background: #f1f5f9;
  }

  .tokens-table {
    width: 100%;
    border-collapse: collapse;
    margin-top: 12px;
    font-size: 13px;
  }

  .tokens-table th {
    text-align: left;
    padding: 8px 12px;
    background: #f8fafc;
    border-bottom: 2px solid #e2e8f0;
    font-weight: 600;
    font-size: 12px;
  }

  .tokens-table td {
    padding: 6px 12px;
    border-bottom: 1px solid #f1f5f9;
  }

  .type-badge {
    display: inline-block;
    padding: 2px 6px;
    border-radius: 3px;
    font-size: 11px;
    font-weight: 600;
    font-family: monospace;
    background: #e0e7ff;
    color: #4338ca;
  }

  .token-value {
    font-family: monospace;
    color: #1e293b;
  }

  .token-line,
  .token-position {
    font-family: monospace;
    color: #64748b;
    font-size: 12px;
  }

  .no-data {
    padding: 16px;
    color: #64748b;
    font-style: italic;
    text-align: center;
  }
</style>
