import { useState, type FormEvent } from 'react'
import { Link } from 'react-router'
import { api } from '../../api/client'
import type { Location, Movement, MovementDetail, Product } from '../../api/types'
import { AsideHint, MovementForm, Step, Wide } from '../../components/MovementForm'
import { BinStock, ProductSelect, SourceBinSelect } from '../../components/MovementFields'
import { PlacementPlan } from '../../components/PlacementPlan'
import { Field, Input, SuccessAlert } from '../../components/ui'
import { loadCreatedBy } from '../../lib/createdBy'
import { formatMoney, formatQuantity } from '../../lib/format'
import { describeEntries } from '../../lib/layout'
import { useMovementSubmit } from '../../lib/useMovementSubmit'
import { usePlacementPreview } from '../../lib/usePlacementPreview'
import { checkDecimal, checkText, onlyErrors } from '../../lib/validate'

export function ShipForm({ products, locations, initialProduct }: { products: Product[]; locations: Location[]; initialProduct: string }) {
  const [productId, setProductId] = useState(initialProduct)
  const [locationId, setLocationId] = useState('')
  const [quantity, setQuantity] = useState('')
  const [reference, setReference] = useState('')
  const [createdBy, setCreatedBy] = useState(loadCreatedBy)
  const [done, setDone] = useState<{ movement: Movement; picked: string } | null>(null)
  const { submit, errors, serverError, saving } = useMovementSubmit('/api/movements/out')

  const unit = products.find((p) => String(p.id) === productId)?.unit ?? ''
  const shipped = done ? products.find((p) => p.id === done.movement.product_id) : undefined
  const previewable = productId !== '' && checkDecimal(quantity, unit) === ''
  const preview = usePlacementPreview(
    '/api/picks/preview',
    previewable ? { product_id: Number(productId), quantity: quantity.trim(), location_id: locationId ? Number(locationId) : undefined } : null,
  )

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    const movement = await submit(
      {
        product_id: Number(productId),
        location_id: locationId ? Number(locationId) : undefined,
        quantity: quantity.trim(),
        reference: reference.trim(),
        created_by: createdBy.trim(),
      },
      onlyErrors({
        product_id: productId ? '' : 'Choose a product',
        quantity: checkDecimal(quantity, unit),
        reference: checkText(reference, 'Reference', 128, false),
        created_by: checkText(createdBy, 'Your name', 128),
      }),
      unit,
    )
    if (movement) {
      const detail = await api.get<MovementDetail>(`/api/movements/${movement.id}`)
      setDone({ movement, picked: describeEntries(detail, locations) })
      setQuantity('')
      setReference('')
    }
  }

  return (
    <>
      {done && shipped && (
        <SuccessAlert>
          Shipped {formatQuantity(done.movement.quantity, shipped.unit)} of{' '}
          <Link to={`/products/${shipped.id}`} className="font-medium underline underline-offset-2">
            {shipped.sku}
          </Link>{' '}
          from {done.picked} at {formatMoney(done.movement.unit_cost)} each, <span className="font-medium">{formatMoney(done.movement.total_value)}</span> in
          total. That cost is fixed on{' '}
          <Link to={`/movements/${done.movement.id}`} className="underline underline-offset-2">
            movement #{done.movement.id}
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
          productId ? (
            <>
              <BinStock productId={productId} locationId={locationId} version={done?.movement.id ?? 0} />
              <PlacementPlan plan={preview.plan} error={preview.error} mode="pick" />
              {!preview.plan && !preview.error && <AsideHint>Enter a quantity to see which bins it will come from.</AsideHint>}
              <AsideHint>The shipment is valued at the average cost when you post it.</AsideHint>
            </>
          ) : (
            <AsideHint>Choose a product to see where it is stored.</AsideHint>
          )
        }
      >
        <Step index="01" title="What is leaving">
          <Wide>
            <ProductSelect
              products={products}
              value={productId}
              onChange={(id) => {
                setProductId(id)
                setLocationId('')
              }}
              error={errors.product_id}
            />
          </Wide>
        </Step>
        <Step index="02" title="From where">
          <SourceBinSelect
            label="Pick from"
            productId={productId}
            locations={locations}
            value={locationId}
            onChange={setLocationId}
            error={errors.location_id}
            version={done?.movement.id ?? 0}
            autoLabel="Let the system choose"
          />
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
