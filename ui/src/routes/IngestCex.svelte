<script>
  import FormLine from '../components/FormLine.svelte'
  import Message from '../components/Message.svelte'

  let inputRef, formRef
  let cexFile = ''
  let complete = false
  let loading = false
  let modalVisible = false
  let errorMessage = null

  const errorMessages = {
    bad_file_ext:
      'The submitted file did not have the corresponding .cex file extension.',
    bad_file_body: 'The submitted file could not be read.',
    file_not_found: 'The submitted file could not be read.',
    bad_cex_data:
      'The CEX data contained erroneous data and could not be processed.',
    unknown:
      'An unknown error occurred. This is not necessarily related to the uploaded CEX data',
  }

  function showModal() {
    modalVisible = true
  }
  function hideModal() {
    modalVisible = false
    errorMessage = null
  }

  async function handleSubmit(event) {
    event.preventDefault()
    if (inputRef.files.length < 1) {
      return
    }

    const file = inputRef.files[0]
    const formData = new FormData()
    formData.append('file', file)

    loading = true
    const res = await fetch('/api/v1/cex/upload', {
      method: 'POST',
      body: formData,
    })
    loading = false

    if (res.status !== 200) {
      try {
        const data = await res.json()
        if (typeof errorMessages[data.message] !== 'undefined') {
          errorMessage = errorMessages[data.message]
        } else {
          errorMessage = errorMessages.unknown
        }
      } catch (err) {
        errorMessage = errorMessages.unknown
      }
    } else {
      formRef.reset()
    }

    showModal()
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

<div class="container">
  <section>
    <div class="columns">
      <div class="column form-column">
        <p>
          Ingest references created previously to set up a project on Brucheion.
          Please select the respective CEX file below and submit the file to
          start the ingestion process.
        </p>

        <form class="form" bind:this={formRef} on:submit={handleSubmit}>
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
            <Message
              text="Processing CEX data may take up to several minutes. Please
              don't refresh the page while ingestion is in progress." />
          </FormLine>
          <FormLine>
            <button
              class="button is-success"
              type="submit"
              class:is-loading={loading}
              disabled={!complete || loading}
              on:click={handleSubmit}>
              Ingest
            </button>
          </FormLine>
        </form>

        <div class="modal" class:is-active={modalVisible}>
          <div class="modal-background" on:click={hideModal} />
          <div class="modal-card">
            <header class="modal-card-head">
              <p class="modal-card-title">
                {#if errorMessage !== null}Error{:else}Ingestion Successful{/if}
              </p>
              <button class="delete" aria-label="close" on:click={hideModal} />
            </header>
            <section class="modal-card-body">
              {#if errorMessage !== null}
                <p>{errorMessage}</p>
              {:else}
                <p>The CEX data has been ingested successfully.</p>
              {/if}
            </section>
            <footer class="modal-card-foot">
              <button class="button" on:click={hideModal}>Dismiss</button>
            </footer>
            <button
              class="modal-close is-large"
              aria-label="close"
              on:click={hideModal} />
          </div>
        </div>
      </div>
    </div>
  </section>
</div>
