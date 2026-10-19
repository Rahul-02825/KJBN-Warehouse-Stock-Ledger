import { useCallback, useEffect, useState } from 'react'
import { ApiError, api } from '../api/client'

// Loads `path` on mount and whenever it changes. Pass null to skip loading.
export function useApi<T>(path: string | null) {
  const [data, setData] = useState<T | null>(null)
  const [error, setError] = useState<ApiError | null>(null)
  const [loading, setLoading] = useState(path !== null)
  const [version, setVersion] = useState(0)

  useEffect(() => {
    if (path === null) return
    let cancelled = false
    setLoading(true)
    setError(null)
    api
      .get<T>(path)
      .then((result) => !cancelled && setData(result))
      .catch((e: ApiError) => !cancelled && setError(e))
      .finally(() => !cancelled && setLoading(false))
    return () => {
      cancelled = true
    }
  }, [path, version])

  const reload = useCallback(() => setVersion((v) => v + 1), [])
  return { data, error, loading, reload }
}
