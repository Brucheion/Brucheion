<script>
  import FormLine from '../components/FormLine.svelte'
  import TextInput from '../components/TextInput.svelte'

  let inputRef = undefined
  let cexFile = ''
  let fileName = ''
  let uploadFile = true
  let complete = false

  function handleSubmit(event) {
    event.preventDefault()
  }

  $: if (cexFile && inputRef) {
    const [file] = inputRef.files

    if (file.name.match(/\.cex$/)) {
      if (!fileName) {
        fileName = file.name
      }
    }
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
</style>

<div class="container is-fluid">
  <section>
    <div class="columns is-desktop">

      <div class="column form-column">
        <p>
          Ingest previously created references to set up project on Brucheion.
          Please select the respective CEX file below.
        </p>

        <form class="form" on:submit={handleSubmit}>
          <h4 class="title is-4">Media Data</h4>
          <FormLine id="cex-file" label="CEX File">
            <input
              id="cex-file"
              type="file"
              accept=".cex"
              bind:value={cexFile}
              bind:this={inputRef} />
          </FormLine>

          <FormLine id="file-name" label="File name">
            <TextInput
              id="file-name"
              placeholder="References file name"
              bind:value={fileName} />
          </FormLine>

          <FormLine id="upload-file" label="Upload file?">
            <input type="checkbox" id="upload-file" bind:checked={uploadFile} />
          </FormLine>

          <FormLine offset>
            <button
              class="button is-success"
              disabled={!complete}
              on:click={handleSubmit}>
              Upload CEX File
            </button>
          </FormLine>
        </form>
      </div>
    </div>
  </section>
</div>
