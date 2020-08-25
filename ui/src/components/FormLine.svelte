<script>
  export let id = ''
  export let label = ''
  export let offset = false
</script>

<style>
  .offset {
    margin-top: 50px;
  }

  .label-wrapper {
    flex-grow: 2;
  }

  .line-label {
    padding: 0;
    text-align: right;
  }

  @media screen and (max-width: 768px) {
    .line-label {
      text-align: left;
    }
  }
</style>

<div class="field is-horizontal" class:offset>
  <div class="field-label is-normal label-wrapper">
    {#if label}
      <label class="label line-label" for={id}>{label}</label>
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
