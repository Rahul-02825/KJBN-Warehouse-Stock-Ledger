import type { Location, Product, ProductBalances, StockCell } from '../api/types'
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
  exclude,
}: {
  label: string
  locations: Location[]
  value: string
  onChange: (v: string) => void
  error?: string
  emptyLabel?: string
  hint?: string
  exclude?: string
}) {
  return (
    <Field label={label} error={error} hint={hint}>
      <Select value={value} onChange={(e) => onChange(e.target.value)} invalid={!!error}>
        <option value="">{emptyLabel}</option>
        {locations.filter((l) => String(l.id) !== exclude).map((l) => (
          <option key={l.id} value={l.id}>
            {binLabel(l)}
          </option>
        ))}
      </Select>
    </Field>
  )
}

export function SourceBinSelect({
  label,
  productId,
  locations,
  value,
  onChange,
  error,
  version,
}: {
  label: string
  productId: string
  locations: Location[]
  value: string
  onChange: (v: string) => void
  error?: string
  version: number
}) {
  const { data, loading } = useApi<ProductBalances>(productId ? `/api/products/${productId}/balances?v=${version}` : null)
  const held = (data?.data ?? []).filter((b) => !/^0+(\.0+)?$/.test(b.quantity))
  const description = (id: number) => locations.find((l) => l.id === id)?.description

  const placeholder = !productId ? 'Choose a product first' : loading && !data ? 'Loading bins…' : held.length === 0 ? 'No bin holds this product yet' : 'Choose a bin…'
  return (
    <Field label={label} error={error} hint={productId && data && held.length > 0 ? 'Only bins that hold this product' : undefined}>
      <Select value={value} onChange={(e) => onChange(e.target.value)} invalid={!!error} disabled={!productId || held.length === 0}>
        <option value="">{placeholder}</option>
        {held.map((b) => (
          <option key={b.location_id} value={b.location_id}>
            {[b.location_code, description(b.location_id), formatQuantity(b.quantity, data?.unit)].filter(Boolean).join(' · ')}
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
