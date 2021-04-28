<script>
  import { validateUrn } from '../lib/cts-urn'
  import PassageDesk from '../components/PassageDesk.svelte'

  export let urn
  let passage, err

  if (!validateUrn(urn, { nid: 'cts' })) {
    // TODO
    err = new Error('Not found')
  }

  $: getPassage(urn)
    .then((p) => (passage = p))
    .catch((e) => (err = e))

  async function getPassage(urn) {
    const res = await fetch(`/api/v1/passage/${urn}`)
    const d = await res.json()
    return d.data
  }
</script>

<div>
  <code>{urn}</code>

  {#if passage && !err}
    <PassageDesk {passage} />
  {:else if err}
    <p>An error occurred: {err}</p>
  {/if}
</div>
