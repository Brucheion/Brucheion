<script>
  import { fade } from 'svelte/transition'

  export let text = ''
  export let error = false
</script>

<em class:error transition:fade={{duration: 125}}>{text}</em>

<style>
  em {
    display: inline-block;
    padding: 5px;

    font-style: normal;
  }

  .error {
    color: rgb(200, 10, 10);
  }
</style>
