import { useState, type FormEvent } from 'react'
import { ApiError, api } from '../api/client'
import { UNITS, type Product } from '../api/types'
import { checkDecimal, checkText, onlyErrors } from '../lib/validate'
import { Button, Card, ErrorAlert, Field, Input, Select } from './ui'

// Creates a product, or edits one when `product` is given. SKU and unit cannot change after creation.
export function ProductForm({ product, onSaved, onCancel }: { product?: Product; onSaved: (p: Product) => void; onCancel: () => void }) {
  const editing = product !== undefined
  const [sku, setSku] = useState(product?.sku ?? '')
  const [name, setName] = useState(product?.name ?? '')
  const [unit, setUnit] = useState<string>(product?.unit ?? 'pcs')
  const [reorderLevel, setReorderLevel] = useState(product?.reorder_level ?? '0')
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [serverError, setServerError] = useState<ApiError | null>(null)
  const [saving, setSaving] = useState(false)

  async function submit(e: FormEvent) {
    e.preventDefault()
    const found = onlyErrors({
      sku: editing ? '' : checkText(sku, 'SKU', 64),
      name: checkText(name, 'Name', 200),
      reorder_level: checkDecimal(reorderLevel, unit, { allowZero: true }),
    })
    setErrors(found)
    setServerError(null)
    if (Object.keys(found).length > 0) return

    setSaving(true)
    try {
      const saved = editing
        ? await api.patch<Product>(`/api/products/${product.id}`, { name: name.trim(), reorder_level: reorderLevel.trim() })
        : await api.post<Product>('/api/products', { sku: sku.trim(), name: name.trim(), unit, reorder_level: reorderLevel.trim() })
      onSaved(saved)
    } catch (err) {
      const apiErr = err as ApiError
      if (apiErr.code === 'DUPLICATE') setErrors({ sku: 'A product with this SKU already exists' })
      else if (apiErr.code === 'VALIDATION_ERROR') setErrors(apiErr.fields)
      else setServerError(apiErr)
    } finally {
      setSaving(false)
    }
  }

  return (
    <Card title={editing ? `Edit ${product.sku}` : 'New product'} className="mb-6">
      <ErrorAlert error={serverError} />
      <form onSubmit={submit} noValidate className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Field label="SKU" error={errors.sku} hint={editing ? 'Cannot be changed' : 'Unique code, e.g. SPRAYER-16L'}>
          <Input value={sku} onChange={(e) => setSku(e.target.value)} disabled={editing} invalid={!!errors.sku} autoFocus={!editing} />
        </Field>
        <Field label="Name" error={errors.name}>
          <Input value={name} onChange={(e) => setName(e.target.value)} invalid={!!errors.name} autoFocus={editing} />
        </Field>
        <Field label="Unit" error={errors.unit} hint={editing ? 'Cannot be changed' : 'pcs must be whole numbers'}>
          <Select value={unit} onChange={(e) => setUnit(e.target.value)} disabled={editing} invalid={!!errors.unit}>
            {UNITS.map((u) => (
              <option key={u} value={u}>
                {u}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Reorder level" error={errors.reorder_level} hint="Low-stock alert at or below this total">
          <Input value={reorderLevel} onChange={(e) => setReorderLevel(e.target.value)} inputMode="decimal" invalid={!!errors.reorder_level} />
        </Field>
        <div className="flex gap-2 sm:col-span-2 lg:col-span-4">
          <Button type="submit" disabled={saving}>
            {saving ? 'Saving…' : editing ? 'Save changes' : 'Create product'}
          </Button>
          <Button type="button" variant="secondary" onClick={onCancel}>
            Cancel
          </Button>
        </div>
      </form>
    </Card>
  )
}
