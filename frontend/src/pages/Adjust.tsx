import { useState, type FormEvent } from 'react'
import { Link, useSearchParams } from 'react-router'
import { ADJUSTMENT_REASONS, type Movement } from '../api/types'
import { BinStock, LocationSelect, ProductSelect } from '../components/MovementFields'
import { MovementFormCard } from '../components/MovementForm'
import { ErrorAlert, Field, Input, PageHeader, Select, Spinner, SuccessAlert } from '../components/ui'
import { loadCreatedBy } from '../lib/createdBy'
import { formatMoney, formatQuantity } from '../lib/format'
import { useMovementSubmit } from '../lib/useMovementSubmit'
import { useOptions } from '../lib/useOptions'
import { checkDecimal, checkText, onlyErrors } from '../lib/validate'

export function Adjust() {
  const [params] = useSearchParams()
  const { products, locations, error } = useOptions()
  const [productId, setProductId] = useState(params.get('product') ?? '')
  const [locationId, setLocationId] = useState('')
  const [direction, setDirection] = useState<'add' | 'remove'>('remove')
  const [quantity, setQuantity] = useState('')
  const [reason, setReason] = useState('')
  const [note, setNote] = useState('')
  const [createdBy, setCreatedBy] = useState(loadCreatedBy)
  const [done, setDone] = useState<Movement | null>(null)
  const { submit, errors, serverError, saving } = useMovementSubmit('/api/movements/adjustment')

  if (error) return <ErrorAlert error={error} />
  if (!products || !locations) return <Spinner />
  const unit = products.find((p) => String(p.id) === productId)?.unit ?? ''
  const adjusted = done ? products.find((p) => p.id === done.product_id) : undefined

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    // The API takes a signed quantity: negative removes stock.
    const signed = direction === 'remove' ? `-${quantity.trim()}` : quantity.trim()
    const movement = await submit(
      {
        product_id: Number(productId),
        location_id: Number(locationId),
        quantity: signed,
        reason,
        note: note.trim(),
        created_by: createdBy.trim(),
      },
      onlyErrors({
        product_id: productId ? '' : 'Choose a product',
        location_id: locationId ? '' : 'Choose a bin',
        quantity: checkDecimal(quantity, unit),
        reason: reason ? '' : 'Choose a reason',
        note: reason === 'other' && note.trim() === '' ? 'A note is required when the reason is Other' : checkText(note, 'Note', 1000, false),
        created_by: checkText(createdBy, 'Your name', 128),
      }),
      unit,
    )
    if (movement) {
      setDone(movement)
      setQuantity('')
      setNote('')
    }
  }

  const reasonLabel = (value: string | null) => ADJUSTMENT_REASONS.find((r) => r.value === value)?.label.replace(' (note required)', '') ?? value

  return (
    <>
      <PageHeader title="Stock adjustment" description="Correct stock after a count or an incident. The average cost does not change; the adjustment is valued at the current average." />
      {done && adjusted && (
        <SuccessAlert>
          Movement #{done.id}: {done.to_location_id ? 'added' : 'removed'} {formatQuantity(done.quantity, adjusted.unit)} of{' '}
          <Link to={`/products/${adjusted.id}`} className="font-semibold underline">
            {adjusted.sku}
          </Link>{' '}
          ({reasonLabel(done.reason)}), valued at {formatMoney(done.total_value)}.
        </SuccessAlert>
      )}
      <MovementFormCard onSubmit={onSubmit} saving={saving} serverError={serverError} submitLabel="Post adjustment">
        <ProductSelect products={products} value={productId} onChange={setProductId} error={errors.product_id} />
        <LocationSelect label="Bin" locations={locations} value={locationId} onChange={setLocationId} error={errors.location_id} />
        <BinStock productId={productId} locationId={locationId} version={done?.id ?? 0} />
        <fieldset className="sm:col-span-2">
          <legend className="mb-1 text-sm font-medium text-slate-700">Direction</legend>
          <div className="flex gap-4 text-sm">
            {(['remove', 'add'] as const).map((d) => (
              <label key={d} className="flex items-center gap-2">
                <input type="radio" name="direction" checked={direction === d} onChange={() => setDirection(d)} className="accent-emerald-700" />
                {d === 'remove' ? 'Remove stock' : 'Add stock'}
              </label>
            ))}
          </div>
        </fieldset>
        <Field label={`Quantity to ${direction}${unit ? ` (${unit})` : ''}`} error={errors.quantity} hint={unit === 'pcs' ? 'Whole numbers only' : 'Up to 4 decimals'}>
          <Input value={quantity} onChange={(e) => setQuantity(e.target.value)} inputMode="decimal" invalid={!!errors.quantity} />
        </Field>
        <Field label="Reason" error={errors.reason}>
          <Select value={reason} onChange={(e) => setReason(e.target.value)} invalid={!!errors.reason}>
            <option value="">Choose a reason…</option>
            {ADJUSTMENT_REASONS.map((r) => (
              <option key={r.value} value={r.value}>
                {r.label}
              </option>
            ))}
          </Select>
        </Field>
        <div className="sm:col-span-2">
          <Field label="Note" error={errors.note} hint={reason === 'other' ? 'Required for Other' : 'Optional'}>
            <Input value={note} onChange={(e) => setNote(e.target.value)} invalid={!!errors.note} />
          </Field>
        </div>
        <Field label="Your name" error={errors.created_by}>
          <Input value={createdBy} onChange={(e) => setCreatedBy(e.target.value)} invalid={!!errors.created_by} />
        </Field>
      </MovementFormCard>
    </>
  )
}
