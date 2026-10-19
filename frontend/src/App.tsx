import { BrowserRouter, Navigate, Route, Routes, useLocation } from 'react-router'
import { Layout } from './components/Layout'
import { Dashboard } from './pages/Dashboard'
import { Locations } from './pages/Locations'
import { MovementDetailPage } from './pages/MovementDetailPage'
import { MovementHistory } from './pages/MovementHistory'
import { NotFound } from './pages/NotFound'
import { ProductDetail } from './pages/ProductDetail'
import { Products } from './pages/Products'
import { RecordMovement } from './pages/record/RecordMovement'
import { CostCheck } from './pages/reports/CostCheck'
import { LowStock } from './pages/reports/LowStock'
import { Reconciliation } from './pages/reports/Reconciliation'
import { SeasonForecast } from './pages/reports/SeasonForecast'
import { Valuation } from './pages/reports/Valuation'
import { StockOverview } from './pages/StockOverview'

function RedirectToRecord({ kind }: { kind: string }) {
  const { search } = useLocation()
  return <Navigate to={`/record/${kind}${search}`} replace />
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<Layout />}>
          <Route index element={<Dashboard />} />
          <Route path="stock" element={<StockOverview />} />
          <Route path="record" element={<Navigate to="/record/receive" replace />} />
          <Route path="record/:kind" element={<RecordMovement />} />
          <Route path="movements" element={<MovementHistory />} />
          <Route path="movements/:id" element={<MovementDetailPage />} />
          <Route path="products" element={<Products />} />
          <Route path="products/:id" element={<ProductDetail />} />
          <Route path="locations" element={<Locations />} />
          <Route path="reports" element={<Navigate to="/reports/low-stock" replace />} />
          <Route path="reports/low-stock" element={<LowStock />} />
          <Route path="reports/valuation" element={<Valuation />} />
          <Route path="reports/reconciliation" element={<Reconciliation />} />
          <Route path="reports/cost-check" element={<CostCheck />} />
          <Route path="reports/season-forecast" element={<SeasonForecast />} />
          {['receive', 'ship', 'transfer', 'adjust'].map((kind) => (
            <Route key={kind} path={kind} element={<RedirectToRecord kind={kind} />} />
          ))}
          <Route path="*" element={<NotFound />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}
