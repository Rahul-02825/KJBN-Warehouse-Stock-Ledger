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
          <Route path="*" element={<NotFound />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}
