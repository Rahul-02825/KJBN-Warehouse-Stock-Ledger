import { Link } from 'react-router'
import { EmptyState } from '../components/ui'

export function NotFound() {
  return (
    <EmptyState title="This page does not exist" action={<Link to="/" className="link text-sm">Go to the overview</Link>}>
      The link may be old. Everything is reachable from the bar at the top.
    </EmptyState>
  )
}
