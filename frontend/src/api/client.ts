import type { ApiErrorBody, QueryValue } from '@/types'
import { useFailureStore } from '@/stores/failure'

/**
 * A 503 from the API (or the dev proxy answering for a down API) is a
 * service-level failure, not a request-level one. Shows the 503 page; any
 * successful response clears it again.
 */
function trackAvailability(status: number, ok: boolean): void {
  try {
    const failure = useFailureStore()
    if (!ok && status === 503) failure.markServiceUnavailable()
    else if (ok) failure.markServiceUnavailable(false)
  } catch {
    // No active pinia (unit tests calling the client directly).
  }
}

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
 * Stable, ordered query strings for list endpoints. Empty, null and undefined
 * values are omitted so callers can pass partial filter objects directly.
 */
export function buildQuery(params: Record<string, QueryValue> = {}): string {
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === '') continue
    search.set(key, String(value))
  }
  const query = search.toString()
  return query ? `?${query}` : ''
}

/**
 * Typed request helper. Sends cookies, always asks for JSON, and normalises
 * errors into ApiError so callers never branch on raw Response objects.
 */
export async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  if (!headers.has('Accept')) headers.set('Accept', 'application/json')
  // Let the browser set multipart boundaries for FormData uploads.
  const isFormData = typeof FormData !== 'undefined' && init.body instanceof FormData
  if (init.body !== undefined && !isFormData && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }

  const response = await fetch(joinPath(path), {
    ...init,
    headers,
    credentials: 'include',
  })

  if (!response.ok) {
    trackAvailability(response.status, false)
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

  trackAvailability(response.status, true)
  if (response.status === 204) return undefined as T
  return (await response.json()) as T
}

/** Stable human-readable message for error toasts and inline states. */
export function errorMessage(error: unknown, fallback = 'Something went wrong. Try again.'): string {
  return error instanceof Error && error.message ? error.message : fallback
}

/** POST a FormData body. The browser supplies the multipart Content-Type. */
export function requestForm<T>(path: string, form: FormData, init: RequestInit = {}): Promise<T> {
  return request<T>(path, { ...init, method: 'POST', body: form })
}

function withMethod<T>(path: string, method: string, init: RequestInit = {}): Promise<T> {
  return request<T>(path, { ...init, method })
}

/** Thin, extensible facade. Domain modules can add typed helpers later. */
export const api = {
  request,
  requestForm,
  get: <T>(path: string, init?: RequestInit) => withMethod<T>(path, 'GET', init),
  post: <T>(path: string, body?: unknown, init?: RequestInit) =>
    withMethod<T>(path, 'POST', { ...init, body: body === undefined ? undefined : JSON.stringify(body) }),
  patch: <T>(path: string, body?: unknown, init?: RequestInit) =>
    withMethod<T>(path, 'PATCH', { ...init, body: body === undefined ? undefined : JSON.stringify(body) }),
  put: <T>(path: string, body?: unknown, init?: RequestInit) =>
    withMethod<T>(path, 'PUT', { ...init, body: body === undefined ? undefined : JSON.stringify(body) }),
  delete: <T>(path: string, init?: RequestInit) => withMethod<T>(path, 'DELETE', init),
}
