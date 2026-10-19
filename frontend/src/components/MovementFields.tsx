import type { Location, Product, StockCell } from '../api/types'
import { formatMoney, formatQuantity } from '../lib/format'
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
}: {
  label: string
  locations: Location[]
  value: string
  onChange: (v: string) => void
  error?: string
}) {
  return (
    <Field label={label} error={error}>
      <Select value={value} onChange={(e) => onChange(e.target.value)} invalid={!!error}>
        <option value="">Choose a bin…</option>
        {locations.map((l) => (
          <option key={l.id} value={l.id}>
            {l.code}
            {l.description ? ` · ${l.description}` : ''}
          </option>
        ))}
      </Select>
    </Field>
  )
}

// Shows what is in the chosen bin right now, so the user knows the limit before posting.
export function BinStock({ productId, locationId, version }: { productId: string; locationId: string; version: number }) {
  const path = productId && locationId ? `/api/inventory/${productId}/${locationId}?v=${version}` : null
  const { data } = useApi<StockCell>(path)
  if (!path || !data) return null
  return (
    <p className="rounded-md bg-slate-100 px-3 py-2 text-sm text-slate-700 sm:col-span-2">
      In <span className="font-mono">{data.location_code}</span> now: <span className="font-semibold">{formatQuantity(data.quantity, data.unit)}</span>
      <span className="text-slate-400"> · </span>
      Average cost <span className="font-semibold">{formatMoney(data.average_cost)}</span>
    </p>
  )
}
