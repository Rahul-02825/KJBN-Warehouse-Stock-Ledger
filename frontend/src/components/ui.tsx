import type { ButtonHTMLAttributes, InputHTMLAttributes, ReactNode, SelectHTMLAttributes } from 'react'
import { Link, type LinkProps } from 'react-router'
import { ApiError } from '../api/client'

export function Eyebrow({ children, className = '' }: { children: ReactNode; className?: string }) {
  return (
    <p className={`flex items-center gap-3 font-mono text-[11.5px] uppercase tracking-[0.2em] text-muted ${className}`}>
      <span className="h-px w-[30px] bg-saffron" aria-hidden />
      {children}
    </p>
  )
}

export function PageHeader({ eyebrow, title, description, actions }: { eyebrow?: string; title: string; description?: string; actions?: ReactNode }) {
  return (
    <header className="mb-10 flex flex-wrap items-end justify-between gap-6">
      <div className="max-w-2xl">
        {eyebrow && <Eyebrow className="mb-4">{eyebrow}</Eyebrow>}
        <h1 className="text-[clamp(30px,4.4vw,48px)] font-semibold leading-[1.06] tracking-[-0.025em]">{title}</h1>
        {description && <p className="mt-3 text-[15.5px] text-muted">{description}</p>}
      </div>
      {actions && <div className="flex flex-wrap gap-2">{actions}</div>}
    </header>
  )
}

export function SectionHeading({ title, description, action }: { title: string; description?: string; action?: ReactNode }) {
  return (
    <div className="mb-4 flex flex-wrap items-end justify-between gap-3">
      <div>
        <h2 className="font-serif text-[clamp(22px,2.6vw,30px)] leading-[1.16] tracking-[-0.012em]">{title}</h2>
        {description && <p className="mt-1 text-sm text-muted">{description}</p>}
      </div>
      {action}
    </div>
  )
}

export function Panel({ title, children, className = '' }: { title?: string; children: ReactNode; className?: string }) {
  return (
    <section className={`rounded-panel border border-line bg-paper p-5 sm:p-6 ${className}`}>
      {title && <h3 className="mb-4 font-serif text-xl font-medium leading-[1.24] tracking-[-0.005em]">{title}</h3>}
      {children}
    </section>
  )
}

export function FeaturePanel({ children, className = '' }: { children: ReactNode; className?: string }) {
  return (
    <section className={`rounded-panel bg-petrol px-6 py-7 text-on-petrol-strong shadow-[0_40px_90px_-48px_rgba(14,42,40,.5)] sm:px-10 sm:py-9 ${className}`}>
      {children}
    </section>
  )
}

type Variant = 'primary' | 'ghost' | 'danger'

const buttonBase =
  'inline-flex items-center justify-center gap-2 rounded-btn px-5 py-2.5 text-[14.5px] font-medium transition-all duration-200 ease-kjbn disabled:cursor-not-allowed disabled:opacity-50 disabled:hover:translate-y-0'

const buttonVariants: Record<Variant, string> = {
  primary: 'bg-petrol text-on-petrol hover:-translate-y-px hover:bg-ink',
  ghost: 'border border-line text-ink hover:border-petrol',
  danger: 'bg-danger text-on-petrol hover:-translate-y-px hover:bg-[#8E3327]',
}

export function Button({ variant = 'primary', className = '', ...props }: ButtonHTMLAttributes<HTMLButtonElement> & { variant?: Variant }) {
  return <button className={`${buttonBase} ${buttonVariants[variant]} ${className}`} {...props} />
}

export function ButtonLink({ variant = 'primary', className = '', ...props }: LinkProps & { variant?: Variant }) {
  return <Link className={`${buttonBase} ${buttonVariants[variant]} ${className}`} {...props} />
}

export function Field({ label, error, hint, children }: { label: string; error?: string; hint?: string; children: ReactNode }) {
  return (
    <label className="block">
      <span className="mb-1.5 block text-[13px] font-medium text-ink">{label}</span>
      {children}
      {error ? (
        <span className="mt-1.5 block text-[13px] text-danger">{error}</span>
      ) : (
        hint && <span className="mt-1.5 block text-[12.5px] text-muted">{hint}</span>
      )}
    </label>
  )
}

const controlBase =
  'h-[42px] w-full rounded-btn border bg-field px-3 text-[14.5px] text-ink transition-colors duration-200 placeholder:text-muted/70 focus:border-petrol focus:outline-none disabled:bg-paper-2 disabled:text-muted'

export function Input({ invalid, className = '', ...props }: InputHTMLAttributes<HTMLInputElement> & { invalid?: boolean }) {
  return <input className={`${controlBase} ${invalid ? 'border-danger' : 'border-line'} ${className}`} {...props} />
}

export function Select({ invalid, className = '', ...props }: SelectHTMLAttributes<HTMLSelectElement> & { invalid?: boolean }) {
  return <select className={`${controlBase} ${invalid ? 'border-danger' : 'border-line'} ${className}`} {...props} />
}

export function Checkbox({ label, ...props }: InputHTMLAttributes<HTMLInputElement> & { label: string }) {
  return (
    <label className="inline-flex items-center gap-2.5 text-sm text-ink">
      <input type="checkbox" className="h-4 w-4 accent-petrol" {...props} />
      {label}
    </label>
  )
}

export function Spinner({ label = 'Loading…' }: { label?: string }) {
  return (
    <div className="flex items-center gap-3 py-10 text-sm text-muted" role="status">
      <span className="h-4 w-4 animate-spin rounded-full border-2 border-line border-t-petrol" />
      {label}
    </div>
  )
}

export function EmptyState({ title, children, action }: { title: string; children?: ReactNode; action?: ReactNode }) {
  return (
    <div className="rounded-panel border border-dashed border-line px-6 py-12 text-center">
      <p className="font-serif text-xl">{title}</p>
      {children && <div className="mx-auto mt-2 max-w-md text-sm text-muted">{children}</div>}
      {action && <div className="mt-5">{action}</div>}
    </div>
  )
}

export function ErrorAlert({ error, onRetry }: { error: ApiError | Error | null; onRetry?: () => void }) {
  if (!error) return null
  const code = error instanceof ApiError ? error.code : undefined
  const message = code === 'INTERNAL' ? 'The server hit an unexpected error. Check the API logs for details.' : error.message
  return (
    <div className="mb-6 flex items-start justify-between gap-4 rounded-btn border border-danger/25 bg-danger-soft px-4 py-3 text-sm text-danger" role="alert">
      <div>
        <p className="font-medium">{message}</p>
        {code && code !== 'NETWORK' && code !== 'INTERNAL' && <p className="code mt-0.5 opacity-80">{code}</p>}
      </div>
      {onRetry && (
        <button className="shrink-0 font-medium underline underline-offset-2" onClick={onRetry}>
          Retry
        </button>
      )}
    </div>
  )
}

export function SuccessAlert({ children }: { children: ReactNode }) {
  return (
    <div className="mb-6 rounded-btn border border-success/25 bg-success-soft px-4 py-3 text-sm text-success" role="status">
      {children}
    </div>
  )
}

export function Notice({ children }: { children: ReactNode }) {
  return (
    <div className="rounded-btn border border-saffron/40 bg-saffron-soft px-4 py-3 text-sm text-ink" role="alert">
      {children}
    </div>
  )
}

type Tone = 'default' | 'warning' | 'danger' | 'success'

const toneText: Record<Tone, string> = {
  default: 'text-ink',
  warning: 'text-saffron-ink',
  danger: 'text-danger',
  success: 'text-success',
}

export function Stat({ label, value, tone = 'default' }: { label: string; value: ReactNode; tone?: Tone }) {
  return (
    <div className="border-t border-line pt-4">
      <p className="text-[13px] text-muted">{label}</p>
      <p className={`num mt-1 font-serif text-[30px] leading-tight ${toneText[tone]}`}>{value}</p>
    </div>
  )
}

const badgeTones = {
  default: 'bg-paper-2 text-ink',
  neutral: 'bg-paper-2 text-muted',
  success: 'bg-success-soft text-success',
  danger: 'bg-danger-soft text-danger',
  warning: 'bg-saffron-soft text-saffron-ink',
  info: 'bg-petrol-line/12 text-petrol-line',
}

export function Badge({ tone = 'default', children }: { tone?: keyof typeof badgeTones; children: ReactNode }) {
  return <span className={`inline-block rounded-btn px-1.5 py-0.5 font-mono text-[10.5px] uppercase tracking-[0.12em] ${badgeTones[tone]}`}>{children}</span>
}
