import { Badge } from '@/components/ui/badge'
import { Link } from 'react-router-dom'
import { usePairingOptional } from '@/lib/pairing-context'
import { derivePairingStatus } from '@/lib/pairing-types'
import { cn } from '@/lib/utils'

const labels: Record<string, string> = {
  checking: 'WhatsApp…',
  connected: 'WhatsApp connected',
  scan: 'Scan to pair',
  waiting: 'Pairing…',
  unreachable: 'WhatsApp offline',
  timeout: 'QR expired',
}

export function PairingStatusChip({ className }: { className?: string }) {
  const ctx = usePairingOptional()
  const status = derivePairingStatus(ctx?.pairing ?? null)

  const variant =
    status === 'connected'
      ? 'default'
      : status === 'unreachable' || status === 'timeout'
        ? 'destructive'
        : status === 'scan'
          ? 'secondary'
          : 'outline'

  return (
    <Link to="/pair" className={cn('inline-flex', className)}>
      <Badge variant={variant} className="cursor-pointer hover:opacity-90">
        {labels[status] ?? 'WhatsApp'}
        {!ctx?.streamConnected && status !== 'connected' ? ' · reconnecting' : ''}
      </Badge>
    </Link>
  )
}
