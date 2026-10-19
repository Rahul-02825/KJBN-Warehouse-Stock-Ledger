import { RefreshCw } from 'lucide-react'
import { icon } from '../../components/icon'
import { Link } from 'react-router'
import type { CostCheckRow } from '../../api/types'
import { Row, Table, Td } from '../../components/Table'
import { Badge, Button, ErrorAlert, Spinner, Stat } from '../../components/ui'
import { formatMoney } from '../../lib/format'
import { useApi } from '../../lib/useApi'
import { ReportHeader } from './ReportHeader'

export function CostCheck() {
  const { data, error, loading, reload } = useApi<{ data: CostCheckRow[]; mismatch_count: number }>('/api/reports/cost-check')
  return (
    <>
      <ReportHeader
        title="Cost check"
        description="Replays every movement with the posting rules and compares the result with each product's stored average cost."
        actions={
          <Button variant="ghost" onClick={reload} disabled={loading}>
            <RefreshCw {...icon} className={loading ? 'animate-spin' : ''} />
            {loading ? 'Checking…' : 'Check again'}
          </Button>
        }
      />
      <ErrorAlert error={error} onRetry={reload} />
      {!data ? (
        !error && <Spinner />
      ) : (
        <>
          <div className="mb-8 w-56">
            <Stat label="Products that differ" value={data.mismatch_count} tone={data.mismatch_count ? 'danger' : 'success'} />
          </div>
          <Table head={['SKU', 'Name', 'Stored average', 'From history', 'Difference', '']} numeric={[2, 3, 4]}>
            {data.data.map((r) => (
              <Row key={r.product_id} tone={r.matches ? 'default' : 'danger'}>
                <Td>
                  <Link to={`/products/${r.product_id}`} className="link-quiet code">
                    {r.sku}
                  </Link>
                </Td>
                <Td>{r.name}</Td>
                <Td numeric>{formatMoney(r.stored_average)}</Td>
                <Td numeric>{formatMoney(r.recomputed_average)}</Td>
                <Td numeric className={r.matches ? 'text-muted' : 'font-medium text-danger'}>
                  {formatMoney(r.difference)}
                </Td>
                <Td>{r.matches ? <Badge tone="success">Matches</Badge> : <Badge tone="danger">Differs</Badge>}</Td>
              </Row>
            ))}
          </Table>
        </>
      )}
    </>
  )
}
