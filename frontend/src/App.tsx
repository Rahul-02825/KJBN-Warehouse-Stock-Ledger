import { BrowserRouter, Route, Routes } from 'react-router'
import { Layout } from './components/Layout'
import { Adjust } from './pages/Adjust'
import { Dashboard } from './pages/Dashboard'
import { Locations } from './pages/Locations'
import { MovementDetailPage } from './pages/MovementDetailPage'
import { MovementHistory } from './pages/MovementHistory'
import { NotFound } from './pages/NotFound'
import { ProductDetail } from './pages/ProductDetail'
import { Products } from './pages/Products'
import { CostCheck } from './pages/reports/CostCheck'
import { LowStock } from './pages/reports/LowStock'
import { Reconciliation } from './pages/reports/Reconciliation'
import { SeasonForecast } from './pages/reports/SeasonForecast'
import { Valuation } from './pages/reports/Valuation'
import { Receive } from './pages/Receive'
import { Ship } from './pages/Ship'
import { StockOverview } from './pages/StockOverview'
import { Transfer } from './pages/Transfer'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<Layout />}>
          <Route index element={<Dashboard />} />
          <Route path="products" element={<Products />} />
          <Route path="products/:id" element={<ProductDetail />} />
          <Route path="locations" element={<Locations />} />
          <Route path="stock" element={<StockOverview />} />
          <Route path="receive" element={<Receive />} />
          <Route path="ship" element={<Ship />} />
          <Route path="transfer" element={<Transfer />} />
          <Route path="adjust" element={<Adjust />} />
          <Route path="movements" element={<MovementHistory />} />
          <Route path="movements/:id" element={<MovementDetailPage />} />
          <Route path="reports/low-stock" element={<LowStock />} />
          <Route path="reports/valuation" element={<Valuation />} />
          <Route path="reports/reconciliation" element={<Reconciliation />} />
          <Route path="reports/cost-check" element={<CostCheck />} />
          <Route path="reports/season-forecast" element={<SeasonForecast />} />
          <Route path="*" element={<NotFound />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}
