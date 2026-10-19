import { Link } from 'react-router'
import type { ListResponse, LowStockRow } from '../../api/types'
import { CsvLink } from '../../components/CsvLink'
import { Table, Td } from '../../components/Table'
import { EmptyState, ErrorAlert, PageHeader, Spinner } from '../../components/ui'
import { formatDecimal } from '../../lib/format'
import { useApi } from '../../lib/useApi'

export function LowStock() {
  const { data, error, reload } = useApi<ListResponse<LowStockRow>>('/api/reports/low-stock')
  return (
    <>
      <PageHeader
        title="Low stock"
        description="Products whose total across all bins is at or below their reorder level, largest shortfall first."
        actions={<CsvLink path="/api/reports/low-stock?format=csv" />}
      />
      <ErrorAlert error={error} onRetry={reload} />
      {!data ? (
        !error && <Spinner />
      ) : data.data.length === 0 ? (
        <EmptyState title="Nothing is low">Every product is above its reorder level.</EmptyState>
      ) : (
        <Table head={['SKU', 'Name', 'Total on hand', 'Reorder level', 'Shortfall', '']} numeric={[2, 3, 4]}>
          {data.data.map((r) => (
            <tr key={r.product_id} className="hover:bg-slate-50">
              <Td>
                <Link to={`/products/${r.product_id}`} className="font-mono text-emerald-700 hover:underline">
                  {r.sku}
                </Link>
              </Td>
              <Td>{r.name}</Td>
              <Td className="text-right tabular-nums">{formatDecimal(r.total_quantity)}</Td>
              <Td className="text-right tabular-nums">{formatDecimal(r.reorder_level)}</Td>
              <Td className="text-right font-semibold tabular-nums text-amber-700">
                {formatDecimal(r.shortfall)} {r.unit}
              </Td>
              <Td>
                <Link to={`/receive?product=${r.product_id}`} className="text-emerald-700 hover:underline">
                  Receive
                </Link>
              </Td>
            </tr>
          ))}
        </Table>
      )}
    </>
  )
}
