import { useMemo, useState } from 'react'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Page, PageHeader } from '@/components/page'
import { apiFetch, setSessionToken } from '@/lib/api'

function normalizePhone(raw: string) {
  return raw.replace(/\D+/g, '')
}

type VerifyOtpResponse = {
  token: string
  role?: string
  permissions?: Record<string, boolean>
}

export function LoginPage({ onLoggedIn }: { onLoggedIn: () => void }) {
  const [phone, setPhone] = useState('')
  const [code, setCode] = useState('')
  const [step, setStep] = useState<'request' | 'verify'>('request')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const normalized = useMemo(() => normalizePhone(phone), [phone])

  async function requestOtp() {
    setLoading(true)
    setError(null)
    try {
      await apiFetch('/auth/request-otp', { method: 'POST', json: { phone: normalized } })
      setStep('verify')
    } catch (e: any) {
      setError(e?.message ?? 'Could not request OTP')
    } finally {
      setLoading(false)
    }
  }

  async function verifyOtp() {
    setLoading(true)
    setError(null)
    try {
      const res = await apiFetch<VerifyOtpResponse>('/auth/verify-otp', {
        method: 'POST',
        json: { phone: normalized, code: code.trim() },
      })
      setSessionToken(res.token)
      onLoggedIn()
    } catch (e: any) {
      setError(e?.message ?? 'Could not verify OTP')
    } finally {
      setLoading(false)
    }
  }

  return (
    <Page>
      <PageHeader
        title="Sign in"
        description="Managers can sign in via OTP. Administrators can still use the operator auth bypass."
      />

      {error && (
        <Alert>
          <AlertTitle>Authentication failed</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <Card className="max-w-lg">
        <CardHeader>
          <CardTitle className="text-base">OTP login</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="phone">Phone (digits only)</Label>
            <Input
              id="phone"
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
              placeholder="e.g. 6590013157"
              autoComplete="tel"
              inputMode="tel"
            />
            <div className="text-xs text-muted-foreground">
              Normalized as <code>{normalized || '—'}</code>
            </div>
          </div>

          {step === 'verify' ? (
            <div className="space-y-1.5">
              <Label htmlFor="code">OTP code</Label>
              <Input
                id="code"
                value={code}
                onChange={(e) => setCode(e.target.value)}
                placeholder="6-digit code"
                autoComplete="one-time-code"
                inputMode="numeric"
              />
            </div>
          ) : null}

          <div className="flex items-center justify-end gap-2">
            {step === 'verify' ? (
              <Button
                type="button"
                variant="secondary"
                onClick={() => {
                  setStep('request')
                  setCode('')
                  setError(null)
                }}
                disabled={loading}
              >
                Back
              </Button>
            ) : null}

            {step === 'request' ? (
              <Button
                type="button"
                onClick={() => void requestOtp()}
                disabled={loading || normalized.length < 7}
              >
                {loading ? 'Sending…' : 'Send OTP'}
              </Button>
            ) : (
              <Button
                type="button"
                onClick={() => void verifyOtp()}
                disabled={loading || normalized.length < 7 || code.trim().length < 4}
              >
                {loading ? 'Verifying…' : 'Verify'}
              </Button>
            )}
          </div>
        </CardContent>
      </Card>
    </Page>
  )
}

