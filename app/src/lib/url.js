// adopted from: <https://stackoverflow.com/a/43467144>
export function validateHttpUrl(url) {
  let _url

  try {
    _url = new URL(url)
  } catch (_) {
    return false
  }

  return _url.protocol === 'http:' || _url.protocol === 'https:'
}
