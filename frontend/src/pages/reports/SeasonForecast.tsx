import { useState } from 'react'
import { Link } from 'react-router'
import type { ForecastRow } from '../../api/types'
import { Row, Table, Td } from '../../components/Table'
import { EmptyState, ErrorAlert, Field, Select, Spinner } from '../../components/ui'
import { formatMoney, formatQuantity } from '../../lib/format'
import { useApi } from '../../lib/useApi'
import { ReportHeader } from './ReportHeader'

const istDate = new Intl.DateTimeFormat('en-IN', { timeZone: 'Asia/Kolkata', day: 'numeric', month: 'short', year: 'numeric' })

export function SeasonForecast() {
  const [days, setDays] = useState('30')
  const { data, error, reload } = useApi<{ window_start: string; window_end: string; data: ForecastRow[] }>(`/api/reports/season-forecast?days=${days}`)

  return (
    <>
      <ReportHeader
        title="Season forecast"
        description="Compares stock on hand with what was shipped in the same weeks last year, so you can buy before the season starts."
      />
      <div className="mb-8 flex flex-wrap items-end gap-6">
        <div className="w-40">
          <Field label="Look ahead">
            <Select value={days} onChange={(e) => setDays(e.target.value)}>
              {['15', '30', '60', '90', '180'].map((d) => (
                <option key={d} value={d}>
                  {d} days
                </option>
              ))}
            </Select>
          </Field>
        </div>
        {data && (
          <p className="pb-2.5 text-sm text-muted">
            Compared with shipments from {istDate.format(new Date(data.window_start))} to {istDate.format(new Date(data.window_end))}
          </p>
        )}
      </div>
      <ErrorAlert error={error} onRetry={reload} />
      {!data ? (
        !error && <Spinner />
      ) : data.data.length === 0 ? (
        <EmptyState title="Nothing to stock up on">Stock on hand covers last year's shipments for this period, or there is no history from last year yet.</EmptyState>
      ) : (
        <Table head={['SKU', 'Name', 'Shipped last year', 'On hand', 'Short by', 'Cost to cover', '']} numeric={[2, 3, 4, 5]}>
          {data.data.map((r) => (
            <Row key={r.product_id}>
              <Td>
                <Link to={`/products/${r.product_id}`} className="link-quiet code">
                  {r.sku}
                </Link>
              </Td>
              <Td>{r.name}</Td>
              <Td numeric>{formatQuantity(r.last_year_demand, r.unit)}</Td>
              <Td numeric>{formatQuantity(r.current_stock, r.unit)}</Td>
              <Td numeric className="font-medium text-saffron-ink">
                {formatQuantity(r.gap, r.unit)}
              </Td>
              <Td numeric>{formatMoney(r.gap_value)}</Td>
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
