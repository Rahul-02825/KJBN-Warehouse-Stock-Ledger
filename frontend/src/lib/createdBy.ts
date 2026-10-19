// Remembers the operator's name in this browser so it need not be typed on every form.
const KEY = 'stock-ledger.created-by'

export function loadCreatedBy(): string {
  try {
    return localStorage.getItem(KEY) ?? ''
  } catch {
    return ''
  }
}

export function saveCreatedBy(name: string) {
  try {
    localStorage.setItem(KEY, name)
  } catch {
    // Storage can be blocked (private mode); the form still works.
  }
}
