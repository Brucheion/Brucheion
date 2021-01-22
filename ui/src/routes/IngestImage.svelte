<script>
  import { stringify as stringifyQuery } from 'qs'
  import OpenSeadragon from 'openseadragon'
  import { onMount } from 'svelte'
  import { navigate } from 'svelte-routing'
  import FormLine from '../components/FormLine.svelte'
  import Message from '../components/Message.svelte'
  import { validateUrn } from '../lib/cts-urn'
  import TextInput from '../components/TextInput.svelte'
  import { validateHttpUrl } from '../lib/url'
  import { isIIIFImage } from '../lib/iiif'
  import { getStaticOpts, getIIIFOpts, getInternalOpts } from '../lib/osd'

  let collection = ''
  let imageName = ''
  let imageUrl = ''
  let external = true
  let protocol = 'static'

  let statusMessage = null,
    timeoutHandle = null
  let collectionRef, imageNameRef
  let collections = []
  let nameExists = false
  let previewViewer = undefined,
    viewerOpts = undefined,
    previewVisible = false,
    previewErrored = false

  $: validNames =
    validateUrn(collection, { noPassage: true }) && validateUrn(imageName)
  $: validSource = validateUrn(imageUrl) || validateHttpUrl(imageUrl)
  $: complete = validNames && validSource

  $: if (statusMessage !== null) {
    clearTimeout(timeoutHandle)
    timeoutHandle = setTimeout(() => (statusMessage = null), 10000)
  }
  $: errorMessage =
    statusMessage && statusMessage.toLowerCase().includes('error')
  $: external = !validateUrn(imageUrl)
  $: if (validNames) {
    fetch(`/getImageInfo/${collection}/${imageName}`).then(async (res) => {
      const imageInfo = await res.json()
      nameExists = !!imageInfo.urn
    })
  } else if (nameExists) {
    nameExists = false
  }

  async function displayExternalMedia(imageUrl) {
    try {
      const [isManifest, imageManifest] = await isIIIFImage(imageUrl)
      if (isManifest) {
        viewerOpts = getIIIFOpts('preview', imageManifest)
        protocol = 'iiif'
      } else {
        viewerOpts = getStaticOpts('preview', imageUrl)
        protocol = 'static'
      }
    } catch (err) {
      if (!err.message.includes('NetworkError')) {
        console.error(err.message)
      }

      viewerOpts = getStaticOpts('preview', imageUrl)
      protocol = 'static'
    }
  }

  $: if (validSource) {
    previewErrored = false

    if (validateHttpUrl(imageUrl)) {
      displayExternalMedia(imageUrl)
    } else if (validateUrn(imageUrl)) {
      viewerOpts = getInternalOpts('preview', imageUrl)
      protocol = 'localDZ'
    }
  }

  function createViewer(opts) {
    const { tileSources, ...otherOpts } = opts
    previewViewer = OpenSeadragon(otherOpts)

    previewViewer.addHandler('open-failed', () => {
      previewVisible = false
      previewViewer.destroy()
      previewErrored = true
    })

    previewViewer.addHandler('open', () => {
      previewVisible = true
    })

    previewViewer.open(tileSources)
  }

  /* We'll need to trick the Svelte reactivity here, since destroying a prior viewer before creating a new one will result
   * in a circular dependency within the $-statement. Hence, above we just create the viewer options and handle viewer
   * lifecycles in the below $-statement.
   */
  $: if (validSource && viewerOpts) {
    if (previewViewer) {
      previewVisible = false
      previewViewer.destroy()
    }

    createViewer(viewerOpts)
  }

  onMount(async () => {
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

    await fetchCollections()
  })

  async function fetchCollections() {
    const res = await fetch('/requestImgCollection/')
    collections = (await res.json()).item
  }

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
  .form-column {
    max-width: 724px;
  }

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

  input[type='checkbox'] {
    position: relative;
    margin: 0 6px 3px 3px;
  }

  .preview-container {
    margin-top: 25px;
    padding: 25px;
    opacity: 0;

    transition: opacity 125ms ease-out;
  }

  .preview-container.visible {
    opacity: 1;
  }

  @media screen and (min-width: 1088px) {
    .preview-container {
      margin-top: 0;
    }
  }

  .preview {
    box-sizing: border-box;
    max-width: 700px;
    height: 600px;
    border: 1px solid rgba(230, 230, 230);
    border-radius: 3px;
    padding: 3px;

    background: rgba(245, 245, 245);
    box-shadow: 0px 0px 5px rgba(0, 0, 0, 0.15);
  }
</style>

<div class="container is-fluid">
  <section>
    <div class="columns is-desktop">
      <div class="column form-column">
        <form class="form" on:submit={handleSubmit}>
          <h4 class="title is-4">Media Data</h4>
          <FormLine id="collection" label="Collection">
            <TextInput
              id="collection"
              placeholder="Collection CITE URN"
              bind:value={collection}
              bind:inputRef={collectionRef}
              validate={(value) => validateUrn(value, { noPassage: true })}
              invalidMessage="Please enter a valid CITE collection URN."
              items={collections} />
          </FormLine>

          <FormLine id="name" label="Image Name">
            <TextInput
              id="name"
              placeholder="Image CITE URN"
              bind:value={imageName}
              bind:inputRef={imageNameRef}
              validate={(value) => validateUrn(value)}
              invalidMessage="Please enter a valid CITE object URN."
              autocomplete="off" />
            {#if nameExists}
              <Message
                text="This URN already exists and will be replaced if submitted." />
            {/if}
          </FormLine>

          <FormLine id="source" label="Source">
            <TextInput
              id="source"
              placeholder="Resource URL"
              bind:value={imageUrl}
              validate={(value) => validateUrn(value) || validateHttpUrl(value)}
              invalidMessage="Please enter a valid CITE object URN or a HTTP(S)
              URL." />
            {#if previewErrored}
              <Message
                text="The media could not be loaded for preview due to errors.
                You can ingest it nonetheless." />
            {/if}
          </FormLine>

          <FormLine id="protocol" label="Type">
            <div class="select">
              <select id="protocol" bind:value={protocol}>
                <option value="static">Static</option>
                <option value="localDZ">Deep Zoom (local)</option>
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
      </div>
      <div class="column form-column">
        <div class="preview-container" class:visible={previewVisible}>
          <h3 class="title is-4">Preview</h3>
          <div id="preview" class="preview" />
        </div>
      </div>
    </div>
  </section>
</div>
