// Small shared building blocks so every page looks and behaves the same.
import type { ButtonHTMLAttributes, InputHTMLAttributes, ReactNode, SelectHTMLAttributes } from 'react'
import { ApiError } from '../api/client'

export function PageHeader({ title, description, actions }: { title: string; description?: string; actions?: ReactNode }) {
  return (
    <div className="mb-6 flex flex-wrap items-end justify-between gap-3">
      <div>
        <h1 className="text-2xl font-semibold">{title}</h1>
        {description && <p className="mt-1 text-sm text-slate-500">{description}</p>}
      </div>
      {actions && <div className="flex flex-wrap gap-2">{actions}</div>}
    </div>
  )
}

export function Card({ title, children, className = '' }: { title?: string; children: ReactNode; className?: string }) {
  return (
    <section className={`rounded-lg border border-slate-200 bg-white p-4 shadow-sm ${className}`}>
      {title && <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-slate-500">{title}</h2>}
      {children}
    </section>
  )
}

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & { variant?: 'primary' | 'secondary' | 'danger' }

export function Button({ variant = 'primary', className = '', ...props }: ButtonProps) {
  const styles = {
    primary: 'bg-emerald-700 text-white hover:bg-emerald-800',
    secondary: 'border border-slate-300 bg-white text-slate-700 hover:bg-slate-100',
    danger: 'bg-red-600 text-white hover:bg-red-700',
  }
  return (
    <button
      className={`rounded-md px-4 py-2 text-sm font-medium disabled:cursor-not-allowed disabled:opacity-50 ${styles[variant]} ${className}`}
      {...props}
    />
  )
}

// Field wraps an input with its label and the error message for that field.
export function Field({ label, error, hint, children }: { label: string; error?: string; hint?: string; children: ReactNode }) {
  return (
    <label className="block">
      <span className="mb-1 block text-sm font-medium text-slate-700">{label}</span>
      {children}
      {error ? <span className="mt-1 block text-sm text-red-600">{error}</span> : hint && <span className="mt-1 block text-xs text-slate-500">{hint}</span>}
    </label>
  )
}

const inputStyle =
  'w-full rounded-md border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-600 disabled:bg-slate-100'

export function Input({ invalid, className = '', ...props }: InputHTMLAttributes<HTMLInputElement> & { invalid?: boolean }) {
  return <input className={`${inputStyle} ${invalid ? 'border-red-500' : 'border-slate-300'} ${className}`} {...props} />
}

export function Select({ invalid, className = '', ...props }: SelectHTMLAttributes<HTMLSelectElement> & { invalid?: boolean }) {
  return <select className={`${inputStyle} bg-white ${invalid ? 'border-red-500' : 'border-slate-300'} ${className}`} {...props} />
}

export function Spinner({ label = 'Loading…' }: { label?: string }) {
  return (
    <div className="flex items-center gap-2 py-8 text-sm text-slate-500" role="status">
      <span className="h-4 w-4 animate-spin rounded-full border-2 border-slate-300 border-t-emerald-700" />
      {label}
    </div>
  )
}

export function EmptyState({ title, children }: { title: string; children?: ReactNode }) {
  return (
    <div className="rounded-lg border border-dashed border-slate-300 bg-white px-6 py-10 text-center">
      <p className="font-medium text-slate-700">{title}</p>
      {children && <div className="mt-2 text-sm text-slate-500">{children}</div>}
    </div>
  )
}

// Shows a server error in plain words; never raw JSON.
export function ErrorAlert({ error, onRetry }: { error: ApiError | Error | null; onRetry?: () => void }) {
  if (!error) return null
  const code = error instanceof ApiError ? error.code : undefined
  return (
    <div className="mb-4 flex items-start justify-between gap-4 rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800" role="alert">
      <div>
        <p className="font-medium">{error.message}</p>
        {code && code !== 'NETWORK' && <p className="mt-0.5 text-xs text-red-600">{code}</p>}
      </div>
      {onRetry && (
        <button className="shrink-0 font-medium underline" onClick={onRetry}>
          Retry
        </button>
      )}
    </div>
  )
}

export function SuccessAlert({ children }: { children: ReactNode }) {
  return <div className="mb-4 rounded-md border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-800" role="status">{children}</div>
}

export function StatCard({ label, value, tone = 'default' }: { label: string; value: ReactNode; tone?: 'default' | 'warning' | 'danger' }) {
  const tones = { default: 'text-slate-900', warning: 'text-amber-600', danger: 'text-red-600' }
  return (
    <div className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
      <p className="text-sm text-slate-500">{label}</p>
      <p className={`mt-1 text-2xl font-semibold ${tones[tone]}`}>{value}</p>
    </div>
  )
}
