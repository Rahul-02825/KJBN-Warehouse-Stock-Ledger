import type { ApiError } from '../api/client'
import type { PlacementPreview } from '../api/types'

export function PlacementPlan({ plan, error }: { plan: PlacementPreview | null; error: ApiError | null }) {
  if (error?.code === 'NO_SPACE') {
    return (
      <div className="rounded-md border border-amber-300 bg-amber-50 p-3 text-sm text-amber-900 sm:col-span-2" role="alert">
        Cannot accommodate this receipt: only {error.details.fit} of {error.details.requested} items fit in the warehouse.
      </div>
    )
  }
  if (!plan) return null

  const spills = plan.allocations.length > 1
  return (
    <div className="rounded-md border border-emerald-200 bg-emerald-50 p-3 text-sm sm:col-span-2">
      <p className="mb-2 text-emerald-900">
        Starts at <span className="font-mono font-semibold">{plan.start_location_code}</span> ({plan.reason})
        {spills ? ', then fills the nearest bins with space:' : ':'}
      </p>
      <ul className="space-y-1">
        {plan.allocations.map((a) => (
          <li key={a.location_id} className="flex justify-between gap-4 tabular-nums">
            <span className="font-mono">{a.location_code}</span>
            <span>
              {a.quantity} pcs
              {a.distance_m > 0 && <span className="text-slate-500"> · {a.distance_m} m away</span>}
            </span>
          </li>
        ))}
      </ul>
    </div>
  )
}
