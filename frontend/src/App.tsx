import { BrowserRouter, Route, Routes } from 'react-router'
import { Layout } from './components/Layout'
import { Dashboard } from './pages/Dashboard'
import { Locations } from './pages/Locations'
import { NotFound } from './pages/NotFound'
import { Products } from './pages/Products'
import { StockOverview } from './pages/StockOverview'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<Layout />}>
          <Route index element={<Dashboard />} />
          <Route path="products" element={<Products />} />
          <Route path="locations" element={<Locations />} />
          <Route path="stock" element={<StockOverview />} />
          <Route path="*" element={<NotFound />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}
