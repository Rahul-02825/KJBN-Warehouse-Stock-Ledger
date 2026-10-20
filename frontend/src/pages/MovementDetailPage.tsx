import { ArrowLeft } from 'lucide-react'
import { icon } from '../components/icon'
import type { ReactNode } from 'react'
import { Link, useParams } from 'react-router'
import type { ListResponse, Location, MovementDetail } from '../api/types'
import { Delta, MovementBadge } from '../components/MovementBadge'
import { ReversePanel } from '../components/ReversePanel'
import { Row, Table, Td } from '../components/Table'
import { EmptyState, ErrorAlert, Eyebrow, Spinner } from '../components/ui'
import { formatDateTime, formatDecimal, formatMoney, formatQuantity } from '../lib/format'
import { useApi } from '../lib/useApi'

export function MovementDetailPage() {
  const { id } = useParams()
  const { data: m, error, reload } = useApi<MovementDetail>(`/api/movements/${id}`)
  const locations = useApi<ListResponse<Location>>('/api/locations')

  if (error?.code === 'NOT_FOUND' || error?.code === 'BAD_REQUEST') {
    return (
      <EmptyState title="Movement not found" action={<Link to="/movements" className="link text-sm">Back to movement history</Link>}>
        It may have been typed wrongly. Movements are never deleted.
      </EmptyState>
    )
  }
  if (error) return <ErrorAlert error={error} onRetry={reload} />
  if (!m || !locations.data) return <Spinner />

  const binCode = (locationId: number) => locations.data?.data.find((l) => l.id === locationId)?.code ?? `#${locationId}`

  return (
    <>
      <Link to="/movements" className="inline-flex items-center gap-1.5 text-sm text-muted transition-colors duration-200 hover:text-ink">
        <ArrowLeft {...icon} />
        Movement history
      </Link>
      <header className="mb-8 mt-6">
        <Eyebrow className="mb-4">{formatDateTime(m.created_at)}</Eyebrow>
        <div className="flex flex-wrap items-center gap-3">
          <h1 className="text-[clamp(30px,4.4vw,48px)] font-semibold leading-[1.06] tracking-[-0.025em]">Movement #{m.id}</h1>
          <MovementBadge type={m.type} />
        </div>
        <p className="mt-3 text-[15.5px] text-muted">
          {formatQuantity(m.quantity, m.unit)} of{' '}
          <Link to={`/products/${m.product_id}`} className="link">
            {m.sku}, {m.product_name}
          </Link>
        </p>
      </header>

      {m.is_reversed && m.reversed_by_id && (
        <p className="mb-6 border-l-2 border-saffron pl-4 text-sm">
          Reversed by{' '}
          <Link to={`/movements/${m.reversed_by_id}`} className="link">
            movement #{m.reversed_by_id}
          </Link>
          . This record stays in history unchanged.
        </p>
      )}
      {m.reverses_movement_id && (
        <p className="mb-6 border-l-2 border-saffron pl-4 text-sm">
          This undoes{' '}
          <Link to={`/movements/${m.reverses_movement_id}`} className="link">
            movement #{m.reverses_movement_id}
          </Link>
          .
        </p>
      )}

      <dl className="mb-12 grid gap-x-8 gap-y-5 border-y border-line py-6 text-sm sm:grid-cols-2 lg:grid-cols-4">
        <Item label="From bin">
          <span className="code">{m.from_location_code ?? '–'}</span>
        </Item>
        <Item label="To bin">
          <span className="code">{m.to_location_code ?? '–'}</span>
        </Item>
        <Item label="Unit cost">
          <span className="num">{formatMoney(m.unit_cost)}</span>
        </Item>
        <Item label="Total value">
          <span className="num">{formatMoney(m.total_value)}</span>
        </Item>
        <Item label="Reference">{m.reference || '–'}</Item>
        <Item label="Reason">{m.reason?.replace('_', ' ') ?? '–'}</Item>
        <Item label="Note">{m.note || '–'}</Item>
        <Item label="Posted by">{m.created_by}</Item>
      </dl>

      <section className="mb-12">
        <h2 className="mb-1 font-serif text-[clamp(22px,2.6vw,30px)] leading-tight">Ledger entries</h2>
        <p className="mb-4 text-sm text-muted">What this movement did to each bin. A transfer or a split receipt touches more than one.</p>
        <Table head={['Bin', 'Change', 'Unit cost']} numeric={[1, 2]}>
          {(m.entries ?? []).map((e) => (
            <Row key={e.location_id}>
              <Td className="code">{binCode(e.location_id)}</Td>
              <Td numeric>
                <Delta value={formatDecimal(e.quantity_delta)} />
              </Td>
              <Td numeric>{formatMoney(e.unit_cost)}</Td>
            </Row>
          ))}
        </Table>
      </section>

      <ReversePanel key={m.id} movement={m} onReversed={reload} />
    </>
  )
}

function Item({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div>
      <dt className="text-[12.5px] text-muted">{label}</dt>
      <dd className="mt-1 font-medium">{children}</dd>
    </div>
  )
}
