import type { MovementType } from '../api/types'
import { Badge } from './ui'

const tones = {
  IN: 'success',
  OUT: 'danger',
  TRANSFER: 'info',
  ADJUSTMENT: 'warning',
  REVERSAL: 'neutral',
} as const

export function MovementBadge({ type }: { type: MovementType }) {
  return <Badge tone={tones[type]}>{type}</Badge>
}

export function Delta({ value }: { value: string }) {
  const negative = value.startsWith('-')
  return <span className={`num font-medium ${negative ? 'text-danger' : 'text-success'}`}>{negative ? `−${value.slice(1)}` : `+${value}`}</span>
}
