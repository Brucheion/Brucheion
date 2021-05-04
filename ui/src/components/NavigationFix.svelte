<script>
  /* the HTML code below is copied directly from the Go template (`navigation.html`), which is not good practice due
   * to duplicate code fragments, but i haven't seen any better way for handling this since we're dealing with data that
   * is loaded client-side and we somehow need to update the navigation anchors. see this as a simple expense while
   * Brucheion's UI is being transitioned to the client side :^)
   */
  import { onMount } from 'svelte'
  export let passageURN
  let templateElement

  onMount(() => {
    const nav = document.getElementById('navMenu')
    nav.innerHTML = ''
    nav.appendChild(templateElement.content.firstChild)
  })
</script>

<template bind:this={templateElement}>
  <div class="navbar-end">
    <div class="navbar-item has-dropdown is-hoverable">
      <p class="navbar-link">Tools</p>
      <div class="navbar-dropdown">
        <a href="/view/{passageURN}" class="navbar-item">Passage Overview</a>
        <a href="/edit2/{passageURN}" class="navbar-item">Image Citation</a>
        <a href="/edit/{passageURN}" class="navbar-item">Transcribe</a>
        <a href="/multicompare/{passageURN}" class="navbar-item">
          Multicompare
        </a>
        <a href="/tree/" class="navbar-item">Morpho-syntactic analysis</a>
        <a href="/editcat/{passageURN}" class="navbar-item">Edit Metadata</a>
        <a href="/ingest/cex" class="navbar-item">Ingest CEX</a>
        <a href="/ingest/image" class="navbar-item">Ingest Image</a>
      </div>
    </div>
    <div class="navbar-item has-dropdown is-hoverable">
      <p class="navbar-link">
        <i class="fa fa-lg fa-fw fa-user-circle" />
        <span class="navbar-item-label">(username)</span>
      </p>
      <div class="navbar-dropdown">
        <a href="/export/exported/" class="navbar-item">Download CEX</a>
        <a href="#" class="navbar-item">Account Settings</a>
        <a href="#" class="navbar-item">Display Settings</a>
        <a href="/logout/" class="navbar-item">Logout</a>
      </div>
    </div>
  </div>
</template>
