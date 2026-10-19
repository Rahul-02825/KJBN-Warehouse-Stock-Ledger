import { useEffect, useState } from 'react'
import { ApiError, api } from '../api/client'
import type { ListResponse, Location, PagedResponse, Product } from '../api/types'

// Loads every product (page by page) and every bin for the movement form dropdowns.
export function useOptions() {
  const [products, setProducts] = useState<Product[] | null>(null)
  const [locations, setLocations] = useState<Location[] | null>(null)
  const [error, setError] = useState<ApiError | null>(null)

  useEffect(() => {
    async function load() {
      const all: Product[] = []
      for (let page = 1; ; page++) {
        const res = await api.get<PagedResponse<Product>>(`/api/products?page_size=100&page=${page}`)
        all.push(...res.data)
        if (all.length >= res.total || res.data.length === 0) break
      }
      const bins = await api.get<ListResponse<Location>>('/api/locations')
      setProducts(all)
      setLocations(bins.data)
    }
    load().catch((e: ApiError) => setError(e))
  }, [])

  return { products, locations, error }
}
