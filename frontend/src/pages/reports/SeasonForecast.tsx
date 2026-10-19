import { useState } from 'react'
import { Link } from 'react-router'
import type { ForecastRow } from '../../api/types'
import { Table, Td } from '../../components/Table'
import { EmptyState, ErrorAlert, PageHeader, Select, Spinner } from '../../components/ui'
import { formatMoney, formatQuantity } from '../../lib/format'
import { useApi } from '../../lib/useApi'

const istDate = new Intl.DateTimeFormat('en-IN', { timeZone: 'Asia/Kolkata', day: '2-digit', month: 'short', year: 'numeric' })

export function SeasonForecast() {
  const [days, setDays] = useState('30')
  const { data, error, reload } = useApi<{ window_start: string; window_end: string; data: ForecastRow[] }>(`/api/reports/season-forecast?days=${days}`)

  return (
    <>
      <PageHeader title="Season forecast" description="Compares current stock with what was shipped in the same period last year, so stock can be bought before the season starts." />
      <div className="mb-4 flex flex-wrap items-center gap-3 text-sm">
        <span>Look ahead</span>
        <div className="w-32">
          <Select value={days} onChange={(e) => setDays(e.target.value)} aria-label="Days to look ahead">
            {['15', '30', '60', '90', '180'].map((d) => (
              <option key={d} value={d}>
                {d} days
              </option>
            ))}
          </Select>
        </div>
        {data && (
          <span className="text-slate-500">
            Compared with shipments from {istDate.format(new Date(data.window_start))} to {istDate.format(new Date(data.window_end))}
          </span>
        )}
      </div>
      <ErrorAlert error={error} onRetry={reload} />
      {!data ? (
        !error && <Spinner />
      ) : data.data.length === 0 ? (
        <EmptyState title="Nothing to stock up on">
          Current stock covers last year's shipments for this period, or there is no history from last year yet.
        </EmptyState>
      ) : (
        <Table head={['SKU', 'Name', 'Shipped last year', 'In stock now', 'Short by', 'Cost to cover', '']} numeric={[2, 3, 4, 5]}>
          {data.data.map((r) => (
            <tr key={r.product_id} className="hover:bg-slate-50">
              <Td>
                <Link to={`/products/${r.product_id}`} className="font-mono text-emerald-700 hover:underline">
                  {r.sku}
                </Link>
              </Td>
              <Td>{r.name}</Td>
              <Td className="text-right tabular-nums">{formatQuantity(r.last_year_demand, r.unit)}</Td>
              <Td className="text-right tabular-nums">{formatQuantity(r.current_stock, r.unit)}</Td>
              <Td className="text-right font-semibold tabular-nums text-amber-700">{formatQuantity(r.gap, r.unit)}</Td>
              <Td className="text-right tabular-nums">{formatMoney(r.gap_value)}</Td>
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
