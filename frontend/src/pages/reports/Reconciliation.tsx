import { useState } from 'react'
import { Link } from 'react-router'
import type { ReconciliationResponse } from '../../api/types'
import { Row, Table, Td } from '../../components/Table'
import { Button, Checkbox, EmptyState, ErrorAlert, Spinner, Stat } from '../../components/ui'
import { formatDecimal } from '../../lib/format'
import { isZero } from '../../lib/grid'
import { useApi } from '../../lib/useApi'
import { ReportHeader } from './ReportHeader'

export function Reconciliation() {
  const [onlyDrift, setOnlyDrift] = useState(true)
  const { data, error, loading, reload } = useApi<ReconciliationResponse>(`/api/reports/reconciliation?only_drift=${onlyDrift}`)

  return (
    <>
      <ReportHeader
        title="Reconciliation"
        description="Recounts every bin from the ledger and compares it with the stored balance. It only reports; it never changes anything."
        actions={
          <Button variant="ghost" onClick={reload} disabled={loading}>
            {loading ? 'Recounting…' : 'Recount now'}
          </Button>
        }
      />
      <ErrorAlert error={error} onRetry={reload} />
      {!data ? (
        !error && <Spinner />
      ) : (
        <>
          <div className="mb-8 flex flex-wrap items-end gap-8">
            <div className="w-56">
              <Stat label="Bins out of balance" value={data.drift_count} tone={data.drift_count ? 'danger' : 'success'} />
            </div>
            <Checkbox label="Show only differences" checked={onlyDrift} onChange={(e) => setOnlyDrift(e.target.checked)} />
          </div>
          {data.data.length === 0 ? (
            <EmptyState title="Ledger and balances agree">Every stored balance equals the sum of its ledger entries.</EmptyState>
          ) : (
            <Table head={['SKU', 'Bin', 'Stored', 'From ledger', 'Difference']} numeric={[2, 3, 4]}>
              {data.data.map((r) => {
                const drift = !isZero(r.difference.replace('-', ''))
                return (
                  <Row key={`${r.product_id}-${r.location_id}`} tone={drift ? 'danger' : 'default'}>
                    <Td>
                      <Link to={`/products/${r.product_id}`} className="link-quiet code">
                        {r.sku}
                      </Link>
                    </Td>
                    <Td className="code">{r.location_code}</Td>
                    <Td numeric>{formatDecimal(r.cached_quantity)}</Td>
                    <Td numeric>{formatDecimal(r.ledger_quantity)}</Td>
                    <Td numeric className={drift ? 'font-medium text-danger' : 'text-muted'}>
                      {formatDecimal(r.difference)}
                    </Td>
                  </Row>
                )
              })}
            </Table>
          )}
          <p className="mt-6 max-w-2xl text-sm text-muted">
            Matching quantities do not prove the average cost is right. The{' '}
            <Link to="/reports/cost-check" className="link">
              cost check
            </Link>{' '}
            covers that.
          </p>
        </>
      )}
    </>
  )
}
