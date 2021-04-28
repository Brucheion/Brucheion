<script>
  import { onMount } from 'svelte'
  import { Link, navigate } from 'svelte-routing'
  import OpenSeadragon from 'openseadragon'
  import { getInternalOpts } from '../lib/osd'

  export let passage

  // TODO debug
  $: console.log('passage', passage)

  let previewViewer = undefined,
    viewerOpts = undefined,
    previewVisible = false,
    previewFailed = false,
    selectedImageRef = undefined,
    didMount = false,
    selectedCatalogUrn = passage.catalog.urn

  function createViewer(opts) {
    const { tileSources, ...otherOpts } = opts
    previewViewer = OpenSeadragon(otherOpts)

    previewViewer.addHandler('open-failed', () => {
      previewVisible = false
      previewViewer.destroy()
      previewFailed = true
    })

    previewViewer.addHandler('open', () => {
      previewVisible = true
    })

    previewViewer.open(tileSources)
  }

  // this should update the folio viewer a) once after mounting and b) when `passage` changes due to reactivity.
  $: if (didMount) {
    updateViewer(passage.imageRefs)
  }

  function updateViewer(refs) {
    if (previewViewer) {
      previewViewer.destroy()
    }

    if (Array.isArray(refs) && refs.length > 0) {
      viewerOpts = getInternalOpts('preview', refs[0])
      selectedImageRef = refs[0]

      createViewer(viewerOpts)
    }
  }

  function handleWitnessSelection(e) {
    console.log('witness selected', e.target.value)

    // FIXME: this is a pretty naive attempt to catch the passage ID
    const p = passage.id.split(':').pop()
    console.log('passage ID', passage.id, selectedCatalogUrn, p)

    navigate(`/view/${selectedCatalogUrn}${p}`)
  }

  onMount(() => {
    didMount = true
  })
</script>

<style>
  .toolbar {
    display: flex;
    flex-direction: row;
  }

  .witnesses {
  }

  .desk {
    display: flex;
    flex-direction: column;
    width: 100%;
    height: 100%;
  }

  .preview {
    box-sizing: border-box;
    width: 100%;
    height: 601px;
    border: 2px solid rgba(230, 230, 230);
    border-radius: 4px;
    padding: 4px;

    background: rgba(246, 245, 245);
    box-shadow: 0px 0px 5px rgba(0, 0, 0, 0.15);
  }
</style>

<div>
  <nav role="navigation" class="toolbar">
    <Link to={`/view/${passage.previousPassage}`}>Previous Passage</Link>
    <Link to={`/view/${passage.nextPassage}`}>Next Passage</Link>

    <form class="witnesses">
      <div class="control">
        <div class="select">
          <select
            bind:value={selectedCatalogUrn}
            on:change={handleWitnessSelection}>
            {#each passage.textRefs as ref}
              <option value={ref}>{ref}</option>
            {/each}
          </select>
        </div>
      </div>
    </form>
  </nav>

  <div class="desk">
    <section>
      <div>
        <h3>Folio selection</h3>
        <div class="select">
          <select bind:value={selectedImageRef}>
            {#if !!passage.imageRefs}
              {#each passage.imageRefs as ref}
                <option value={ref}>{ref}</option>
              {/each}
            {:else}
              <option disabled>No image references</option>
            {/if}
          </select>
        </div>
      </div>
      <div id="preview" class="preview" />
    </section>

    <section>
      <div class="tile is-ancestor" id="work-row-1">
        <p>
          {#each passage.transcriptionLines as line}
            {line}
            <br />
          {/each}
        </p>
      </div>
    </section>
  </div>
</div>
