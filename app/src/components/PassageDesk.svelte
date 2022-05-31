<script>
  import { onMount } from 'svelte'
  import { Link, navigate } from 'svelte-routing'
  import OpenSeadragon from 'openseadragon'
  import { getInternalOpts } from '../lib/osd'
  import ResizeBar from './ResizeBar.svelte'

  export let passage

  let previewViewer = undefined,
    viewerOpts = undefined,
    previewVisible = false,
    previewFailed = false,
    selectedImageRef = undefined,
    didMount = false,
    selectedCatalogUrn = passage.catalog.urn,
    showMetadata = false,
    previewContainer = undefined,
    previewHeight = 350

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

  function handleToggleMetadata() {
    showMetadata = !showMetadata
  }
  function handleHideMetadata() {
    showMetadata = false
  }

  /* this should update the folio viewer a) once after mounting and b) when `passage` changes due to reactivity.
   * this is just a lazy trick to trigger the viewer update in coordination with svelte's reactivity */
  $: if (didMount) {
    updateViewer(passage.imageRefs)
  }
  onMount(() => {
    didMount = true
  })

  function handleResize(e) {
    previewHeight = e.detail.y - previewContainer.offsetTop
  }
</script>

<style>
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
  .toolbar li.pl {
    margin-left: 8px;
  }
  /* border left */
  .toolbar li.bl {
    border-left: 1px solid var(--toolbar-border-color);
  }
  /* fill left */
  .toolbar li.fl {
    margin-left: auto;
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

  .vertical-split {
    display: flex;
    flex-direction: row;
    height: 100%;
  }

  .vertical-split .pane {
    border-left: 1px solid var(--toolbar-border-color);
  }
  .vertical-split .pane:first-child {
    border-left: 0;
  }

  .preview {
    display: flex;
    flex-direction: column;
    flex-grow: 0;
    flex-shrink: 0;

    box-sizing: border-box;
    width: 100%;
    height: var(--height, 500px);
    padding: 4px;

    background: var(--pane-bg-color);
  }

  :global(.openseadragon-container) {
    flex-grow: 1;
  }

  .transcription {
    box-sizing: border-box;
    height: 100%;
    padding: 16px;
    overflow-y: scroll;
  }

  .metadata {
    font: 14px/130% 'Inter', sans-serif;
    background: white;
  }

  .metadata dl dt {
    padding: 8px 16px 4px;
  }

  .metadata dl dd {
    padding: 0px 16px 8px;
  }

  .metadata dl dt:nth-child(4n + 1),
  .metadata dl dt:nth-child(4n + 1) + dd {
    background: var(--pane-bg-color);
  }

  .close-pane {
    font-size: 20px;
    font-weight: 600;
  }

  .close-pane:hover {
    text-decoration: none;
  }
</style>

<nav role="navigation">
  <ul class="toolbar">
    <li>
      <label for="passage.id">Passage</label>
    </li>

    <li class="pl">
      <div class="select">
        <select
          bind:value={selectedCatalogUrn}
          on:blur={handleWitnessSelection}>
          {#each passage.textRefs as ref}
            <option value={ref}>{ref}</option>
          {/each}
        </select>
      </div>
    </li>
    <li class="jl">
      <code>{passageId}</code>
    </li>

    <li class="pl">
      <Link to={`/view/${passage.previousPassage}`}>← Previous Passage</Link>
    </li>
    <li class="bl">
      <Link to={`/view/${passage.nextPassage}`}>Next Passage →</Link>
    </li>

      <li>
        <label for="passage.id">Folio</label>
      </li>
      <li>
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
      </li>

      <li class="pl">
        <Link to={`/edit2/${passage.id}`}>Edit References</Link>
      </li>

    <li class="pl">
      <a href="#" on:click|preventDefault={handleToggleMetadata}>
        {#if showMetadata}Hide{:else}Show{/if}
        Metadata
      </a>
    </li>

  </ul>
</nav>

<div class="desk">
  <section class="pane grow static">
    <div
      bind:this={previewContainer}
      id="preview"
      class="preview"
      style="--height: {previewHeight}px" />
  </section>

  <ResizeBar on:resize={handleResize} />

  <section class="pane">
    <div class="vertical-split">
      <div class="pane">
        <ul class="toolbar">
          <li>
            <label for="passage.id">Transcription</label>
          </li>
          <li>
            <a href={`/edit/${passage.id}`}>Edit</a>
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
      </div>
      {#if showMetadata}
        <div class="pane">
          <ul class="toolbar">
            <li>
              <label for="passage.id">Metadata</label>
            </li>
            <li>
              <a href={`/editcat/${passage.id}`}>Edit</a>
            </li>
            <li class="fl">
              <a
                href="#"
                class="close-pane"
                on:click|preventDefault={handleHideMetadata}>
                ×
              </a>
            </li>
          </ul>
          <div class="metadata">
            <dl>
              <dt>Work URN</dt>
              <dd>{passage.catalog.urn}</dd>
              <dt>Scheme</dt>
              <dd>{passage.catalog.citationScheme}</dd>
              <dt>Workgroup</dt>
              <dd>{passage.catalog.groupName}</dd>
              <dt>Title</dt>
              <dd>{passage.catalog.workTitle}</dd>
              <dt>Version Label</dt>
              <dd>{passage.catalog.versionLabel}</dd>
              <dt>Exemplar Label</dt>
              <dd>{passage.catalog.exemplarLabel}</dd>
              <dt>Online</dt>
              <dd>{passage.catalog.online}</dd>
              <dt>Language</dt>
              <dd>{passage.catalog.language}</dd>
            </dl>
          </div>
        </div>
      {/if}
    </div>
  </section>
</div>
