import { Undo2 } from 'lucide-react'
import { icon } from './icon'
import { useState, type FormEvent } from 'react'
import { Link } from 'react-router'
import type { MovementDetail } from '../api/types'
import { loadCreatedBy } from '../lib/createdBy'
import { useMovementSubmit } from '../lib/useMovementSubmit'
import { checkText, onlyErrors } from '../lib/validate'
import { Button, Field, Input, SuccessAlert } from './ui'

export function ReversePanel({ movement, onReversed }: { movement: MovementDetail; onReversed: () => void }) {
  const [open, setOpen] = useState(false)
  const [note, setNote] = useState('')
  const [createdBy, setCreatedBy] = useState(loadCreatedBy)
  const [reversalId, setReversalId] = useState<number | null>(null)
  const { submit, errors, serverError, saving, lastError } = useMovementSubmit(`/api/movements/${movement.id}/reverse`)

  const expectedRejection = movement.is_reversed
    ? `Already reversed by movement #${movement.reversed_by_id}. Another reversal will be refused.`
    : movement.type === 'REVERSAL'
      ? 'This is itself a reversal, so it cannot be reversed.'
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

  let rejection = ''
  if (serverError?.code === 'ALREADY_REVERSED') rejection = `Refused: movement #${movement.id} has already been reversed. A movement can be reversed only once.`
  else if (serverError?.code === 'NOT_REVERSIBLE') rejection = `Refused: ${serverError.message}.`
  else if (lastError?.code === 'INSUFFICIENT_STOCK') rejection = `Refused: reversing would take stock below zero (${lastError.message}).`
  else if (serverError) rejection = serverError.message

  if (reversalId) {
    return (
      <SuccessAlert>
        Reversed. The effect of movement #{movement.id} has been undone by{' '}
        <Link to={`/movements/${reversalId}`} className="font-medium underline underline-offset-2">
          movement #{reversalId}
        </Link>
        .
      </SuccessAlert>
    )
  }

  return (
    <section className="border-t border-line pt-8">
      <h2 className="mb-1 font-serif text-[clamp(22px,2.6vw,30px)] leading-tight">Correct a mistake</h2>
      <p className="mb-5 max-w-2xl text-sm text-muted">
        Reversing posts a new movement with the opposite effect. Movement #{movement.id} stays in history, marked as reversed.
        {movement.type === 'IN' && ' Reversing a receipt also takes it out of the average cost.'}
      </p>

      {!open ? (
        <div className="flex flex-wrap items-center gap-4">
          <Button variant={expectedRejection ? 'ghost' : 'danger'} onClick={() => setOpen(true)}>
            <Undo2 {...icon} />
            {expectedRejection ? 'Try to reverse' : 'Reverse this movement'}
          </Button>
          {expectedRejection && <span className="text-sm text-muted">{expectedRejection}</span>}
        </div>
      ) : (
        <form onSubmit={onSubmit} noValidate className="max-w-2xl rounded-panel border border-danger/30 p-5">
          {rejection && (
            <p className="mb-5 rounded-btn bg-danger-soft px-4 py-3 text-sm font-medium text-danger" role="alert">
              {rejection}
            </p>
          )}
          <div className="grid gap-5 sm:grid-cols-2">
            <Field label="Why is it being reversed?" error={errors.note} hint="Optional">
              <Input value={note} onChange={(e) => setNote(e.target.value)} invalid={!!errors.note} />
            </Field>
            <Field label="Your name" error={errors.created_by}>
              <Input value={createdBy} onChange={(e) => setCreatedBy(e.target.value)} invalid={!!errors.created_by} />
            </Field>
          </div>
          <div className="mt-6 flex gap-2">
            <Button type="submit" variant="danger" disabled={saving}>
              {saving ? 'Reversing…' : 'Reverse movement'}
            </Button>
            <Button type="button" variant="ghost" onClick={() => setOpen(false)}>
              Cancel
            </Button>
          </div>
        </form>
      )}
    </section>
  )
}
