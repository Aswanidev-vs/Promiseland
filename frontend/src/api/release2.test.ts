import { afterEach, describe, expect, it, vi } from 'vitest'
import { listCreators, featurePost, unfeaturePost } from './creators'
import { cancelRsvp, createEvent, listEvents, listEventComments, rsvpEvent } from './events'
import { createPost, getFeed } from './posts'
import { applyToTeamUp, listTeamUps, reviewApplication } from './teamups'

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

function mockJson(body: unknown, status = 200): void {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse(status, body)))
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('R2 endpoint wiring', () => {
  it('builds event filters in the documented q,type,from,to,city,country,online,limit,cursor order', async () => {
    mockJson({ items: [], next_cursor: '' })
    await listEvents({
      q: 'comic con',
      type: 'convention',
      from: '2026-12-01',
      to: '2026-12-05',
      city: 'Osaka',
      country: 'JP',
      online: false,
      limit: 20,
      cursor: 'evt-1',
    })
    expect(lastCall()[0]).toBe(
      '/api/v1/events?q=comic+con&type=convention&from=2026-12-01&to=2026-12-05&city=Osaka&country=JP&online=false&limit=20&cursor=evt-1',
    )

    mockJson({ items: [], next_cursor: '' })
    await listEvents({ online: true, city: '', cursor: '' })
    expect(lastCall()[0]).toBe('/api/v1/events?online=true')
  })

  it('maps createEvent client fields to snake_case wire keys', async () => {
    mockJson({ id: 1 })
    await createEvent({
      title: 'Winter meetup',
      description: 'Annual gathering',
      eventType: 'meetup',
      startsAt: '2026-12-20T10:00:00Z',
      endsAt: '2026-12-20T16:00:00Z',
      timezone: 'Asia/Tokyo',
      venueText: 'Harbour cafe',
      city: 'Tokyo',
      country: 'JP',
      isOnline: false,
      capacity: 40,
      coverMediaId: 9,
    })
    const [url, init] = lastCall()
    expect(url).toBe('/api/v1/events')
    expect(init.method).toBe('POST')
    expect(JSON.parse(init.body as string)).toEqual({
      title: 'Winter meetup',
      description: 'Annual gathering',
      event_type: 'meetup',
      starts_at: '2026-12-20T10:00:00Z',
      ends_at: '2026-12-20T16:00:00Z',
      timezone: 'Asia/Tokyo',
      venue_text: 'Harbour cafe',
      city: 'Tokyo',
      country: 'JP',
      is_online: false,
      capacity: 40,
      cover_media_id: 9,
    })
  })

  it('posts and deletes RSVPs on the documented subresource', async () => {
    mockJson({ id: 7, rsvp_status: 'maybe' })
    await rsvpEvent(7, 'maybe')
    let [url, init] = lastCall()
    expect(url).toBe('/api/v1/events/7/rsvp')
    expect(init.method).toBe('POST')
    expect(JSON.parse(init.body as string)).toEqual({ status: 'maybe' })

    await cancelRsvp(7)
    ;[url, init] = lastCall()
    expect(url).toBe('/api/v1/events/7/rsvp')
    expect(init.method).toBe('DELETE')

    mockJson({ items: [], next_cursor: '' })
    await listEventComments(7, { limit: 5, cursor: 'cmt-1' })
    expect(lastCall()[0]).toBe('/api/v1/events/7/comments?limit=5&cursor=cmt-1')
  })

  it('builds creator filters in the documented category,q,limit,cursor order', async () => {
    mockJson({ items: [], next_cursor: '' })
    await listCreators({ category: 'cosplay', q: 'mira', limit: 12, cursor: 'cr-1' })
    expect(lastCall()[0]).toBe('/api/v1/creators?category=cosplay&q=mira&limit=12&cursor=cr-1')
  })

  it('toggles featured posts through /users/me/featured/{postId}', async () => {
    mockJson(undefined, 204)
    await featurePost(42)
    let [url, init] = lastCall()
    expect(url).toBe('/api/v1/users/me/featured/42')
    expect(init.method).toBe('POST')

    await unfeaturePost(42)
    ;[url, init] = lastCall()
    expect(url).toBe('/api/v1/users/me/featured/42')
    expect(init.method).toBe('DELETE')
  })

  it('propagates creatorCategory through createPost and the feed query', async () => {
    mockJson({ id: 3 })
    await createPost({
      type: 'text',
      title: 'Cosplay build',
      body: 'Finished the wig',
      creatorCategory: 'cosplay',
    })
    let [url, init] = lastCall()
    expect(url).toBe('/api/v1/posts')
    expect(init.method).toBe('POST')
    expect(JSON.parse(init.body as string)).toEqual({
      type: 'text',
      title: 'Cosplay build',
      body: 'Finished the wig',
      creator_category: 'cosplay',
    })

    mockJson({ items: [], next_cursor: '' })
    await getFeed({ creatorCategory: 'fan_art', limit: 10 })
    ;[url] = lastCall()
    expect(url).toBe('/api/v1/posts?creatorCategory=fan_art&limit=10')
  })

  it('wires team-up pagination and application review', async () => {
    mockJson({ items: [], next_cursor: '' })
    await listTeamUps({ limit: 8, cursor: 'tu-1' })
    expect(lastCall()[0]).toBe('/api/v1/team-ups?limit=8&cursor=tu-1')

    mockJson({ id: 5 })
    await applyToTeamUp(3, 'I can bring a camera')
    let [url, init] = lastCall()
    expect(url).toBe('/api/v1/team-ups/3/apply')
    expect(init.method).toBe('POST')
    expect(JSON.parse(init.body as string)).toEqual({ message: 'I can bring a camera' })

    await reviewApplication(3, 5, 'accepted')
    ;[url, init] = lastCall()
    expect(url).toBe('/api/v1/team-ups/3/applications/5')
    expect(init.method).toBe('PATCH')
    expect(JSON.parse(init.body as string)).toEqual({ status: 'accepted' })
  })
})
