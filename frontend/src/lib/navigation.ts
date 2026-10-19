import { ArrowDownToLine, ArrowLeftRight, ArrowUpFromLine, Calculator, CalendarRange, IndianRupee, Package, Scale, SlidersHorizontal, TrendingDown, Warehouse } from 'lucide-react'

export const catalogueTabs = [
  { to: '/products', label: 'Products', Icon: Package },
  { to: '/locations', label: 'Bins', Icon: Warehouse },
]

export const reportTabs = [
  { to: '/reports/low-stock', label: 'Low stock', Icon: TrendingDown },
  { to: '/reports/valuation', label: 'Valuation', Icon: IndianRupee },
  { to: '/reports/reconciliation', label: 'Reconciliation', Icon: Scale },
  { to: '/reports/cost-check', label: 'Cost check', Icon: Calculator },
  { to: '/reports/season-forecast', label: 'Season forecast', Icon: CalendarRange },
]

export const movementIcons = {
  receive: ArrowDownToLine,
  ship: ArrowUpFromLine,
  transfer: ArrowLeftRight,
  adjust: SlidersHorizontal,
}
