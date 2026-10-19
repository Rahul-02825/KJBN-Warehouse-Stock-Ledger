import { X } from 'lucide-react'
import { icon } from './icon'
import { useEffect, useRef } from 'react'
import { Link } from 'react-router'
import type { Location, LocationBalances } from '../api/types'
import { formatDateTime, formatDecimal, formatMoney, formatQuantity } from '../lib/format'
import { fillText } from '../lib/layout'
import { useApi } from '../lib/useApi'
import { Row, Table, Td } from './Table'
import { ButtonLink, EmptyState, ErrorAlert, Spinner } from './ui'

export function BinContents({ bin, onClose }: { bin: Location; onClose: () => void }) {
  const dialog = useRef<HTMLDialogElement>(null)
  const { data, error, reload } = useApi<LocationBalances>(`/api/locations/${bin.id}/balances`)
  const held = (data?.data ?? []).filter((p) => !/^0+(\.0+)?$/.test(p.quantity))

  useEffect(() => {
    dialog.current?.showModal()
  }, [])

  return (
    <dialog
      ref={dialog}
      onClose={onClose}
      onClick={(e) => e.target === dialog.current && dialog.current?.close()}
      aria-labelledby="bin-contents-title"
      className="m-auto w-[min(720px,calc(100vw-32px))] rounded-panel border border-line bg-paper p-0 text-ink shadow-[0_40px_90px_-48px_rgba(14,42,40,.5)] backdrop:bg-ink/30 backdrop:backdrop-blur-[2px]"
    >
      <div className="max-h-[80vh] overflow-y-auto p-6 sm:p-8">
        <div className="mb-6 flex items-start justify-between gap-4">
          <div>
            <h2 id="bin-contents-title" className="font-serif text-[28px] leading-tight">
              <span className="code text-[24px]">{bin.code}</span>
            </h2>
            <p className="mt-1 text-sm text-muted">
              {[bin.description, usage(bin), bin.auto_place ? null : 'manual only'].filter(Boolean).join(', ')}
            </p>
          </div>
          <button
            onClick={() => dialog.current?.close()}
            className="inline-flex items-center gap-1.5 rounded-btn border border-line px-3 py-1.5 text-sm text-muted transition-colors duration-200 hover:border-petrol hover:text-ink"
          >
            <X {...icon} />
            Close
          </button>
        </div>

        <ErrorAlert error={error} onRetry={reload} />
        {!data ? (
          !error && <Spinner />
        ) : held.length === 0 ? (
          <EmptyState title="This bin is empty" action={<ButtonLink to="/record/receive">Receive stock</ButtonLink>}>
            Nothing is stored here right now.
          </EmptyState>
        ) : (
          <Table head={['Product', 'Quantity', 'Average cost', 'Last change (IST)']} numeric={[1, 2]}>
            {held.map((p) => (
              <Row key={p.product_id}>
                <Td>
                  <Link to={`/products/${p.product_id}`} className="link-quiet code">
                    {p.sku}
                  </Link>
                  <p className="mt-0.5 text-[12.5px] text-muted">{p.product_name}</p>
                </Td>
                <Td numeric className="font-medium">
                  {formatQuantity(p.quantity, p.unit)}
                </Td>
                <Td numeric className="text-muted">
                  {formatMoney(p.average_cost)}
                </Td>
                <Td className="text-muted">{formatDateTime(p.updated_at)}</Td>
              </Row>
            ))}
          </Table>
        )}
      </div>
    </dialog>
  )
}

function usage(bin: Location): string {
  const used = bin.used ?? '0'
  return bin.capacity === null ? `${formatDecimal(used)} units, no limit` : `${fillText(used, bin.capacity)} units`
}
