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

  const res = await fetch(`/api${path}`, {
    ...init,
    headers,
    body: init.json !== undefined ? JSON.stringify(init.json) : init.body,
  })

  if (!res.ok) {
    const body = await parseJsonSafe(res)
    const msg =
      typeof body === 'object' && body && 'error' in body
        ? String((body as any).error)
        : res.statusText || 'Request failed'
    const err: ApiError = { status: res.status, message: msg, details: body }
    throw err
  }

  const json = (await parseJsonSafe(res)) as T
  return json
}

