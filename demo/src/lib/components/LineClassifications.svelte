<script>
  let { lineClassifications = [] } = $props();
</script>

<details>
  <summary><strong>Line Classifications</strong> ({lineClassifications.length} lines)</summary>
  {#if lineClassifications.length > 0}
    <table class="classifications-table">
      <thead>
        <tr>
          <th>Line</th>
          <th>Type</th>
        </tr>
      </thead>
      <tbody>
        {#each lineClassifications as classification, idx}
          <tr>
            <td class="line-number">{idx + 1}</td>
            <td class="line-type">
              <span class="type-badge type-{classification.lineType.toLowerCase()}">
                {classification.lineType}
              </span>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  {:else}
    <p class="no-data">No classifications yet.</p>
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

  .classifications-table {
    width: 100%;
    border-collapse: collapse;
    margin-top: 12px;
  }

  .classifications-table th {
    text-align: left;
    padding: 8px 12px;
    background: #f8fafc;
    border-bottom: 2px solid #e2e8f0;
    font-weight: 600;
    font-size: 14px;
  }

  .classifications-table td {
    padding: 8px 12px;
    border-bottom: 1px solid #e2e8f0;
  }

  .line-number {
    width: 60px;
    color: #64748b;
    font-family: monospace;
    font-size: 14px;
  }

  .type-badge {
    display: inline-block;
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 12px;
    font-weight: 600;
    font-family: monospace;
  }

  .type-calculation {
    background: #dbeafe;
    color: #1e40af;
  }

  .type-markdown {
    background: #f3f4f6;
    color: #374151;
  }

  .type-blank {
    background: #fef3c7;
    color: #92400e;
  }

  .no-data {
    padding: 16px;
    color: #64748b;
    font-style: italic;
    text-align: center;
  }
</style>
