import { useState, type FormEvent } from 'react'
import { Link } from 'react-router'
import type { MovementDetail } from '../api/types'
import { loadCreatedBy } from '../lib/createdBy'
import { useMovementSubmit } from '../lib/useMovementSubmit'
import { checkText, onlyErrors } from '../lib/validate'
import { Button, Card, Field, Input, SuccessAlert } from './ui'

// Posts the inverse of a movement. The original is never changed; it is only marked as reversed.
export function ReversePanel({ movement, onReversed }: { movement: MovementDetail; onReversed: () => void }) {
  const [open, setOpen] = useState(false)
  const [note, setNote] = useState('')
  const [createdBy, setCreatedBy] = useState(loadCreatedBy)
  const [reversalId, setReversalId] = useState<number | null>(null)
  const { submit, errors, serverError, saving, lastError } = useMovementSubmit(`/api/movements/${movement.id}/reverse`)

  // Explain up front when the server is going to refuse, but still let the user try.
  const expectedRejection = movement.is_reversed
    ? `Already reversed by movement #${movement.reversed_by_id}. The server will reject another reversal.`
    : movement.type === 'REVERSAL'
      ? 'This is itself a reversal. The server will reject reversing it.'
      : ''

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    const result = await submit(
      { note: note.trim(), created_by: createdBy.trim() },
      onlyErrors({ note: checkText(note, 'Note', 1000, false), created_by: checkText(createdBy, 'Your name', 128) }),
      movement.unit,
    )
    if (result) {
      setReversalId(result.id)
      setOpen(false)
      onReversed()
    }
  }

  // The API answers with a code; turn it into a sentence about this movement.
  let rejection = ''
  if (serverError?.code === 'ALREADY_REVERSED') rejection = `Rejected: movement #${movement.id} has already been reversed. A movement can be reversed only once.`
  else if (serverError?.code === 'NOT_REVERSIBLE') rejection = `Rejected: ${serverError.message}.`
  else if (lastError?.code === 'INSUFFICIENT_STOCK') rejection = `Rejected: reversing would make stock negative (${lastError.message}).`
  else if (serverError) rejection = serverError.message

  if (reversalId) {
    return (
      <SuccessAlert>
        Posted reversal{' '}
        <Link to={`/movements/${reversalId}`} className="font-semibold underline">
          movement #{reversalId}
        </Link>
        . The stock effect of movement #{movement.id} has been undone.
      </SuccessAlert>
    )
  }

  if (!open) {
    return (
      <div className="mb-6 flex flex-wrap items-center gap-3">
        <Button variant={expectedRejection ? 'secondary' : 'danger'} onClick={() => setOpen(true)}>
          {expectedRejection ? 'Try to reverse' : 'Reverse this movement'}
        </Button>
        {expectedRejection && <span className="text-sm text-slate-500">{expectedRejection}</span>}
      </div>
    )
  }

  return (
    <Card title={`Reverse movement #${movement.id}`} className="mb-6 border-red-200">
      <p className="mb-4 text-sm text-slate-600">
        This posts a new movement with the opposite effect. Movement #{movement.id} stays in history, marked as reversed.
        {movement.type === 'IN' && ' Reversing a receipt also removes it from the average cost.'}
      </p>
      {rejection && (
        <div className="mb-4 rounded-md border border-red-200 bg-red-50 p-3 text-sm font-medium text-red-800" role="alert">
          {rejection}
        </div>
      )}
      <form onSubmit={onSubmit} noValidate className="grid gap-4 sm:grid-cols-2">
        <Field label="Note" error={errors.note} hint="Why it is being reversed (optional)">
          <Input value={note} onChange={(e) => setNote(e.target.value)} invalid={!!errors.note} />
        </Field>
        <Field label="Your name" error={errors.created_by}>
          <Input value={createdBy} onChange={(e) => setCreatedBy(e.target.value)} invalid={!!errors.created_by} />
        </Field>
        <div className="flex gap-2 sm:col-span-2">
          <Button type="submit" variant="danger" disabled={saving}>
            {saving ? 'Posting…' : 'Confirm reversal'}
          </Button>
          <Button type="button" variant="secondary" onClick={() => setOpen(false)}>
            Cancel
          </Button>
        </div>
      </form>
    </Card>
  )
}
