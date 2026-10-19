import { useState, type FormEvent } from 'react'
import { ApiError, api } from '../api/client'
import type { Location } from '../api/types'
import { checkText, onlyErrors } from '../lib/validate'
import { Button, Card, ErrorAlert, Field, Input } from './ui'

export function LocationForm({ onSaved, onCancel }: { onSaved: (l: Location) => void; onCancel: () => void }) {
  const [code, setCode] = useState('')
  const [description, setDescription] = useState('')
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [serverError, setServerError] = useState<ApiError | null>(null)
  const [saving, setSaving] = useState(false)

  async function submit(e: FormEvent) {
    e.preventDefault()
    const found = onlyErrors({
      code: checkText(code, 'Code', 64) || (/\s/.test(code.trim()) ? 'Code must not contain spaces' : ''),
      description: checkText(description, 'Description', 500, false),
    })
    setErrors(found)
    setServerError(null)
    if (Object.keys(found).length > 0) return

    setSaving(true)
    try {
      onSaved(await api.post<Location>('/api/locations', { code: code.trim(), description: description.trim() }))
    } catch (err) {
      const apiErr = err as ApiError
      if (apiErr.code === 'DUPLICATE') setErrors({ code: 'A bin with this code already exists' })
      else if (apiErr.code === 'VALIDATION_ERROR') setErrors(apiErr.fields)
      else setServerError(apiErr)
    } finally {
      setSaving(false)
    }
  }

  return (
    <Card title="New bin" className="mb-6">
      <ErrorAlert error={serverError} />
      <form onSubmit={submit} noValidate className="grid gap-4 sm:grid-cols-3">
        <Field label="Code" error={errors.code} hint="Unique, no spaces, e.g. A-01 or COLD-02">
          <Input value={code} onChange={(e) => setCode(e.target.value)} invalid={!!errors.code} autoFocus />
        </Field>
        <div className="sm:col-span-2">
          <Field label="Description" error={errors.description} hint="Optional">
            <Input value={description} onChange={(e) => setDescription(e.target.value)} invalid={!!errors.description} />
          </Field>
        </div>
        <div className="flex gap-2 sm:col-span-3">
          <Button type="submit" disabled={saving}>
            {saving ? 'Saving…' : 'Create bin'}
          </Button>
          <Button type="button" variant="secondary" onClick={onCancel}>
            Cancel
          </Button>
        </div>
      </form>
    </Card>
  )
}
