import { Link } from 'react-router'
import type { CostCheckRow } from '../../api/types'
import { Table, Td } from '../../components/Table'
import { Button, ErrorAlert, PageHeader, Spinner, StatCard } from '../../components/ui'
import { formatMoney } from '../../lib/format'
import { useApi } from '../../lib/useApi'

export function CostCheck() {
  const { data, error, loading, reload } = useApi<{ data: CostCheckRow[]; mismatch_count: number }>('/api/reports/cost-check')
  return (
    <>
      <PageHeader
        title="Cost check"
        description="Replays every movement with the same costing rules used when posting and compares the result with each product's stored average cost."
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
          <div className="mb-4 w-56">
            <StatCard label="Products that differ" value={data.mismatch_count} tone={data.mismatch_count ? 'danger' : 'default'} />
          </div>
          <Table head={['SKU', 'Name', 'Stored average', 'Recomputed', 'Difference', 'Status']} numeric={[2, 3, 4]}>
            {data.data.map((r) => (
              <tr key={r.product_id} className={r.matches ? '' : 'bg-red-50'}>
                <Td>
                  <Link to={`/products/${r.product_id}`} className="font-mono text-emerald-700 hover:underline">
                    {r.sku}
                  </Link>
                </Td>
                <Td>{r.name}</Td>
                <Td className="text-right tabular-nums">{formatMoney(r.stored_average)}</Td>
                <Td className="text-right tabular-nums">{formatMoney(r.recomputed_average)}</Td>
                <Td className="text-right tabular-nums">{formatMoney(r.difference)}</Td>
                <Td>{r.matches ? <span className="text-emerald-700">Matches</span> : <span className="font-semibold text-red-700">Differs</span>}</Td>
              </tr>
            ))}
          </Table>
        </>
      )}
    </>
  )
}
