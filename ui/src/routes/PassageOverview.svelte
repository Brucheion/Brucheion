<script>
  import { validateUrn } from '../lib/cts-urn'
  import PassageDesk from '../components/PassageDesk.svelte'

  export let urn
  let passage, err

  $: if (!validateUrn(urn, { nid: 'cts' })) {
    err = new Error('Passage not found')
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

{#if passage && !err}
  <PassageDesk {passage} />
{:else if err}
  <p>An error occurred: {err}</p>
{/if}
