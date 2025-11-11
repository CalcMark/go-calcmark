<script>
  import CalculationLine from './CalculationLine.svelte';
  import { marked } from 'marked';

  let { lines = [], variableContext = {} } = $props();

  // Cache for processed markdown blocks: content hash -> rendered HTML lines
  const markdownCache = new Map();

  // Simple hash function for content
  function hashContent(text) {
    let hash = 0;
    for (let i = 0; i < text.length; i++) {
      const char = text.charCodeAt(i);
      hash = ((hash << 5) - hash) + char;
      hash = hash & hash; // Convert to 32-bit integer
    }
    return hash.toString(36);
  }

  // Group consecutive lines of same type into blocks and process markdown
  function groupIntoBlocks(lines) {
    if (lines.length === 0) return [];

    const blocks = [];
    let currentBlock = {
      type: lines[0].lineType,
      lines: [lines[0]],
      startIndex: 0
    };

    // Helper to finalize a block
    const finalizeBlock = (block) => {
      if (block.type === 'MARKDOWN') {
        // Extract content from lines and process as single markdown block
        const content = block.lines.map(l => l.content).join('\n');
        const contentHash = hashContent(content);

        // Check cache
        let renderedHtml;
        if (markdownCache.has(contentHash)) {
          renderedHtml = markdownCache.get(contentHash);
        } else {
          // Process entire markdown block as a single unit
          renderedHtml = marked.parse(content, { breaks: true, gfm: true }).trim();
          markdownCache.set(contentHash, renderedHtml);
        }

        // Store the rendered HTML on the block itself, not on individual lines
        block.renderedHtml = renderedHtml;
      }
      // BLANK lines don't need processing - they render as-is
      return block;
    };

    // Build blocks
    for (let i = 1; i < lines.length; i++) {
      const line = lines[i];
      if (line.lineType === currentBlock.type) {
        currentBlock.lines.push(line);
      } else {
        blocks.push(finalizeBlock(currentBlock));
        currentBlock = {
          type: line.lineType,
          lines: [line],
          startIndex: i
        };
      }
    }

    // Don't forget the last block
    blocks.push(finalizeBlock(currentBlock));

    return blocks;
  }

  const blocks = $derived(groupIntoBlocks(lines));
</script>

{#each blocks as block, blockIndex}
  {#if block.type === 'CALCULATION'}
    <div class="block calculation-block">
      {#each block.lines as line}
        <CalculationLine
          lineNumber={line.lineNumber}
          tokens={line.tokens}
          diagnostics={line.diagnostics}
          evaluationResult={line.evaluationResult}
          lineText={line.lineText}
          variableContext={variableContext}
        />
      {/each}
    </div>
  {:else if block.type === 'MARKDOWN'}
    <div class="block markdown-block">
      {@html block.renderedHtml}
    </div>
  {/if}
  <!-- BLANK blocks render nothing -->
{/each}

<style>
  .block {
    display: contents;
  }

  .calculation-block {
    display: contents;
  }

  .markdown-block {
    display: block;
    color: #1e293b;
  }

  /* Reset margins on block elements inside markdown blocks */
  .markdown-block :global(p),
  .markdown-block :global(h1),
  .markdown-block :global(h2),
  .markdown-block :global(h3),
  .markdown-block :global(h4),
  .markdown-block :global(h5),
  .markdown-block :global(h6),
  .markdown-block :global(ul),
  .markdown-block :global(ol),
  .markdown-block :global(blockquote) {
    margin-top: 0;
    margin-bottom: 0.5em;
  }

  /* Last element in block has no bottom margin */
  .markdown-block :global(*:last-child) {
    margin-bottom: 0;
  }

  /* Add spacing between markdown and calculation blocks */
  .markdown-block + .calculation-block,
  .calculation-block + .markdown-block {
    margin-top: 1em;
  }
</style>
