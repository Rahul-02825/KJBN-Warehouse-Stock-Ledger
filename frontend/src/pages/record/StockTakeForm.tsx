import { useState, type FormEvent } from 'react'
import { Link } from 'react-router'
import { ApiError, api } from '../../api/client'
import type { Location, LocationBalances, Movement, Product } from '../../api/types'
import { LocationSelect } from '../../components/MovementFields'
import { Delta } from '../../components/MovementBadge'
import { AsideHint, MovementForm, Step, Wide } from '../../components/MovementForm'
import { Row, Table, Td } from '../../components/Table'
import { Field, Input, Select, Spinner, SuccessAlert } from '../../components/ui'
import { loadCreatedBy, saveCreatedBy } from '../../lib/createdBy'
import { isZeroDecimal, subtractDecimal } from '../../lib/decimal'
import { formatDecimal, formatQuantity } from '../../lib/format'
import { useApi } from '../../lib/useApi'
import { checkDecimal, checkText } from '../../lib/validate'

interface Line {
  product: Product
  expected: string
}

interface Outcome {
  adjustments: Movement[]
  unchanged: number
}

export function StockTakeForm({ products, locations }: { products: Product[]; locations: Location[]; initialProduct: string }) {
  const [locationId, setLocationId] = useState('')
  const [counted, setCounted] = useState<Record<number, string>>({})
  const [added, setAdded] = useState<number[]>([])
  const [note, setNote] = useState('')
  const [createdBy, setCreatedBy] = useState(loadCreatedBy)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [serverError, setServerError] = useState<ApiError | null>(null)
  const [saving, setSaving] = useState(false)
  const [outcome, setOutcome] = useState<Outcome | null>(null)
  const [version, setVersion] = useState(0)

  const balances = useApi<LocationBalances>(locationId ? `/api/locations/${locationId}/balances?v=${version}` : null)
  const productById = (id: number) => products.find((p) => p.id === id)
  const lines: Line[] = [
    ...(balances.data?.data ?? []).filter((b) => !isZeroDecimal(b.quantity)).flatMap((b) => {
      const product = productById(b.product_id)
      return product ? [{ product, expected: b.quantity }] : []
    }),
    ...added.flatMap((id) => {
      const product = productById(id)
      return product ? [{ product, expected: '0' }] : []
    }),
  ]
  const listed = new Set(lines.map((l) => l.product.id))
  const lineError = (l: Line) => checkDecimal(counted[l.product.id] ?? '', l.product.unit, { allowZero: true, required: false })
  const entered = lines.filter((l) => (counted[l.product.id] ?? '').trim() !== '' && lineError(l) === '')
  const differences = entered.filter((l) => !isZeroDecimal(subtractDecimal(counted[l.product.id], l.expected)))

  function chooseBin(id: string) {
    setLocationId(id)
    setCounted({})
    setAdded([])
    setErrors({})
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    const found: Record<string, string> = {}
    if (!locationId) found.location_id = 'Choose the bin you counted'
    for (const l of lines) if (lineError(l)) found[`line-${l.product.id}`] = lineError(l)
    if (locationId && entered.length === 0) found.counts = 'Enter at least one counted quantity'
    const name = checkText(createdBy, 'Your name', 128)
    if (name) found.created_by = name
    setErrors(found)
    setServerError(null)
    if (Object.keys(found).length > 0) return

    setSaving(true)
    try {
      const result = await api.post<Outcome>('/api/stock-takes', {
        location_id: Number(locationId),
        counts: entered.map((l) => ({ product_id: l.product.id, counted: counted[l.product.id].trim() })),
        note: note.trim(),
        created_by: createdBy.trim(),
      })
      saveCreatedBy(createdBy.trim())
      setOutcome(result)
      setCounted({})
      setAdded([])
      setVersion((v) => v + 1)
    } catch (err) {
      setServerError(err as ApiError)
    } finally {
      setSaving(false)
    }
  }

  return (
    <>
      {outcome && (
        <SuccessAlert>
          {outcome.adjustments.length === 0
            ? `Everything matched. ${outcome.unchanged} counted, nothing to correct.`
            : `Posted ${outcome.adjustments.length} count ${outcome.adjustments.length === 1 ? 'correction' : 'corrections'}`}
          {outcome.adjustments.length > 0 && (
            <>
              {outcome.unchanged > 0 && `; ${outcome.unchanged} matched`}:{' '}
              {outcome.adjustments.map((m, i) => (
                <span key={m.id}>
                  {i > 0 && ', '}
                  <Link to={`/movements/${m.id}`} className="underline underline-offset-2">
                    #{m.id}
                  </Link>
                </span>
              ))}
              .
            </>
          )}
        </SuccessAlert>
      )}
      <MovementForm
        onSubmit={onSubmit}
        saving={saving}
        serverError={serverError}
        submitLabel={differences.length > 0 ? `Post ${differences.length} ${differences.length === 1 ? 'correction' : 'corrections'}` : 'Post count'}
        aside={
          !locationId ? (
            <AsideHint>Choose a bin, then type what you counted for each product. Only differences are posted.</AsideHint>
          ) : (
            <>
              <dl className="space-y-1.5">
                <div className="flex justify-between gap-3">
                  <dt className="text-muted">Counted</dt>
                  <dd className="num font-medium">
                    {entered.length} of {lines.length}
                  </dd>
                </div>
                <div className="flex justify-between gap-3">
                  <dt className="text-muted">Differences</dt>
                  <dd className="num font-medium">{differences.length}</dd>
                </div>
              </dl>
              {differences.length > 0 && (
                <ul className="divide-y divide-line border-y border-line">
                  {differences.map((l) => (
                    <li key={l.product.id} className="flex animate-rise items-baseline justify-between gap-3 py-2">
                      <span className="code">{l.product.sku}</span>
                      <Delta value={formatDecimal(subtractDecimal(counted[l.product.id], l.expected))} />
                    </li>
                  ))}
                </ul>
              )}
              <AsideHint>Each difference becomes a count-correction adjustment. Counting the same again later posts nothing new.</AsideHint>
            </>
          )
        }
      >
        <Step index="01" title="Which bin">
          <LocationSelect label="Bin you counted" locations={locations} value={locationId} onChange={chooseBin} error={errors.location_id} />
        </Step>
        <Step index="02" title="What you counted">
          <Wide>
            {!locationId ? (
              <p className="text-sm text-muted">Choose a bin to see what the system expects in it.</p>
            ) : balances.loading && !balances.data ? (
              <Spinner label="Loading the bin…" />
            ) : (
              <div className="space-y-4">
                {lines.length === 0 ? (
                  <p className="text-sm text-muted">The system thinks this bin is empty. Add any product you found in it below.</p>
                ) : (
                  <Table head={['Product', 'Expected', 'Counted', 'Difference']} numeric={[1, 3]}>
                    {lines.map((l) => {
                      const value = counted[l.product.id] ?? ''
                      const error = errors[`line-${l.product.id}`] ?? lineError(l)
                      const difference = value.trim() !== '' && !error ? subtractDecimal(value, l.expected) : null
                      return (
                        <Row key={l.product.id}>
                          <Td>
                            <span className="code">{l.product.sku}</span>
                            <p className="mt-0.5 text-[12.5px] text-muted">{l.product.name}</p>
                          </Td>
                          <Td numeric>{formatQuantity(l.expected, l.product.unit)}</Td>
                          <Td className="w-40">
                            <Input
                              value={value}
                              onChange={(e) => setCounted({ ...counted, [l.product.id]: e.target.value })}
                              inputMode="decimal"
                              placeholder="Not counted"
                              invalid={!!error}
                              aria-label={`Counted ${l.product.sku}`}
                            />
                            {error && <p className="mt-1 whitespace-normal text-[12.5px] text-danger">{error}</p>}
                          </Td>
                          <Td numeric>
                            {difference === null ? (
                              <span className="text-line">–</span>
                            ) : isZeroDecimal(difference) ? (
                              <span className="text-muted">Matches</span>
                            ) : (
                              <Delta value={formatDecimal(difference)} />
                            )}
                          </Td>
                        </Row>
                      )
                    })}
                  </Table>
                )}
                <div className="max-w-sm">
                  <Field label="Found something else in this bin?" hint="Adds it with an expected quantity of zero">
                    <Select
                      value=""
                      onChange={(e) => {
                        if (e.target.value) setAdded([...added, Number(e.target.value)])
                      }}
                    >
                      <option value="">Add a product…</option>
                      {products
                        .filter((p) => !listed.has(p.id))
                        .map((p) => (
                          <option key={p.id} value={p.id}>
                            {p.sku} · {p.name}
                          </option>
                        ))}
                    </Select>
                  </Field>
                </div>
                {errors.counts && <p className="text-[13px] text-danger">{errors.counts}</p>}
              </div>
            )}
          </Wide>
        </Step>
        <Step index="03" title="Who counted">
          <Field label="Note" hint="Optional, such as the count date or team">
            <Input value={note} onChange={(e) => setNote(e.target.value)} />
          </Field>
          <Field label="Your name" error={errors.created_by}>
            <Input value={createdBy} onChange={(e) => setCreatedBy(e.target.value)} invalid={!!errors.created_by} />
          </Field>
        </Step>
      </MovementForm>
    </>
  )
}
