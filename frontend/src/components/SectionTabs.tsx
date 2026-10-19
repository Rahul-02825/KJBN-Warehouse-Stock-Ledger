import { NavLink } from 'react-router'

export function SectionTabs({ tabs, label }: { tabs: { to: string; label: string }[]; label: string }) {
  return (
    <nav className="mb-10 flex gap-6 overflow-x-auto border-b border-line" aria-label={label}>
      {tabs.map((t) => (
        <NavLink
          key={t.to}
          to={t.to}
          end
          className={({ isActive }) =>
            `-mb-px whitespace-nowrap border-b-2 pb-3 text-[14px] transition-colors duration-200 ${
              isActive ? 'border-petrol font-medium text-ink' : 'border-transparent text-muted hover:text-ink'
            }`
          }
        >
          {t.label}
        </NavLink>
      ))}
    </nav>
  )
}
