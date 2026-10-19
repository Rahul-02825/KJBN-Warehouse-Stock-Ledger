import type { ReactNode } from 'react'

export function Table({ head, numeric = [], children }: { head: ReactNode[]; numeric?: number[]; children: ReactNode }) {
  return (
    <div className="overflow-x-auto rounded-panel border border-line">
      <table className="min-w-full text-[14px]">
        <thead className="bg-paper-2">
          <tr>
            {head.map((h, i) => (
              <th
                key={i}
                className={`whitespace-nowrap px-4 py-2.5 font-mono text-[11px] font-normal uppercase tracking-[0.14em] text-muted ${numeric.includes(i) ? 'text-right' : 'text-left'}`}
              >
                {h}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-line">{children}</tbody>
      </table>
    </div>
  )
}

export function Row({ children, tone = 'default' }: { children: ReactNode; tone?: 'default' | 'warning' | 'danger' | 'muted' }) {
  const tones = {
    default: 'hover:bg-paper-2/60',
    warning: 'bg-saffron-soft/60',
    danger: 'bg-danger-soft/70',
    muted: 'text-muted',
  }
  return <tr className={`transition-colors duration-200 ${tones[tone]}`}>{children}</tr>
}

export function Td({ children, className = '', numeric = false }: { children: ReactNode; className?: string; numeric?: boolean }) {
  return <td className={`whitespace-nowrap px-4 py-3 ${numeric ? 'num text-right' : ''} ${className}`}>{children}</td>
}
