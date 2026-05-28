import { lazy, Suspense } from 'react'
import { Navigate, Route, Routes } from 'react-router-dom'
import { AppShell } from '@/components/app-shell'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Page, PageHeader } from '@/components/page'

const OverviewPage = lazy(() =>
  import('@/pages/overview').then((m) => ({ default: m.OverviewPage })),
)
const SettingsProvidersPage = lazy(() =>
  import('@/pages/settings-providers').then((m) => ({
    default: m.SettingsProvidersPage,
  })),
)
const SettingsInstructionsPage = lazy(() =>
  import('@/pages/settings-instructions').then((m) => ({
    default: m.SettingsInstructionsPage,
  })),
)
const MemoryRecallPage = lazy(() =>
  import('@/pages/memory-recall').then((m) => ({ default: m.MemoryRecallPage })),
)
const MemoryDreamsPage = lazy(() =>
  import('@/pages/memory-dreams').then((m) => ({ default: m.MemoryDreamsPage })),
)
const IntegrationsComposioPage = lazy(() =>
  import('@/pages/integrations-composio').then((m) => ({
    default: m.IntegrationsComposioPage,
  })),
)

function PageFallback() {
  return (
    <Page>
      <PageHeader title="Loading…" description="Fetching page resources." />
    </Page>
  )
}

function NotFound() {
  return (
    <Page>
      <PageHeader title="Not found" description="This route does not exist." />
      <Card>
        <CardHeader>
          <CardTitle className="text-base">404</CardTitle>
        </CardHeader>
        <CardContent className="text-sm text-muted-foreground">
          The page you requested was not found.
        </CardContent>
      </Card>
    </Page>
  )
}

export default function App() {
  return (
    <AppShell>
      <Suspense fallback={<PageFallback />}>
        <Routes>
          <Route path="/" element={<OverviewPage />} />
          <Route
            path="/settings/providers"
            element={<SettingsProvidersPage />}
          />
          <Route
            path="/settings/instructions"
            element={<SettingsInstructionsPage />}
          />
          <Route path="/memory/recall" element={<MemoryRecallPage />} />
          <Route path="/memory/dreams" element={<MemoryDreamsPage />} />
          <Route
            path="/integrations/composio"
            element={<IntegrationsComposioPage />}
          />
          <Route path="/overview" element={<Navigate to="/" replace />} />
          <Route path="*" element={<NotFound />} />
        </Routes>
      </Suspense>
    </AppShell>
  )
}
