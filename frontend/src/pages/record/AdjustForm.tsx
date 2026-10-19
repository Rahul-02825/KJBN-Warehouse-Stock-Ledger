import { useState, type FormEvent } from 'react'
import { Link } from 'react-router'
import { ADJUSTMENT_REASONS, type Location, type Movement, type Product } from '../../api/types'
import { AsideHint, MovementForm, Step, Wide } from '../../components/MovementForm'
import { BinStock, LocationSelect, ProductSelect } from '../../components/MovementFields'
import { Field, Input, Select, SuccessAlert } from '../../components/ui'
import { loadCreatedBy } from '../../lib/createdBy'
import { formatMoney, formatQuantity } from '../../lib/format'
import { useMovementSubmit } from '../../lib/useMovementSubmit'
import { checkDecimal, checkText, onlyErrors } from '../../lib/validate'

export function AdjustForm({ products, locations, initialProduct }: { products: Product[]; locations: Location[]; initialProduct: string }) {
  const [productId, setProductId] = useState(initialProduct)
  const [locationId, setLocationId] = useState('')
  const [direction, setDirection] = useState<'add' | 'remove'>('remove')
  const [quantity, setQuantity] = useState('')
  const [reason, setReason] = useState('')
  const [note, setNote] = useState('')
  const [createdBy, setCreatedBy] = useState(loadCreatedBy)
  const [done, setDone] = useState<Movement | null>(null)
  const { submit, errors, serverError, saving } = useMovementSubmit('/api/movements/adjustment')

  const unit = products.find((p) => String(p.id) === productId)?.unit ?? ''
  const adjusted = done ? products.find((p) => p.id === done.product_id) : undefined

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    const signed = direction === 'remove' ? `-${quantity.trim()}` : quantity.trim()
    const movement = await submit(
      { product_id: Number(productId), location_id: Number(locationId), quantity: signed, reason, note: note.trim(), created_by: createdBy.trim() },
      onlyErrors({
        product_id: productId ? '' : 'Choose a product',
        location_id: locationId ? '' : 'Choose a bin',
        quantity: checkDecimal(quantity, unit),
        reason: reason ? '' : 'Choose a reason',
        note: reason === 'other' && note.trim() === '' ? 'Add a note when the reason is Other' : checkText(note, 'Note', 1000, false),
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

  const reasonLabel = (value: string | null) => ADJUSTMENT_REASONS.find((r) => r.value === value)?.label.replace(' (note required)', '').toLowerCase() ?? value

  return (
    <>
      {done && adjusted && (
        <SuccessAlert>
          {done.to_location_id ? 'Added' : 'Removed'} {formatQuantity(done.quantity, adjusted.unit)} of{' '}
          <Link to={`/products/${adjusted.id}`} className="font-medium underline underline-offset-2">
            {adjusted.sku}
          </Link>{' '}
          for {reasonLabel(done.reason)}, valued at {formatMoney(done.total_value)}.{' '}
          <Link to={`/movements/${done.id}`} className="underline underline-offset-2">
            Movement #{done.id}
          </Link>
        </SuccessAlert>
      )}
      <MovementForm
        onSubmit={onSubmit}
        saving={saving}
        serverError={serverError}
        submitLabel="Post adjustment"
        aside={
          productId && locationId ? (
            <>
              <BinStock productId={productId} locationId={locationId} version={done?.id ?? 0} />
              <AsideHint>Adjustments are valued at the current average and never change it.</AsideHint>
            </>
          ) : (
            <AsideHint>Choose a product and a bin to see what the system thinks is there.</AsideHint>
          )
        }
      >
        <Step index="01" title="Which stock">
          <ProductSelect products={products} value={productId} onChange={setProductId} error={errors.product_id} />
          <LocationSelect label="Bin" locations={locations} value={locationId} onChange={setLocationId} error={errors.location_id} />
        </Step>
        <Step index="02" title="The correction">
          <Wide>
            <div className="inline-flex rounded-btn border border-line p-0.5" role="radiogroup" aria-label="Direction">
              {(['remove', 'add'] as const).map((d) => (
                <button
                  key={d}
                  type="button"
                  role="radio"
                  aria-checked={direction === d}
                  onClick={() => setDirection(d)}
                  className={`rounded-[2px] px-4 py-1.5 text-sm transition-colors duration-200 ${direction === d ? 'bg-petrol text-on-petrol' : 'text-muted hover:text-ink'}`}
                >
                  {d === 'remove' ? 'Remove stock' : 'Add stock'}
                </button>
              ))}
            </div>
          </Wide>
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
          <Wide>
            <Field label="Note" error={errors.note} hint={reason === 'other' ? 'Required for Other' : 'Optional'}>
              <Input value={note} onChange={(e) => setNote(e.target.value)} invalid={!!errors.note} />
            </Field>
          </Wide>
        </Step>
        <Step index="03" title="Who">
          <Field label="Your name" error={errors.created_by}>
            <Input value={createdBy} onChange={(e) => setCreatedBy(e.target.value)} invalid={!!errors.created_by} />
          </Field>
        </Step>
      </MovementForm>
    </>
  )
}
