import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type PropsWithChildren,
} from 'react'
import { apiFetch, getSessionToken } from '@/lib/api'
import type { PairingSnapshot } from '@/lib/pairing-types'

type PairingContextValue = {
  pairing: PairingSnapshot | null
  streamConnected: boolean
  refresh: () => Promise<void>
  requestNewQR: () => Promise<void>
}

const PairingContext = createContext<PairingContextValue | null>(null)

function apiBase(): string {
  return import.meta.env.BASE_URL.replace(/\/?$/, '/')
}

async function consumePairingStream(
  onEvent: (type: string, data: unknown) => void,
  signal: AbortSignal,
) {
  const headers: HeadersInit = {}
  const token = getSessionToken()
  if (token) headers['Authorization'] = `Bearer ${token}`

  const res = await fetch(`${apiBase()}api/pairing/stream`, {
    headers,
    signal,
    credentials: 'same-origin',
  })
  if (!res.ok || !res.body) {
    throw new Error(`pairing stream failed (${res.status})`)
  }

  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    const parts = buffer.split('\n\n')
    buffer = parts.pop() ?? ''
    for (const block of parts) {
      const lines = block.split('\n')
      let eventType = 'message'
      let dataLine = ''
      for (const line of lines) {
        if (line.startsWith('event:')) eventType = line.slice(6).trim()
        if (line.startsWith('data:')) dataLine = line.slice(5).trim()
      }
      if (dataLine) {
        try {
          onEvent(eventType, JSON.parse(dataLine))
        } catch {
          // ignore malformed chunks
        }
      }
    }
  }
}

export function PairingProvider({ children }: PropsWithChildren) {
  const [pairing, setPairing] = useState<PairingSnapshot | null>(null)
  const [streamConnected, setStreamConnected] = useState(false)
  const mounted = useRef(true)

  const applySnapshot = useCallback((snap: PairingSnapshot) => {
    setPairing(snap)
  }, [])

  const refresh = useCallback(async () => {
    const snap = await apiFetch<PairingSnapshot>('/pairing')
    applySnapshot(snap)
  }, [applySnapshot])

  const requestNewQR = useCallback(async () => {
    const snap = await apiFetch<PairingSnapshot>('/pairing/refresh', {
      method: 'POST',
    })
    applySnapshot(snap)
  }, [applySnapshot])

  useEffect(() => {
    mounted.current = true
    const ac = new AbortController()

    void refresh().catch(() => {
      if (mounted.current) setPairing(null)
    })

    void (async () => {
      while (mounted.current && !ac.signal.aborted) {
        try {
          setStreamConnected(true)
          await consumePairingStream((type, data) => {
            if (!mounted.current) return
            if (type === 'ready_snapshot') {
              const wrap = data as { pairing?: PairingSnapshot }
              if (wrap.pairing) applySnapshot(wrap.pairing)
              return
            }
            if (type === 'pairing_changed') {
              applySnapshot(data as PairingSnapshot)
            }
          }, ac.signal)
        } catch {
          if (ac.signal.aborted) break
          setStreamConnected(false)
          await new Promise((r) => setTimeout(r, 3000))
        }
      }
    })()

    return () => {
      mounted.current = false
      ac.abort()
      setStreamConnected(false)
    }
  }, [applySnapshot, refresh])

  const value = useMemo(
    () => ({ pairing, streamConnected, refresh, requestNewQR }),
    [pairing, streamConnected, refresh, requestNewQR],
  )

  return (
    <PairingContext.Provider value={value}>{children}</PairingContext.Provider>
  )
}

export function usePairing() {
  const ctx = useContext(PairingContext)
  if (!ctx) {
    throw new Error('usePairing must be used within PairingProvider')
  }
  return ctx
}

export function usePairingOptional() {
  return useContext(PairingContext)
}
