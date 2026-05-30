export type PairingSnapshot = {
  supported: boolean
  reachable: boolean
  logged_in: boolean | null
  connected: boolean | null
  qr_available: boolean
  event: string | null
  updated_at: string | null
  expires_at: string | null
  detail: string | null
}

export type PairingStatus =
  | 'checking'
  | 'connected'
  | 'scan'
  | 'waiting'
  | 'unreachable'
  | 'timeout'

export function derivePairingStatus(
  pairing: PairingSnapshot | null,
): PairingStatus {
  if (!pairing) return 'checking'
  if (!pairing.reachable) return 'unreachable'
  if (pairing.logged_in && pairing.connected) return 'connected'
  if (pairing.qr_available) return 'scan'
  if (pairing.event === 'timeout') return 'timeout'
  return 'waiting'
}
