import { useState, type FormEvent } from 'react'
import { Link } from 'react-router'
import type { Location, Movement, Product } from '../../api/types'
import { AsideHint, MovementForm, Step, Wide } from '../../components/MovementForm'
import { BinStock, LocationSelect, ProductSelect, SourceBinSelect } from '../../components/MovementFields'
import { Field, Input, SuccessAlert } from '../../components/ui'
import { loadCreatedBy } from '../../lib/createdBy'
import { formatMoney, formatQuantity } from '../../lib/format'
import { useMovementSubmit } from '../../lib/useMovementSubmit'
import { checkDecimal, checkText, onlyErrors } from '../../lib/validate'

export function TransferForm({ products, locations, initialProduct }: { products: Product[]; locations: Location[]; initialProduct: string }) {
  const [productId, setProductId] = useState(initialProduct)
  const [fromId, setFromId] = useState('')
  const [toId, setToId] = useState('')
  const [quantity, setQuantity] = useState('')
  const [reference, setReference] = useState('')
  const [createdBy, setCreatedBy] = useState(loadCreatedBy)
  const [done, setDone] = useState<Movement | null>(null)
  const { submit, errors, serverError, saving } = useMovementSubmit('/api/movements/transfer')

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
        to_location_id: !toId ? 'Choose the bin to move to' : toId === fromId ? 'Choose a different bin from the source' : '',
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
      {done && moved && (
        <SuccessAlert>
          Moved {formatQuantity(done.quantity, moved.unit)} of{' '}
          <Link to={`/products/${moved.id}`} className="font-medium underline underline-offset-2">
            {moved.sku}
          </Link>{' '}
          from <span className="code">{code(done.from_location_id)}</span> to <span className="code">{code(done.to_location_id)}</span>. The average cost stays at{' '}
          {formatMoney(done.unit_cost)}.{' '}
          <Link to={`/movements/${done.id}`} className="underline underline-offset-2">
            Movement #{done.id}
          </Link>
        </SuccessAlert>
      )}
      <MovementForm
        onSubmit={onSubmit}
        saving={saving}
        serverError={serverError}
        submitLabel="Transfer stock"
        aside={
          productId && (fromId || toId) ? (
            <>
              <BinStock productId={productId} locationId={fromId} version={done?.id ?? 0} heading="Moving from" />
              <BinStock productId={productId} locationId={toId} version={done?.id ?? 0} heading="Moving to" />
              <AsideHint>Both bins change together, or neither does.</AsideHint>
            </>
          ) : (
            <AsideHint>Choose a product and bins to compare what each holds now.</AsideHint>
          )
        }
      >
        <Step index="01" title="What moves">
          <Wide>
            <ProductSelect
              products={products}
              value={productId}
              onChange={(id) => {
                setProductId(id)
                setFromId('')
              }}
              error={errors.product_id}
            />
          </Wide>
        </Step>
        <Step index="02" title="Between bins">
          <SourceBinSelect
            label="From"
            productId={productId}
            locations={locations}
            value={fromId}
            onChange={setFromId}
            error={errors.from_location_id}
            version={done?.id ?? 0}
          />
          <LocationSelect label="To" locations={locations} value={toId} onChange={setToId} error={errors.to_location_id} exclude={fromId} />
        </Step>
        <Step index="03" title="How much">
          <Field label={`Quantity${unit ? ` (${unit})` : ''}`} error={errors.quantity} hint={unit === 'pcs' ? 'Whole numbers only' : 'Up to 4 decimals'}>
            <Input value={quantity} onChange={(e) => setQuantity(e.target.value)} inputMode="decimal" invalid={!!errors.quantity} />
          </Field>
        </Step>
        <Step index="04" title="Paperwork">
          <Field label="Reference" error={errors.reference} hint="Optional">
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
