<script>
  import { createEventDispatcher } from 'svelte'
  import Message from './Message.svelte'
  import TextAheadOverlay from './TextAheadOverlay.svelte'
  import debounce from '../lib/debounce'

  export let id
  export let value
  export let placeholder = ''
  export let inputRef = undefined
  export let validate = () => true
  export let disabled = false
  export let invalidMessage = undefined
  export let autocomplete = 'on'

  export let items = undefined
  export let minLength = 3
  export let maxItems = 10

  const dispatch = createEventDispatcher()

  let changed = false
  let invalid = false
  let overlayOpen = false
  let selectedIndex = 0
  let timer = null
  let results = []

  const setInvalid = debounce(() => (invalid = true), 1000)
  $: changed = changed || (value !== '' && !changed)
  $: if (!items) {
    if (changed && !validate(value)) {
      setInvalid.call()
    } else {
      setInvalid.cancel()
      invalid = false
    }
  }

  const regExpEscape = (s) => {
    return s.replace(/[-\\^$*+?.()|[\]{}]/g, '\\$&')
  }

  function handleBlur() {
    timer = setTimeout(() => {
      overlayOpen = false
      invalid = changed && !validate(value)
    }, 250)
  }

  function handleFocus() {
    clearTimeout(timer)
  }

  const closeOverlay = (index = -1) => {
    overlayOpen = false
    selectedIndex = -1

    if (index > -1) {
      value = results[index].value
    }
  }

  const handleChange = (event) => {
    const term = event.target.value

    if (term.length >= Number(minLength)) {
      overlayOpen = true
      filterResults(term)
    } else {
      results = []
    }
  }

  const filterResults = (term) => {
    results = items
      .filter((item) => {
        if (typeof item !== 'string') item = item.value || ''
        return item.toUpperCase().includes(term.toUpperCase())
      })
      .map((item, i) => {
        const text = typeof item !== 'string' ? item.value : item

        return {
          key: item.key || i,
          value: text,
          label:
            term.trim() === ''
              ? text
              : text.replace(
                  RegExp(regExpEscape(term.trim()), 'i'),
                  "<span style='font-weight: 600;'>$&</span>"
                ),
        }
      })
      .slice(0, maxItems - 1)
  }

  function handleKeyDown(event) {
    if (event.keyCode === 40 && selectedIndex < results.length - 1) {
      selectedIndex += 1
    } else if (event.keyCode === 38 && selectedIndex > 0) {
      selectedIndex -= 1
    } else if (event.keyCode === 13) {
      event.preventDefault()

      if (selectedIndex === -1) selectedIndex = 0
      closeOverlay(selectedIndex)
    } else if (event.keyCode === 27) {
      event.preventDefault()
      closeOverlay()
    }
  }
</script>

<style>
  input {
    background-color: white;
    border-color: #dbdbdb;
    border-radius: 4px;
    color: #363636;
  }
</style>

<div>
  <input
    {id}
    class="input"
    class:is-danger={invalid}
    type="text"
    {disabled}
    {placeholder}
    autocomplete={items ? 'off' : autocomplete}
    bind:value
    bind:this={inputRef}
    on:focus={() => items && handleFocus()}
    on:blur={() => items && handleBlur()}
    on:input={(event) => items && handleChange(event)}
    on:keydown={(event) => items && handleKeyDown(event)} />

  {#if items && overlayOpen && results.length > 0}
    <TextAheadOverlay
      items={results}
      {selectedIndex}
      on:close={(event) => closeOverlay(event.detail)}
      on:select={(event) => (selectedIndex = event.detail)} />
  {/if}

  {#if invalidMessage && invalid}
    <Message text={invalidMessage} error />
  {/if}
</div>
