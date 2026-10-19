import { useState } from 'react'
import { Link } from 'react-router'
import type { ReconciliationResponse } from '../../api/types'
import { Table, Td } from '../../components/Table'
import { Button, EmptyState, ErrorAlert, PageHeader, Spinner, StatCard } from '../../components/ui'
import { formatDecimal } from '../../lib/format'
import { isZero } from '../../lib/grid'
import { useApi } from '../../lib/useApi'

export function Reconciliation() {
  const [onlyDrift, setOnlyDrift] = useState(true)
  const { data, error, loading, reload } = useApi<ReconciliationResponse>(`/api/reports/reconciliation?only_drift=${onlyDrift}`)

  return (
    <>
      <PageHeader
        title="Reconciliation"
        description="Recomputes every balance from the ledger and compares it with the cached balance. It only reports; it never changes data."
        actions={
          <Button onClick={reload} disabled={loading}>
            {loading ? 'Running…' : 'Run again'}
          </Button>
        }
      />
      <ErrorAlert error={error} onRetry={reload} />
      {!data ? (
        !error && <Spinner />
      ) : (
        <>
          <div className="mb-4 flex flex-wrap items-center gap-4">
            <div className="w-56">
              <StatCard label="Balances with drift" value={data.drift_count} tone={data.drift_count ? 'danger' : 'default'} />
            </div>
            <label className="flex items-center gap-2 text-sm text-slate-700">
              <input type="checkbox" checked={onlyDrift} onChange={(e) => setOnlyDrift(e.target.checked)} className="h-4 w-4 accent-emerald-700" />
              Show only differences
            </label>
          </div>
          {data.data.length === 0 ? (
            <EmptyState title="Ledger and balances agree">Every cached balance equals the sum of its ledger entries.</EmptyState>
          ) : (
            <Table head={['SKU', 'Bin', 'Cached', 'Ledger', 'Difference']} numeric={[2, 3, 4]}>
              {data.data.map((r) => {
                const drift = !isZero(r.difference.replace('-', ''))
                return (
                  <tr key={`${r.product_id}-${r.location_id}`} className={drift ? 'bg-red-50' : ''}>
                    <Td>
                      <Link to={`/products/${r.product_id}`} className="font-mono text-emerald-700 hover:underline">
                        {r.sku}
                      </Link>
                    </Td>
                    <Td className="font-mono">{r.location_code}</Td>
                    <Td className="text-right tabular-nums">{formatDecimal(r.cached_quantity)}</Td>
                    <Td className="text-right tabular-nums">{formatDecimal(r.ledger_quantity)}</Td>
                    <Td className={`text-right font-semibold tabular-nums ${drift ? 'text-red-700' : 'text-slate-400'}`}>{formatDecimal(r.difference)}</Td>
                  </tr>
                )
              })}
            </Table>
          )}
          <p className="mt-4 text-sm text-slate-500">
            Matching quantities do not prove the average cost is right. Use the{' '}
            <Link to="/reports/cost-check" className="text-emerald-700 underline">
              cost check
            </Link>{' '}
            for that.
          </p>
        </>
      )}
    </>
  )
}
