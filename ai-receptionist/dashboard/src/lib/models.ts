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
  /** bundled (default) or stacked */
  prompt_layout?: string
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

export type ProviderStatus = {
  provider?: ProviderId
  model?: string
  configured?: boolean
}

export type ProviderPing = {
  ok: boolean
  provider?: ProviderId
  model?: string
  message?: string
  cached?: boolean
}

export type ComposioStatus = {
  ok: boolean
  configured?: boolean
  message?: string
  enabled_tools?: string[]
  verified?: boolean
  calendar_ready?: boolean
  gmail_ready?: boolean
  needs_reauth?: boolean
  expired_accounts?: number
  user_id?: string
  timezone?: string
  calendar_account_id?: string
  gmail_account_id?: string
  connected_accounts?: Array<{
    id: string
    toolkit_slug: string
    status: string
    user_id?: string
  }>
}

export type RecallResult = {
  items: Array<{
    text: string
    score?: number
    source?: 'sqlite' | 'graphiti' | string
    created_at?: string | null
  }>
  snippet?: string
}

