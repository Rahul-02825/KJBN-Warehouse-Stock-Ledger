import { useState, type FormEvent } from 'react'
import { Link, useSearchParams } from 'react-router'
import { api } from '../api/client'
import type { Movement, Product } from '../api/types'
import { BinStock, LocationSelect, ProductSelect } from '../components/MovementFields'
import { MovementFormCard } from '../components/MovementForm'
import { ErrorAlert, Field, Input, PageHeader, Spinner, SuccessAlert } from '../components/ui'
import { loadCreatedBy } from '../lib/createdBy'
import { formatMoney, formatQuantity } from '../lib/format'
import { useMovementSubmit } from '../lib/useMovementSubmit'
import { useOptions } from '../lib/useOptions'
import { checkDecimal, checkText, onlyErrors } from '../lib/validate'

export function Receive() {
  const [params] = useSearchParams()
  const { products, locations, error } = useOptions()
  const [productId, setProductId] = useState(params.get('product') ?? '')
  const [locationId, setLocationId] = useState('')
  const [quantity, setQuantity] = useState('')
  const [unitCost, setUnitCost] = useState('')
  const [reference, setReference] = useState('')
  const [createdBy, setCreatedBy] = useState(loadCreatedBy)
  const [done, setDone] = useState<{ movement: Movement; product: Product } | null>(null)
  const { submit, errors, serverError, saving } = useMovementSubmit('/api/movements/in')

  if (error) return <ErrorAlert error={error} />
  if (!products || !locations) return <Spinner />
  const product = products.find((p) => String(p.id) === productId)
  const unit = product?.unit ?? ''

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    const movement = await submit(
      { product_id: Number(productId), location_id: Number(locationId), quantity: quantity.trim(), unit_cost: unitCost.trim(), reference: reference.trim(), created_by: createdBy.trim() },
      onlyErrors({
        product_id: productId ? '' : 'Choose a product',
        location_id: locationId ? '' : 'Choose a bin',
        quantity: checkDecimal(quantity, unit),
        unit_cost: checkDecimal(unitCost, ''),
        reference: checkText(reference, 'Reference', 128, false),
        created_by: checkText(createdBy, 'Your name', 128),
      }),
      unit,
    )
    if (movement) {
      // Read the product again to show the new weighted-average cost the server calculated.
      const updated = await api.get<Product>(`/api/products/${movement.product_id}`)
      setDone({ movement, product: updated })
      setQuantity('')
      setUnitCost('')
      setReference('')
    }
  }

  return (
    <>
      <PageHeader title="Receive stock" description="Record goods arriving into a bin. Each receipt updates the product's weighted-average cost." />
      {done && (
        <SuccessAlert>
          Movement #{done.movement.id}: received {formatQuantity(done.movement.quantity, done.product.unit)} of{' '}
          <Link to={`/products/${done.product.id}`} className="font-semibold underline">
            {done.product.sku}
          </Link>{' '}
          at {formatMoney(done.movement.unit_cost)} each. New average cost: <span className="font-semibold">{formatMoney(done.product.average_cost)}</span>.
        </SuccessAlert>
      )}
      <MovementFormCard onSubmit={onSubmit} saving={saving} serverError={serverError} submitLabel="Receive stock">
        <ProductSelect products={products} value={productId} onChange={setProductId} error={errors.product_id} />
        <LocationSelect label="Into bin" locations={locations} value={locationId} onChange={setLocationId} error={errors.location_id} />
        <BinStock productId={productId} locationId={locationId} version={done?.movement.id ?? 0} />
        <Field label={`Quantity${unit ? ` (${unit})` : ''}`} error={errors.quantity} hint={unit === 'pcs' ? 'Whole numbers only' : 'Up to 4 decimals'}>
          <Input value={quantity} onChange={(e) => setQuantity(e.target.value)} inputMode="decimal" invalid={!!errors.quantity} />
        </Field>
        <Field label="Unit cost (₹)" error={errors.unit_cost} hint="Must be greater than zero">
          <Input value={unitCost} onChange={(e) => setUnitCost(e.target.value)} inputMode="decimal" invalid={!!errors.unit_cost} />
        </Field>
        <Field label="Reference" error={errors.reference} hint="Invoice number, optional">
          <Input value={reference} onChange={(e) => setReference(e.target.value)} invalid={!!errors.reference} />
        </Field>
        <Field label="Your name" error={errors.created_by}>
          <Input value={createdBy} onChange={(e) => setCreatedBy(e.target.value)} invalid={!!errors.created_by} />
        </Field>
      </MovementFormCard>
    </>
  )
}
