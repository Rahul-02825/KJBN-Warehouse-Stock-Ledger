import { useState, type FormEvent } from 'react'
import { Link, useSearchParams } from 'react-router'
import type { Movement } from '../api/types'
import { BinStock, LocationSelect, ProductSelect } from '../components/MovementFields'
import { MovementFormCard } from '../components/MovementForm'
import { ErrorAlert, Field, Input, PageHeader, Spinner, SuccessAlert } from '../components/ui'
import { loadCreatedBy } from '../lib/createdBy'
import { formatMoney, formatQuantity } from '../lib/format'
import { useMovementSubmit } from '../lib/useMovementSubmit'
import { useOptions } from '../lib/useOptions'
import { checkDecimal, checkText, onlyErrors } from '../lib/validate'

export function Transfer() {
  const [params] = useSearchParams()
  const { products, locations, error } = useOptions()
  const [productId, setProductId] = useState(params.get('product') ?? '')
  const [fromId, setFromId] = useState('')
  const [toId, setToId] = useState('')
  const [quantity, setQuantity] = useState('')
  const [reference, setReference] = useState('')
  const [createdBy, setCreatedBy] = useState(loadCreatedBy)
  const [done, setDone] = useState<Movement | null>(null)
  const { submit, errors, serverError, saving } = useMovementSubmit('/api/movements/transfer')

  if (error) return <ErrorAlert error={error} />
  if (!products || !locations) return <Spinner />
  const unit = products.find((p) => String(p.id) === productId)?.unit ?? ''
  const moved = done ? products.find((p) => p.id === done.product_id) : undefined
  const code = (id: number | null) => locations.find((l) => l.id === id)?.code ?? '?'

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    const movement = await submit(
      {
        product_id: Number(productId),
        from_location_id: Number(fromId),
        to_location_id: Number(toId),
        quantity: quantity.trim(),
        reference: reference.trim(),
        created_by: createdBy.trim(),
      },
      onlyErrors({
        product_id: productId ? '' : 'Choose a product',
        from_location_id: fromId ? '' : 'Choose the bin to move from',
        to_location_id: !toId ? 'Choose the bin to move to' : toId === fromId ? 'Must be different from the source bin' : '',
        quantity: checkDecimal(quantity, unit),
        reference: checkText(reference, 'Reference', 128, false),
        created_by: checkText(createdBy, 'Your name', 128),
      }),
      unit,
    )
    if (movement) {
      setDone(movement)
      setQuantity('')
      setReference('')
    }
  }

  return (
    <>
      <PageHeader title="Transfer stock" description="Move stock between two bins. It is one movement with two ledger entries, and it happens fully or not at all." />
      {done && moved && (
        <SuccessAlert>
          Movement #{done.id}: moved {formatQuantity(done.quantity, moved.unit)} of{' '}
          <Link to={`/products/${moved.id}`} className="font-semibold underline">
            {moved.sku}
          </Link>{' '}
          from <span className="font-mono">{code(done.from_location_id)}</span> to <span className="font-mono">{code(done.to_location_id)}</span>. Average cost
          unchanged at {formatMoney(done.unit_cost)}.
        </SuccessAlert>
      )}
      <MovementFormCard onSubmit={onSubmit} saving={saving} serverError={serverError} submitLabel="Transfer stock">
        <div className="sm:col-span-2">
          <ProductSelect products={products} value={productId} onChange={setProductId} error={errors.product_id} />
        </div>
        <LocationSelect label="From bin" locations={locations} value={fromId} onChange={setFromId} error={errors.from_location_id} />
        <LocationSelect label="To bin" locations={locations} value={toId} onChange={setToId} error={errors.to_location_id} />
        <BinStock productId={productId} locationId={fromId} version={done?.id ?? 0} className="" />
        <BinStock productId={productId} locationId={toId} version={done?.id ?? 0} className="" />
        <Field label={`Quantity${unit ? ` (${unit})` : ''}`} error={errors.quantity} hint={unit === 'pcs' ? 'Whole numbers only' : 'Up to 4 decimals'}>
          <Input value={quantity} onChange={(e) => setQuantity(e.target.value)} inputMode="decimal" invalid={!!errors.quantity} />
        </Field>
        <Field label="Reference" error={errors.reference} hint="Optional">
          <Input value={reference} onChange={(e) => setReference(e.target.value)} invalid={!!errors.reference} />
        </Field>
        <Field label="Your name" error={errors.created_by}>
          <Input value={createdBy} onChange={(e) => setCreatedBy(e.target.value)} invalid={!!errors.created_by} />
        </Field>
      </MovementFormCard>
    </>
  )
}
