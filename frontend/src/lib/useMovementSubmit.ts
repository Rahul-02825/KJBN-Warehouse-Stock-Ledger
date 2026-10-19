import { useRef, useState } from 'react'
import { ApiError, api } from '../api/client'
import type { Movement } from '../api/types'
import { saveCreatedBy } from './createdBy'

// randomUUID needs HTTPS or localhost; fall back to a random string elsewhere.
function newKey(): string {
  return crypto.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(36).slice(2)}`
}

// Shared submit logic for every movement form: client checks, posting with an
// Idempotency-Key, and turning API errors into field messages.
export function useMovementSubmit(path: string) {
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [serverError, setServerError] = useState<ApiError | null>(null)
  const [saving, setSaving] = useState(false)
  const [lastError, setLastError] = useState<ApiError | null>(null)
  // One key per intended posting: a double click or retry cannot post twice.
  const key = useRef(newKey())

  async function submit(body: Record<string, unknown>, clientErrors: Record<string, string>, unit = ''): Promise<Movement | null> {
    setErrors(clientErrors)
    setServerError(null)
    setLastError(null)
    if (Object.keys(clientErrors).length > 0) return null

    setSaving(true)
    try {
      const movement = await api.post<Movement>(path, body, { 'Idempotency-Key': key.current })
      key.current = newKey()
      saveCreatedBy(String(body.created_by))
      return movement
    } catch (err) {
      const e = err as ApiError
      setLastError(e)
      if (e.code === 'VALIDATION_ERROR') setErrors(e.fields)
      else if (e.code === 'INSUFFICIENT_STOCK') {
        const u = unit ? ` ${unit}` : ''
        setErrors({ quantity: `Only ${e.details.available}${u} available in this bin; you asked for ${e.details.requested}${u}.` })
      } else setServerError(e)
      return null
    } finally {
      setSaving(false)
    }
  }

  return { submit, errors, serverError, saving, lastError }
}
