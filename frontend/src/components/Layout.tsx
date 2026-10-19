import { useState } from 'react'
import { NavLink, Outlet } from 'react-router'
import { HealthBadge } from './HealthBadge'

// Nav entries are added here as each page is built.
const navGroups = [
  { title: 'Overview', links: [{ to: '/', label: 'Dashboard' }] },
  {
    title: 'Setup',
    links: [
      { to: '/products', label: 'Products' },
      { to: '/locations', label: 'Locations' },
    ],
  },
  {
    title: 'Stock',
    links: [
      { to: '/stock', label: 'Stock overview' },
      { to: '/receive', label: 'Receive' },
      { to: '/ship', label: 'Ship' },
      { to: '/transfer', label: 'Transfer' },
      { to: '/adjust', label: 'Adjust' },
    ],
  },
  { title: 'History', links: [{ to: '/movements', label: 'Movements' }] },
  {
    title: 'Reports',
    links: [
      { to: '/reports/low-stock', label: 'Low stock' },
      { to: '/reports/valuation', label: 'Valuation' },
      { to: '/reports/reconciliation', label: 'Reconciliation' },
      { to: '/reports/cost-check', label: 'Cost check' },
      { to: '/reports/season-forecast', label: 'Season forecast' },
    ],
  },
]

export function Layout() {
  const [open, setOpen] = useState(false)

  return (
    <div className="min-h-screen md:flex">
      <aside className="bg-slate-900 text-slate-100 md:sticky md:top-0 md:h-screen md:w-60 md:shrink-0 md:overflow-y-auto">
        <div className="flex items-center justify-between px-4 py-4">
          <div>
            <p className="font-semibold">Stock Ledger</p>
            <HealthBadge />
          </div>
          <button className="rounded border border-slate-700 px-2 py-1 text-sm md:hidden" onClick={() => setOpen(!open)} aria-expanded={open}>
            Menu
          </button>
        </div>
        <nav className={`${open ? 'block' : 'hidden'} px-2 pb-4 md:block`}>
          {navGroups.map((group) => (
            <div key={group.title} className="mb-4">
              <p className="px-2 pb-1 text-xs font-semibold uppercase tracking-wide text-slate-400">{group.title}</p>
              {group.links.map((link) => (
                <NavLink
                  key={link.to}
                  to={link.to}
                  end={link.to === '/'}
                  onClick={() => setOpen(false)}
                  className={({ isActive }) =>
                    `block rounded px-2 py-1.5 text-sm ${isActive ? 'bg-emerald-700 text-white' : 'text-slate-300 hover:bg-slate-800'}`
                  }
                >
                  {link.label}
                </NavLink>
              ))}
            </div>
          ))}
        </nav>
      </aside>
      <main className="min-w-0 flex-1 p-4 md:p-8">
        <Outlet />
      </main>
    </div>
  )
}
