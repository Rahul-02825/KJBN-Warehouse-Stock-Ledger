import { RefreshCw, ScrollText } from 'lucide-react'
import { useState } from 'react'
import type { LogsResponse } from '../../api/types'
import { icon } from '../../components/icon'
import { Row, Table, Td } from '../../components/Table'
import { Badge, Button, EmptyState, ErrorAlert, Field, Input, Select, Spinner } from '../../components/ui'
import { formatDateTime } from '../../lib/format'
import { logDetails } from '../../lib/logs'
import { useApi } from '../../lib/useApi'
import { ReportHeader } from './ReportHeader'

const levelTone = { INFO: 'neutral', WARN: 'warning', ERROR: 'danger' } as const

export function Logs() {
  const [level, setLevel] = useState('')
  const [requestId, setRequestId] = useState('')
  const query = new URLSearchParams({ limit: '200' })
  if (level) query.set('level', level)
  if (requestId.trim()) query.set('request_id', requestId.trim())
  const { data, error, loading, reload } = useApi<LogsResponse>(`/api/logs?${query}`)

  return (
    <>
      <ReportHeader
        title="Logs"
        description="The API's most recent log lines, newest first. Every response carries an X-Request-ID; search for it here to see what happened to that request."
        actions={
          <Button variant="ghost" onClick={reload} disabled={loading}>
            <RefreshCw {...icon} className={loading ? 'animate-spin' : ''} />
            Refresh
          </Button>
        }
      />
      <div className="mb-6 grid gap-4 sm:grid-cols-[220px_320px]">
        <Field label="Show">
          <Select value={level} onChange={(e) => setLevel(e.target.value)}>
            <option value="">Everything</option>
            <option value="WARN">Warnings and errors</option>
            <option value="ERROR">Errors only</option>
          </Select>
        </Field>
        <Field label="Request id" hint="Paste an X-Request-ID, or click one in the table">
          <Input value={requestId} onChange={(e) => setRequestId(e.target.value)} placeholder="e.g. 7f3c2a9b1d4e5f60" />
        </Field>
      </div>
      <ErrorAlert error={error} onRetry={reload} />
      {!data ? (
        !error && <Spinner />
      ) : data.data.length === 0 ? (
        <EmptyState title="No log lines" Icon={ScrollText}>
          Nothing matches yet. The server keeps its last {data.capacity} lines in memory, so the list starts empty after a restart.
        </EmptyState>
      ) : (
        <Table head={['Time', 'Level', 'Message', 'Request id', 'Details']}>
          {data.data.map((line, i) => (
            <Row key={`${line.time}-${i}`} tone={line.level === 'ERROR' ? 'danger' : 'default'}>
              <Td className="whitespace-nowrap text-muted">{formatDateTime(line.time)}</Td>
              <Td>
                <Badge tone={levelTone[line.level as keyof typeof levelTone] ?? 'default'}>{line.level}</Badge>
              </Td>
              <Td className="whitespace-nowrap">{line.msg}</Td>
              <Td>
                {line.request_id && (
                  <button type="button" className="link-quiet code text-[12px]" onClick={() => setRequestId(line.request_id ?? '')}>
                    {line.request_id}
                  </button>
                )}
              </Td>
              <Td className="code min-w-[300px] max-w-[460px] whitespace-normal! break-words text-[12px] text-muted">
                {logDetails(line).map(([key, value]) => (
                  <span key={key} className="mr-3 inline-block max-w-full align-top">
                    {key}=<span className="text-ink">{value}</span>
                  </span>
                ))}
              </Td>
            </Row>
          ))}
        </Table>
      )}
      <p className="mt-6 max-w-2xl text-sm text-muted">
        Only the last {data?.capacity ?? 500} lines are kept, and they reset when the server restarts or wakes from sleep. The hosting
        provider keeps the full history.
      </p>
    </>
  )
}
