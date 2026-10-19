import { useState, type ReactNode } from 'react'
import { Link } from 'react-router'
import type { ListResponse, LowStockRow, StockCell, ValuationResponse } from '../api/types'
import { EmptyState, ErrorAlert, Input, PageHeader, Spinner } from '../components/ui'
import { isZero, toGrid } from '../lib/grid'
import { formatDecimal, formatMoney } from '../lib/format'
import { useApi } from '../lib/useApi'

export function StockOverview() {
  const cells = useApi<ListResponse<StockCell>>('/api/inventory')
  const valuation = useApi<ValuationResponse>('/api/reports/valuation')
  const lowStock = useApi<ListResponse<LowStockRow>>('/api/reports/low-stock')
  const [filter, setFilter] = useState('')
  const [onlyLow, setOnlyLow] = useState(false)

  const error = cells.error ?? valuation.error ?? lowStock.error
  const reload = () => {
    cells.reload()
    valuation.reload()
    lowStock.reload()
  }

  if (error) return <Frame><ErrorAlert error={error} onRetry={reload} /></Frame>
  if (!cells.data || !valuation.data || !lowStock.data) return <Frame><Spinner /></Frame>

  const { locations, rows } = toGrid(cells.data.data)
  const totals = new Map(valuation.data.data.map((v) => [v.product_id, v]))
  const low = new Set(lowStock.data.data.map((l) => l.product_id))

  const needle = filter.trim().toLowerCase()
  const visible = rows.filter(
    (r) => (!onlyLow || low.has(r.productId)) && (needle === '' || r.sku.toLowerCase().includes(needle) || r.name.toLowerCase().includes(needle)),
  )

  if (rows.length === 0 || locations.length === 0) {
    return (
      <Frame>
        <EmptyState title="Nothing to show yet">Create at least one product and one bin, then receive stock.</EmptyState>
      </Frame>
    )
  }

  return (
    <Frame>
      <div className="mb-4 flex flex-wrap items-center gap-4">
        <Input className="max-w-xs" placeholder="Filter by SKU or name" value={filter} onChange={(e) => setFilter(e.target.value)} aria-label="Filter products" />
        <label className="flex items-center gap-2 text-sm text-slate-700">
          <input type="checkbox" checked={onlyLow} onChange={(e) => setOnlyLow(e.target.checked)} className="h-4 w-4 accent-emerald-700" />
          Only low stock ({low.size})
        </label>
        <span className="text-sm text-slate-500">
          Total value: <span className="font-semibold text-slate-900">{formatMoney(valuation.data.total_value)}</span>
        </span>
      </div>

      {visible.length === 0 ? (
        <EmptyState title="No products match this filter" />
      ) : (
        <div className="overflow-x-auto rounded-lg border border-slate-200 bg-white shadow-sm">
          <table className="min-w-full text-sm">
            <thead className="bg-slate-50">
              <tr>
                <th className="sticky left-0 z-10 bg-slate-50 px-4 py-2 text-left font-medium text-slate-600">Product</th>
                {locations.map((l) => (
                  <th key={l.id} className="whitespace-nowrap px-4 py-2 text-right font-mono font-medium text-slate-600">
                    {l.code}
                  </th>
                ))}
                <th className="whitespace-nowrap border-l border-slate-200 px-4 py-2 text-right font-medium text-slate-600">Total</th>
                <th className="whitespace-nowrap px-4 py-2 text-right font-medium text-slate-600">Avg cost</th>
                <th className="whitespace-nowrap px-4 py-2 text-right font-medium text-slate-600">Value</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {visible.map((r) => {
                const t = totals.get(r.productId)
                const isLow = low.has(r.productId)
                return (
                  <tr key={r.productId} className={isLow ? 'bg-amber-50' : 'hover:bg-slate-50'}>
                    <td className={`sticky left-0 z-10 min-w-40 px-4 py-2 ${isLow ? 'bg-amber-50' : 'bg-white'}`}>
                      <div className="flex items-center gap-2">
                        <Link to={`/products/${r.productId}`} className="whitespace-nowrap font-mono text-emerald-700 hover:underline">
                          {r.sku}
                        </Link>
                        {isLow && <span className="rounded bg-amber-200 px-1.5 py-0.5 text-xs font-medium text-amber-900">Low</span>}
                      </div>
                      <div className="text-xs text-slate-500">
                        {r.name} · {r.unit}
                      </div>
                    </td>
                    {locations.map((l) => (
                      <td key={l.id} className={`px-4 py-2 text-right tabular-nums ${isZero(r.quantities[l.id]) ? 'text-slate-300' : ''}`}>
                        {isZero(r.quantities[l.id]) ? '–' : formatDecimal(r.quantities[l.id])}
                      </td>
                    ))}
                    <td className="border-l border-slate-200 px-4 py-2 text-right font-semibold tabular-nums">{formatDecimal(t?.total_quantity)}</td>
                    <td className="px-4 py-2 text-right tabular-nums">{formatMoney(t?.average_cost)}</td>
                    <td className="px-4 py-2 text-right tabular-nums">{formatMoney(t?.total_value)}</td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}
    </Frame>
  )
}

function Frame({ children }: { children: ReactNode }) {
  return (
    <>
      <PageHeader title="Stock overview" description="Quantity of every product in every bin. Rows marked Low are at or below their reorder level." />
      {children}
    </>
  )
}
