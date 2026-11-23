import type { ApiErrorBody } from '@/types'

/** All REST calls are versioned under this prefix (PRD §14). */
export const API_BASE = '/api/v1'

/** Thrown for every non-2xx response, with the API's error envelope parsed. */
export class ApiError extends Error {
  readonly status: number
  readonly code: string
  readonly details?: Record<string, unknown>
  readonly body?: ApiErrorBody

  constructor(status: number, code: string, message: string, body?: ApiErrorBody) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.details = body?.error.details
    this.body = body
  }
}

function joinPath(path: string): string {
  return `${API_BASE}${path.startsWith('/') ? path : `/${path}`}`
}

/**
 * Typed request helper. Sends cookies, always asks for JSON, and normalises
 * errors into ApiError so callers never branch on raw Response objects.
 */
export async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  if (!headers.has('Accept')) headers.set('Accept', 'application/json')
  if (init.body !== undefined && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }

  const response = await fetch(joinPath(path), {
    ...init,
    headers,
    credentials: 'include',
  })

  if (!response.ok) {
    let body: ApiErrorBody | undefined
    try {
      const parsed: unknown = await response.json()
      if (
        typeof parsed === 'object' &&
        parsed !== null &&
        'error' in parsed &&
        typeof (parsed as ApiErrorBody).error?.code === 'string'
      ) {
        body = parsed as ApiErrorBody
      }
    } catch {
      // Non-JSON error (proxy 502, maintenance page, etc.) — use HTTP status.
    }

    const code = body?.error.code || `http_${response.status}`
    const message = body?.error.message || response.statusText || 'The request could not be completed'
    throw new ApiError(response.status, code, message, body)
  }

  if (response.status === 204) return undefined as T
  return (await response.json()) as T
}

function withMethod<T>(path: string, method: string, init: RequestInit = {}): Promise<T> {
  return request<T>(path, { ...init, method })
}

/** Thin, extensible facade. Domain modules can add typed helpers later. */
export const api = {
  request,
  get: <T>(path: string, init?: RequestInit) => withMethod<T>(path, 'GET', init),
  post: <T>(path: string, body?: unknown, init?: RequestInit) =>
    withMethod<T>(path, 'POST', { ...init, body: body === undefined ? undefined : JSON.stringify(body) }),
  patch: <T>(path: string, body?: unknown, init?: RequestInit) =>
    withMethod<T>(path, 'PATCH', { ...init, body: body === undefined ? undefined : JSON.stringify(body) }),
  put: <T>(path: string, body?: unknown, init?: RequestInit) =>
    withMethod<T>(path, 'PUT', { ...init, body: body === undefined ? undefined : JSON.stringify(body) }),
  delete: <T>(path: string, init?: RequestInit) => withMethod<T>(path, 'DELETE', init),
}
