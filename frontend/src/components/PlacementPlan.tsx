import type { ApiError } from '../api/client'
import type { PlacementPreview } from '../api/types'
import { formatDecimal, formatQuantity } from '../lib/format'
import { Notice } from './ui'

export function PlacementPlan({ plan, error, mode = 'place' }: { plan: PlacementPreview | null; error: ApiError | null; mode?: 'place' | 'pick' }) {
  if (error?.code === 'INSUFFICIENT_STOCK') {
    return (
      <Notice>
        Not enough stock: only <span className="num font-medium">{formatDecimal(error.details.available)}</span> of{' '}
        <span className="num">{formatDecimal(error.details.requested)}</span> in the warehouse.
      </Notice>
    )
  }
  if (error?.code === 'NO_SPACE') {
    return (
      <Notice>
        Not enough space: only <span className="num font-medium">{formatDecimal(error.details.fit)}</span> of{' '}
        <span className="num">{formatDecimal(error.details.requested)}</span> units fit in the warehouse.
      </Notice>
    )
  }
  if (!plan) return null

  return (
    <div>
      <p className="text-muted">
        {mode === 'pick' ? 'Takes from' : 'Starts at'} <span className="code text-ink">{plan.start_location_code}</span>
        {mode === 'pick' ? ' first' : ''}, {plan.reason}.
        {plan.allocations.length > 1 && (mode === 'pick' ? ' The rest comes from the nearest bins that hold it.' : ' The rest goes to the nearest bins with space.')}
      </p>
      <ul className="mt-3 divide-y divide-line border-y border-line">
        {plan.allocations.map((a) => (
          <li key={a.location_id} className="flex items-baseline justify-between gap-3 py-2">
            <span className="code">{a.location_code}</span>
            <span className="num">
              <span className="font-medium">{formatQuantity(a.quantity, plan.unit)}</span>
              {a.distance_m > 0 && <span className="ml-2 text-muted">{a.distance_m} m</span>}
            </span>
          </li>
        ))}
      </ul>
    </div>
  )
}
