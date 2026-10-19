import { useState, type FormEvent } from 'react'
import { Link } from 'react-router'
import { api } from '../../api/client'
import type { Location, Movement, MovementDetail, Product } from '../../api/types'
import { AsideHint, MovementForm, Step, Wide } from '../../components/MovementForm'
import { BinStock, LocationSelect, ProductSelect } from '../../components/MovementFields'
import { PlacementPlan } from '../../components/PlacementPlan'
import { Field, Input, SuccessAlert } from '../../components/ui'
import { loadCreatedBy } from '../../lib/createdBy'
import { formatMoney, formatQuantity } from '../../lib/format'
import { useMovementSubmit } from '../../lib/useMovementSubmit'
import { usePlacementPreview } from '../../lib/usePlacementPreview'
import { checkDecimal, checkText, onlyErrors } from '../../lib/validate'

export function ReceiveForm({ products, locations, initialProduct }: { products: Product[]; locations: Location[]; initialProduct: string }) {
  const [productId, setProductId] = useState(initialProduct)
  const [locationId, setLocationId] = useState('')
  const [quantity, setQuantity] = useState('')
  const [unitCost, setUnitCost] = useState('')
  const [reference, setReference] = useState('')
  const [createdBy, setCreatedBy] = useState(loadCreatedBy)
  const [done, setDone] = useState<{ movement: Movement; product: Product; placed: string } | null>(null)
  const { submit, errors, serverError, saving } = useMovementSubmit('/api/movements/in')

  const unit = products.find((p) => String(p.id) === productId)?.unit ?? ''
  const autoPlaced = unit === 'pcs'
  const previewable = autoPlaced && checkDecimal(quantity, unit) === ''
  const preview = usePlacementPreview(
    previewable ? { product_id: Number(productId), quantity: quantity.trim(), location_id: locationId ? Number(locationId) : undefined } : null,
  )

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    const movement = await submit(
      {
        product_id: Number(productId),
        location_id: locationId ? Number(locationId) : undefined,
        quantity: quantity.trim(),
        unit_cost: unitCost.trim(),
        reference: reference.trim(),
        created_by: createdBy.trim(),
      },
      onlyErrors({
        product_id: productId ? '' : 'Choose a product',
        location_id: locationId || autoPlaced ? '' : 'Choose a bin',
        quantity: checkDecimal(quantity, unit),
        unit_cost: checkDecimal(unitCost, ''),
        reference: checkText(reference, 'Reference', 128, false),
        created_by: checkText(createdBy, 'Your name', 128),
      }),
      unit,
    )
    if (movement) {
      const [updated, detail] = await Promise.all([
        api.get<Product>(`/api/products/${movement.product_id}`),
        api.get<MovementDetail>(`/api/movements/${movement.id}`),
      ])
      setDone({ movement, product: updated, placed: describePlacement(detail, locations) })
      setQuantity('')
      setUnitCost('')
      setReference('')
    }
  }

  return (
    <>
      {done && (
        <SuccessAlert>
          Received {formatQuantity(done.movement.quantity, done.product.unit)} of{' '}
          <Link to={`/products/${done.product.id}`} className="font-medium underline underline-offset-2">
            {done.product.sku}
          </Link>{' '}
          at {formatMoney(done.movement.unit_cost)} each, placed in {done.placed}. The average cost is now{' '}
          <span className="font-medium">{formatMoney(done.product.average_cost)}</span>.{' '}
          <Link to={`/movements/${done.movement.id}`} className="underline underline-offset-2">
            Movement #{done.movement.id}
          </Link>
        </SuccessAlert>
      )}
      <MovementForm
        onSubmit={onSubmit}
        saving={saving}
        serverError={serverError}
        submitLabel="Receive stock"
        aside={
          productId ? (
            <>
              <BinStock productId={productId} locationId={locationId} version={done?.movement.id ?? 0} />
              <PlacementPlan plan={preview.plan} error={preview.error} />
              {!locationId && !preview.plan && !preview.error && (
                <AsideHint>{autoPlaced ? 'Enter a quantity to see where the items will go.' : 'Choose a bin to see what is in it now.'}</AsideHint>
              )}
            </>
          ) : (
            <AsideHint>Choose a product to see where it will be stored and what the bin holds now.</AsideHint>
          )
        }
      >
        <Step index="01" title="What arrived">
          <Wide>
            <ProductSelect products={products} value={productId} onChange={setProductId} error={errors.product_id} />
          </Wide>
        </Step>
        <Step index="02" title="Where it goes">
          <LocationSelect
            label={autoPlaced ? 'Starting bin' : 'Bin'}
            locations={locations}
            value={locationId}
            onChange={setLocationId}
            error={errors.location_id}
            emptyLabel={autoPlaced ? 'Let the system choose' : 'Choose a bin…'}
            hint={autoPlaced ? 'Anything that does not fit goes to the nearest bins with space' : undefined}
          />
        </Step>
        <Step index="03" title="How much">
          <Field label={`Quantity${unit ? ` (${unit})` : ''}`} error={errors.quantity} hint={unit === 'pcs' ? 'Whole numbers only' : 'Up to 4 decimals'}>
            <Input value={quantity} onChange={(e) => setQuantity(e.target.value)} inputMode="decimal" invalid={!!errors.quantity} />
          </Field>
          <Field label="Unit cost (₹)" error={errors.unit_cost} hint="Updates the weighted-average cost">
            <Input value={unitCost} onChange={(e) => setUnitCost(e.target.value)} inputMode="decimal" invalid={!!errors.unit_cost} />
          </Field>
        </Step>
        <Step index="04" title="Paperwork">
          <Field label="Invoice or reference" error={errors.reference} hint="Optional">
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

function describePlacement(detail: MovementDetail, locations: Location[]): string {
  const code = (id: number) => locations.find((l) => l.id === id)?.code ?? `#${id}`
  return (detail.entries ?? []).map((e) => `${code(e.location_id)} (${e.quantity_delta})`).join(', ')
}
