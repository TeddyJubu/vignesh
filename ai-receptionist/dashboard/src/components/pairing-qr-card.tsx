import { useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { usePairingOptional } from '@/lib/pairing-context'
import { derivePairingStatus } from '@/lib/pairing-types'
import { getSessionToken } from '@/lib/api'
import { Loader2, RefreshCcw, Smartphone } from 'lucide-react'

function apiBase(): string {
  return import.meta.env.BASE_URL.replace(/\/?$/, '/')
}

export function PairingQRCard({ compact = false }: { compact?: boolean }) {
  const ctx = usePairingOptional()
  const pairing = ctx?.pairing ?? null
  const status = derivePairingStatus(pairing)
  const [qrURL, setQrURL] = useState<string | null>(null)
  const [qrLoading, setQrLoading] = useState(false)

  useEffect(() => {
    if (!pairing?.qr_available || !pairing.updated_at) {
      setQrURL(null)
      return
    }
    let cancelled = false
    let objectURL: string | null = null
    setQrLoading(true)

    const headers: HeadersInit = {}
    const token = getSessionToken()
    if (token) headers['Authorization'] = `Bearer ${token}`

    const bust = encodeURIComponent(pairing.updated_at)
    void fetch(`${apiBase()}api/pairing/qr.png?v=${bust}`, {
      headers,
      credentials: 'same-origin',
    })
      .then((res) => {
        if (!res.ok) throw new Error('qr fetch failed')
        return res.blob()
      })
      .then((blob) => {
        if (cancelled) return
        objectURL = URL.createObjectURL(blob)
        setQrURL(objectURL)
      })
      .catch(() => {
        if (!cancelled) setQrURL(null)
      })
      .finally(() => {
        if (!cancelled) setQrLoading(false)
      })

    return () => {
      cancelled = true
      if (objectURL) URL.revokeObjectURL(objectURL)
    }
  }, [pairing?.qr_available, pairing?.updated_at])

  if (status === 'connected') {
    return (
      <Card>
        <CardHeader className={compact ? 'pb-2' : undefined}>
          <CardTitle className="text-base flex items-center gap-2">
            <Smartphone className="h-4 w-4 text-green-600" />
            WhatsApp connected
          </CardTitle>
        </CardHeader>
        <CardContent className="text-sm text-muted-foreground">
          {pairing?.detail ?? 'Session is linked and online.'}
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader className={compact ? 'pb-2' : undefined}>
        <CardTitle className="text-base">Scan to connect WhatsApp</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-sm text-muted-foreground">
          {pairing?.detail ??
            (status === 'timeout'
              ? 'QR expired — generating a new code…'
              : 'Open WhatsApp on your phone → Linked devices → Link a device.')}
        </p>

        {status === 'scan' && (
          <div className="flex justify-center rounded-lg bg-white p-4">
            {qrLoading && (
              <div className="flex h-64 w-64 items-center justify-center">
                <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
              </div>
            )}
            {!qrLoading && qrURL && (
              <img
                src={qrURL}
                alt="WhatsApp pairing QR code"
                className="h-64 w-64"
                width={256}
                height={256}
              />
            )}
          </div>
        )}

        {status === 'waiting' || status === 'checking' ? (
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Loader2 className="h-4 w-4 animate-spin" />
            Waiting for QR code…
          </div>
        ) : null}

        <div className="flex flex-wrap gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={!ctx}
            onClick={() => void ctx?.refresh()}
          >
            <RefreshCcw className="mr-2 h-4 w-4" />
            Refresh status
          </Button>
          <Button
            variant="secondary"
            size="sm"
            disabled={!ctx}
            onClick={() => void ctx?.requestNewQR()}
          >
            New QR
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
