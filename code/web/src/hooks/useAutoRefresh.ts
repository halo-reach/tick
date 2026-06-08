import { useEffect, useRef, useState, useCallback } from 'react'

export function useAutoRefresh<T>(
  fetcher: () => Promise<T>,
  intervalMs = 30000,
) {
  const [data, setData] = useState<T | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null)
  const timer = useRef<ReturnType<typeof setInterval>>()

  const refresh = useCallback(async () => {
    try {
      const result = await fetcher()
      setData(result)
      setError(null)
      setLastUpdated(new Date())
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [fetcher])

  useEffect(() => {
    refresh()
    timer.current = setInterval(refresh, intervalMs)
    return () => clearInterval(timer.current)
  }, [refresh, intervalMs])

  return { data, error, loading, lastUpdated, refresh }
}
