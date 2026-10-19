import { useState } from 'react'
import { Link } from 'react-router'
import type { ListResponse, LowStockRow, StockCell, ValuationResponse } from '../api/types'
import { Badge, ButtonLink, Checkbox, EmptyState, ErrorAlert, Input, PageHeader, Spinner } from '../components/ui'
import { formatDecimal, formatMoney } from '../lib/format'
import { isZero, toGrid } from '../lib/grid'
import { useApi } from '../lib/useApi'

const pinned = {
  product: 'sticky left-0 z-10 w-36 min-w-36 border-r border-line sm:w-48 sm:min-w-48',
  total: 'w-24 min-w-24 border-l border-line md:sticky md:right-[256px] md:z-10',
  average: 'w-28 min-w-28 md:sticky md:right-[144px] md:z-10',
  value: 'w-36 min-w-36 md:sticky md:right-0 md:z-10',
}

export function StockOverview() {
  const cells = useApi<ListResponse<StockCell>>('/api/inventory')
  const valuation = useApi<ValuationResponse>('/api/reports/valuation')
  const lowStock = useApi<ListResponse<LowStockRow>>('/api/reports/low-stock')
  const [filter, setFilter] = useState('')
  const [onlyLow, setOnlyLow] = useState(false)

  const error = cells.error ?? valuation.error ?? lowStock.error
  const reload = () => [cells, valuation, lowStock].forEach((r) => r.reload())
  const header = (
    <PageHeader
      eyebrow="Stock"
      title="Every product in every bin"
      description="Quantities come from the ledger. Rows marked low are at or below their reorder level across all bins."
    />
  )

  if (error) return <>{header}<ErrorAlert error={error} onRetry={reload} /></>
  if (!cells.data || !valuation.data || !lowStock.data) return <>{header}<Spinner /></>

  const { locations, rows } = toGrid(cells.data.data)
  const totals = new Map(valuation.data.data.map((v) => [v.product_id, v]))
  const low = new Set(lowStock.data.data.map((l) => l.product_id))
  const needle = filter.trim().toLowerCase()
  const visible = rows.filter(
    (r) => (!onlyLow || low.has(r.productId)) && (needle === '' || r.sku.toLowerCase().includes(needle) || r.name.toLowerCase().includes(needle)),
  )

  if (rows.length === 0 || locations.length === 0) {
    return (
      <>
        {header}
        <EmptyState title="No stock yet" action={<ButtonLink to="/products">Set up products and bins</ButtonLink>}>
          Add at least one product and one bin, then receive stock.
        </EmptyState>
      </>
    )
  }

  return (
    <>
      {header}
      <div className="mb-5 flex flex-wrap items-center gap-x-6 gap-y-3">
        <Input className="max-w-xs" placeholder="Filter by SKU or name" value={filter} onChange={(e) => setFilter(e.target.value)} aria-label="Filter products" />
        <Checkbox label={`Only low stock (${low.size})`} checked={onlyLow} onChange={(e) => setOnlyLow(e.target.checked)} />
        <p className="ml-auto text-sm text-muted">
          {locations.length > 6 && <span className="mr-4">Scroll sideways to see all {locations.length} bins.</span>}
          Valued at <span className="num font-medium text-ink">{formatMoney(valuation.data.total_value)}</span>
        </p>
      </div>

      {visible.length === 0 ? (
        <EmptyState title="No products match">Clear the filter to see every product.</EmptyState>
      ) : (
        <div className="overflow-x-auto rounded-panel border border-line">
          <table className="min-w-full border-separate border-spacing-0 text-[14px]">
            <thead>
              <tr className="font-mono text-[11px] uppercase tracking-[0.14em] text-muted">
                <th className={`${pinned.product} bg-paper-2 px-4 py-2.5 text-left font-normal`}>Product</th>
                {locations.map((l) => (
                  <th key={l.id} className="min-w-24 whitespace-nowrap bg-paper-2 px-4 py-2.5 text-right font-normal">
                    {l.code}
                  </th>
                ))}
                <th className={`${pinned.total} bg-paper-2 px-4 py-2.5 text-right font-normal`}>Total</th>
                <th className={`${pinned.average} bg-paper-2 px-4 py-2.5 text-right font-normal`}>Avg cost</th>
                <th className={`${pinned.value} bg-paper-2 px-4 py-2.5 text-right font-normal`}>Value</th>
              </tr>
            </thead>
            <tbody>
              {visible.map((r) => {
                const t = totals.get(r.productId)
                const isLow = low.has(r.productId)
                const cell = 'border-t border-line bg-paper px-4 py-3 transition-colors duration-200 group-hover:bg-paper-2'
                return (
                  <tr key={r.productId} className="group">
                    <td className={`${pinned.product} ${cell}`}>
                      <div className="flex items-center gap-2">
                        <Link to={`/products/${r.productId}`} className="link-quiet code whitespace-nowrap">
                          {r.sku}
                        </Link>
                        {isLow && <Badge tone="warning">Low</Badge>}
                      </div>
                      <p className="mt-0.5 text-[12.5px] text-muted">
                        {r.name}, {r.unit}
                      </p>
                    </td>
                    {locations.map((l) => (
                      <td key={l.id} className={`${cell} num text-right ${isZero(r.quantities[l.id]) ? 'text-line' : ''}`}>
                        {isZero(r.quantities[l.id]) ? '–' : formatDecimal(r.quantities[l.id])}
                      </td>
                    ))}
                    <td className={`${pinned.total} ${cell} num text-right font-medium ${isLow ? 'text-saffron-ink' : ''}`}>{formatDecimal(t?.total_quantity)}</td>
                    <td className={`${pinned.average} ${cell} num text-right text-muted`}>{formatMoney(t?.average_cost)}</td>
                    <td className={`${pinned.value} ${cell} num text-right`}>{formatMoney(t?.total_value)}</td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}
    </>
  )
}
