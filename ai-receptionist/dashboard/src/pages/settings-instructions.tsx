import { useMemo, useState } from 'react'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { Page, PageHeader } from '@/components/page'
import { apiFetch } from '@/lib/api'
import type { Instructions } from '@/lib/models'
import { useApiState } from '@/lib/use-api'

const runbookKeys = ['julia-default', 'julia-qualify', 'julia-support']

export function SettingsInstructionsPage() {
  const state = useApiState<Instructions>(() => apiFetch('/instructions'), [])
  const [saving, setSaving] = useState(false)

  const instructions = state.data ?? {}
  const runbooks = instructions.runbooks ?? {}

  const activeContact = useMemo(
    () => instructions.sample_contact ?? '',
    [instructions.sample_contact],
  )
  const [sampleContact, setSampleContact] = useState(activeContact)

  async function save(next: Instructions) {
    setSaving(true)
    try {
      const updated = await apiFetch<Instructions>('/instructions', {
        method: 'PUT',
        json: next,
      })
      state.setData(updated)
    } finally {
      setSaving(false)
    }
  }

  return (
    <Page>
      <PageHeader
        title="Instructions"
        description="Edit identity + runbooks + client instructions, and preview the built system prompt."
        right={<Badge variant="secondary">Review-gated dreams apply here</Badge>}
      />

      {state.error && (
        <Alert>
          <AlertTitle>Could not load instructions</AlertTitle>
          <AlertDescription>
            The Go API should expose <code>GET /api/instructions</code> and{' '}
            <code>PUT /api/instructions</code>.
          </AlertDescription>
        </Alert>
      )}

      <div className="grid gap-4 lg:grid-cols-2">
        <Card className="min-w-0">
          <CardHeader>
            <CardTitle className="text-base">Editable sources</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <Tabs defaultValue="soul">
              <TabsList>
                <TabsTrigger value="soul">Identity</TabsTrigger>
                <TabsTrigger value="runbooks">Runbooks</TabsTrigger>
                <TabsTrigger value="client">Client</TabsTrigger>
              </TabsList>

              <TabsContent value="soul" className="space-y-3">
                <div className="text-sm text-muted-foreground">
                  Stored as <code>identity_soul</code>.
                </div>
                <Textarea
                  className="min-h-72 font-mono text-xs"
                  value={instructions.identity_soul ?? ''}
                  onChange={(e) =>
                    state.setData({ ...instructions, identity_soul: e.target.value })
                  }
                  placeholder="Assistant identity, boundaries, tone, and defaults…"
                />
              </TabsContent>

              <TabsContent value="runbooks" className="space-y-4">
                <div className="text-sm text-muted-foreground">
                  Stored under agent notes (keyed by mode).
                </div>
                <div className="space-y-4">
                  {runbookKeys.map((key) => (
                    <div key={key} className="space-y-2">
                      <div className="text-sm font-medium">{key}</div>
                      <Textarea
                        className="min-h-40 font-mono text-xs"
                        value={runbooks[key] ?? ''}
                        onChange={(e) =>
                          state.setData({
                            ...instructions,
                            runbooks: { ...runbooks, [key]: e.target.value },
                          })
                        }
                        placeholder={`Runbook content for ${key}…`}
                      />
                    </div>
                  ))}
                </div>
              </TabsContent>

              <TabsContent value="client" className="space-y-3">
                <div className="flex items-center justify-between gap-2">
                  <div className="text-sm text-muted-foreground">
                    {instructions.client_instructions?.source ? (
                      <>
                        Source:{' '}
                        <span className="font-medium text-foreground">
                          {instructions.client_instructions.source}
                        </span>
                      </>
                    ) : (
                      <>Client instructions</>
                    )}
                  </div>
                </div>
                <Textarea
                  className="min-h-72 font-mono text-xs"
                  value={instructions.client_instructions?.content ?? ''}
                  onChange={(e) =>
                    state.setData({
                      ...instructions,
                      client_instructions: {
                        source:
                          instructions.client_instructions?.source ?? 'unknown',
                        content: e.target.value,
                      },
                    })
                  }
                  placeholder="Client instructions (e.g. knowledge/instructions.md)…"
                />
              </TabsContent>
            </Tabs>

            <div className="flex justify-end gap-2">
              <Button
                variant="secondary"
                onClick={() => void state.refresh()}
                disabled={saving || state.loading}
              >
                Reload
              </Button>
              <Button
                onClick={() => void save(instructions)}
                disabled={saving || state.loading}
              >
                {saving ? 'Saving…' : 'Save'}
              </Button>
            </div>
          </CardContent>
        </Card>

        <Card className="min-w-0">
          <CardHeader>
            <CardTitle className="text-base">Preview</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="grid gap-3 sm:grid-cols-2">
              <div className="space-y-2">
                <div className="text-sm font-medium">Sample contact</div>
                <Input
                  value={sampleContact}
                  placeholder="e.g. +15551234567"
                  onChange={(e) => setSampleContact(e.target.value)}
                />
              </div>
              <div className="flex items-end justify-end">
                <Button
                  variant="secondary"
                  onClick={() =>
                    void save({
                      ...instructions,
                      sample_contact: sampleContact,
                    })
                  }
                  disabled={saving}
                >
                  Refresh preview
                </Button>
              </div>
            </div>

            <Textarea
              readOnly
              className="min-h-96 font-mono text-xs"
              value={instructions.preview ?? ''}
              placeholder="Preview will appear here…"
            />
          </CardContent>
        </Card>
      </div>
    </Page>
  )
}

