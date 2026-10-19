import { Link } from 'react-router'
import type { ActivityDay, CostCheckRow, ForecastRow, ListResponse, LowStockRow, ReconciliationResponse, TopMover, ValuationResponse, Warehouse } from '../api/types'
import { ActivityChart, TopMoversChart } from '../components/Charts'
import { ButtonLink, ErrorAlert, FeaturePanel, PageHeader, Panel, Spinner } from '../components/ui'
import { formatDecimal, formatMoney } from '../lib/format'
import { fillText } from '../lib/layout'
import { useApi } from '../lib/useApi'

export function Dashboard() {
  const valuation = useApi<ValuationResponse>('/api/reports/valuation')
  const lowStock = useApi<ListResponse<LowStockRow>>('/api/reports/low-stock')
  const recon = useApi<ReconciliationResponse>('/api/reports/reconciliation?only_drift=true')
  const costs = useApi<{ data: CostCheckRow[]; mismatch_count: number }>('/api/reports/cost-check')
  const forecast = useApi<{ data: ForecastRow[] }>('/api/reports/season-forecast?days=30')
  const warehouses = useApi<ListResponse<Warehouse>>('/api/warehouses')
  const activity = useApi<ListResponse<ActivityDay>>('/api/reports/daily-activity?days=14')
  const movers = useApi<ListResponse<TopMover>>('/api/reports/top-movers?days=30&limit=5')

  const all = [valuation, lowStock, recon, costs, forecast, warehouses, activity, movers]
  const error = all.find((r) => r.error)?.error ?? null
  const ready = all.every((r) => r.data)
  const warehouse = warehouses.data?.data[0]

  return (
    <>
      <PageHeader
        eyebrow={warehouse ? warehouse.name : 'Overview'}
        title="Stock position"
        description="Everything in the warehouse, valued at weighted-average cost, and what needs a look today."
        actions={
          <ButtonLink to="/stock" variant="ghost">
            View stock
          </ButtonLink>
        }
      />
      <ErrorAlert error={error} onRetry={() => all.forEach((r) => r.reload())} />
      {!ready ? (
        !error && <Spinner />
      ) : (
        <div className="space-y-14">
          <FeaturePanel>
            <div className="grid gap-8 md:grid-cols-[1.5fr_1fr] md:items-end">
              <div>
                <p className="text-sm text-on-petrol-muted">Inventory value</p>
                <p className="num mt-2 font-serif text-[clamp(40px,6.4vw,68px)] leading-none tracking-[-0.02em]">{formatMoney(valuation.data!.total_value)}</p>
                <p className="mt-4 text-sm text-on-petrol-muted">
                  {valuation.data!.data.length} products
                  {warehouse && <> in a {warehouse.radius_m} m warehouse</>}
                </p>
              </div>
              <dl className="grid grid-cols-2 gap-6 border-petrol-line md:border-l md:pl-8">
                <HeroFigure label="Items stored" value={warehouse ? fillText(warehouse.used, warehouse.capacity) : '—'} />
                <HeroFigure label="Below reorder level" value={String(lowStock.data!.data.length)} />
                <HeroFigure label="Movements, 14 days" value={String(activity.data!.data.reduce((n, d) => n + d.in_count + d.out_count + d.transfer_count + d.adjustment_count + d.reversal_count, 0))} />
                <HeroFigure label="Bins out of balance" value={String(recon.data!.drift_count)} />
              </dl>
            </div>
          </FeaturePanel>

          <NeedsAttention
            lowStock={lowStock.data!.data}
            drift={recon.data!.drift_count}
            costMismatches={costs.data!.mismatch_count}
            seasonGaps={forecast.data!.data.length}
          />

          <div className="grid gap-6 lg:grid-cols-3">
            <Panel title="Movements per day" className="lg:col-span-2">
              <ActivityChart days={activity.data!.data} />
            </Panel>
            <Panel title="Most shipped this month">
              <TopMoversChart movers={movers.data!.data} />
            </Panel>
          </div>
        </div>
      )}
    </>
  )
}

function HeroFigure({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-[12.5px] text-on-petrol-muted">{label}</dt>
      <dd className="num mt-1 font-serif text-[26px] leading-tight text-saffron">{value}</dd>
    </div>
  )
}

function NeedsAttention({ lowStock, drift, costMismatches, seasonGaps }: { lowStock: LowStockRow[]; drift: number; costMismatches: number; seasonGaps: number }) {
  const items = [
    {
      count: lowStock.length,
      text: lowStock.length === 1 ? `${lowStock[0].sku} is at or below its reorder level` : 'products are at or below their reorder level',
      detail:
        lowStock.length === 1
          ? `${formatDecimal(lowStock[0].total_quantity)} of ${formatDecimal(lowStock[0].reorder_level)} ${lowStock[0].unit} on hand`
          : lowStock
              .slice(0, 3)
              .map((r) => `${r.sku} ${formatDecimal(r.total_quantity)} of ${formatDecimal(r.reorder_level)} ${r.unit}`)
              .join(', '),
      to: '/reports/low-stock',
      action: 'Review low stock',
    },
    { count: drift, text: drift === 1 ? 'bin balance differs from the ledger' : 'bin balances differ from the ledger', to: '/reports/reconciliation', action: 'Reconcile' },
    { count: costMismatches, text: costMismatches === 1 ? 'average cost does not match its history' : 'average costs do not match their history', to: '/reports/cost-check', action: 'Check costs' },
    { count: seasonGaps, text: seasonGaps === 1 ? 'product has less stock than last season' : 'products have less stock than last season', to: '/reports/season-forecast', action: 'Plan stock' },
  ].filter((i) => i.count > 0)

  return (
    <section>
      <h2 className="mb-1 font-serif text-[clamp(22px,2.6vw,30px)] leading-tight">Needs attention</h2>
      {items.length === 0 ? (
        <p className="text-muted">Nothing right now. Every product is above its reorder level and the ledger agrees with every bin.</p>
      ) : (
        <ul className="mt-4 divide-y divide-line border-y border-line">
          {items.map((i) => (
            <li key={i.to} className="grid gap-x-6 gap-y-1 py-4 sm:grid-cols-[64px_1fr_auto] sm:items-baseline">
              <span className="num font-serif text-[30px] leading-none">{i.count}</span>
              <div>
                <p>{i.text}</p>
                {i.detail && <p className="mt-0.5 text-[13px] text-muted">{i.detail}</p>}
              </div>
              <Link to={i.to} className="link justify-self-start text-sm">
                {i.action}
              </Link>
            </li>
          ))}
        </ul>
      )}
    </section>
  )
}
