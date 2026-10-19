import { Link } from 'react-router'
import type { ValuationResponse } from '../../api/types'
import { CsvLink } from '../../components/CsvLink'
import { Table, Td } from '../../components/Table'
import { EmptyState, ErrorAlert, PageHeader, Spinner, StatCard } from '../../components/ui'
import { formatMoney, formatQuantity } from '../../lib/format'
import { useApi } from '../../lib/useApi'

export function Valuation() {
  const { data, error, reload } = useApi<ValuationResponse>('/api/reports/valuation')
  return (
    <>
      <PageHeader title="Inventory valuation" description="Quantity × weighted-average cost for each product." actions={<CsvLink path="/api/reports/valuation?format=csv" />} />
      <ErrorAlert error={error} onRetry={reload} />
      {!data ? (
        !error && <Spinner />
      ) : data.data.length === 0 ? (
        <EmptyState title="No products yet" />
      ) : (
        <>
          <div className="mb-4 max-w-xs">
            <StatCard label="Total inventory value" value={formatMoney(data.total_value)} />
          </div>
          <Table head={['SKU', 'Name', 'Total on hand', 'Average cost', 'Value']} numeric={[2, 3, 4]}>
            {data.data.map((r) => (
              <tr key={r.product_id} className="hover:bg-slate-50">
                <Td>
                  <Link to={`/products/${r.product_id}`} className="font-mono text-emerald-700 hover:underline">
                    {r.sku}
                  </Link>
                </Td>
                <Td>{r.name}</Td>
                <Td className="text-right tabular-nums">{formatQuantity(r.total_quantity, r.unit)}</Td>
                <Td className="text-right tabular-nums">{formatMoney(r.average_cost)}</Td>
                <Td className="text-right font-semibold tabular-nums">{formatMoney(r.total_value)}</Td>
              </tr>
            ))}
            <tr className="bg-slate-50 font-semibold">
              <Td>Total</Td>
              <Td>{''}</Td>
              <Td>{''}</Td>
              <Td>{''}</Td>
              <Td className="text-right tabular-nums">{formatMoney(data.total_value)}</Td>
            </tr>
          </Table>
        </>
      )}
    </>
  )
}
