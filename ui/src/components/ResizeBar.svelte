<script>
  import { createEventDispatcher } from 'svelte'

  const dispatch = createEventDispatcher()
  let barElement = undefined,
    innerOffsetY = null
  $: resizing = innerOffsetY !== null

  function handleMouseDown(e) {
    innerOffsetY = e.y - barElement.offsetTop
    document.addEventListener('mousemove', handleMouseMove, false)
  }

  function handleMouseMove(e) {
    dispatch('resize', {
      y: e.y - innerOffsetY,
    })
  }

  function handleMouseUp() {
    innerOffsetY = null
    document.removeEventListener('mousemove', handleMouseMove, false)
  }
</script>

<style>
  .bar {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;

    box-sizing: border-box;
    width: 100%;
    height: 12px;
    padding: 4px;

    background: var(--toolbar-bg-color);
    border-top: 1px solid var(--toolbar-border-color);
    cursor: ns-resize;
  }

  .handle {
    flex-shrink: 0;
    flex-grow: 0;
    width: 32px;
    height: 4px;
    border-radius: 2px;
    background: rgb(100, 100, 100);
  }

  .resizing {
    cursor: ns-resize;
  }
</style>

<svelte:window on:mouseup={handleMouseUp} />
<svelte:body class:resizing />

<div class="bar" bind:this={barElement} on:mousedown={handleMouseDown}>
  <div class="handle" />
</div>
