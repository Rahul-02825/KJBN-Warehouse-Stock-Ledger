import { Plus } from 'lucide-react'
import { useState } from 'react'
import { Link, NavLink, Outlet, useLocation } from 'react-router'
import { icon } from './icon'
import { ButtonLink } from './ui'

const destinations = [
  { to: '/', label: 'Overview', match: (path: string) => path === '/' },
  { to: '/stock', label: 'Stock', match: (path: string) => path.startsWith('/stock') },
  { to: '/movements', label: 'Movements', match: (path: string) => path.startsWith('/movements') },
  { to: '/products', label: 'Catalogue', match: (path: string) => path.startsWith('/products') || path.startsWith('/locations') },
  { to: '/reports/low-stock', label: 'Reports', match: (path: string) => path.startsWith('/reports') },
]

export function Layout() {
  const { pathname } = useLocation()
  const [open, setOpen] = useState(false)

  return (
    <div className="min-h-screen">
      <header className="sticky top-0 z-30 border-b border-line bg-[color-mix(in_srgb,var(--color-paper)_80%,transparent)] backdrop-blur-[10px]">
        <div className="mx-auto flex h-[62px] max-w-[1100px] items-center gap-8 px-4 sm:px-8">
          <div className="flex items-center gap-8">
            <Link
              to="/"
              onClick={() => setOpen(false)}
              className="whitespace-nowrap border border-saffron px-2.5 py-1.5 text-[17px] font-light uppercase leading-none tracking-[0.08em] text-petrol"
            >
              Stock Ledger
            </Link>
            <span className="hidden h-6 w-px bg-line md:block" aria-hidden />
          </div>

          <nav className="hidden flex-1 items-center gap-6 md:flex" aria-label="Main">
            {destinations.map((d) => (
              <NavItem key={d.to} to={d.to} label={d.label} active={d.match(pathname)} />
            ))}
          </nav>

          <div className="ml-auto flex items-center gap-4">
            <div className="hidden sm:block">
              <ButtonLink to="/record/receive" className="px-4 py-2">
                <Plus {...icon} />
                Record movement
              </ButtonLink>
            </div>
            <button
              className="rounded-btn border border-line px-3 py-1.5 text-sm md:hidden"
              onClick={() => setOpen(!open)}
              aria-expanded={open}
              aria-controls="mobile-nav"
            >
              Menu
            </button>
          </div>
        </div>

        {open && (
          <nav id="mobile-nav" className="border-t border-line px-4 pb-4 pt-2 md:hidden" aria-label="Main">
            {destinations.map((d) => (
              <Link
                key={d.to}
                to={d.to}
                onClick={() => setOpen(false)}
                className={`block border-b border-line py-3 text-[13px] uppercase tracking-[0.14em] ${d.match(pathname) ? 'font-medium text-ink' : 'text-muted'}`}
              >
                {d.label}
              </Link>
            ))}
            <ButtonLink to="/record/receive" className="mt-4 w-full" onClick={() => setOpen(false)}>
              <Plus {...icon} />
              Record movement
            </ButtonLink>
          </nav>
        )}
      </header>

      <main className="mx-auto max-w-[1100px] animate-reveal px-4 pb-24 pt-10 sm:px-8 sm:pt-14">
        <Outlet />
      </main>
    </div>
  )
}

function NavItem({ to, label, active }: { to: string; label: string; active: boolean }) {
  return (
    <NavLink
      to={to}
      className={`relative py-1 text-[12.5px] uppercase tracking-[0.14em] transition-colors duration-200 ${active ? 'text-ink' : 'text-muted hover:text-ink'}`}
      aria-current={active ? 'page' : undefined}
    >
      {label}
      {active && <span className="absolute inset-x-0 -bottom-[19px] h-0.5 bg-saffron" aria-hidden />}
    </NavLink>
  )
}
