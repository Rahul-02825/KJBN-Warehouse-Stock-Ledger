import { useState, type FormEvent } from 'react'
import type { PagedResponse, Product } from '../api/types'
import { Pagination } from '../components/Pagination'
import { ProductForm } from '../components/ProductForm'
import { Table, Td } from '../components/Table'
import { Button, EmptyState, ErrorAlert, Input, PageHeader, Spinner, SuccessAlert } from '../components/ui'
import { formatDateTime, formatDecimal, formatMoney } from '../lib/format'
import { useApi } from '../lib/useApi'

const PAGE_SIZE = 20

export function Products() {
  const [page, setPage] = useState(1)
  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const [editing, setEditing] = useState<Product | 'new' | null>(null)
  const [message, setMessage] = useState('')

  const query = new URLSearchParams({ page: String(page), page_size: String(PAGE_SIZE) })
  if (search) query.set('search', search)
  const { data, error, loading, reload } = useApi<PagedResponse<Product>>(`/api/products?${query}`)

  function applySearch(e: FormEvent) {
    e.preventDefault()
    setMessage('')
    setPage(1)
    setSearch(searchInput.trim())
  }

  function saved(p: Product) {
    setMessage(editing === 'new' ? `Created ${p.sku}.` : `Saved ${p.sku}.`)
    setEditing(null)
    reload()
  }

  return (
    <>
      <PageHeader
        title="Products"
        description="SKU and unit are fixed once created. Average cost changes only through stock receipts."
        actions={
          editing === null && (
            <Button
              onClick={() => {
                setEditing('new')
                setMessage('')
              }}
            >
              New product
            </Button>
          )
        }
      />
      {message && <SuccessAlert>{message}</SuccessAlert>}
      {editing !== null && (
        <ProductForm key={editing === 'new' ? 'new' : editing.id} product={editing === 'new' ? undefined : editing} onSaved={saved} onCancel={() => setEditing(null)} />
      )}

      <form onSubmit={applySearch} className="mb-4 flex max-w-md gap-2">
        <Input placeholder="Search SKU or name" value={searchInput} onChange={(e) => setSearchInput(e.target.value)} aria-label="Search products" />
        <Button type="submit" variant="secondary">
          Search
        </Button>
      </form>

      <ErrorAlert error={error} onRetry={reload} />
      {loading && !data ? (
        <Spinner />
      ) : data && data.data.length === 0 ? (
        <EmptyState title={search ? `No products match "${search}"` : 'No products yet'}>
          {search ? 'Try a different search.' : 'Create your first product to start receiving stock.'}
        </EmptyState>
      ) : (
        data && (
          <>
            <Table head={['SKU', 'Name', 'Unit', 'Reorder level', 'Average cost', 'Updated', '']} numeric={[3, 4]}>
              {data.data.map((p) => (
                <tr key={p.id} className="hover:bg-slate-50">
                  <Td className="font-mono">{p.sku}</Td>
                  <Td>{p.name}</Td>
                  <Td>{p.unit}</Td>
                  <Td className="text-right tabular-nums">{formatDecimal(p.reorder_level)}</Td>
                  <Td className="text-right tabular-nums">{formatMoney(p.average_cost)}</Td>
                  <Td className="text-slate-500">{formatDateTime(p.updated_at)}</Td>
                  <Td>
                    <button
                      className="text-emerald-700 hover:underline"
                      onClick={() => {
                        setEditing(p)
                        setMessage('')
                        window.scrollTo({ top: 0, behavior: 'smooth' })
                      }}
                    >
                      Edit
                    </button>
                  </Td>
                </tr>
              ))}
            </Table>
            <Pagination page={page} pageSize={PAGE_SIZE} total={data.total} onPage={setPage} />
          </>
        )
      )}
    </>
  )
}
