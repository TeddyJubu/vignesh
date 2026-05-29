import { useMemo, useState } from 'react'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Page, PageHeader } from '@/components/page'
import { apiFetch, getSettings, putSettings } from '@/lib/api'
import type { AppSettings, ComposioStatus } from '@/lib/models'
import { useApiState } from '@/lib/use-api'

export function IntegrationsComposioPage() {
  const status = useApiState<ComposioStatus>(() => apiFetch('/composio/status'), [])
  const settings = useApiState<AppSettings>(() => getSettings(), [])
  const [saving, setSaving] = useState(false)
  const [verifying, setVerifying] = useState(false)

  const currentKey = settings.data?.['composio.api_key'] ?? ''
  const currentAllow = settings.data?.['composio.allowlist'] ?? ''
  const currentUserID = settings.data?.['composio.user_id'] ?? ''
  const currentCalendarAcct =
    settings.data?.['composio.calendar_connected_account_id'] ?? ''
  const currentGmailAcct = settings.data?.['composio.gmail_connected_account_id'] ?? ''
  const currentTZ = settings.data?.['composio.timezone'] ?? ''

  const allowlist = useMemo(() => {
    return currentAllow
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean)
  }, [currentAllow])

  async function save(next: AppSettings) {
    setSaving(true)
    try {
      await putSettings(next)
      settings.setData(next)
      await status.refresh()
    } finally {
      setSaving(false)
    }
  }

  async function verify() {
    setVerifying(true)
    try {
      const data = await apiFetch<ComposioStatus>('/composio/status?verify=1')
      status.setData(data)
    } finally {
      setVerifying(false)
    }
  }

  const ready = Boolean(status.data?.ok)
  const configured = Boolean(status.data?.configured ?? status.data?.ok)
  const calendarReady = Boolean(status.data?.calendar_ready)
  const gmailReady = Boolean(status.data?.gmail_ready)
  const needsReauth = Boolean(status.data?.needs_reauth)

  return (
    <Page>
      <PageHeader
        title="Composio"
        description="Connect Google Calendar and Gmail via Composio for live booking and confirmation emails."
        right={
          <Badge variant={ready ? 'default' : needsReauth ? 'destructive' : configured ? 'secondary' : 'secondary'}>
            {ready ? 'Live' : needsReauth ? 'Re-auth needed' : configured ? 'Key only' : 'Not configured'}
          </Badge>
        }
      />

      {(status.error || settings.error) && (
        <Alert>
          <AlertTitle>API not reachable</AlertTitle>
          <AlertDescription>
            This page expects <code>/api/composio/status</code> and{' '}
            <code>/api/settings</code>.
          </AlertDescription>
        </Alert>
      )}

      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Connection</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            <div className="flex items-center justify-between">
              <div className="text-muted-foreground">API key</div>
              <Badge variant={configured ? 'default' : 'secondary'}>
                {configured ? 'Set' : 'Missing'}
              </Badge>
            </div>
            <div className="flex items-center justify-between">
              <div className="text-muted-foreground">Google Calendar</div>
              <Badge variant={calendarReady ? 'default' : 'secondary'}>
                {calendarReady ? 'Ready' : 'Not connected'}
              </Badge>
            </div>
            <div className="flex items-center justify-between">
              <div className="text-muted-foreground">Gmail</div>
              <Badge variant={gmailReady ? 'default' : 'secondary'}>
                {gmailReady ? 'Ready' : 'Not connected'}
              </Badge>
            </div>
            {status.data?.needs_reauth ? (
              <div className="rounded-md border border-destructive/40 bg-destructive/5 p-3 text-destructive">
                Google connections in Composio are expired or revoked. Open Composio → Users and reconnect
                Google Calendar and Gmail.
                {typeof status.data.expired_accounts === 'number' && status.data.expired_accounts > 0
                  ? ` (${status.data.expired_accounts} stale connection(s) found.)`
                  : null}
              </div>
            ) : null}
            <div className="text-muted-foreground">
              {status.data?.message ??
                'Add your Composio API key, then connect Google Calendar and Gmail in the Composio dashboard.'}
            </div>
            {status.data?.calendar_account_id ? (
              <div className="text-xs text-muted-foreground">
                Calendar account:{' '}
                <code className="text-foreground">{status.data.calendar_account_id}</code>
              </div>
            ) : null}
            {status.data?.gmail_account_id ? (
              <div className="text-xs text-muted-foreground">
                Gmail account:{' '}
                <code className="text-foreground">{status.data.gmail_account_id}</code>
              </div>
            ) : null}
            {status.data?.connected_accounts?.length ? (
              <div className="space-y-1 text-xs text-muted-foreground">
                <div>Active Composio accounts:</div>
                <ul className="list-inside list-disc text-foreground">
                  {status.data.connected_accounts.map((acct) => (
                    <li key={acct.id}>
                      {acct.toolkit_slug} — <code>{acct.id}</code>
                    </li>
                  ))}
                </ul>
              </div>
            ) : null}
            {status.data?.enabled_tools?.length ? (
              <div className="text-muted-foreground">
                Local allowlist:{' '}
                <span className="text-foreground">
                  {status.data.enabled_tools.join(', ')}
                </span>
              </div>
            ) : null}
            <div className="flex justify-end gap-2">
              <Button variant="secondary" size="sm" onClick={() => void status.refresh()}>
                Refresh
              </Button>
              <Button size="sm" onClick={() => void verify()} disabled={verifying}>
                {verifying ? 'Verifying…' : 'Verify & resolve'}
              </Button>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Configuration</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-1.5">
              <Label htmlFor="composio-api-key">API key</Label>
              <Input
                id="composio-api-key"
                type="password"
                value={currentKey}
                placeholder="comp_..."
                autoComplete="off"
                onChange={(e) =>
                  settings.setData({ ...(settings.data ?? {}), 'composio.api_key': e.target.value })
                }
              />
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="composio-user-id">Composio user ID</Label>
              <Input
                id="composio-user-id"
                value={currentUserID}
                placeholder="default"
                onChange={(e) =>
                  settings.setData({
                    ...(settings.data ?? {}),
                    'composio.user_id': e.target.value,
                  })
                }
              />
              <div className="text-xs text-muted-foreground">
                Must match the user ID used when connecting accounts in Composio.
              </div>
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="composio-timezone">Calendar timezone</Label>
              <Input
                id="composio-timezone"
                value={currentTZ}
                placeholder="Asia/Singapore"
                onChange={(e) =>
                  settings.setData({
                    ...(settings.data ?? {}),
                    'composio.timezone': e.target.value,
                  })
                }
              />
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="composio-calendar-acct">Calendar connected account ID</Label>
              <Input
                id="composio-calendar-acct"
                value={currentCalendarAcct}
                placeholder="(auto-resolve if blank)"
                onChange={(e) =>
                  settings.setData({
                    ...(settings.data ?? {}),
                    'composio.calendar_connected_account_id': e.target.value,
                  })
                }
              />
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="composio-gmail-acct">Gmail connected account ID</Label>
              <Input
                id="composio-gmail-acct"
                value={currentGmailAcct}
                placeholder="(auto-resolve if blank)"
                onChange={(e) =>
                  settings.setData({
                    ...(settings.data ?? {}),
                    'composio.gmail_connected_account_id': e.target.value,
                  })
                }
              />
              <div className="text-xs text-muted-foreground">
                Leave blank to auto-detect the first active account per toolkit.
              </div>
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="composio-allowlist">Tool group allowlist</Label>
              <Textarea
                id="composio-allowlist"
                value={currentAllow}
                placeholder="(optional) gmail, calendar, slack"
                onChange={(e) =>
                  settings.setData({
                    ...(settings.data ?? {}),
                    'composio.allowlist': e.target.value,
                  })
                }
              />
              <div className="text-xs text-muted-foreground">
                Optional local safety gate.
                {allowlist.length ? ` Currently: ${allowlist.join(', ')}` : ''}
              </div>
            </div>

            <div className="flex justify-end gap-2">
              <Button variant="secondary" onClick={() => void settings.refresh()} disabled={saving}>
                Reload
              </Button>
              <Button
                onClick={() => void save(settings.data ?? {})}
                disabled={saving || settings.loading}
              >
                {saving ? 'Saving…' : 'Save'}
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </Page>
  )
}
