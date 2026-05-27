import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Page, PageHeader } from '@/components/page'
import { apiFetch } from '@/lib/api'
import type { ComposioStatus, ProviderPing } from '@/lib/models'
import { useApiState } from '@/lib/use-api'
import { RefreshCcw } from 'lucide-react'

export function OverviewPage() {
  const ping = useApiState<ProviderPing>(() => apiFetch('/providers/ping'), [])
  const composio = useApiState<ComposioStatus>(
    () => apiFetch('/composio/status'),
    [],
  )

  return (
    <Page>
      <PageHeader
        title="Overview"
        description="Quick status for providers, memory, and integrations."
        right={
          <Button
            variant="secondary"
            size="sm"
            onClick={() => {
              void ping.refresh()
              void composio.refresh()
            }}
          >
            <RefreshCcw className="mr-2 h-4 w-4" />
            Refresh
          </Button>
        }
      />

      {(ping.error || composio.error) && (
        <Alert>
          <AlertTitle>API not reachable</AlertTitle>
          <AlertDescription>
            Start the Go app that serves this dashboard and exposes the REST API
            under <code>/api</code>.
          </AlertDescription>
        </Alert>
      )}

      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader className="flex flex-row items-start justify-between gap-2">
            <CardTitle className="text-base">Provider</CardTitle>
            <Badge variant={ping.data?.ok ? 'default' : 'secondary'}>
              {ping.data?.ok ? 'Healthy' : 'Unknown'}
            </Badge>
          </CardHeader>
          <CardContent className="space-y-2 text-sm">
            <div className="flex items-center justify-between">
              <div className="text-muted-foreground">Current</div>
              <div className="font-medium">{ping.data?.provider ?? '—'}</div>
            </div>
            <div className="flex items-center justify-between">
              <div className="text-muted-foreground">Model</div>
              <div className="font-medium">{ping.data?.model ?? '—'}</div>
            </div>
            {ping.data?.message ? (
              <div className="text-muted-foreground">{ping.data.message}</div>
            ) : null}
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-start justify-between gap-2">
            <CardTitle className="text-base">Composio</CardTitle>
            <Badge variant={composio.data?.ok ? 'default' : 'secondary'}>
              {composio.data?.ok ? 'Connected' : 'Not configured'}
            </Badge>
          </CardHeader>
          <CardContent className="space-y-2 text-sm">
            {composio.data?.message ? (
              <div className="text-muted-foreground">{composio.data.message}</div>
            ) : (
              <div className="text-muted-foreground">
                Configure an API key in Integrations → Composio.
              </div>
            )}
            {composio.data?.enabled_tools?.length ? (
              <div className="text-muted-foreground">
                Enabled tool groups:{' '}
                <span className="text-foreground">
                  {composio.data.enabled_tools.join(', ')}
                </span>
              </div>
            ) : null}
          </CardContent>
        </Card>
      </div>
    </Page>
  )
}

