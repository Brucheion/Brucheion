const URN = require('urn-lib')

const citeNid = 'cite2'
const citeUrn = URN.create('urn', {
  components: ['nid', 'namespace', 'work', 'passage'],
})

function validateUrn(urn, opts = {}) {
  const noPassage = opts.noPassage || false
  const components = citeUrn.parse(urn)

  return (
    !!components &&
    components.nid === citeNid &&
    !!components.namespace &&
    !!components.work &&
    ((noPassage && !components.passage) || (!noPassage && !!components.passage))
  )
}

module.exports = { validateUrn }
