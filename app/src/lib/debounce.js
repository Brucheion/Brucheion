export default function debounce(fn, delay) {
  let timeout

  const call = (...args) => {
    if (timeout) {
      clearTimeout(timeout)
    }
    timeout = setTimeout(() => fn(...args), delay)
  }

  call.cancel = () => {
    if (timeout) {
      clearTimeout(timeout)
    }
  }

  return call
}
