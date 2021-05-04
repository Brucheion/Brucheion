<script>
  import { validateUrn } from '../lib/cts-urn'
  import PassageDesk from '../components/PassageDesk.svelte'
  import NavigationFix from '../components/NavigationFix.svelte'

  export let urn
  let passage, user, err

  $: if (!validateUrn(urn, { nid: 'cts' })) {
    err = new Error('Passage not found')
  }

  $: Promise.all([getPassage(urn), getUser()])
    .then(([p, u]) => {
      passage = p
      user = u
    })
    .catch((e) => (err = e))

  async function getPassage(urn) {
    const res = await fetch(`/api/v1/passage/${urn}`)
    const d = await res.json()
    return d.data
  }

  async function getUser() {
    const res = await fetch(`/api/v1/user`)
    const d = await res.json()
    return d.data
  }
</script>

{#if passage && !err}
  <PassageDesk {passage} />
  <NavigationFix passageURN={passage.id} userName={user.name} />
{:else if err}
  <p>An error occurred: {err}</p>
{/if}
