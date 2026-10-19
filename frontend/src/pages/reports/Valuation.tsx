import { Link } from 'react-router'
import type { ValuationResponse } from '../../api/types'
import { CsvLink } from '../../components/CsvLink'
import { Row, Table, Td } from '../../components/Table'
import { EmptyState, ErrorAlert, Spinner, Stat } from '../../components/ui'
import { formatMoney, formatQuantity } from '../../lib/format'
import { useApi } from '../../lib/useApi'
import { ReportHeader } from './ReportHeader'

export function Valuation() {
  const { data, error, reload } = useApi<ValuationResponse>('/api/reports/valuation')
  return (
    <>
      <ReportHeader title="Inventory valuation" description="Each product's quantity on hand times its weighted-average cost." actions={<CsvLink path="/api/reports/valuation?format=csv" />} />
      <ErrorAlert error={error} onRetry={reload} />
      {!data ? (
        !error && <Spinner />
      ) : data.data.length === 0 ? (
        <EmptyState title="No products yet">Valuation appears once products have stock.</EmptyState>
      ) : (
        <>
          <div className="mb-8 max-w-xs">
            <Stat label="Total value" value={formatMoney(data.total_value)} />
          </div>
          <Table head={['SKU', 'Name', 'On hand', 'Average cost', 'Value']} numeric={[2, 3, 4]}>
            {data.data.map((r) => (
              <Row key={r.product_id}>
                <Td>
                  <Link to={`/products/${r.product_id}`} className="link-quiet code">
                    {r.sku}
                  </Link>
                </Td>
                <Td>{r.name}</Td>
                <Td numeric>{formatQuantity(r.total_quantity, r.unit)}</Td>
                <Td numeric className="text-muted">
                  {formatMoney(r.average_cost)}
                </Td>
                <Td numeric className="font-medium">
                  {formatMoney(r.total_value)}
                </Td>
              </Row>
            ))}
            <tr className="bg-paper-2">
              <Td className="font-medium">Total</Td>
              <Td>{''}</Td>
              <Td>{''}</Td>
              <Td>{''}</Td>
              <Td numeric className="font-serif text-[17px]">
                {formatMoney(data.total_value)}
              </Td>
            </tr>
          </Table>
        </>
      )}
    </>
  )
}
