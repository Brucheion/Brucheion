<script>
  export let id = ''
  export let label = ''
  export let offset = false
</script>

<style>
  .offset {
    margin-top: 25px;
  }

  .label-wrapper {
    flex-grow: 1;
  }

  @media screen and (min-width: 769px) {
    .label-wrapper {
      display: flex;
      flex-flow: row;
      align-items: flex-start;
      justify-content: flex-end;

      padding: 0;
    }
  }

  .line-label {
    width: 125px;
    padding: 6px 0;
    text-align: right;
    line-height: 1.5;
  }

  @media screen and (max-width: 768px) {
    .line-label {
      width: 100%;
      text-align: left;
      padding: 2px 0;
    }
  }
</style>

<div class="field is-horizontal" class:offset>
  <div class="field-label is-normal label-wrapper">
    {#if label}
      <label class="label line-label" for={id}>{label}</label>
    {:else}
      <div class="line-label" />
    {/if}
  </div>
  <div class="field-body">
    <div class="field">
      <div class="control">
        <slot />
      </div>
    </div>
  </div>
</div>
