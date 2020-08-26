<script>
  import { createEventDispatcher } from 'svelte'
  import { fade, fly } from 'svelte/transition'
  import { quintOut } from 'svelte/easing'

  export let items
  export let selectedIndex

  const dispatch = createEventDispatcher()

  function handleClose(index) {
    dispatch('close', index)
  }

  function handleMouseMove(index) {
    dispatch('select', index)
  }
</script>

<style>
  .search-container {
    position: relative;
  }

  .search {
    position: absolute;
    left: 0;
    right: 0;
    top: 0;
    z-index: 50;

    padding: 3px;
  }

  .item {
    box-sizing: border-box;
    border-radius: 4px;
    padding: 3px 5px;

    cursor: pointer;
  }

  .item.selected {
    background: rgba(230, 230, 255);
  }
</style>

<div class="search-container">
  <div
    class="box search"
    in:fly={{ y: -5, duration: 150, delay: 0, opacity: 0.2, start: 0.0, easing: quintOut }}
    out:fade={{ duration: 125, easing: quintOut }}>
    {#each items as item, i}
      <div
        class="item"
        class:selected={i === selectedIndex}
        on:click={() => handleClose(i)}
        on:mousemove={() => handleMouseMove(i)}>
        {@html item.label}
      </div>
    {/each}
  </div>
</div>
