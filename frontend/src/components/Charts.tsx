import { useState } from 'react'
import { Link } from 'react-router'
import type { ActivityDay, TopMover } from '../api/types'
import { formatMoney, formatQuantity } from '../lib/format'
import { MovementBadge } from './MovementBadge'

const kinds = [
  { key: 'in_count', type: 'IN' },
  { key: 'out_count', type: 'OUT' },
  { key: 'transfer_count', type: 'TRANSFER' },
  { key: 'adjustment_count', type: 'ADJUSTMENT' },
  { key: 'reversal_count', type: 'REVERSAL' },
] as const

const dayLabel = new Intl.DateTimeFormat('en-IN', { day: 'numeric', month: 'short', timeZone: 'UTC' })

function total(day: ActivityDay): number {
  return kinds.reduce((sum, k) => sum + day[k.key], 0)
}

export function ActivityChart({ days }: { days: ActivityDay[] }) {
  const [active, setActive] = useState<number | null>(null)
  const totals = days.map(total)
  const max = Math.max(1, ...totals)
  if (totals.every((t) => t === 0)) return <p className="py-12 text-center text-sm text-muted">No movements in this period.</p>

  const focused = active === null ? null : days[active]
  const periodTotals = kinds.map((k) => ({ type: k.type, count: days.reduce((sum, d) => sum + d[k.key], 0) }))

  return (
    <div>
      <div className="relative">
        <div className="flex h-44 items-end gap-0.5 border-b border-line" onMouseLeave={() => setActive(null)}>
          {days.map((d, i) => (
            <button
              key={d.day}
              type="button"
              className="group flex h-full flex-1 items-end focus-visible:outline-offset-0"
              onMouseEnter={() => setActive(i)}
              onFocus={() => setActive(i)}
              onBlur={() => setActive(null)}
              aria-label={`${dayLabel.format(new Date(d.day))}: ${totals[i]} movements`}
            >
              <span
                className={`block w-full origin-bottom animate-grow-y rounded-t-[4px] transition-colors duration-200 ${active === i ? 'bg-ink' : 'bg-petrol'}`}
                style={{ height: totals[i] === 0 ? 0 : `max(${(totals[i] / max) * 100}%, 3px)`, animationDelay: `${i * 30}ms` }}
              />
            </button>
          ))}
        </div>
        {focused && active !== null && (
          <div
            className="pointer-events-none absolute -top-2 z-10 w-44 -translate-y-full rounded-btn border border-line bg-paper px-3 py-2 text-[12.5px] shadow-[0_12px_30px_-18px_rgba(14,42,40,.45)]"
            style={{ left: `clamp(0px, calc(${((active + 0.5) / days.length) * 100}% - 88px), calc(100% - 176px))` }}
          >
            <p className="mb-1 font-medium">
              {dayLabel.format(new Date(focused.day))} · <span className="num">{totals[active]}</span> movements
            </p>
            {kinds
              .filter((k) => focused[k.key] > 0)
              .map((k) => (
                <p key={k.key} className="flex justify-between text-muted">
                  <span>{k.type.toLowerCase()}</span>
                  <span className="num text-ink">{focused[k.key]}</span>
                </p>
              ))}
          </div>
        )}
      </div>
      <div className="mt-2 flex gap-0.5 font-mono text-[10.5px] text-muted">
        {days.map((d, i) => (
          <span key={d.day} className="flex-1 text-center">
            {(days.length - 1 - i) % 3 === 0 ? d.day.slice(8) : ''}
          </span>
        ))}
      </div>
      <ul className="mt-5 flex flex-wrap gap-x-5 gap-y-2 text-[13px] text-muted">
        {periodTotals.map((p) => (
          <li key={p.type} className="flex items-center gap-2">
            <MovementBadge type={p.type} />
            <span className="num text-ink">{p.count}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}

export function TopMoversChart({ movers }: { movers: TopMover[] }) {
  if (movers.length === 0) return <p className="py-12 text-center text-sm text-muted">Nothing shipped in this period.</p>
  const max = Math.max(...movers.map((m) => Number(m.shipped_value)), 1)
  return (
    <ul className="space-y-4">
      {movers.map((m, i) => (
        <li key={m.product_id}>
          <div className="mb-1.5 flex items-baseline justify-between gap-3 text-sm">
            <Link to={`/products/${m.product_id}`} className="link-quiet code">
              {m.sku}
            </Link>
            <span className="num text-muted">
              {formatQuantity(m.shipped_quantity, m.unit)} <span className="ml-1 font-medium text-ink">{formatMoney(m.shipped_value)}</span>
            </span>
          </div>
          <div className="h-1.5 rounded-full bg-paper-2">
            <div
              className="h-1.5 origin-left animate-grow-x rounded-full bg-petrol"
              style={{ width: `${(Number(m.shipped_value) / max) * 100}%`, animationDelay: `${150 + i * 80}ms` }}
            />
          </div>
        </li>
      ))}
    </ul>
  )
}
