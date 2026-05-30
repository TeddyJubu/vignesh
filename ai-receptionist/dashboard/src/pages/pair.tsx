import { useEffect } from 'react'
import { PairingQRCard } from '@/components/pairing-qr-card'
import { Page, PageHeader } from '@/components/page'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { derivePairingStatus } from '@/lib/pairing-types'
import { usePairing } from '@/lib/pairing-context'
import { ShieldAlert } from 'lucide-react'

export function PairPage() {
  const { pairing, streamConnected, refresh } = usePairing()
  const status = derivePairingStatus(pairing)

  useEffect(() => {
    if (status === 'connected' || status === 'scan') return
    const id = window.setInterval(() => {
      void refresh()
    }, 3000)
    return () => window.clearInterval(id)
  }, [status, refresh])

  return (
    <Page>
      <PageHeader
        title="WhatsApp pairing"
        description="Scan the QR code with your phone to link this bot. The code refreshes automatically every ~60 seconds."
      />

      <Alert>
        <ShieldAlert className="h-4 w-4" />
        <AlertTitle>Sensitive operation</AlertTitle>
        <AlertDescription>
          Anyone who scans this QR links their WhatsApp as the bot identity. Only
          admins should open this page. Keep the dashboard URL private.
        </AlertDescription>
      </Alert>

      {!streamConnected && status !== 'connected' && (
        <p className="text-sm text-muted-foreground">
          Live updates reconnecting… status may lag briefly.
        </p>
      )}

      <div className="max-w-md">
        <PairingQRCard />
      </div>

      <ol className="mt-6 list-decimal space-y-2 pl-5 text-sm text-muted-foreground">
        <li>Open WhatsApp on your phone.</li>
        <li>Go to Settings → Linked devices → Link a device.</li>
        <li>Scan the QR above — it updates automatically when it expires.</li>
        <li>When connected, this page shows a green connected state.</li>
      </ol>
    </Page>
  )
}
