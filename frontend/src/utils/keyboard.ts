export function isEditableTarget(target: EventTarget | null): boolean {
  if (!target || !(target instanceof HTMLElement)) return false
  const tag = target.tagName
  if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return true
  if (target.isContentEditable) return true
  return false
}

export function shouldIgnoreHotkey(event: KeyboardEvent): boolean {
  if (event.defaultPrevented) return true
  if (isEditableTarget(event.target)) return true
  return false
}
