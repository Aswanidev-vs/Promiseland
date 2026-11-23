import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError, api, buildQuery, requestForm, request } from './client'

function jsonResponse(status: number, body: unknown, statusText = 'Status text'): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    statusText,
    json: async () => body,
  } as unknown as Response
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('request', () => {
  it('prefixes the versioned path and sends cookies plus JSON headers', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(200, { id: 'u1', username: 'kai' }))
    vi.stubGlobal('fetch', fetchMock)

    const result = await request<{ id: string }>('/auth/me')

    expect(result).toEqual({ id: 'u1', username: 'kai' })
    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/auth/me')
    expect(init.credentials).toBe('include')
    expect(new Headers(init.headers).get('Accept')).toBe('application/json')
  })

  it('throws ApiError parsed from the error envelope', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        jsonResponse(
          401,
          { error: { code: 'invalid_credentials', message: 'Email or password is incorrect' } },
          'Unauthorized',
        ),
      ),
    )

    await expect(request('/auth/login', { method: 'POST' })).rejects.toMatchObject({
      name: 'ApiError',
      status: 401,
      code: 'invalid_credentials',
      message: 'Email or password is incorrect',
    })

    try {
      await request('/auth/login')
    } catch (error) {
      expect(error).toBeInstanceOf(ApiError)
      expect((error as ApiError).body?.error.code).toBe('invalid_credentials')
    }
  })

  it('falls back to the HTTP status when the error body has no envelope', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse(502, { oops: true }, 'Bad Gateway')))

    await expect(request('/posts')).rejects.toMatchObject({
      status: 502,
      code: 'http_502',
      message: 'Bad Gateway',
    })
  })

  it('survives a non-JSON error body', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 503,
        statusText: 'Service Unavailable',
        json: async () => {
          throw new SyntaxError('Unexpected token')
        },
      }),
    )

    await expect(request('/posts')).rejects.toMatchObject({ status: 503, code: 'http_503' })
  })

  it('returns undefined for 204 and serialises JSON bodies in api.post', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(204, undefined))
    vi.stubGlobal('fetch', fetchMock)

    await expect(api.post<void>('/auth/logout')).resolves.toBeUndefined()
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/auth/logout')
    expect(init.method).toBe('POST')
    expect(init.body).toBeUndefined()

    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse(201, { ok: true })))
    await api.post('/communities', { name: 'Sailor Moon' })
    const postInit = vi.mocked(fetch).mock.calls[0]![1] as RequestInit
    expect(postInit.body).toBe('{"name":"Sailor Moon"}')
    expect(new Headers(postInit.headers).get('Content-Type')).toBe('application/json')
  })
})

describe('buildQuery', () => {
  it('serialises filters in order and skips empty values', () => {
    expect(
      buildQuery({ q: 'sailor moon', type: 'communities', limit: 20, cursor: '', scope: undefined }),
    ).toBe('?q=sailor+moon&type=communities&limit=20')
    expect(buildQuery({ cursor: 'abc', spoiler: false, n: 0 })).toBe('?cursor=abc&spoiler=false&n=0')
    expect(buildQuery({})).toBe('')
    expect(buildQuery()).toBe('')
  })
})

describe('requestForm', () => {
  it('leaves multipart Content-Type to the browser', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(200, { id: 1 }))
    vi.stubGlobal('fetch', fetchMock)
    const form = new FormData()
    form.append('file', new File(['image'], 'art.png', { type: 'image/png' }))

    await requestForm('/media', form)

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/media')
    expect(init.method).toBe('POST')
    expect(init.body).toBe(form)
    expect(new Headers(init.headers).has('Content-Type')).toBe(false)
    expect(new Headers(init.headers).get('Accept')).toBe('application/json')
  })
})
