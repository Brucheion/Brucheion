<script>
  import FormLine from '../components/FormLine.svelte'
  import TextInput from '../components/TextInput.svelte'
  import Message from '../components/Message.svelte'
  import debounce from '../lib/debounce'
  import { purename } from '../lib/fs'

  let inputRef = undefined
  let cexFile = ''
  let fileName = ''
  let complete = false
  let validName = true

  function handleSubmit(event) {
    event.preventDefault()
  }

  async function handleFileName(fileName) {
    let res
    try {
      res = await fetch(`/api/v1/cex/exists?name=${fileName}`)
    } catch (err) {
      validName = true
      return
    }

    if (res.status !== 200) {
      validName = false
      return
    }
    const d = await res.json()
    if (d.status !== 'success' || d.data.exists) {
      validName = false
      return
    }
    validName = true
  }
  const debouncedHandleFileName = debounce(handleFileName, 500)

  $: if (cexFile && inputRef) {
    const [file] = inputRef.files

    if (file.name.match(/\.cex$/)) {
      if (!fileName) {
        fileName = purename(file.name)
      }
    }
  }

  $: if (!!fileName) {
    debouncedHandleFileName(fileName)
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

          <FormLine id="file-name" label="File name">
            <TextInput
              id="file-name"
              placeholder="References file name"
              bind:value={fileName}
              invalid={!validName} />

            {#if !validName}
              <Message
                error={true}
                text="This name is not valid. Please choose a different one." />
            {/if}
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
