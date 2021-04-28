<script>
  import { Router, Route } from 'svelte-routing'
  import IngestCex from './routes/IngestCex.svelte'
  import IngestImage from './routes/IngestImage.svelte'
  import PassageOverview from './routes/PassageOverview.svelte'
  import NotFound from './routes/NotFound.svelte'

  export let url = location.pathname
</script>

<Router {url}>
  <Route path="/view/:urn" component={PassageOverview} />
  <Route path="/ingest/cex" component={IngestCex} />
  <Route path="/ingest/image" component={IngestImage} />
  <Route path="*" component={NotFound} />
</Router>
