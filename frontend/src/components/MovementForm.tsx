import type { FormEvent, ReactNode } from 'react'
import type { ApiError } from '../api/client'
import { Button, ErrorAlert } from './ui'

export function MovementForm({
  onSubmit,
  saving,
  serverError,
  submitLabel,
  aside,
  children,
}: {
  onSubmit: (e: FormEvent) => void
  saving: boolean
  serverError: ApiError | null
  submitLabel: string
  aside: ReactNode
  children: ReactNode
}) {
  return (
    <div className="grid gap-10 lg:grid-cols-[minmax(0,1fr)_300px]">
      <form onSubmit={onSubmit} noValidate>
        <ErrorAlert error={serverError} />
        {children}
        <div className="border-t border-line pt-6">
          <Button type="submit" disabled={saving}>
            {saving ? 'Posting…' : submitLabel}
          </Button>
        </div>
      </form>
      <aside className="self-start lg:sticky lg:top-[96px]">
        <div className="rounded-panel bg-paper-2 p-5">
          <h2 className="mb-3 font-serif text-lg">Before you post</h2>
          <div className="space-y-4 text-sm">{aside}</div>
        </div>
      </aside>
    </div>
  )
}

export function Step({ index, title, children }: { index: string; title: string; children: ReactNode }) {
  return (
    <section className="grid gap-x-8 gap-y-4 border-t border-line pb-8 pt-6 sm:grid-cols-[140px_minmax(0,1fr)]">
      <div>
        <span className="font-mono text-[12px] tracking-[0.08em] text-saffron-ink">{index}</span>
        <h3 className="mt-1 font-serif text-[20px] leading-tight">{title}</h3>
      </div>
      <div className="grid gap-5 sm:grid-cols-2">{children}</div>
    </section>
  )
}

export function Wide({ children }: { children: ReactNode }) {
  return <div className="sm:col-span-2">{children}</div>
}

export function AsideHint({ children }: { children: ReactNode }) {
  return <p className="text-muted">{children}</p>
}
