<script>
  let { evaluationResults = [] } = $props();
</script>

<details>
  <summary><strong>Evaluation Results</strong> ({evaluationResults.length} results)</summary>
  {#if evaluationResults.length > 0}
    <table class="results-table">
      <thead>
        <tr>
          <th>Line</th>
          <th>Result</th>
        </tr>
      </thead>
      <tbody>
        {#each evaluationResults as result}
          <tr>
            <td class="result-line">{result.OriginalLine}</td>
            <td class="result-value">
              {#if result.error}
                <span class="value-error">{result.error}</span>
              {:else if result.Value}
                {#if result.Value.SourceFormat}
                  <span class="value-currency">{result.Value.SourceFormat}</span>
                {:else if result.Value.Symbol}
                  <span class="value-currency">{result.Value.Symbol}{result.Value.Value}</span>
                {:else if typeof result.Value.Value === 'boolean'}
                  <span class="value-boolean">{result.Value.Value ? 'true' : 'false'}</span>
                {:else if result.Value.Value !== undefined}
                  <span class="value-number">{result.Value.Value}</span>
                {:else}
                  <span class="value-other">—</span>
                {/if}
              {:else}
                <span class="value-other">—</span>
              {/if}
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  {:else}
    <p class="no-results">No evaluation results yet.</p>
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

  .results-table {
    width: 100%;
    border-collapse: collapse;
    margin-top: 12px;
  }

  .results-table th {
    text-align: left;
    padding: 8px 12px;
    background: #f8fafc;
    border-bottom: 2px solid #e2e8f0;
    font-weight: 600;
    font-size: 14px;
  }

  .results-table td {
    padding: 8px 12px;
    border-bottom: 1px solid #e2e8f0;
  }

  .result-line {
    width: 60px;
    color: #64748b;
    font-family: monospace;
    font-size: 14px;
  }

  .result-value {
    font-family: monospace;
    font-size: 16px;
  }

  .value-number {
    color: #0066cc;
    font-weight: 600;
  }

  .value-currency {
    color: #0d9488;
    font-weight: 700;
    font-size: 18px;
  }

  .value-boolean {
    color: #9333ea;
    font-weight: 600;
  }

  .value-other {
    color: #64748b;
    font-family: monospace;
    font-size: 12px;
  }

  .value-error {
    color: #dc2626;
    font-weight: 500;
    font-style: italic;
  }

  .no-results {
    padding: 16px;
    color: #64748b;
    font-style: italic;
    text-align: center;
  }
</style>
