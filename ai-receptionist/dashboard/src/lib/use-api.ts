import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { ApiError } from '@/lib/api'

export type UseApiOptions = {
  /** When false, data is loaded only via refresh() (default true). */
  autoLoad?: boolean
}

export function useApiState<T>(
  load: () => Promise<T>,
  deps: unknown[] = [],
  options?: UseApiOptions,
) {
  const autoLoad = options?.autoLoad !== false
  const [data, setData] = useState<T | null>(null)
  const [loading, setLoading] = useState(autoLoad)
  const [error, setError] = useState<ApiError | Error | null>(null)
  const inFlight = useRef<Promise<void> | null>(null)

  const refresh = useCallback(async () => {
    if (inFlight.current) {
      return inFlight.current
    }
    const run = (async () => {
      setLoading(true)
      setError(null)
      try {
        const next = await load()
        setData(next)
      } catch (e) {
        setError(e as ApiError | Error)
      } finally {
        setLoading(false)
        inFlight.current = null
      }
    })()
    inFlight.current = run
    return run
  }, deps) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (autoLoad) {
      void refresh()
    }
  }, [refresh, autoLoad])

  return useMemo(
    () => ({ data, loading, error, refresh, setData }),
    [data, loading, error, refresh],
  )
}
