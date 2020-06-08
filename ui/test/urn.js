const test = require('ava')
const { validateUrn } = require('../src/lib/cts-urn.js')

test('validateUrn()', (t) => {
  const validUrns = [
    'urn:cite2:sktlit:skt0001.nyaya002.msC3D:foo',
    'urn:cite2:sktlit:skt0001.nyaya002.msC3D:1.2-1.5',
  ]
  const invalidUrns = ['foobar', 'foo:bar:', 'urn:cite2:foo::']
  t.plan(validUrns.length + invalidUrns.length)

  for (const urn of validUrns) {
    t.is(validateUrn(urn), true, `${urn} not validated correctly`)
  }
  for (const urn of invalidUrns) {
    t.is(validateUrn(urn), false, `${urn} not validated correctly`)
  }
})

test('validateUrn() without passages', (t) => {
  const validUrns = [
    'urn:cite2:sktlit:skt0001.nyaya002.msC3D:',
    'urn:cite2:sktlit:skt0001.nyaya002.msC3D',
  ]
  const invalidUrns = ['foobar', 'foo:bar:', 'urn:cite2:foo::']
  t.plan(validUrns.length + invalidUrns.length)

  for (const urn of validUrns) {
    t.is(validateUrn(urn, { noPassage: true }), true)
  }
  for (const urn of invalidUrns) {
    t.is(validateUrn(urn, { noPassage: true }), false)
  }
})
