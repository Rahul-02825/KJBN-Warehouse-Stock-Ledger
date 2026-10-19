import { useEffect, useState } from 'react'
import { ApiError, api } from '../api/client'
import type { PlacementPreview } from '../api/types'

export interface PreviewRequest {
  product_id: number
  quantity: string
  location_id?: number
}

export function usePlacementPreview(request: PreviewRequest | null) {
  const [plan, setPlan] = useState<PlacementPreview | null>(null)
  const [error, setError] = useState<ApiError | null>(null)
  const key = request ? JSON.stringify(request) : ''

  useEffect(() => {
    let cancelled = false
    const timer = setTimeout(() => {
      if (!key) {
        setPlan(null)
        setError(null)
        return
      }
      api
        .post<PlacementPreview>('/api/placements/preview', JSON.parse(key))
        .then((result) => {
          if (cancelled) return
          setPlan(result)
          setError(null)
        })
        .catch((e: ApiError) => {
          if (cancelled) return
          setPlan(null)
          setError(e)
        })
    }, 300)
    return () => {
      cancelled = true
      clearTimeout(timer)
    }
  }, [key])

  return { plan, error }
}
