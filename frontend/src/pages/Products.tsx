import { Plus, Search } from 'lucide-react'
import { icon } from '../components/icon'
import { useState, type FormEvent } from 'react'
import { Link } from 'react-router'
import type { PagedResponse, Product } from '../api/types'
import { Pagination } from '../components/Pagination'
import { ProductForm } from '../components/ProductForm'
import { SectionTabs } from '../components/SectionTabs'
import { catalogueTabs } from '../lib/navigation'
import { Row, Table, Td } from '../components/Table'
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

  function startEditing(target: Product | 'new') {
    setEditing(target)
    setMessage('')
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  function saved(p: Product) {
    setMessage(editing === 'new' ? `Created ${p.sku}.` : `Saved ${p.sku}.`)
    setEditing(null)
    reload()
  }

  return (
    <>
      <PageHeader
        eyebrow="Catalogue"
        title="Products"
        description="What the warehouse stores. The SKU and unit are fixed once created; the average cost changes only through receipts."
        actions={
          editing === null && (
            <Button onClick={() => startEditing('new')}>
              <Plus {...icon} />
              Add product
            </Button>
          )
        }
      />
      <SectionTabs tabs={catalogueTabs} label="Catalogue" />
      {message && <SuccessAlert>{message}</SuccessAlert>}
      {editing !== null && (
        <ProductForm key={editing === 'new' ? 'new' : editing.id} product={editing === 'new' ? undefined : editing} onSaved={saved} onCancel={() => setEditing(null)} />
      )}

      <form onSubmit={applySearch} className="mb-5 flex max-w-md gap-2">
        <Input placeholder="Search by SKU or name" value={searchInput} onChange={(e) => setSearchInput(e.target.value)} aria-label="Search products" />
        <Button type="submit" variant="ghost">
          <Search {...icon} />
          Search
        </Button>
      </form>

      <ErrorAlert error={error} onRetry={reload} />
      {loading && !data ? (
        <Spinner />
      ) : data && data.data.length === 0 ? (
        search ? (
          <EmptyState title={`Nothing matches “${search}”`}>Try part of the SKU or a different word from the name.</EmptyState>
        ) : (
          <EmptyState title="No products yet" action={
              <Button onClick={() => startEditing('new')}>
                <Plus {...icon} />
                Add product
              </Button>
            }>
            Add the first product, then receive stock into a bin.
          </EmptyState>
        )
      ) : (
        data && (
          <>
            <Table head={['SKU', 'Name', 'Unit', 'Reorder level', 'Average cost', 'Updated (IST)', '']} numeric={[3, 4]}>
              {data.data.map((p) => (
                <Row key={p.id}>
                  <Td>
                    <Link to={`/products/${p.id}`} className="link-quiet code">
                      {p.sku}
                    </Link>
                  </Td>
                  <Td>{p.name}</Td>
                  <Td className="text-muted">{p.unit}</Td>
                  <Td numeric>{formatDecimal(p.reorder_level)}</Td>
                  <Td numeric>{formatMoney(p.average_cost)}</Td>
                  <Td className="text-muted">{formatDateTime(p.updated_at)}</Td>
                  <Td>
                    <button className="text-sm text-muted transition-colors duration-200 hover:text-ink" onClick={() => startEditing(p)}>
                      Edit
                    </button>
                  </Td>
                </Row>
              ))}
            </Table>
            <Pagination page={page} pageSize={PAGE_SIZE} total={data.total} onPage={setPage} />
          </>
        )
      )}
    </>
  )
}
