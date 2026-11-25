import { afterEach, describe, expect, it, vi } from 'vitest'
import { verifyEmail } from './auth'
import { uploadMedia } from './media'
import { getFeed, votePost } from './posts'
import { search } from './search'
import { updateAnimeInterests } from './users'

function jsonResponse(status: number, body: unknown): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    statusText: 'OK',
    json: async () => body,
  } as unknown as Response
}

function lastCall(): [string, RequestInit] {
  return vi.mocked(fetch).mock.calls.at(-1) as [string, RequestInit]
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('R1 endpoint wiring', () => {
  it('builds feed and search query strings from partial params', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(jsonResponse(200, { items: [], next_cursor: '' })),
    )
    await getFeed({ scope: 'following', community: 'sailor-moon', limit: 20, cursor: '' })
    expect(lastCall()[0]).toBe('/api/v1/posts?scope=following&community=sailor-moon&limit=20')

    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(jsonResponse(200, { users: [], communities: [], posts: [] })),
    )
    await search({ q: 'kai', type: 'users' })
    expect(lastCall()[0]).toBe('/api/v1/search?q=kai&type=users')
  })

  it('uploads media as multipart FormData on the file field', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse(200, {
        id: 1,
        url: '/media/1',
        mime_type: 'image/png',
        size_bytes: 4,
        width: 1,
        height: 1,
      }),
    )
    vi.stubGlobal('fetch', fetchMock)
    const file = new File(['image'], 'art.png', { type: 'image/png' })

    const media = await uploadMedia(file)

    expect(media.id).toBe(1)
    const [url, init] = lastCall()
    expect(url).toBe('/api/v1/media')
    expect(init.method).toBe('POST')
    expect(new Headers(init.headers).has('Content-Type')).toBe(false)
    const form = init.body as FormData
    expect(form.get('file')).toBe(file)
  })

  it('maps camelCase client inputs to the documented wire keys', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(jsonResponse(200, { id: 1 })),
    )
    await updateAnimeInterests([{ animeId: 7, interestType: 'watching' }])
    const [url, init] = lastCall()
    expect(url).toBe('/api/v1/users/me/anime-interests')
    expect(init.method).toBe('PUT')
    expect(JSON.parse(init.body as string)).toEqual({
      items: [{ anime_id: 7, interest_type: 'watching' }],
    })
  })

  it('uses the conventions auth and voting paths', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse(204, undefined)))
    await verifyEmail('token-1')
    expect(lastCall()[0]).toBe('/api/v1/auth/verify')

    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(jsonResponse(200, { id: 4 })),
    )
    await votePost(4, 1)
    expect(lastCall()[0]).toBe('/api/v1/posts/4/reactions')
    expect(JSON.parse(lastCall()[1].body as string)).toEqual({ vote: 1 })
  })
})
