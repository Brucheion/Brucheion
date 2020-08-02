<script>
  import { stringify as stringifyQuery } from 'qs'
  import { onMount } from 'svelte'
  import { navigate } from 'svelte-routing'
  import FormLine from '../components/FormLine.svelte'
  import Message from '../components/Message.svelte'
  import { validateUrn } from '../lib/cts-urn'
  import TextInput from '../components/TextInput.svelte'
  import { validateHttpUrl } from '../lib/url'

  let collection = ''
  let imageName = ''
  let imageUrl = ''
  let external = true
  let protocol = 'static'

  let statusMessage = null,
    timeoutHandle
  let collectionRef, imageNameRef

  $: complete =
    validateUrn(collection, { noPassage: true }) &&
    validateUrn(imageName) &&
    imageUrl

  $: if (statusMessage !== null) {
    clearTimeout(timeoutHandle)
    timeoutHandle = setTimeout(() => (statusMessage = null), 10000)
  }
  $: errorMessage =
    statusMessage && statusMessage.toLowerCase().includes('error')
  $: external = !validateUrn(imageUrl)

  onMount(() => {
    const query = new URLSearchParams(location.search)
    if (query.has('collection')) {
      if (validateUrn(query.get('collection'), { noPassage: true })) {
        collection = query.get('collection')
        imageNameRef.focus()
      } else {
        query.delete('collection')
        navigate(`/ingest?${query.toString()}`, { replace: true })
      }
    } else {
      collectionRef.focus()
    }
  })

  async function handleSubmit(event) {
    event.preventDefault()
    if (!complete) {
      return
    }

    const query = {
      name: collection,
      urn: imageName,
      location: imageUrl,
      external,
      protocol,
    }
    const res = await fetch(`/addtoCITE?${stringifyQuery(query)}`)
    if (res.status !== 200) {
      console.error(`Ingestion failed: HTTP ${res.status} ${await res.text()}`)
      statusMessage = 'An error occurred. Please try later.'
      return
    }
    statusMessage = 'Image ingested.'
    window.setTimeout(() => {
      imageName = ''
      imageNameRef.focus()
    }, 2500)
  }
</script>

<style>
  .form {
    box-sizing: border-box;
    max-width: 700px;
    padding: 25px;
  }

  select {
    background-color: white;
    border-color: #dbdbdb;
    border-radius: 4px;
    color: #363636;
  }

  .checkbox-label {
    padding: 0;
    text-align: left;
  }

  input[type='checkbox'] {
    position: relative;
    margin: 0 6px 3px 3px;
  }
</style>

<div class="container is-fluid">
  <section>
    <form class="form" on:submit={handleSubmit}>
      <FormLine id="collection" label="Collection">
        <TextInput
          id="collection"
          placeholder="Collection CITE URN"
          bind:value={collection}
          bind:inputRef={collectionRef}
          validate={(value) => validateUrn(value, { noPassage: true })}
          invalidMessage="Please enter a valid CITE collection URN." />
      </FormLine>

      <FormLine id="name" label="Image Name">
        <TextInput
          id="name"
          placeholder="Image CITE URN"
          bind:value={imageName}
          bind:inputRef={imageNameRef}
          validate={(value) => validateUrn(value)}
          invalidMessage="Please enter a valid CITE object URN."
          autocomplete={false} />
      </FormLine>

      <FormLine id="source" label="Source">
        <TextInput
          id="source"
          placeholder="Resource URL"
          bind:value={imageUrl}
          validate={(value) => validateUrn(value) || validateHttpUrl(value)}
          invalidMessage="Please enter a valid CITE object URN or a HTTP(S) URL." />
      </FormLine>

      <FormLine id="protocol" label="Type">
        <div class="select">
          <select id="protocol" bind:value={protocol}>
            <option value="static">Static</option>
            <option value="localDZ">Deep Zoom</option>
            <option value="iiif">IIIF</option>
          </select>
        </div>
      </FormLine>

      <FormLine offset>
        <button
          class="button is-success"
          disabled={!complete}
          on:click={handleSubmit}>
          Add Image
        </button>
        {#if statusMessage}
          <Message text={statusMessage} error={errorMessage} />
        {/if}
      </FormLine>
    </form>
  </section>
</div>
