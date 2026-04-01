/**
 * Browser-local operator workspace focus — continuity across pages on this device only.
 * Not shared across operators; not canonical truth.
 */
const STORAGE_KEY = 'mel_operator_workspace_focus_v1'

export interface OperatorWorkspaceFocus {
  incidentId: string
  incidentTitle?: string
  /** ISO timestamp when focus was set */
  savedAt: string
}

export function readOperatorWorkspaceFocus(): OperatorWorkspaceFocus | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return null
    const j = JSON.parse(raw) as unknown
    if (!j || typeof j !== 'object') return null
    const id = (j as { incidentId?: string }).incidentId
    if (typeof id !== 'string' || !id.trim()) return null
    const title = (j as { incidentTitle?: string }).incidentTitle
    const savedAt = (j as { savedAt?: string }).savedAt
    return {
      incidentId: id.trim(),
      incidentTitle: typeof title === 'string' ? title : undefined,
      savedAt: typeof savedAt === 'string' && savedAt ? savedAt : new Date().toISOString(),
    }
  } catch {
    return null
  }
}

export function writeOperatorWorkspaceFocus(f: OperatorWorkspaceFocus): void {
  try {
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({
        incidentId: f.incidentId,
        incidentTitle: f.incidentTitle,
        savedAt: f.savedAt,
      }),
    )
  } catch {
    /* quota / private mode */
  }
}

export function clearOperatorWorkspaceFocus(): void {
  try {
    localStorage.removeItem(STORAGE_KEY)
  } catch {
    /* ignore */
  }
}
