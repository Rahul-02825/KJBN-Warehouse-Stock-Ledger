import type { FormEvent, ReactNode } from 'react'
import type { ApiError } from '../api/client'
import { Button, Card, ErrorAlert } from './ui'

export function MovementFormCard({
  onSubmit,
  saving,
  serverError,
  submitLabel,
  children,
}: {
  onSubmit: (e: FormEvent) => void
  saving: boolean
  serverError: ApiError | null
  submitLabel: string
  children: ReactNode
}) {
  return (
    <Card className="max-w-3xl">
      <ErrorAlert error={serverError} />
      <form onSubmit={onSubmit} noValidate className="grid gap-4 sm:grid-cols-2">
        {children}
        <div className="sm:col-span-2">
          <Button type="submit" disabled={saving}>
            {saving ? 'Posting…' : submitLabel}
          </Button>
        </div>
      </form>
    </Card>
  )
}
