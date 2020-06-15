export default function debounce(callback, delay) {
  let timeoutHandle

  return {
    call: (...args) => {
      if (timeoutHandle) {
        clearTimeout(timeoutHandle)
      }
      timeoutHandle = setTimeout(() => callback(...args), delay)
    },
    cancel: () => {
      if (timeoutHandle) {
        clearTimeout(timeoutHandle)
      }
    },
  }
}
