import type { ReactNode } from 'react'
import { Link, useParams } from 'react-router'
import type { ListResponse, Location, MovementDetail } from '../api/types'
import { Delta, MovementBadge } from '../components/MovementBadge'
import { Table, Td } from '../components/Table'
import { Card, EmptyState, ErrorAlert, PageHeader, Spinner } from '../components/ui'
import { formatDateTime, formatDecimal, formatMoney, formatQuantity } from '../lib/format'
import { useApi } from '../lib/useApi'

export function MovementDetailPage() {
  const { id } = useParams()
  const { data: m, error, reload } = useApi<MovementDetail>(`/api/movements/${id}`)
  const locations = useApi<ListResponse<Location>>('/api/locations')

  if (error?.code === 'NOT_FOUND' || error?.code === 'BAD_REQUEST') {
    return (
      <EmptyState title="Movement not found">
        <Link to="/movements" className="text-emerald-700 underline">
          Back to history
        </Link>
      </EmptyState>
    )
  }
  if (error) return <ErrorAlert error={error} onRetry={reload} />
  if (!m || !locations.data) return <Spinner />

  const binCode = (locationId: number) => locations.data?.data.find((l) => l.id === locationId)?.code ?? `#${locationId}`

  return (
    <>
      <Link to="/movements" className="text-sm text-emerald-700 hover:underline">
        ← Movement history
      </Link>
      <PageHeader title={`Movement #${m.id}`} description={formatDateTime(m.created_at)} />

      {m.is_reversed && m.reversed_by_id && (
        <div className="mb-4 rounded-md border border-slate-300 bg-slate-100 p-3 text-sm">
          This movement was reversed by{' '}
          <Link to={`/movements/${m.reversed_by_id}`} className="font-semibold text-emerald-700 underline">
            movement #{m.reversed_by_id}
          </Link>
          . It stays in history unchanged.
        </div>
      )}
      {m.reverses_movement_id && (
        <div className="mb-4 rounded-md border border-slate-300 bg-slate-100 p-3 text-sm">
          This is a reversal of{' '}
          <Link to={`/movements/${m.reverses_movement_id}`} className="font-semibold text-emerald-700 underline">
            movement #{m.reverses_movement_id}
          </Link>
          .
        </div>
      )}

      <Card className="mb-6">
        <dl className="grid gap-x-6 gap-y-3 text-sm sm:grid-cols-2 lg:grid-cols-3">
          <Item label="Type">
            <MovementBadge type={m.type} />
          </Item>
          <Item label="Product">
            <Link to={`/products/${m.product_id}`} className="text-emerald-700 hover:underline">
              {m.sku} · {m.product_name}
            </Link>
          </Item>
          <Item label="Quantity">{formatQuantity(m.quantity, m.unit)}</Item>
          <Item label="From bin">
            <span className="font-mono">{m.from_location_code ?? '—'}</span>
          </Item>
          <Item label="To bin">
            <span className="font-mono">{m.to_location_code ?? '—'}</span>
          </Item>
          <Item label="Unit cost">{formatMoney(m.unit_cost)}</Item>
          <Item label="Total value">{formatMoney(m.total_value)}</Item>
          <Item label="Reference">{m.reference || '—'}</Item>
          <Item label="Reason">{m.reason?.replace('_', ' ') ?? '—'}</Item>
          <Item label="Note">{m.note || '—'}</Item>
          <Item label="Posted by">{m.created_by}</Item>
        </dl>
      </Card>

      <h2 className="mb-1 text-lg font-semibold">Ledger entries</h2>
      <p className="mb-3 text-sm text-slate-500">The stock effect of this movement on each bin. A transfer has two entries.</p>
      <Table head={['Bin', 'Change', 'Unit cost']} numeric={[1, 2]}>
        {(m.entries ?? []).map((e) => (
          <tr key={e.location_id}>
            <Td className="font-mono">{binCode(e.location_id)}</Td>
            <Td className="text-right">
              <Delta value={formatDecimal(e.quantity_delta)} />
            </Td>
            <Td className="text-right tabular-nums">{formatMoney(e.unit_cost)}</Td>
          </tr>
        ))}
      </Table>
    </>
  )
}

function Item({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div>
      <dt className="text-slate-500">{label}</dt>
      <dd className="mt-0.5 font-medium">{children}</dd>
    </div>
  )
}
