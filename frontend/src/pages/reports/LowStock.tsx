import { Link } from 'react-router'
import type { ListResponse, LowStockRow } from '../../api/types'
import { CsvLink } from '../../components/CsvLink'
import { Row, Table, Td } from '../../components/Table'
import { EmptyState, ErrorAlert, Spinner } from '../../components/ui'
import { formatDecimal } from '../../lib/format'
import { useApi } from '../../lib/useApi'
import { ReportHeader } from './ReportHeader'

export function LowStock() {
  const { data, error, reload } = useApi<ListResponse<LowStockRow>>('/api/reports/low-stock')
  return (
    <>
      <ReportHeader
        title="Low stock"
        description="Products whose total across all bins is at or below their reorder level, the largest shortfall first."
        actions={<CsvLink path="/api/reports/low-stock?format=csv" />}
      />
      <ErrorAlert error={error} onRetry={reload} />
      {!data ? (
        !error && <Spinner />
      ) : data.data.length === 0 ? (
        <EmptyState title="Nothing is low">Every product is above its reorder level.</EmptyState>
      ) : (
        <Table head={['SKU', 'Name', 'On hand', 'Reorder level', 'Short by', '']} numeric={[2, 3, 4]}>
          {data.data.map((r) => (
            <Row key={r.product_id}>
              <Td>
                <Link to={`/products/${r.product_id}`} className="link-quiet code">
                  {r.sku}
                </Link>
              </Td>
              <Td>{r.name}</Td>
              <Td numeric>{formatDecimal(r.total_quantity)}</Td>
              <Td numeric className="text-muted">
                {formatDecimal(r.reorder_level)}
              </Td>
              <Td numeric className="font-medium text-saffron-ink">
                {formatDecimal(r.shortfall)} {r.unit}
              </Td>
              <Td>
                <Link to={`/record/receive?product=${r.product_id}`} className="link text-sm">
                  Receive
                </Link>
              </Td>
            </Row>
          ))}
        </Table>
      )}
    </>
  )
}
