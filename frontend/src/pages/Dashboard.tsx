import { Link } from 'react-router'
import type { ActivityDay, ListResponse, LowStockRow, ReconciliationResponse, TopMover, ValuationResponse } from '../api/types'
import { ActivityChart, TopMoversChart } from '../components/Charts'
import { Card, ErrorAlert, PageHeader, Spinner, StatCard } from '../components/ui'
import { formatDecimal, formatMoney } from '../lib/format'
import { useApi } from '../lib/useApi'

export function Dashboard() {
  const valuation = useApi<ValuationResponse>('/api/reports/valuation')
  const lowStock = useApi<ListResponse<LowStockRow>>('/api/reports/low-stock')
  const recon = useApi<ReconciliationResponse>('/api/reports/reconciliation?only_drift=true')
  const activity = useApi<ListResponse<ActivityDay>>('/api/reports/daily-activity?days=14')
  const movers = useApi<ListResponse<TopMover>>('/api/reports/top-movers?days=30&limit=5')

  const all = [valuation, lowStock, recon, activity, movers]
  const error = all.find((r) => r.error)?.error ?? null
  const ready = all.every((r) => r.data)

  return (
    <>
      <PageHeader title="Dashboard" description="Stock position across all bins." />
      <ErrorAlert error={error} onRetry={() => all.forEach((r) => r.reload())} />
      {!ready ? (
        !error && <Spinner />
      ) : (
        <>
          <div className="mb-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <StatCard label="Products" value={valuation.data!.data.length} />
            <StatCard label="Inventory value" value={formatMoney(valuation.data!.total_value)} />
            <StatCard label="Low-stock products" value={lowStock.data!.data.length} tone={lowStock.data!.data.length ? 'warning' : 'default'} />
            <StatCard label="Balances with drift" value={recon.data!.drift_count} tone={recon.data!.drift_count ? 'danger' : 'default'} />
          </div>
          <div className="grid gap-6 lg:grid-cols-3">
            <Card title="Movements per day · last 14 days" className="lg:col-span-2">
              <ActivityChart days={activity.data!.data} />
            </Card>
            <Card title="Top shipped · last 30 days">
              <TopMoversChart movers={movers.data!.data} />
            </Card>
            <Card title="Low stock" className="lg:col-span-3">
              {lowStock.data!.data.length === 0 ? (
                <p className="text-sm text-slate-500">Every product is above its reorder level.</p>
              ) : (
                <ul className="divide-y divide-slate-100 text-sm">
                  {lowStock.data!.data.slice(0, 5).map((r) => (
                    <li key={r.product_id} className="flex justify-between py-2">
                      <Link to={`/products/${r.product_id}`} className="font-mono text-emerald-700 hover:underline">
                        {r.sku}
                      </Link>
                      <span className="tabular-nums text-slate-600">
                        {formatDecimal(r.total_quantity)} of {formatDecimal(r.reorder_level)} {r.unit} · short {formatDecimal(r.shortfall)}
                      </span>
                    </li>
                  ))}
                </ul>
              )}
              <Link to="/reports/low-stock" className="mt-2 inline-block text-sm text-emerald-700 hover:underline">
                See the full report →
              </Link>
            </Card>
          </div>
        </>
      )}
    </>
  )
}
