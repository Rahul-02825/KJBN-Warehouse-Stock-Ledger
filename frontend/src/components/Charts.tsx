import { Link } from 'react-router'
import type { ActivityDay, TopMover } from '../api/types'
import { formatMoney, formatQuantity } from '../lib/format'

// Plain HTML bar charts: no chart library. Numbers are used only to size bars;
// every value shown is still formatted from the API's decimal strings.

const series = [
  { key: 'in_count', label: 'IN', color: 'bg-emerald-500' },
  { key: 'out_count', label: 'OUT', color: 'bg-sky-500' },
  { key: 'transfer_count', label: 'Transfer', color: 'bg-violet-500' },
  { key: 'adjustment_count', label: 'Adjustment', color: 'bg-amber-500' },
  { key: 'reversal_count', label: 'Reversal', color: 'bg-slate-400' },
] as const

export function ActivityChart({ days }: { days: ActivityDay[] }) {
  const totals = days.map((d) => series.reduce((sum, s) => sum + d[s.key], 0))
  const max = Math.max(1, ...totals)
  if (totals.every((t) => t === 0)) return <p className="py-8 text-center text-sm text-slate-500">No movements in this period.</p>

  return (
    <div>
      <div className="flex h-40 items-end gap-1">
        {days.map((d, i) => (
          <div key={d.day} className="group relative flex h-full flex-1 flex-col justify-end" title={`${d.day}: ${totals[i]} movements`}>
            {series.map((s) =>
              d[s.key] > 0 ? <div key={s.key} className={s.color} style={{ height: `${(d[s.key] / max) * 100}%` }} /> : null,
            )}
          </div>
        ))}
      </div>
      <div className="mt-1 flex gap-1 text-[10px] text-slate-400">
        {days.map((d, i) => (
          <span key={d.day} className="flex-1 text-center">
            {(days.length - 1 - i) % 2 === 0 ? d.day.slice(8) : ''}
          </span>
        ))}
      </div>
      <div className="mt-3 flex flex-wrap gap-3 text-xs text-slate-600">
        {series.map((s) => (
          <span key={s.key} className="flex items-center gap-1">
            <span className={`h-2.5 w-2.5 rounded-sm ${s.color}`} />
            {s.label}
          </span>
        ))}
      </div>
    </div>
  )
}

export function TopMoversChart({ movers }: { movers: TopMover[] }) {
  if (movers.length === 0) return <p className="py-8 text-center text-sm text-slate-500">Nothing shipped in this period.</p>
  const max = Math.max(...movers.map((m) => Number(m.shipped_value)), 1)
  return (
    <ul className="space-y-3">
      {movers.map((m) => (
        <li key={m.product_id}>
          <div className="mb-1 flex justify-between gap-2 text-sm">
            <Link to={`/products/${m.product_id}`} className="font-mono text-emerald-700 hover:underline">
              {m.sku}
            </Link>
            <span className="tabular-nums text-slate-600">
              {formatQuantity(m.shipped_quantity, m.unit)} · <span className="font-semibold text-slate-900">{formatMoney(m.shipped_value)}</span>
            </span>
          </div>
          <div className="h-2 rounded bg-slate-100">
            <div className="h-2 rounded bg-sky-500" style={{ width: `${(Number(m.shipped_value) / max) * 100}%` }} />
          </div>
        </li>
      ))}
    </ul>
  )
}
