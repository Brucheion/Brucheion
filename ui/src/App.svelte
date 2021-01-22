<script>
  import { Router, Link, Route } from 'svelte-routing'
  import IngestImage from './routes/IngestImage.svelte'
  import NotFound from './routes/NotFound.svelte'

  export let url = location.pathname
</script>

<Router {url}>
  <Route path="/ingest/image" component={IngestImage} />
  <Route path="*" component={NotFound} />
</Router>
