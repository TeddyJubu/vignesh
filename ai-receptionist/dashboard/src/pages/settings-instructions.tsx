import { useEffect, useMemo, useState } from 'react'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { Page, PageHeader } from '@/components/page'
import { apiFetch } from '@/lib/api'
import type { Instructions } from '@/lib/models'
import { useApiState } from '@/lib/use-api'

const runbookKeys = ['julia-sales', 'julia-cs', 'julia-booking'] as const

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

  useEffect(() => {
    setSampleContact(activeContact)
  }, [activeContact])

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

  const layoutLabel =
    instructions.prompt_layout === 'stacked'
      ? 'Legacy stacked layout'
      : 'Bundled layout (SOUL → system, Knowledge → user)'

  return (
    <Page>
      <PageHeader
        title="Instructions"
        description="Edit SOUL.md and the Epicware knowledge base. Preview shows how prompts are sent to the model on each WhatsApp turn."
        right={
          <div className="flex flex-col items-end gap-1">
            <Badge variant="secondary">{layoutLabel}</Badge>
            <Badge variant="outline">Dreams are review-gated</Badge>
          </div>
        }
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
            <Tabs defaultValue="soul" className="w-full">
              <TabsList className="grid h-10 w-full grid-cols-3">
                <TabsTrigger value="soul">Soul</TabsTrigger>
                <TabsTrigger value="knowledge">Knowledge</TabsTrigger>
                <TabsTrigger value="runbooks">Runbooks</TabsTrigger>
              </TabsList>

              <TabsContent value="soul" className="mt-4 space-y-3">
                <div className="text-sm text-muted-foreground">
                  <code>knowledge/SOUL.md</code> → <code>identity_soul</code>. Injected as the{' '}
                  <span className="font-medium text-foreground">system</span> prompt (persona,
                  tone, rules).
                </div>
                <Textarea
                  className="min-h-72 font-mono text-xs"
                  value={instructions.identity_soul ?? ''}
                  onChange={(e) =>
                    state.setData({ ...instructions, identity_soul: e.target.value })
                  }
                  placeholder="Julia soul document…"
                />
              </TabsContent>

              <TabsContent value="knowledge" className="mt-4 space-y-3">
                <div className="text-sm text-muted-foreground">
                  <code>knowledge/KNOWLEDGE.md</code> + operational rules →{' '}
                  <code>client_instructions</code>. Sent in the{' '}
                  <span className="font-medium text-foreground">user</span> turn as{' '}
                  <code>EPICWARE KNOWLEDGE BASE</code>
                  {instructions.client_instructions?.source ? (
                    <>
                      {' '}
                      (source:{' '}
                      <span className="font-medium text-foreground">
                        {instructions.client_instructions.source}
                      </span>
                      )
                    </>
                  ) : null}
                  .
                </div>
                <Textarea
                  className="min-h-72 font-mono text-xs"
                  value={instructions.client_instructions?.content ?? ''}
                  onChange={(e) =>
                    state.setData({
                      ...instructions,
                      client_instructions: {
                        source:
                          instructions.client_instructions?.source ?? 'db',
                        content: e.target.value,
                      },
                    })
                  }
                  placeholder="Product facts, pricing, FAQs, escalation list…"
                />
              </TabsContent>

              <TabsContent value="runbooks" className="mt-4 space-y-4">
                <div className="text-sm text-muted-foreground">
                  Mode runbooks appended to the system prompt (<code>julia-sales</code>,{' '}
                  <code>julia-cs</code>, <code>julia-booking</code>).
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
                        placeholder={`Runbook for ${key}…`}
                      />
                    </div>
                  ))}
                </div>
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
            <CardTitle className="text-base">Prompt preview</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="grid gap-3 sm:grid-cols-2">
              <div className="space-y-1.5">
                <Label htmlFor="sample-contact">Sample contact (history)</Label>
                <Input
                  id="sample-contact"
                  value={sampleContact}
                  placeholder="e.g. 6580286424"
                  autoComplete="off"
                  onChange={(e) => setSampleContact(e.target.value)}
                />
                <div className="text-xs text-muted-foreground">
                  Optional conv id for last 5 turns in the preview history block.
                </div>
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
              className="min-h-[32rem] font-mono text-xs leading-relaxed"
              value={instructions.preview ?? ''}
              placeholder="Bundled SYSTEM + USER preview will appear here…"
            />
          </CardContent>
        </Card>
      </div>
    </Page>
  )
}
