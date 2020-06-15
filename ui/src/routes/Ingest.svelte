<script>
  import { stringify as stringifyQuery } from 'qs'
  import { onMount } from 'svelte'
  import { navigate } from 'svelte-routing'
  import FormLine from '../components/FormLine.svelte'
  import Message from '../components/Message.svelte'
  import { validateUrn } from '../lib/cts-urn'

  let collection = ''
  let imageName = ''
  let imageUrl = ''
  let external = true
  let protocol = 'static'

  let errorMessage = null, timeoutHandle
  let collectionRef, imageNameRef

  $: complete = validateUrn(collection, { noPassage: true }) && validateUrn(imageName) && imageUrl

  $: if (errorMessage !== null) {
    clearTimeout(timeoutHandle)
    timeoutHandle = setTimeout(() => errorMessage = null, 10000)
  }

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
      errorMessage = 'An error occurred. Please try later.'
      return
    }
    errorMessage = 'Image ingested!'
  }
</script>

<div class="container is-fluid">
  <section>
    <form class="form" on:submit={handleSubmit}>
      <FormLine id="collection" label="Collection">
        <input id="collection" class="input" type="text" placeholder="Collection CITE URN"
               bind:value={collection} bind:this={collectionRef}/>
      </FormLine>

      <FormLine id="name" label="Image Name">
        <input id="name" class="input" type="text" placeholder="Image CITE URN" bind:value={imageName}
               bind:this={imageNameRef}/>
      </FormLine>

      <FormLine id="source" label="Source">
        <input id="source" class="input" type="text" placeholder="Resource URL" bind:value={imageUrl}/>
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

      <FormLine>
        <label class="checkbox label">
          <input type="checkbox" bind:checked={external}>
          External resource
        </label>
      </FormLine>

      <FormLine offset>
        <button class="button is-success" disabled={!complete} on:click={handleSubmit}>Add Image</button>
        {#if errorMessage}
          <Message text={errorMessage} error/>
        {/if}
      </FormLine>
    </form>
  </section>
</div>

<style>
  .form {
    box-sizing: border-box;
    max-width: 700px;
    padding: 25px;
  }

  input, select {
    background-color: white;
    border-color: #dbdbdb;
    border-radius: 4px;
    color: #363636;
  }

  input[type=checkbox] {
    margin: 0 6px 3px 3px;
  }
</style>
