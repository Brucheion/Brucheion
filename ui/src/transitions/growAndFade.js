import { expoInOut } from 'svelte/easing'

export default function growAndFade(
  node,
  { delay = 0, duration = 250, easing = expoInOut }
) {
  const opacity = +getComputedStyle(node).opacity || 1
  const height = node.offsetHeight

  return {
    delay,
    duration,
    easing,
    css: (t) => `
      opacity: ${t * opacity};
      height: ${t * height}px
    `,
  }
}
