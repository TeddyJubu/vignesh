export type ProviderId =
  | 'ollama'
  | 'openai'
  | 'anthropic'
  | 'openrouter'
  | 'custom'

export type AppSettings = Record<string, string | undefined>

export type Instructions = {
  identity_soul?: string
  runbooks?: Record<string, string>
  client_instructions?: {
    source: 'file' | 'db' | 'unknown'
    content: string
  }
  sample_contact?: string
  preview?: string
}

export type DreamProposal = {
  id: string
  created_at: string
  status: 'proposed' | 'applied' | 'rejected' | string
  title: string
  rationale?: string
  diff?: string
  patch?: unknown
}

export type ProviderPing = {
  ok: boolean
  provider?: ProviderId
  model?: string
  message?: string
}

export type ComposioStatus = {
  ok: boolean
  message?: string
  enabled_tools?: string[]
}

export type RecallResult = {
  query: string
  items: Array<{
    id: string
    title?: string
    snippet: string
    score?: number
    meta?: Record<string, unknown>
  }>
}

