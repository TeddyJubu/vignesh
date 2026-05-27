import { Navigate, Route, Routes } from 'react-router-dom'
import { AppShell } from '@/components/app-shell'
import { OverviewPage } from '@/pages/overview'
import { SettingsProvidersPage } from '@/pages/settings-providers'
import { SettingsInstructionsPage } from '@/pages/settings-instructions'
import { MemoryRecallPage } from '@/pages/memory-recall'
import { MemoryDreamsPage } from '@/pages/memory-dreams'
import { IntegrationsComposioPage } from '@/pages/integrations-composio'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Page, PageHeader } from '@/components/page'

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
      <Routes>
        <Route path="/" element={<OverviewPage />} />
        <Route path="/settings/providers" element={<SettingsProvidersPage />} />
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
    </AppShell>
  )
}
