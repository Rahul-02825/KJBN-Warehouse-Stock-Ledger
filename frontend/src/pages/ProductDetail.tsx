import { ArrowLeft } from 'lucide-react'
import { icon } from '../components/icon'
import { movementIcons } from '../lib/navigation'
import { Link, useParams } from 'react-router'
import type { ListResponse, LowStockRow, Product, ProductBalances, ProductLedger } from '../api/types'
import { CsvLink } from '../components/CsvLink'
import { Delta, MovementBadge } from '../components/MovementBadge'
import { Row, Table, Td } from '../components/Table'
import { ButtonLink, EmptyState, ErrorAlert, Eyebrow, Spinner, Stat } from '../components/ui'
import { formatDateTime, formatDecimal, formatMoney, formatQuantity } from '../lib/format'
import { useApi } from '../lib/useApi'

export function ProductDetail() {
  const { id } = useParams()
  const product = useApi<Product>(`/api/products/${id}`)
  const balances = useApi<ProductBalances>(`/api/products/${id}/balances`)
  const ledger = useApi<ProductLedger>(`/api/products/${id}/ledger`)
  const lowStock = useApi<ListResponse<LowStockRow>>('/api/reports/low-stock')

  if (product.error?.code === 'NOT_FOUND' || product.error?.code === 'BAD_REQUEST') {
    return (
      <EmptyState title="Product not found" action={<Link to="/products" className="link text-sm">Back to products</Link>}>
        Check the link, or find the product in the catalogue.
      </EmptyState>
    )
  }
  const all = [product, balances, ledger, lowStock]
  const error = all.find((r) => r.error)?.error ?? null
  if (error) return <ErrorAlert error={error} onRetry={() => all.forEach((r) => r.reload())} />
  if (!product.data || !balances.data || !ledger.data || !lowStock.data) return <Spinner />

  const p = product.data
  const isLow = lowStock.data.data.some((row) => row.product_id === p.id)
  const bins = balances.data.data.filter((b) => !/^0+(\.0+)?$/.test(b.quantity))

  return (
    <>
      <Link to="/products" className="inline-flex items-center gap-1.5 text-sm text-muted transition-colors duration-200 hover:text-ink">
        <ArrowLeft {...icon} />
        Products
      </Link>
      <header className="mb-10 mt-6 flex flex-wrap items-end justify-between gap-6">
        <div>
          <Eyebrow className="mb-4">{p.sku}</Eyebrow>
          <h1 className="text-[clamp(30px,4.4vw,48px)] font-semibold leading-[1.06] tracking-[-0.025em]">{p.name}</h1>
          <p className="mt-3 text-[15.5px] text-muted">Counted in {p.unit}. Every number on this page is explained by the ledger below.</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <ButtonLink to={`/record/receive?product=${p.id}`}>
            <movementIcons.receive {...icon} />
            Receive
          </ButtonLink>
          <ButtonLink to={`/record/ship?product=${p.id}`} variant="ghost">
            <movementIcons.ship {...icon} />
            Ship
          </ButtonLink>
          <ButtonLink to={`/record/transfer?product=${p.id}`} variant="ghost">
            <movementIcons.transfer {...icon} />
            Transfer
          </ButtonLink>
          <ButtonLink to={`/record/adjust?product=${p.id}`} variant="ghost">
            <movementIcons.adjust {...icon} />
            Adjust
          </ButtonLink>
        </div>
      </header>

      <div className="mb-12 grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
        <Stat label="On hand" value={formatQuantity(balances.data.total_quantity, p.unit)} tone={isLow ? 'warning' : 'default'} />
        <Stat label="Reorder level" value={formatQuantity(p.reorder_level, p.unit)} />
        <Stat label="Average cost" value={formatMoney(p.average_cost)} />
        <Stat label="Status" value={isLow ? 'Reorder now' : 'In stock'} tone={isLow ? 'warning' : 'success'} />
      </div>

      <section className="mb-12">
        <h2 className="mb-4 font-serif text-[clamp(22px,2.6vw,30px)] leading-tight">Where it is</h2>
        {bins.length === 0 ? (
          <p className="text-muted">Not in any bin right now.</p>
        ) : (
          <ul className="flex flex-wrap gap-3">
            {bins.map((b) => (
              <li key={b.location_id} className="min-w-32 rounded-panel border border-line px-4 py-3">
                <p className="code text-muted">{b.location_code}</p>
                <p className="num mt-0.5 font-serif text-xl">{formatQuantity(b.quantity, p.unit)}</p>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section>
        <div className="mb-4 flex flex-wrap items-end justify-between gap-3">
          <div>
            <h2 className="mb-1 font-serif text-[clamp(22px,2.6vw,30px)] leading-tight">Ledger</h2>
            <p className="text-sm text-muted">Oldest first. Each row is one change; the balances show what was left right after it.</p>
          </div>
          {ledger.data.data.length > 0 && <CsvLink path={`/api/products/${p.id}/ledger?format=csv`} />}
        </div>
        {ledger.data.data.length === 0 ? (
          <EmptyState title="No movements yet" action={<ButtonLink to={`/record/receive?product=${p.id}`}>Receive stock</ButtonLink>}>
            The ledger starts with the first receipt.
          </EmptyState>
        ) : (
          <Table head={['Posted (IST)', 'No.', 'Type', 'Bin', 'Change', 'Unit cost', 'Bin balance', 'Total', 'Reference', 'By']} numeric={[4, 5, 6, 7]}>
            {ledger.data.data.map((e, i) => (
              <Row key={i}>
                <Td className="text-muted">{formatDateTime(e.created_at)}</Td>
                <Td>
                  <Link to={`/movements/${e.movement_id}`} className="link-quiet code">
                    #{e.movement_id}
                  </Link>
                </Td>
                <Td>
                  <MovementBadge type={e.type} />
                </Td>
                <Td className="code">{e.location_code}</Td>
                <Td numeric>
                  <Delta value={formatDecimal(e.quantity_delta)} />
                </Td>
                <Td numeric className="text-muted">
                  {formatMoney(e.unit_cost)}
                </Td>
                <Td numeric>{formatDecimal(e.running_location_balance)}</Td>
                <Td numeric className="font-medium">
                  {formatDecimal(e.running_total)}
                </Td>
                <Td className="max-w-56 truncate text-muted">{[e.reference, e.reason?.replace('_', ' ')].filter(Boolean).join(', ') || '–'}</Td>
                <Td className="text-muted">{e.created_by}</Td>
              </Row>
            ))}
          </Table>
        )}
      </section>
    </>
  )
}
