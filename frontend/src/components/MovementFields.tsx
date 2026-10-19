import type { Location, Product, StockCell } from '../api/types'
import { formatMoney, formatQuantity } from '../lib/format'
import { binLabel } from '../lib/layout'
import { useApi } from '../lib/useApi'
import { Field, Select } from './ui'

export function ProductSelect({ products, value, onChange, error }: { products: Product[]; value: string; onChange: (v: string) => void; error?: string }) {
  return (
    <Field label="Product" error={error}>
      <Select value={value} onChange={(e) => onChange(e.target.value)} invalid={!!error}>
        <option value="">Choose a product…</option>
        {products.map((p) => (
          <option key={p.id} value={p.id}>
            {p.sku} · {p.name} ({p.unit})
          </option>
        ))}
      </Select>
    </Field>
  )
}

export function LocationSelect({
  label,
  locations,
  value,
  onChange,
  error,
  emptyLabel = 'Choose a bin…',
  hint,
}: {
  label: string
  locations: Location[]
  value: string
  onChange: (v: string) => void
  error?: string
  emptyLabel?: string
  hint?: string
}) {
  return (
    <Field label={label} error={error} hint={hint}>
      <Select value={value} onChange={(e) => onChange(e.target.value)} invalid={!!error}>
        <option value="">{emptyLabel}</option>
        {locations.map((l) => (
          <option key={l.id} value={l.id}>
            {binLabel(l)}
          </option>
        ))}
      </Select>
    </Field>
  )
}

export function BinStock({ productId, locationId, version, heading }: { productId: string; locationId: string; version: number; heading?: string }) {
  const path = productId && locationId ? `/api/inventory/${productId}/${locationId}?v=${version}` : null
  const { data } = useApi<StockCell>(path)
  if (!path || !data) return null
  return (
    <dl className="space-y-1.5">
      {heading && <p className="text-[12.5px] text-muted">{heading}</p>}
      <div className="flex justify-between gap-3">
        <dt className="text-muted">
          In <span className="code text-ink">{data.location_code}</span> now
        </dt>
        <dd className="num font-medium">{formatQuantity(data.quantity, data.unit)}</dd>
      </div>
      <div className="flex justify-between gap-3">
        <dt className="text-muted">Average cost</dt>
        <dd className="num font-medium">{formatMoney(data.average_cost)}</dd>
      </div>
    </dl>
  )
}
