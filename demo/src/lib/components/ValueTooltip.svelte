<script>
  let { visible = false, text = '', x = 0, y = 0 } = $props();
</script>

{#if visible}
  <div
    class="tooltip"
    style="left: {x}px; top: {y}px;"
  >
    {text}
  </div>
{/if}

<style>
  .tooltip {
    position: fixed;
    background: #1e293b;
    color: white;
    padding: 6px 12px;
    border-radius: 6px;
    font-size: 13px;
    font-weight: 500;
    pointer-events: none;
    z-index: 1000;
    white-space: pre-wrap;
    max-width: 400px;
    word-wrap: break-word;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
    transform: translate(-50%, -100%);
  }

  .tooltip::before {
    content: '';
    position: absolute;
    top: 100%;
    left: 50%;
    transform: translateX(-50%);
    border: 6px solid transparent;
    border-top-color: #1e293b;
  }
</style>
