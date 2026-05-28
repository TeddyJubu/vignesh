export type ApiError = {
  status: number
  message: string
  details?: unknown
}

async function parseJsonSafe(res: Response): Promise<unknown> {
  const contentType = res.headers.get('content-type') ?? ''
  if (!contentType.includes('application/json')) return undefined
  try {
    return await res.json()
  } catch {
    return undefined
  }
}

export async function apiFetch<T>(
  path: string,
  init: RequestInit & { json?: unknown } = {},
): Promise<T> {
  const headers = new Headers(init.headers)
  if (init.json !== undefined) headers.set('content-type', 'application/json')

  const base = import.meta.env.BASE_URL.replace(/\/?$/, '/')
  const res = await fetch(`${base}api${path}`, {
    ...init,
    headers,
    body: init.json !== undefined ? JSON.stringify(init.json) : init.body,
  })

  const body = await parseJsonSafe(res)

  if (!res.ok) {
    const msg =
      typeof body === 'object' && body && 'error' in body
        ? String((body as any).error)
        : res.statusText || 'Request failed'
    const err: ApiError = { status: res.status, message: msg, details: body }
    throw err
  }

  if (body === undefined) {
    const err: ApiError = {
      status: res.status,
      message: 'API returned a non-JSON response (is the Go server running?)',
    }
    throw err
  }

  return body as T
}

// Convenience wrappers to normalize Go API response shapes.
export async function getSettings() {
  const res = await apiFetch<{ settings: Record<string, string> }>('/settings')
  return res.settings
}

export async function putSettings(next: Record<string, string | undefined>) {
  await apiFetch('/settings', { method: 'PUT', json: { settings: next } })
}

export async function getDreams() {
  const res = await apiFetch<{ dreams: unknown[] }>('/dreams')
  return res.dreams
}

