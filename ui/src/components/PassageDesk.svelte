<script>
  import { onMount } from 'svelte'
  import { Link, navigate } from 'svelte-routing'
  import OpenSeadragon from 'openseadragon'
  import { getInternalOpts } from '../lib/osd'

  export let passage

  let previewViewer = undefined,
    viewerOpts = undefined,
    previewVisible = false,
    previewFailed = false,
    selectedImageRef = undefined,
    didMount = false,
    selectedCatalogUrn = passage.catalog.urn

  // FIXME: this is a pretty naive attempt to catch the passage ID
  $: passageId = passage.id.split(':').pop()

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

  function updateViewer(refs) {
    if (previewViewer) {
      previewViewer.destroy()
    }

    if (Array.isArray(refs) && refs.length > 0) {
      if (!selectedImageRef || !refs.includes(selectedImageRef)) {
        selectedImageRef = refs[0]
      }
      viewerOpts = getInternalOpts('preview', selectedImageRef)

      createViewer(viewerOpts)
    }
  }

  function handleWitnessSelection() {
    navigate(`/view/${selectedCatalogUrn}${passageId}`)
  }

  /* this should update the folio viewer a) once after mounting and b) when `passage` changes due to reactivity.
   * this is just a lazy trick to trigger the viewer update in coordination with svelte's reactivity */
  $: if (didMount) {
    updateViewer(passage.imageRefs)
  }
  onMount(() => {
    didMount = true
  })
</script>

<style>
  :global(:root) {
    --toolbar-bg-color: rgb(240, 240, 240);
    --toolbar-text-color: rgb(50, 50, 50);
    --toolbar-border-color: rgb(200, 200, 200);
  }

  .toolbar {
    display: flex;
    flex-direction: row;
    align-items: center;

    height: 40px;
    margin: 0;
    border-top: 1px solid var(--toolbar-border-color);
    border-bottom: 1px solid var(--toolbar-border-color);
    padding: 0;

    list-style: none;
    background: var(--toolbar-bg-color, white);
  }

  .toolbar,
  .toolbar select {
    font: 400 14px/100% 'Inter', sans-serif;
    color: var(--toolbar-text-color, black);
  }

  .toolbar.stacked-below {
    border-top: 0;
  }

  .toolbar li {
    flex-shrink: 0;
    margin: 0;
    padding: 2px 8px;
  }

  /* join left */
  .toolbar li.jl {
    margin-left: 2px;
  }
  /* space left */
  .toolbar li.sl {
    margin-left: 8px;
  }
  /* border left */
  .toolbar li.bl {
    border-left: 1px solid var(--toolbar-border-color);
  }

  .toolbar li:last-child {
    border-right: 0;
  }

  .toolbar li label {
    margin: 0;
    padding: 0;

    font-weight: 600;
    font-size: inherit;
  }

  .toolbar li code {
    font: 14px/100% 'IBM Plex Mono', monospace;
    color: inherit;
  }

  .desk {
    display: flex;
    flex-direction: column;
    flex-grow: 1;

    width: 100%;
    height: 100%;
    min-height: 600px;
  }

  .pane {
    flex-basis: 50%;
    flex-shrink: 0;
    flex-grow: 1;
  }

  .pane.static {
    flex-grow: 0;
  }

  .pane.grow {
    display: flex;
    flex-direction: column;
  }

  .preview {
    display: flex;
    flex-direction: column;
    flex-grow: 0;
    flex-shrink: 0;

    box-sizing: border-box;
    width: 100%;
    height: 450px;
    padding: 4px;
    background: rgba(246, 245, 245);
  }

  :global(.openseadragon-container) {
    flex-grow: 1;
  }

  .transcription {
    box-sizing: border-box;
    padding: 16px;
    overflow-y: scroll;
  }
</style>

<nav role="navigation">
  <ul class="toolbar">
    <li>
      <label>Passage</label>
    </li>

    <li class="sl">
      <div class="select">
        <select
          bind:value={selectedCatalogUrn}
          on:change={handleWitnessSelection}>
          {#each passage.textRefs as ref}
            <option value={ref}>{ref}</option>
          {/each}
        </select>
      </div>
    </li>
    <li class="jl">
      <code>{passageId}</code>
    </li>

    <li class="sl">
      <Link to={`/view/${passage.previousPassage}`}>← Previous Passage</Link>
    </li>
    <li class="bl">
      <Link to={`/view/${passage.nextPassage}`}>Next Passage →</Link>
    </li>

    <li class="sl">
      <a href="#">Metadata</a>
    </li>
  </ul>
</nav>

<div class="desk">
  <section class="pane grow static">
    <ul class="toolbar stacked-below">
      <li>
        <label>Folio</label>
      </li>
      <li>
        <form on:submit|preventDefault={() => console.log('help?')}>
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
        </form>
      </li>

      <li class="sl">
        <Link to={`/edit2/${passage.id}`}>Edit References</Link>
      </li>
    </ul>
    <div id="preview" class="preview" />
  </section>

  <section class="pane">
    <ul class="toolbar">
      <li>
        <label>Transcription</label>
      </li>
    </ul>
    <div class="transcription">
      <p>
        {#each passage.transcriptionLines as line}
          {line}
          <br />
        {/each}
      </p>
    </div>
  </section>
</div>
