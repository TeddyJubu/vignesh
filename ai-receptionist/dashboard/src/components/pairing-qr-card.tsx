import { useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { usePairingOptional } from '@/lib/pairing-context'
import { derivePairingStatus } from '@/lib/pairing-types'
import { getSessionToken } from '@/lib/api'
import { Loader2, RefreshCcw, Smartphone, Unlink } from 'lucide-react'

function apiBase(): string {
  return import.meta.env.BASE_URL.replace(/\/?$/, '/')
}

export function PairingQRCard({ compact = false }: { compact?: boolean }) {
  const ctx = usePairingOptional()
  const pairing = ctx?.pairing ?? null
  const status = derivePairingStatus(pairing)
  const linked = pairing?.logged_in === true
  const [qrURL, setQrURL] = useState<string | null>(null)
  const [qrLoading, setQrLoading] = useState(false)
  const [unlinkOpen, setUnlinkOpen] = useState(false)
  const [unlinkBusy, setUnlinkBusy] = useState(false)
  const [unlinkError, setUnlinkError] = useState<string | null>(null)

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

  async function confirmUnlink() {
    if (!ctx) return
    setUnlinkBusy(true)
    setUnlinkError(null)
    try {
      await ctx.unlinkWhatsApp()
      setUnlinkOpen(false)
    } catch (e: any) {
      setUnlinkError(e?.message ?? 'Could not unlink WhatsApp')
    } finally {
      setUnlinkBusy(false)
    }
  }

  if (status === 'connected') {
    return (
      <Card>
        <CardHeader className={compact ? 'pb-2' : undefined}>
          <CardTitle className="text-base flex items-center gap-2">
            <Smartphone className="h-4 w-4 text-green-600" />
            WhatsApp connected
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-muted-foreground">
            {pairing?.detail ?? 'Session is linked and online.'}
          </p>
          <UnlinkButton
            open={unlinkOpen}
            onOpenChange={setUnlinkOpen}
            busy={unlinkBusy}
            error={unlinkError}
            disabled={!ctx}
            onConfirm={() => void confirmUnlink()}
          />
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
          {linked ? (
            <UnlinkButton
              open={unlinkOpen}
              onOpenChange={setUnlinkOpen}
              busy={unlinkBusy}
              error={unlinkError}
              disabled={!ctx}
              onConfirm={() => void confirmUnlink()}
            />
          ) : null}
        </div>
      </CardContent>
    </Card>
  )
}

function UnlinkButton({
  open,
  onOpenChange,
  busy,
  error,
  disabled,
  onConfirm,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  busy: boolean
  error: string | null
  disabled: boolean
  onConfirm: () => void
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogTrigger
        render={
          <Button variant="destructive" size="sm" disabled={disabled}>
            <Unlink className="mr-2 h-4 w-4" />
            Unlink WhatsApp
          </Button>
        }
      />
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Unlink WhatsApp?</DialogTitle>
          <DialogDescription>
            This removes the linked device from WhatsApp and stops the bot from
            sending or receiving messages. You will need to scan a new QR code to
            connect again.
          </DialogDescription>
        </DialogHeader>
        {error ? (
          <p className="text-sm text-destructive">{error}</p>
        ) : null}
        <DialogFooter>
          <Button
            variant="secondary"
            onClick={() => onOpenChange(false)}
            disabled={busy}
          >
            Cancel
          </Button>
          <Button variant="destructive" onClick={onConfirm} disabled={busy}>
            {busy ? 'Unlinking…' : 'Unlink'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
