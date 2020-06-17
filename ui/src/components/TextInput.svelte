<script>
  import { createEventDispatcher } from 'svelte'
  import Message from './Message.svelte'
  import debounce from '../lib/debounce'

  export let id
  export let value
  export let placeholder = ''
  export let inputRef = undefined
  export let validate = () => true
  export let disabled = false
  export let invalidMessage = undefined
  export let autocomplete = true

  let changed = false
  let invalid = false

  const dispatch = createEventDispatcher()
  const setInvalid = debounce(() => (invalid = true), 1000)
  $: changed = changed || (value !== '' && !changed)
  $: if (changed && !validate(value)) {
    setInvalid.call()
  } else {
    setInvalid.cancel()
    invalid = false
  }
</script>

<div>
  <input {id} class="input" class:is-danger={invalid} type="text" {disabled} {placeholder} {autocomplete}
         bind:value={value}
         bind:this={inputRef}/>
  {#if invalidMessage && invalid}
    <Message text={invalidMessage} error/>
  {/if}
</div>

<style>
  input {
    background-color: white;
    border-color: #dbdbdb;
    border-radius: 4px;
    color: #363636;
  }
</style>
