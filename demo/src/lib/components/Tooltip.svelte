<script>
  import { onMount } from 'svelte';

  let { children, message } = $props();

  let showTooltip = $state(false);
  let tooltipX = $state(0);
  let tooltipY = $state(0);
  let containerElement = $state(null);

  function handleMouseEnter(event) {
    if (!message) return;
    showTooltip = true;
    updatePosition(event);
  }

  function handleMouseMove(event) {
    if (!showTooltip) return;
    updatePosition(event);
  }

  function handleMouseLeave() {
    showTooltip = false;
  }

  function updatePosition(event) {
    // Position tooltip relative to cursor
    tooltipX = event.clientX;
    tooltipY = event.clientY;
  }
</script>

<span
  bind:this={containerElement}
  onmouseenter={handleMouseEnter}
  onmousemove={handleMouseMove}
  onmouseleave={handleMouseLeave}
  class="tooltip-container"
>
  {@render children()}
</span>

{#if showTooltip && message}
  <div
    class="tooltip"
    style:left="{tooltipX + 10}px"
    style:top="{tooltipY - 10}px"
  >
    {message}
  </div>
{/if}

<style>
  .tooltip-container {
    display: inline;
    position: relative;
  }

  .tooltip {
    position: fixed;
    z-index: 9999;
    background: #1e293b;
    color: #f1f5f9;
    padding: 6px 10px;
    border-radius: 6px;
    font-size: 13px;
    line-height: 1.4;
    max-width: 300px;
    word-wrap: break-word;
    white-space: pre-wrap;
    pointer-events: none;
    box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
  }
</style>
