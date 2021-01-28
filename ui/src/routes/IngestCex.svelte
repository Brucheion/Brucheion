<script>
  import FormLine from '../components/FormLine.svelte'

  let inputRef
  let cexFile = ''
  let complete = false

  async function handleSubmit(event) {
    event.preventDefault()
    if (inputRef.files.length < 1) {
      return
    }

    const file = inputRef.files[0]
    const formData = new FormData()
    formData.append('file', file)

    const res = await fetch('/api/v1/cex/upload', {
      method: 'POST',
      body: formData,
    })

    const data = await res.json()
    if (res.status !== 200) {
      console.error('HALP', data)
    } else {
      console.log('cool :)', data)
    }
  }

  $: complete = !!cexFile
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

  .input-file {
    padding: 4px 0;
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
              class="input-file"
              bind:value={cexFile}
              bind:this={inputRef} />
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
