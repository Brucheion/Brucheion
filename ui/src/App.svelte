<script>
  import { Router, Link, Route } from 'svelte-routing'
  import Ingest from './routes/Ingest.svelte'
  import NotFound from './routes/NotFound.svelte'

  export let url = location.pathname
</script>

<Router url="{url}">
  <Route path="ingest" component={Ingest}/>
  <Route path="*" component={NotFound}/>
</Router>
