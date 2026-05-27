import { useCallback, useEffect, useMemo, useState } from 'react'
import type { ApiError } from '@/lib/api'

export function useApiState<T>(load: () => Promise<T>, deps: unknown[] = []) {
  const [data, setData] = useState<T | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<ApiError | Error | null>(null)

  const refresh = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const next = await load()
      setData(next)
    } catch (e) {
      setError(e as any)
    } finally {
      setLoading(false)
    }
  }, deps) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    void refresh()
  }, [refresh])

  return useMemo(
    () => ({ data, loading, error, refresh, setData }),
    [data, loading, error, refresh],
  )
}

