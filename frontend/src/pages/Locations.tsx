import { useState } from 'react'
import type { ListResponse, Location } from '../api/types'
import { LocationForm } from '../components/LocationForm'
import { Table, Td } from '../components/Table'
import { Button, EmptyState, ErrorAlert, PageHeader, Spinner, SuccessAlert } from '../components/ui'
import { formatDateTime } from '../lib/format'
import { useApi } from '../lib/useApi'

export function Locations() {
  const [creating, setCreating] = useState(false)
  const [message, setMessage] = useState('')
  const { data, error, loading, reload } = useApi<ListResponse<Location>>('/api/locations')

  function saved(l: Location) {
    setMessage(`Created bin ${l.code}.`)
    setCreating(false)
    reload()
  }

  return (
    <>
      <PageHeader
        title="Locations"
        description="Warehouse bins. A product can be stored in several bins."
        actions={
          !creating && (
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
      {message && <SuccessAlert>{message}</SuccessAlert>}
      {creating && <LocationForm onSaved={saved} onCancel={() => setCreating(false)} />}

      <ErrorAlert error={error} onRetry={reload} />
      {loading && !data ? (
        <Spinner />
      ) : data && data.data.length === 0 ? (
        <EmptyState title="No bins yet">Create a bin before receiving stock.</EmptyState>
      ) : (
        data && (
          <Table head={['Code', 'Description', 'Created']}>
            {data.data.map((l) => (
              <tr key={l.id} className="hover:bg-slate-50">
                <Td className="font-mono">{l.code}</Td>
                <Td className="whitespace-normal">{l.description || <span className="text-slate-400">—</span>}</Td>
                <Td className="text-slate-500">{formatDateTime(l.created_at)}</Td>
              </tr>
            ))}
          </Table>
        )
      )}
    </>
  )
}
