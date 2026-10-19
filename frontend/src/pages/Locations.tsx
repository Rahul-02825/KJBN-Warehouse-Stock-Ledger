import { useState } from 'react'
import type { ListResponse, Location, Warehouse } from '../api/types'
import { BinContents } from '../components/BinContents'
import { LocationForm } from '../components/LocationForm'
import { SectionTabs } from '../components/SectionTabs'
import { catalogueTabs } from '../lib/navigation'
import { Row, Table, Td } from '../components/Table'
import { Badge, Button, EmptyState, ErrorAlert, PageHeader, Spinner, Stat, SuccessAlert } from '../components/ui'
import { distanceFromCentre, fillPercent, fillText } from '../lib/layout'
import { useApi } from '../lib/useApi'

export function Locations() {
  const [creating, setCreating] = useState(false)
  const [message, setMessage] = useState('')
  const [opened, setOpened] = useState<Location | null>(null)
  const locations = useApi<ListResponse<Location>>('/api/locations')
  const warehouses = useApi<ListResponse<Warehouse>>('/api/warehouses')
  const warehouse = warehouses.data?.data[0]
  const error = locations.error ?? warehouses.error

  function saved(l: Location) {
    setMessage(`Added bin ${l.code}.`)
    setCreating(false)
    locations.reload()
    warehouses.reload()
  }

  return (
    <>
      <PageHeader
        eyebrow="Catalogue"
        title="Bins"
        description="Where stock is kept. Every piece, kilogram or litre takes one unit of space. Select a bin to see what it holds."
        actions={
          !creating &&
          warehouse && (
            <Button
              onClick={() => {
                setCreating(true)
                setMessage('')
              }}
            >
              Add bin
            </Button>
          )
        }
      />
      <SectionTabs tabs={catalogueTabs} label="Catalogue" />

      {warehouse && (
        <div className="mb-10 grid gap-6 sm:grid-cols-3">
          <Stat label="Warehouse" value={warehouse.name} />
          <Stat label="Perimeter" value={`${warehouse.radius_m} m`} />
          <Stat label="Units stored" value={fillText(warehouse.used, warehouse.capacity)} />
        </div>
      )}
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
        <EmptyState title="No bins yet" action={<Button onClick={() => setCreating(true)}>Add bin</Button>}>
          Add a bin before receiving stock.
        </EmptyState>
      ) : (
        <Table head={['Code', 'Description', 'Position (m)', 'From centre', 'Fill', 'Placement']} numeric={[3]}>
          {locations.data.data.map((l) => (
            <Row key={l.id}>
              <Td>
                <button onClick={() => setOpened(l)} className="link-quiet code" aria-haspopup="dialog">
                  {l.code}
                </button>
              </Td>
              <Td className="whitespace-normal text-muted">{l.description || '–'}</Td>
              <Td className="num text-muted">
                {l.x}, {l.y}
              </Td>
              <Td numeric>{distanceFromCentre(warehouse, l.x, l.y).toFixed(1)} m</Td>
              <Td>
                <FillBar used={l.used ?? '0'} capacity={l.capacity} />
              </Td>
              <Td>{l.auto_place ? <span className="text-muted">Automatic</span> : <Badge tone="neutral">Manual only</Badge>}</Td>
            </Row>
          ))}
        </Table>
      )}
      {opened && <BinContents key={opened.id} bin={opened} onClose={() => setOpened(null)} />}
    </>
  )
}

function FillBar({ used, capacity }: { used: string; capacity: number | null }) {
  const percent = fillPercent(used, capacity)
  const color = percent >= 100 ? 'bg-danger' : percent >= 80 ? 'bg-saffron' : 'bg-petrol'
  return (
    <div className="min-w-36">
      <p className="num">{fillText(used, capacity)}</p>
      {capacity !== null && (
        <div className="mt-1.5 h-1 rounded-full bg-paper-2" role="img" aria-label={`${percent}% full`}>
          <div className={`h-1 rounded-full ${color}`} style={{ width: `${percent}%` }} />
        </div>
      )}
    </div>
  )
}
