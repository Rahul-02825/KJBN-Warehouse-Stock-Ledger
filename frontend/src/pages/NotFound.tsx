import { Link } from 'react-router'
import { EmptyState } from '../components/ui'

export function NotFound() {
  return (
    <EmptyState title="Page not found">
      <Link to="/" className="text-emerald-700 underline">
        Back to the dashboard
      </Link>
    </EmptyState>
  )
}
