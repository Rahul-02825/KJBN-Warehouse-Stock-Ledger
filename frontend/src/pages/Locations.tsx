import { useState } from 'react'
import type { ListResponse, Location, Warehouse } from '../api/types'
import { LocationForm } from '../components/LocationForm'
import { Table, Td } from '../components/Table'
import { Button, EmptyState, ErrorAlert, PageHeader, Spinner, StatCard, SuccessAlert } from '../components/ui'
import { formatDateTime } from '../lib/format'
import { distanceFromCentre, fillPercent, fillText } from '../lib/layout'
import { useApi } from '../lib/useApi'

export function Locations() {
  const [creating, setCreating] = useState(false)
  const [message, setMessage] = useState('')
  const locations = useApi<ListResponse<Location>>('/api/locations')
  const warehouses = useApi<ListResponse<Warehouse>>('/api/warehouses')
  const warehouse = warehouses.data?.data[0]

  function saved(l: Location) {
    setMessage(`Created bin ${l.code}.`)
    setCreating(false)
    locations.reload()
    warehouses.reload()
  }

  const error = locations.error ?? warehouses.error
  return (
    <>
      <PageHeader
        title="Locations"
        description="Warehouse bins with their position and capacity. Capacity counts items of pcs products."
        actions={
          !creating &&
          warehouse && (
            <Button
              onClick={() => {
                setCreating(true)
                setMessage('')
              }}
            >
              New bin
            </Button>
          )
        }
      />
      {warehouse && <WarehouseSummary warehouse={warehouse} />}
      {message && <SuccessAlert>{message}</SuccessAlert>}
      {creating && warehouse && <LocationForm warehouse={warehouse} onSaved={saved} onCancel={() => setCreating(false)} />}

      <ErrorAlert
        error={error}
        onRetry={() => {
          locations.reload()
          warehouses.reload()
        }}
      />
      {!locations.data || !warehouse ? (
        !error && <Spinner />
      ) : locations.data.data.length === 0 ? (
        <EmptyState title="No bins yet">Create a bin before receiving stock.</EmptyState>
      ) : (
        <Table head={['Code', 'Description', 'Position (m)', 'From centre', 'Fill (pcs)', 'Placement', 'Created']} numeric={[3]}>
          {locations.data.data.map((l) => (
            <tr key={l.id} className="hover:bg-slate-50">
              <Td className="font-mono">{l.code}</Td>
              <Td className="whitespace-normal">{l.description || <span className="text-slate-400">—</span>}</Td>
              <Td className="tabular-nums">
                ({l.x}, {l.y})
              </Td>
              <Td className="text-right tabular-nums">{distanceFromCentre(warehouse, l.x, l.y).toFixed(1)} m</Td>
              <Td>
                <FillBar used={l.used ?? 0} capacity={l.capacity} />
              </Td>
              <Td>{l.auto_place ? 'Automatic' : <span className="text-slate-500">Manual only</span>}</Td>
              <Td className="text-slate-500">{formatDateTime(l.created_at)}</Td>
            </tr>
          ))}
        </Table>
      )}
    </>
  )
}

function WarehouseSummary({ warehouse }: { warehouse: Warehouse }) {
  return (
    <div className="mb-6 grid gap-4 sm:grid-cols-3">
      <StatCard label="Warehouse" value={`${warehouse.code} · ${warehouse.name}`} />
      <StatCard label="Perimeter" value={`${warehouse.radius_m} m around (${warehouse.x}, ${warehouse.y})`} />
      <StatCard label="Items stored" value={fillText(warehouse.used, warehouse.capacity)} />
    </div>
  )
}

function FillBar({ used, capacity }: { used: number; capacity: number | null }) {
  const percent = fillPercent(used, capacity)
  const color = percent >= 100 ? 'bg-red-500' : percent >= 80 ? 'bg-amber-500' : 'bg-emerald-500'
  return (
    <div className="min-w-36">
      <p className="tabular-nums">{fillText(used, capacity)}</p>
      {capacity !== null && (
        <div className="mt-1 h-1.5 rounded bg-slate-100">
          <div className={`h-1.5 rounded ${color}`} style={{ width: `${percent}%` }} />
        </div>
      )}
    </div>
  )
}
