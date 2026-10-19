import { useState, type FormEvent } from 'react'
import { Link, useSearchParams } from 'react-router'
import { MOVEMENT_TYPES, type MovementDetail, type PagedResponse } from '../api/types'
import { CsvLink } from '../components/CsvLink'
import { MovementBadge } from '../components/MovementBadge'
import { Pagination } from '../components/Pagination'
import { Row, Table, Td } from '../components/Table'
import { Badge, Button, ButtonLink, EmptyState, ErrorAlert, Field, Input, PageHeader, Select, Spinner } from '../components/ui'
import { formatDateTime, formatMoney, formatQuantity } from '../lib/format'
import { useApi } from '../lib/useApi'
import { useOptions } from '../lib/useOptions'

const PAGE_SIZE = 25
const FILTERS = ['product_id', 'location_id', 'type', 'reference', 'from', 'to'] as const
type Filters = Record<(typeof FILTERS)[number], string>

export function MovementHistory() {
  const [params, setParams] = useSearchParams()
  const { products, locations } = useOptions()
  const [draft, setDraft] = useState<Filters>(() => Object.fromEntries(FILTERS.map((f) => [f, params.get(f) ?? ''])) as Filters)
  const page = Number(params.get('page') ?? '1') || 1

  const filterQuery = new URLSearchParams()
  for (const f of FILTERS) if (params.get(f)) filterQuery.set(f, params.get(f)!)
  const { data, error, loading, reload } = useApi<PagedResponse<MovementDetail>>(`/api/movements?${filterQuery}&page=${page}&page_size=${PAGE_SIZE}`)
  const filtered = filterQuery.toString() !== ''

  function apply(e: FormEvent) {
    e.preventDefault()
    const next = new URLSearchParams()
    for (const f of FILTERS) if (draft[f].trim()) next.set(f, draft[f].trim())
    setParams(next)
  }

  function clear() {
    setDraft(Object.fromEntries(FILTERS.map((f) => [f, ''])) as Filters)
    setParams(new URLSearchParams())
  }

  const set = (f: (typeof FILTERS)[number]) => (value: string) => setDraft({ ...draft, [f]: value })

  return (
    <>
      <PageHeader
        eyebrow="Movements"
        title="Movement history"
        description="Every posted movement, newest first. Nothing is ever edited or deleted. A mistake is corrected by reversing it."
        actions={<CsvLink path={`/api/movements?${filterQuery}${filtered ? '&' : ''}format=csv`} />}
      />

      <form onSubmit={apply} className="mb-8 grid gap-4 border-y border-line py-5 sm:grid-cols-3 lg:grid-cols-6">
        <Field label="Product">
          <Select value={draft.product_id} onChange={(e) => set('product_id')(e.target.value)}>
            <option value="">All</option>
            {products?.map((p) => (
              <option key={p.id} value={p.id}>
                {p.sku}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Bin">
          <Select value={draft.location_id} onChange={(e) => set('location_id')(e.target.value)}>
            <option value="">All</option>
            {locations?.map((l) => (
              <option key={l.id} value={l.id}>
                {l.code}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Type">
          <Select value={draft.type} onChange={(e) => set('type')(e.target.value)}>
            <option value="">All</option>
            {MOVEMENT_TYPES.map((t) => (
              <option key={t} value={t}>
                {t.charAt(0) + t.slice(1).toLowerCase()}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Reference contains">
          <Input value={draft.reference} onChange={(e) => set('reference')(e.target.value)} />
        </Field>
        <Field label="From (IST)">
          <Input type="date" value={draft.from} onChange={(e) => set('from')(e.target.value)} />
        </Field>
        <Field label="To (IST)">
          <Input type="date" value={draft.to} onChange={(e) => set('to')(e.target.value)} />
        </Field>
        <div className="flex gap-2 sm:col-span-3 lg:col-span-6">
          <Button type="submit">Apply filters</Button>
          {filtered && (
            <Button type="button" variant="ghost" onClick={clear}>
              Clear filters
            </Button>
          )}
        </div>
      </form>

      <ErrorAlert error={error} onRetry={reload} />
      {loading && !data ? (
        <Spinner />
      ) : data && data.data.length === 0 ? (
        filtered ? (
          <EmptyState title="No movements match">Widen the dates or clear a filter.</EmptyState>
        ) : (
          <EmptyState title="No movements yet" action={<ButtonLink to="/record/receive">Receive stock</ButtonLink>}>
            Receiving stock creates the first movement.
          </EmptyState>
        )
      ) : (
        data && (
          <>
            <Table head={['No.', 'Posted (IST)', 'Type', 'Product', 'From', 'To', 'Quantity', 'Value', 'Reference']} numeric={[6, 7]}>
              {data.data.map((m) => (
                <Row key={m.id} tone={m.is_reversed ? 'muted' : 'default'}>
                  <Td>
                    <Link to={`/movements/${m.id}`} className="link-quiet code">
                      #{m.id}
                    </Link>
                  </Td>
                  <Td className="text-muted">{formatDateTime(m.created_at)}</Td>
                  <Td>
                    <span className="flex items-center gap-1.5">
                      <MovementBadge type={m.type} />
                      {m.is_reversed && <Badge tone="neutral">Reversed</Badge>}
                    </span>
                  </Td>
                  <Td>
                    <Link to={`/products/${m.product_id}`} className="code hover:text-petrol">
                      {m.sku}
                    </Link>
                  </Td>
                  <Td className="code">{m.from_location_code ?? <span className="text-line">–</span>}</Td>
                  <Td className="code">{m.to_location_code ?? <span className="text-line">–</span>}</Td>
                  <Td numeric>{formatQuantity(m.quantity, m.unit)}</Td>
                  <Td numeric>{formatMoney(m.total_value)}</Td>
                  <Td className="max-w-56 truncate text-muted">{[m.reference, m.reason?.replace('_', ' ')].filter(Boolean).join(', ') || '–'}</Td>
                </Row>
              ))}
            </Table>
            <Pagination
              page={page}
              pageSize={PAGE_SIZE}
              total={data.total}
              onPage={(p) => {
                const next = new URLSearchParams(params)
                next.set('page', String(p))
                setParams(next)
              }}
            />
          </>
        )
      )}
    </>
  )
}
