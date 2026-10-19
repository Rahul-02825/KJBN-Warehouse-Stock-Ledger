import { useEffect, useState } from 'react'
import { api } from '../api/client'

// Polls /healthz so a sleeping or broken backend is obvious at a glance.
export function HealthBadge() {
  const [ok, setOk] = useState<boolean | null>(null)

  useEffect(() => {
    const check = () => api.get('/healthz').then(() => setOk(true), () => setOk(false))
    check()
    const timer = setInterval(check, 10000)
    return () => clearInterval(timer)
  }, [])

  const text = ok === null ? 'Checking API…' : ok ? 'API online' : 'API offline'
  const dot = ok === null ? 'bg-slate-400' : ok ? 'bg-emerald-500' : 'bg-red-500'
  return (
    <span className="inline-flex items-center gap-2 text-xs text-slate-300">
      <span className={`h-2 w-2 rounded-full ${dot}`} />
      {text}
    </span>
  )
}
