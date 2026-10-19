import { useState, type FormEvent } from 'react'
import { ApiError, api } from '../api/client'
import type { Location, Warehouse } from '../api/types'
import { distanceFromCentre } from '../lib/layout'
import { checkText, onlyErrors } from '../lib/validate'
import { Button, Checkbox, ErrorAlert, Field, Input, Panel } from './ui'

const coordinatePattern = /^-?\d+(\.\d+)?$/

export function LocationForm({ warehouse, onSaved, onCancel }: { warehouse: Warehouse; onSaved: (l: Location) => void; onCancel: () => void }) {
  const [code, setCode] = useState('')
  const [description, setDescription] = useState('')
  const [x, setX] = useState('0')
  const [y, setY] = useState('0')
  const [capacity, setCapacity] = useState('')
  const [autoPlace, setAutoPlace] = useState(true)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [serverError, setServerError] = useState<ApiError | null>(null)
  const [saving, setSaving] = useState(false)

  function checkLayout(): Record<string, string> {
    const position = coordinatePattern.test(x.trim()) && coordinatePattern.test(y.trim())
    if (!position) return { x: 'Enter x and y in metres, e.g. 5 or -2.5' }
    const outside = distanceFromCentre(warehouse, Number(x), Number(y)) > warehouse.radius_m
    return { x: outside ? `Must be within ${warehouse.radius_m} m of the warehouse centre` : '' }
  }

  function checkCapacity(): string {
    const value = capacity.trim()
    if (value === '') return ''
    if (!/^\d+$/.test(value) || Number(value) === 0) return 'Enter a whole number above zero, or leave empty for no limit'
    if (warehouse.capacity !== null && Number(value) > warehouse.capacity) return `Must not exceed the warehouse capacity of ${warehouse.capacity}`
    return ''
  }

  async function submit(e: FormEvent) {
    e.preventDefault()
    const found = onlyErrors({
      code: checkText(code, 'Code', 64) || (/\s/.test(code.trim()) ? 'Code must not contain spaces' : ''),
      description: checkText(description, 'Description', 500, false),
      ...checkLayout(),
      capacity: checkCapacity(),
    })
    setErrors(found)
    setServerError(null)
    if (Object.keys(found).length > 0) return

    setSaving(true)
    try {
      onSaved(
        await api.post<Location>('/api/locations', {
          code: code.trim(),
          description: description.trim(),
          warehouse_id: warehouse.id,
          x: Number(x),
          y: Number(y),
          capacity: capacity.trim() === '' ? null : Number(capacity),
          auto_place: autoPlace,
        }),
      )
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
    <Panel title="New bin" className="mb-10">
      <ErrorAlert error={serverError} />
      <form onSubmit={submit} noValidate className="grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
        <Field label="Code" error={errors.code} hint="Unique, without spaces, such as A-01">
          <Input value={code} onChange={(e) => setCode(e.target.value)} invalid={!!errors.code} autoFocus className="font-mono" />
        </Field>
        <div className="sm:col-span-1 lg:col-span-3">
          <Field label="Description" error={errors.description} hint="Optional">
            <Input value={description} onChange={(e) => setDescription(e.target.value)} invalid={!!errors.description} />
          </Field>
        </div>
        <Field label="Position x (m)" error={errors.x} hint={`Within ${warehouse.radius_m} m of (${warehouse.x}, ${warehouse.y})`}>
          <Input value={x} onChange={(e) => setX(e.target.value)} inputMode="decimal" invalid={!!errors.x} />
        </Field>
        <Field label="Position y (m)">
          <Input value={y} onChange={(e) => setY(e.target.value)} inputMode="decimal" invalid={!!errors.x} />
        </Field>
        <Field label="Capacity (pieces)" error={errors.capacity} hint="Leave empty for no limit">
          <Input value={capacity} onChange={(e) => setCapacity(e.target.value)} inputMode="numeric" invalid={!!errors.capacity} />
        </Field>
        <div className="self-center">
          <Checkbox label="Use for automatic placement" checked={autoPlace} onChange={(e) => setAutoPlace(e.target.checked)} />
        </div>
        <div className="flex gap-2 sm:col-span-2 lg:col-span-4">
          <Button type="submit" disabled={saving}>
            {saving ? 'Saving…' : 'Add bin'}
          </Button>
          <Button type="button" variant="ghost" onClick={onCancel}>
            Cancel
          </Button>
        </div>
      </form>
    </Panel>
  )
}
