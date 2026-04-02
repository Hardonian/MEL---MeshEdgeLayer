import { useEffect } from 'react'
import { shouldIgnoreHotkey } from '@/utils/keyboard'

export interface PageHotkey {
  key: string
  description: string
  handler: () => void
  preventDefault?: boolean
  allowWithModifiers?: boolean
}

export function usePageHotkeys(bindings: PageHotkey[]) {
  useEffect(() => {
    if (bindings.length === 0) return
    const onKeyDown = (event: KeyboardEvent) => {
      if (shouldIgnoreHotkey(event)) return
      for (const binding of bindings) {
        const sameKey = event.key.toLowerCase() === binding.key.toLowerCase()
        if (!sameKey) continue
        const hasModifier = event.metaKey || event.ctrlKey || event.altKey
        if (hasModifier && !binding.allowWithModifiers) continue
        if (binding.preventDefault ?? true) event.preventDefault()
        binding.handler()
        return
      }
    }
    document.addEventListener('keydown', onKeyDown)
    return () => document.removeEventListener('keydown', onKeyDown)
  }, [bindings])
}
