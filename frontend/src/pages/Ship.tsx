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

export function Ship() {
  const [params] = useSearchParams()
  const { products, locations, error } = useOptions()
  const [productId, setProductId] = useState(params.get('product') ?? '')
  const [locationId, setLocationId] = useState('')
  const [quantity, setQuantity] = useState('')
  const [reference, setReference] = useState('')
  const [createdBy, setCreatedBy] = useState(loadCreatedBy)
  const [done, setDone] = useState<Movement | null>(null)
  const { submit, errors, serverError, saving } = useMovementSubmit('/api/movements/out')

  if (error) return <ErrorAlert error={error} />
  if (!products || !locations) return <Spinner />
  const product = products.find((p) => String(p.id) === productId)
  const unit = product?.unit ?? ''
  const shipped = done ? products.find((p) => p.id === done.product_id) : undefined

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    const movement = await submit(
      { product_id: Number(productId), location_id: Number(locationId), quantity: quantity.trim(), reference: reference.trim(), created_by: createdBy.trim() },
      onlyErrors({
        product_id: productId ? '' : 'Choose a product',
        location_id: locationId ? '' : 'Choose a bin',
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
      <PageHeader title="Ship stock" description="Record goods leaving a bin. The shipment is valued at the average cost right now, and that cost is stored on the movement." />
      {done && shipped && (
        <SuccessAlert>
          Movement #{done.id}: shipped {formatQuantity(done.quantity, shipped.unit)} of{' '}
          <Link to={`/products/${shipped.id}`} className="font-semibold underline">
            {shipped.sku}
          </Link>{' '}
          at {formatMoney(done.unit_cost)} each, total <span className="font-semibold">{formatMoney(done.total_value)}</span>. This cost is frozen on the
          movement and will not change if the average changes later.
        </SuccessAlert>
      )}
      <MovementFormCard onSubmit={onSubmit} saving={saving} serverError={serverError} submitLabel="Ship stock">
        <ProductSelect products={products} value={productId} onChange={setProductId} error={errors.product_id} />
        <LocationSelect label="From bin" locations={locations} value={locationId} onChange={setLocationId} error={errors.location_id} />
        <BinStock productId={productId} locationId={locationId} version={done?.id ?? 0} />
        <Field label={`Quantity${unit ? ` (${unit})` : ''}`} error={errors.quantity} hint={unit === 'pcs' ? 'Whole numbers only' : 'Up to 4 decimals'}>
          <Input value={quantity} onChange={(e) => setQuantity(e.target.value)} inputMode="decimal" invalid={!!errors.quantity} />
        </Field>
        <Field label="Reference" error={errors.reference} hint="Order number, optional">
          <Input value={reference} onChange={(e) => setReference(e.target.value)} invalid={!!errors.reference} />
        </Field>
        <Field label="Your name" error={errors.created_by}>
          <Input value={createdBy} onChange={(e) => setCreatedBy(e.target.value)} invalid={!!errors.created_by} />
        </Field>
      </MovementFormCard>
    </>
  )
}
