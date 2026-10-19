import { useState, type FormEvent } from 'react'
import { Link, useSearchParams } from 'react-router'
import { apiURL } from '../api/client'
import { MOVEMENT_TYPES, type MovementDetail, type PagedResponse } from '../api/types'
import { MovementBadge } from '../components/MovementBadge'
import { Pagination } from '../components/Pagination'
import { Table, Td } from '../components/Table'
import { Button, Card, EmptyState, ErrorAlert, Field, Input, PageHeader, Select, Spinner } from '../components/ui'
import { formatDateTime, formatMoney, formatQuantity } from '../lib/format'
import { useApi } from '../lib/useApi'
import { useOptions } from '../lib/useOptions'

const PAGE_SIZE = 25
const FILTERS = ['product_id', 'location_id', 'type', 'reference', 'from', 'to'] as const

export function MovementHistory() {
  // Filters live in the URL so back/forward and shared links keep them.
  const [params, setParams] = useSearchParams()
  const { products, locations } = useOptions()
  const [draft, setDraft] = useState(() => Object.fromEntries(FILTERS.map((f) => [f, params.get(f) ?? ''])) as Record<(typeof FILTERS)[number], string>)
  const page = Number(params.get('page') ?? '1') || 1

  const filterQuery = new URLSearchParams()
  for (const f of FILTERS) if (params.get(f)) filterQuery.set(f, params.get(f)!)
  const { data, error, loading, reload } = useApi<PagedResponse<MovementDetail>>(`/api/movements?${filterQuery}&page=${page}&page_size=${PAGE_SIZE}`)

  function apply(e: FormEvent) {
    e.preventDefault()
    const next = new URLSearchParams()
    for (const f of FILTERS) if (draft[f].trim()) next.set(f, draft[f].trim())
    setParams(next)
  }

  function clear() {
    setDraft(Object.fromEntries(FILTERS.map((f) => [f, ''])) as typeof draft)
    setParams(new URLSearchParams())
  }

  const set = (f: (typeof FILTERS)[number]) => (value: string) => setDraft({ ...draft, [f]: value })
  const filtered = filterQuery.toString() !== ''

  return (
    <>
      <PageHeader
        title="Movement history"
        description="Every posted movement, newest first. Movements are never edited or deleted; mistakes are corrected with a reversal."
        actions={
          <a href={apiURL(`/api/movements?${filterQuery}${filterQuery.toString() ? '&' : ''}format=csv`)} className="rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100">
            Download CSV
          </a>
        }
      />

      <Card className="mb-4">
        <form onSubmit={apply} className="grid gap-3 sm:grid-cols-3 lg:grid-cols-6">
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
                  {t}
                </option>
              ))}
            </Select>
          </Field>
          <Field label="Reference">
            <Input value={draft.reference} onChange={(e) => set('reference')(e.target.value)} placeholder="Contains…" />
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
              <Button type="button" variant="secondary" onClick={clear}>
                Clear
              </Button>
            )}
          </div>
        </form>
      </Card>

      <ErrorAlert error={error} onRetry={reload} />
      {loading && !data ? (
        <Spinner />
      ) : data && data.data.length === 0 ? (
        <EmptyState title={filtered ? 'No movements match these filters' : 'No movements yet'}>{filtered ? 'Try widening the filters.' : 'Receive stock to create the first movement.'}</EmptyState>
      ) : (
        data && (
          <>
            <Table head={['#', 'Date (IST)', 'Type', 'Product', 'From → To', 'Quantity', 'Unit cost', 'Value', 'Reference / reason', 'By']} numeric={[5, 6, 7]}>
              {data.data.map((m) => (
                <tr key={m.id} className={m.is_reversed ? 'bg-slate-50 text-slate-500' : 'hover:bg-slate-50'}>
                  <Td>
                    <Link to={`/movements/${m.id}`} className="tabular-nums text-emerald-700 hover:underline">
                      #{m.id}
                    </Link>
                  </Td>
                  <Td>{formatDateTime(m.created_at)}</Td>
                  <Td>
                    <div className="flex items-center gap-1.5">
                      <MovementBadge type={m.type} />
                      {m.is_reversed && <span className="rounded border border-slate-300 px-1.5 py-0.5 text-xs">Reversed</span>}
                    </div>
                  </Td>
                  <Td>
                    <Link to={`/products/${m.product_id}`} className="font-mono hover:underline">
                      {m.sku}
                    </Link>
                  </Td>
                  <Td className="font-mono">
                    {m.from_location_code ?? '—'} → {m.to_location_code ?? '—'}
                  </Td>
                  <Td className="text-right tabular-nums">{formatQuantity(m.quantity, m.unit)}</Td>
                  <Td className="text-right tabular-nums">{formatMoney(m.unit_cost)}</Td>
                  <Td className="text-right tabular-nums">{formatMoney(m.total_value)}</Td>
                  <Td className="max-w-xs truncate">{[m.reference, m.reason?.replace('_', ' ')].filter(Boolean).join(' · ') || '—'}</Td>
                  <Td>{m.created_by}</Td>
                </tr>
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
