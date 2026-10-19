import { useState, type FormEvent } from 'react'
import { Link } from 'react-router'
import type { Location, Movement, Product } from '../../api/types'
import { AsideHint, MovementForm, Step, Wide } from '../../components/MovementForm'
import { BinStock, LocationSelect, ProductSelect } from '../../components/MovementFields'
import { Field, Input, SuccessAlert } from '../../components/ui'
import { loadCreatedBy } from '../../lib/createdBy'
import { formatMoney, formatQuantity } from '../../lib/format'
import { useMovementSubmit } from '../../lib/useMovementSubmit'
import { checkDecimal, checkText, onlyErrors } from '../../lib/validate'

export function ShipForm({ products, locations, initialProduct }: { products: Product[]; locations: Location[]; initialProduct: string }) {
  const [productId, setProductId] = useState(initialProduct)
  const [locationId, setLocationId] = useState('')
  const [quantity, setQuantity] = useState('')
  const [reference, setReference] = useState('')
  const [createdBy, setCreatedBy] = useState(loadCreatedBy)
  const [done, setDone] = useState<Movement | null>(null)
  const { submit, errors, serverError, saving } = useMovementSubmit('/api/movements/out')

  const unit = products.find((p) => String(p.id) === productId)?.unit ?? ''
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
      {done && shipped && (
        <SuccessAlert>
          Shipped {formatQuantity(done.quantity, shipped.unit)} of{' '}
          <Link to={`/products/${shipped.id}`} className="font-medium underline underline-offset-2">
            {shipped.sku}
          </Link>{' '}
          at {formatMoney(done.unit_cost)} each, <span className="font-medium">{formatMoney(done.total_value)}</span> in total. That cost is fixed on{' '}
          <Link to={`/movements/${done.id}`} className="underline underline-offset-2">
            movement #{done.id}
          </Link>{' '}
          even if the average changes later.
        </SuccessAlert>
      )}
      <MovementForm
        onSubmit={onSubmit}
        saving={saving}
        serverError={serverError}
        submitLabel="Ship stock"
        aside={
          productId && locationId ? (
            <>
              <BinStock productId={productId} locationId={locationId} version={done?.id ?? 0} />
              <AsideHint>The shipment is valued at the average cost when you post it.</AsideHint>
            </>
          ) : (
            <AsideHint>Choose a product and a bin to see how much you can ship.</AsideHint>
          )
        }
      >
        <Step index="01" title="What is leaving">
          <Wide>
            <ProductSelect products={products} value={productId} onChange={setProductId} error={errors.product_id} />
          </Wide>
        </Step>
        <Step index="02" title="From where">
          <LocationSelect label="Bin" locations={locations} value={locationId} onChange={setLocationId} error={errors.location_id} />
        </Step>
        <Step index="03" title="How much">
          <Field label={`Quantity${unit ? ` (${unit})` : ''}`} error={errors.quantity} hint={unit === 'pcs' ? 'Whole numbers only' : 'Up to 4 decimals'}>
            <Input value={quantity} onChange={(e) => setQuantity(e.target.value)} inputMode="decimal" invalid={!!errors.quantity} />
          </Field>
        </Step>
        <Step index="04" title="Paperwork">
          <Field label="Order or reference" error={errors.reference} hint="Optional">
            <Input value={reference} onChange={(e) => setReference(e.target.value)} invalid={!!errors.reference} />
          </Field>
          <Field label="Your name" error={errors.created_by}>
            <Input value={createdBy} onChange={(e) => setCreatedBy(e.target.value)} invalid={!!errors.created_by} />
          </Field>
        </Step>
      </MovementForm>
    </>
  )
}
