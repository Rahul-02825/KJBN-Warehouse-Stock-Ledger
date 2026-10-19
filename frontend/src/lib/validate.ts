// Quick checks so the user sees mistakes before submitting. The backend
// checks everything again; these only mirror its rules.

const decimalPattern = /^\d+(\.\d{1,4})?$/

// Returns an error message, or '' when the value is fine.
export function checkDecimal(value: string, unit: string, { allowZero = false, required = true } = {}): string {
  const v = value.trim()
  if (v === '') return required ? 'Required' : ''
  if (v.startsWith('-')) return 'Must not be negative'
  if (!decimalPattern.test(v)) return 'Enter a number with at most 4 decimal places'
  if (!allowZero && /^0+(\.0+)?$/.test(v)) return 'Must be greater than zero'
  if (unit === 'pcs' && v.includes('.') && !/\.0+$/.test(v)) return 'Must be a whole number for pcs'
  return ''
}

export function checkText(value: string, label: string, max: number, required = true): string {
  const v = value.trim()
  if (required && v === '') return `${label} is required`
  if (v.length > max) return `${label} must be at most ${max} characters`
  return ''
}

// Keeps only the messages that are set.
export function onlyErrors(errors: Record<string, string>): Record<string, string> {
  return Object.fromEntries(Object.entries(errors).filter(([, message]) => message !== ''))
}
