import type { ListResponse, LowStockRow, ReconciliationResponse, ValuationResponse } from '../api/types'
import { ErrorAlert, PageHeader, Spinner, StatCard } from '../components/ui'
import { formatMoney } from '../lib/format'
import { useApi } from '../lib/useApi'

export function Dashboard() {
  const valuation = useApi<ValuationResponse>('/api/reports/valuation')
  const lowStock = useApi<ListResponse<LowStockRow>>('/api/reports/low-stock')
  const recon = useApi<ReconciliationResponse>('/api/reports/reconciliation?only_drift=true')

  const error = valuation.error ?? lowStock.error ?? recon.error
  const loading = valuation.loading || lowStock.loading || recon.loading

  return (
    <>
      <PageHeader title="Dashboard" description="Stock position across all bins." />
      <ErrorAlert
        error={error}
        onRetry={() => {
          valuation.reload()
          lowStock.reload()
          recon.reload()
        }}
      />
      {loading ? (
        <Spinner />
      ) : (
        !error && (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <StatCard label="Products" value={valuation.data?.data.length ?? 0} />
            <StatCard label="Inventory value" value={formatMoney(valuation.data?.total_value)} />
            <StatCard label="Low-stock products" value={lowStock.data?.data.length ?? 0} tone={lowStock.data?.data.length ? 'warning' : 'default'} />
            <StatCard label="Balances with drift" value={recon.data?.drift_count ?? 0} tone={recon.data?.drift_count ? 'danger' : 'default'} />
          </div>
        )
      )}
    </>
  )
}
