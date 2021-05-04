import contentType from 'content-type'

export async function isIIIFImage(url) {
  const res = await fetch(url, { mode: 'cors' })
  const ct = res.headers.get('content-type')

  if (res.ok && ct && contentType.parse(ct).type === 'application/json') {
    const imageManifest = await res.json()
    return [
      imageManifest['@context'] === 'http://iiif.io/api/image/2/context.json',
      imageManifest,
    ]
  } else {
    return [false, null]
  }
}
