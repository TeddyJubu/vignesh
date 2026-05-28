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

  const currentKey = settings.data?.['composio.api_key'] ?? ''
  const currentAllow = settings.data?.['composio.allowlist'] ?? ''

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

  const ready = Boolean(status.data?.ok)

  return (
    <Page>
      <PageHeader
        title="Composio"
        description="Store the Composio API key. Manage integrations and tool access in the Composio dashboard."
        right={
          <Badge variant={ready ? 'default' : 'secondary'}>
            {ready ? 'Connected' : 'Not configured'}
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
              <div className="text-muted-foreground">Status</div>
              <div className="font-medium">{ready ? 'Healthy' : 'Unknown'}</div>
            </div>
            <div className="text-muted-foreground">
              {status.data?.message ??
                'Add your Composio API key to enable Composio-backed tools.'}
            </div>
            {status.data?.enabled_tools?.length ? (
              <div className="text-muted-foreground">
                Enabled:{' '}
                <span className="text-foreground">
                  {status.data.enabled_tools.join(', ')}
                </span>
              </div>
            ) : null}
            <div className="flex justify-end">
              <Button variant="secondary" size="sm" onClick={() => void status.refresh()}>
                Refresh
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
              <div className="text-xs text-muted-foreground">
                Stored in DB by default; can be overridden by env at runtime.
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
                Optional local safety gate. Leave blank to avoid restricting tool groups here; control tools in Composio.
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

