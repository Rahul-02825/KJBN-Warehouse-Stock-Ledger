import { Button } from './ui'

export function Pagination({ page, pageSize, total, onPage }: { page: number; pageSize: number; total: number; onPage: (page: number) => void }) {
  if (total === 0) return null
  const first = (page - 1) * pageSize + 1
  const last = Math.min(page * pageSize, total)
  return (
    <div className="mt-4 flex items-center justify-between text-sm text-muted">
      <span className="num">
        {first}–{last} of {total}
      </span>
      <div className="flex gap-2">
        <Button variant="ghost" className="px-3.5 py-1.5" disabled={page <= 1} onClick={() => onPage(page - 1)}>
          Previous
        </Button>
        <Button variant="ghost" className="px-3.5 py-1.5" disabled={last >= total} onClick={() => onPage(page + 1)}>
          Next
        </Button>
      </div>
    </div>
  )
}
