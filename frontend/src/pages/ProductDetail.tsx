import { Link, useParams } from 'react-router'
import type { ListResponse, LowStockRow, Product, ProductBalances, ProductLedger } from '../api/types'
import { Delta, MovementBadge } from '../components/MovementBadge'
import { Table, Td } from '../components/Table'
import { Card, EmptyState, ErrorAlert, PageHeader, Spinner, StatCard } from '../components/ui'
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
      <EmptyState title="Product not found">
        <Link to="/products" className="text-emerald-700 underline">
          Back to products
        </Link>
      </EmptyState>
    )
  }
  const error = product.error ?? balances.error ?? ledger.error ?? lowStock.error
  if (error) {
    return (
      <ErrorAlert
        error={error}
        onRetry={() => {
          product.reload()
          balances.reload()
          ledger.reload()
          lowStock.reload()
        }}
      />
    )
  }
  if (!product.data || !balances.data || !ledger.data || !lowStock.data) return <Spinner />

  const p = product.data
  const isLow = lowStock.data.data.some((row) => row.product_id === p.id)
  const bins = balances.data.data.filter((b) => !/^0+(\.0+)?$/.test(b.quantity))

  return (
    <>
      <Link to="/products" className="text-sm text-emerald-700 hover:underline">
        ← Products
      </Link>
      <PageHeader
        title={`${p.sku} · ${p.name}`}
        description={`Measured in ${p.unit}. Every quantity below is explained by the ledger at the bottom.`}
        actions={
          <>
            <Link to={`/receive?product=${p.id}`} className="rounded-md bg-emerald-700 px-4 py-2 text-sm font-medium text-white hover:bg-emerald-800">
              Receive
            </Link>
            <Link to={`/ship?product=${p.id}`} className="rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100">
              Ship
            </Link>
          </>
        }
      />

      <div className="mb-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard label="Total on hand" value={formatQuantity(balances.data.total_quantity, p.unit)} tone={isLow ? 'warning' : 'default'} />
        <StatCard label="Reorder level" value={formatQuantity(p.reorder_level, p.unit)} />
        <StatCard label="Average cost" value={formatMoney(p.average_cost)} />
        <StatCard label="Status" value={isLow ? 'Low stock' : 'OK'} tone={isLow ? 'warning' : 'default'} />
      </div>

      <Card title="Stock by bin" className="mb-6">
        {bins.length === 0 ? (
          <p className="text-sm text-slate-500">No stock in any bin.</p>
        ) : (
          <div className="flex flex-wrap gap-3">
            {bins.map((b) => (
              <div key={b.location_id} className="rounded-md border border-slate-200 px-3 py-2">
                <p className="font-mono text-xs text-slate-500">{b.location_code}</p>
                <p className="font-semibold tabular-nums">{formatQuantity(b.quantity, p.unit)}</p>
              </div>
            ))}
          </div>
        )}
      </Card>

      <h2 className="mb-1 text-lg font-semibold">Ledger</h2>
      <p className="mb-3 text-sm text-slate-500">Oldest first. Each row is one stock change; running totals show the balance right after it.</p>
      {ledger.data.data.length === 0 ? (
        <EmptyState title="No movements yet">Receive stock to start this product's ledger.</EmptyState>
      ) : (
        <Table head={['Date (IST)', 'Movement', 'Type', 'Bin', 'Change', 'Unit cost', 'Bin balance', 'Total balance', 'Reference', 'By']} numeric={[4, 5, 6, 7]}>
          {ledger.data.data.map((e, i) => (
            <tr key={i} className="hover:bg-slate-50">
              <Td className="text-slate-500">{formatDateTime(e.created_at)}</Td>
              <Td className="tabular-nums">#{e.movement_id}</Td>
              <Td>
                <MovementBadge type={e.type} />
              </Td>
              <Td className="font-mono">{e.location_code}</Td>
              <Td className="text-right">
                <Delta value={formatDecimal(e.quantity_delta)} />
              </Td>
              <Td className="text-right tabular-nums">{formatMoney(e.unit_cost)}</Td>
              <Td className="text-right tabular-nums">{formatDecimal(e.running_location_balance)}</Td>
              <Td className="text-right font-semibold tabular-nums">{formatDecimal(e.running_total)}</Td>
              <Td className="text-slate-600">{[e.reference, e.reason?.replace('_', ' ')].filter(Boolean).join(' · ') || '—'}</Td>
              <Td className="text-slate-500">{e.created_by}</Td>
            </tr>
          ))}
        </Table>
      )}
    </>
  )
}
