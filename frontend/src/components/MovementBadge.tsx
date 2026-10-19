import type { MovementType } from '../api/types'

const styles: Record<MovementType, string> = {
  IN: 'bg-emerald-100 text-emerald-800',
  OUT: 'bg-sky-100 text-sky-800',
  TRANSFER: 'bg-violet-100 text-violet-800',
  ADJUSTMENT: 'bg-amber-100 text-amber-800',
  REVERSAL: 'bg-slate-200 text-slate-700',
}

export function MovementBadge({ type }: { type: MovementType }) {
  return <span className={`rounded px-1.5 py-0.5 text-xs font-medium ${styles[type]}`}>{type}</span>
}

// A signed quantity: green for stock added, red for stock removed.
export function Delta({ value }: { value: string }) {
  const negative = value.startsWith('-')
  return <span className={`tabular-nums font-medium ${negative ? 'text-red-600' : 'text-emerald-700'}`}>{negative ? value : `+${value}`}</span>
}
