import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  deleteConnection,
  getDiscoverySettings,
  listConnections,
  listFriends,
  reviewConnection,
  sendConnection,
  suppressCandidate,
  unsuppressCandidate,
  updateDiscoverySettings,
} from './discovery'

function jsonResponse(status: number, body: unknown): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    statusText: 'OK',
    json: async () => body,
  } as unknown as Response
}

function mockJson(body: unknown, status = 200): void {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse(status, body)))
}

function lastCall(): [string, RequestInit] {
  return vi.mocked(fetch).mock.calls.at(-1) as [string, RequestInit]
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('discovery API wiring', () => {
  it('builds the friends query with limit and cursor, omitting empty values', async () => {
    mockJson({ items: [], next_cursor: '' })
    await listFriends({ limit: 20, cursor: 'cand-1' })
    expect(lastCall()[0]).toBe('/api/v1/discovery/friends?limit=20&cursor=cand-1')

    mockJson({ items: [], next_cursor: '' })
    await listFriends({ limit: 0, cursor: '' })
    expect(lastCall()[0]).toBe('/api/v1/discovery/friends?limit=0')
  })

  it('reads and patches discovery settings on the documented path', async () => {
    mockJson({ discoverable: true })
    await expect(getDiscoverySettings()).resolves.toEqual({ discoverable: true })
    let [url, init] = lastCall()
    expect(url).toBe('/api/v1/discovery/settings')
    expect(init.method).toBeUndefined()

    mockJson({ discoverable: false })
    await expect(updateDiscoverySettings({ discoverable: false })).resolves.toEqual({
      discoverable: false,
    })
    ;[url, init] = lastCall()
    expect(url).toBe('/api/v1/discovery/settings')
    expect(init.method).toBe('PATCH')
    expect(JSON.parse(init.body as string)).toEqual({ discoverable: false })
  })

  it('posts and deletes suppression on /discovery/{userId}/suppress', async () => {
    mockJson(undefined, 204)
    await suppressCandidate(42)
    let [url, init] = lastCall()
    expect(url).toBe('/api/v1/discovery/42/suppress')
    expect(init.method).toBe('POST')

    await unsuppressCandidate(42)
    ;[url, init] = lastCall()
    expect(url).toBe('/api/v1/discovery/42/suppress')
    expect(init.method).toBe('DELETE')
  })

  it('maps sendConnection input to the user_id wire key and drops an absent message', async () => {
    mockJson({ id: 1 })
    await sendConnection({ userId: 42 })
    let [url, init] = lastCall()
    expect(url).toBe('/api/v1/connections')
    expect(init.method).toBe('POST')
    expect(JSON.parse(init.body as string)).toEqual({ user_id: 42 })

    mockJson({ id: 2 })
    await sendConnection({ userId: 43, message: 'Let us swap concert stories' })
    ;[url, init] = lastCall()
    expect(url).toBe('/api/v1/connections')
    expect(JSON.parse(init.body as string)).toEqual({
      user_id: 43,
      message: 'Let us swap concert stories',
    })
  })

  it('builds connection filters in the documented status,direction,limit,cursor order', async () => {
    mockJson({ items: [], next_cursor: '' })
    await listConnections({
      status: 'pending',
      direction: 'incoming',
      limit: 10,
      cursor: 'conn-1',
    })
    expect(lastCall()[0]).toBe(
      '/api/v1/connections?status=pending&direction=incoming&limit=10&cursor=conn-1',
    )

    mockJson({ items: [], next_cursor: '' })
    await listConnections({ direction: 'all', status: undefined, cursor: '' })
    expect(lastCall()[0]).toBe('/api/v1/connections?direction=all')
  })

  it('reviews and deletes a connection on /connections/{id}', async () => {
    mockJson({ id: 9, status: 'accepted' })
    await expect(reviewConnection(9, 'accepted')).resolves.toEqual({ id: 9, status: 'accepted' })
    let [url, init] = lastCall()
    expect(url).toBe('/api/v1/connections/9')
    expect(init.method).toBe('PATCH')
    expect(JSON.parse(init.body as string)).toEqual({ status: 'accepted' })

    mockJson(undefined, 204)
    await deleteConnection(9)
    ;[url, init] = lastCall()
    expect(url).toBe('/api/v1/connections/9')
    expect(init.method).toBe('DELETE')
  })
})
