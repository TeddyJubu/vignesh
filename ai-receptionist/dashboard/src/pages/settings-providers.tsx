import { useMemo, useState } from 'react'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { Page, PageHeader } from '@/components/page'
import { getSettings, putSettings } from '@/lib/api'
import type { AppSettings, ProviderId } from '@/lib/models'
import { useApiState } from '@/lib/use-api'

const providerOptions: Array<{ id: ProviderId; label: string }> = [
  { id: 'ollama', label: 'Ollama Cloud' },
  { id: 'openai', label: 'OpenAI' },
  { id: 'anthropic', label: 'Anthropic' },
  { id: 'openrouter', label: 'OpenRouter' },
  { id: 'custom', label: 'Custom (OpenAI-compatible)' },
]

function get(settings: AppSettings, key: string) {
  return settings[key] ?? ''
}

function set(settings: AppSettings, key: string, value: string) {
  return { ...settings, [key]: value || undefined }
}

export function SettingsProvidersPage() {
  const state = useApiState<AppSettings>(() => getSettings(), [])
  const [saving, setSaving] = useState(false)
  const settings = state.data ?? {}

  const provider = useMemo(
    () => (settings['ai.provider'] as ProviderId | undefined) ?? 'ollama',
    [settings],
  )

  async function save(next: AppSettings) {
    setSaving(true)
    try {
      await putSettings(next)
      state.setData(next)
    } finally {
      setSaving(false)
    }
  }

  return (
    <Page>
      <PageHeader
        title="Providers"
        description="Choose the active LLM provider and store local default credentials."
        right={
          <Badge variant="secondary">
            Env vars can override secrets
          </Badge>
        }
      />

      {state.error && (
        <Alert>
          <AlertTitle>Could not load settings</AlertTitle>
          <AlertDescription>
            The Go API should expose <code>GET /api/settings</code> and{' '}
            <code>PUT /api/settings</code>.
          </AlertDescription>
        </Alert>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Active provider</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-4 md:grid-cols-2">
          <FieldShell
            label="Provider"
            description="Used for new turns. Secrets can be overridden by env at runtime."
          >
            <Select
              value={provider}
              onValueChange={(v) => { if (v) void save(set(settings, 'ai.provider', v)) }}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select provider" />
              </SelectTrigger>
              <SelectContent>
                {providerOptions.map((p) => (
                  <SelectItem key={p.id} value={p.id}>
                    {p.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FieldShell>

          <FieldShell
            label="Default model"
            description={
              <>
                Saved per-provider under <code>{provider}.model</code>.
              </>
            }
          >
            <Input
              value={get(settings, `${provider}.model`)}
              placeholder="e.g. gpt-4.1-mini, claude-3-5-sonnet..."
              onChange={(e) =>
                state.setData(set(settings, `${provider}.model`, e.target.value))
              }
            />
          </FieldShell>

          <div className="md:col-span-2">
            <Separator className="my-2" />
          </div>

          {provider === 'ollama' && (
            <>
              <Field
                label="Ollama Cloud API key"
                placeholder="ollama_..."
                value={get(settings, 'ollama.api_key')}
                onChange={(v) => state.setData(set(settings, 'ollama.api_key', v))}
                type="password"
                description="Stored in DB by default; env `OLLAMA_API_KEY` overrides at runtime."
              />
              <Field
                label="Ollama Cloud API URL"
                placeholder="https://..."
                value={get(settings, 'ollama.api_url')}
                onChange={(v) => state.setData(set(settings, 'ollama.api_url', v))}
                description="Optional. Defaults to https://ollama.com/api/chat"
              />
            </>
          )}

          {provider === 'openai' && (
            <Field
              label="OpenAI API key"
              placeholder="sk-..."
              value={get(settings, 'openai.api_key')}
              onChange={(v) => state.setData(set(settings, 'openai.api_key', v))}
              type="password"
              description="Stored in DB by default; env `OPENAI_API_KEY` overrides at runtime."
            />
          )}

          {provider === 'anthropic' && (
            <Field
              label="Anthropic API key"
              placeholder="sk-ant-..."
              value={get(settings, 'anthropic.api_key')}
              onChange={(v) =>
                state.setData(set(settings, 'anthropic.api_key', v))
              }
              type="password"
              description="Stored in DB by default; env `ANTHROPIC_API_KEY` overrides at runtime."
            />
          )}

          {provider === 'openrouter' && (
            <Field
              label="OpenRouter API key"
              placeholder="sk-or-..."
              value={get(settings, 'openrouter.api_key')}
              onChange={(v) =>
                state.setData(set(settings, 'openrouter.api_key', v))
              }
              type="password"
              description="Stored in DB by default; env `OPENROUTER_API_KEY` overrides at runtime."
            />
          )}

          {provider === 'custom' && (
            <>
              <Field
                label="Custom base URL"
                placeholder="https://your-openai-compatible-host/v1"
                value={get(settings, 'custom.base_url')}
                onChange={(v) =>
                  state.setData(set(settings, 'custom.base_url', v))
                }
                description="OpenAI-compatible base URL (should include /v1)."
              />
              <Field
                label="Custom API key"
                placeholder="..."
                value={get(settings, 'custom.api_key')}
                onChange={(v) =>
                  state.setData(set(settings, 'custom.api_key', v))
                }
                type="password"
                description="Stored in DB by default; env `CUSTOM_API_KEY` overrides at runtime."
              />
            </>
          )}

          <div className="md:col-span-2 flex justify-end gap-2">
            <Button
              variant="secondary"
              onClick={() => void state.refresh()}
              disabled={saving || state.loading}
            >
              Reload
            </Button>
            <Button
              onClick={() => void save(settings)}
              disabled={saving || state.loading}
            >
              {saving ? 'Saving…' : 'Save'}
            </Button>
          </div>
        </CardContent>
      </Card>
    </Page>
  )
}

function FieldShell({
  label,
  description,
  children,
  htmlFor,
}: {
  label: React.ReactNode
  description?: React.ReactNode
  children: React.ReactNode
  htmlFor?: string
}) {
  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between gap-2">
        <Label htmlFor={htmlFor}>{label}</Label>
      </div>
      {children}
      {description ? (
        <div className="text-xs text-muted-foreground">{description}</div>
      ) : null}
    </div>
  )
}

function Field({
  label,
  value,
  onChange,
  placeholder,
  description,
  type,
}: {
  label: string
  value: string
  onChange: (v: string) => void
  placeholder?: string
  description?: React.ReactNode
  type?: React.ComponentProps<typeof Input>['type']
}) {
  const id = useMemo(
    () => `field-${label.toLowerCase().replace(/[^a-z0-9]+/g, '-')}`,
    [label],
  )
  return (
    <FieldShell label={label} description={description} htmlFor={id}>
      <Input
        id={id}
        type={type}
        value={value}
        placeholder={placeholder}
        autoComplete="off"
        onChange={(e) => onChange(e.target.value)}
      />
    </FieldShell>
  )
}

