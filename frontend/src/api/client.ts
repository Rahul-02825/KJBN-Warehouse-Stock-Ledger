// All HTTP calls go through here, so error handling is the same on every page.

const baseURL = import.meta.env.VITE_API_URL ?? ''

// Matches the backend error body: {"error": {"code", "message", "fields"?, "details"?}}
export class ApiError extends Error {
  status: number
  code: string
  fields: Record<string, string>
  details: Record<string, string>

  constructor(status: number, code: string, message: string, fields = {}, details = {}) {
    super(message)
    this.status = status
    this.code = code
    this.fields = fields
    this.details = details
  }
}

async function request<T>(method: string, path: string, body?: unknown, headers: Record<string, string> = {}): Promise<T> {
  let res: Response
  try {
    res = await fetch(baseURL + path, {
      method,
      headers: body === undefined ? headers : { 'Content-Type': 'application/json', ...headers },
      body: body === undefined ? undefined : JSON.stringify(body),
    })
  } catch {
    throw new ApiError(0, 'NETWORK', 'Cannot reach the server. Check that the API is running.')
  }

  const data = await res.json().catch(() => null)
  if (!res.ok) {
    const e = data?.error
    if (!e) {
      // No API error body: a proxy or gateway answered because the API itself did not.
      throw new ApiError(res.status, 'NETWORK', `The server is not responding (${res.status}). Check that the API is running.`)
    }
    throw new ApiError(res.status, e.code, e.message, e.fields, e.details)
  }
  return data as T
}

export const api = {
  get: <T>(path: string) => request<T>('GET', path),
  post: <T>(path: string, body: unknown, headers?: Record<string, string>) => request<T>('POST', path, body, headers),
  patch: <T>(path: string, body: unknown) => request<T>('PATCH', path, body),
}

// Full URL for links such as CSV downloads.
export function apiURL(path: string): string {
  return baseURL + path
}
