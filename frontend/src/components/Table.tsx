import type { ReactNode } from 'react'

// A plain table that scrolls sideways on small screens instead of breaking the layout.
// `numeric` lists the column indexes whose header should be right-aligned like their numbers.
export function Table({ head, numeric = [], children }: { head: ReactNode[]; numeric?: number[]; children: ReactNode }) {
  return (
    <div className="overflow-x-auto rounded-lg border border-slate-200 bg-white shadow-sm">
      <table className="min-w-full divide-y divide-slate-200 text-sm">
        <thead className="bg-slate-50">
          <tr>
            {head.map((h, i) => (
              <th key={i} className={`whitespace-nowrap px-4 py-2 font-medium text-slate-600 ${numeric.includes(i) ? 'text-right' : 'text-left'}`}>
                {h}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100">{children}</tbody>
      </table>
    </div>
  )
}

export function Td({ children, className = '' }: { children: ReactNode; className?: string }) {
  return <td className={`whitespace-nowrap px-4 py-2 ${className}`}>{children}</td>
}
