import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  getAdultProfile,
  getAdultStatus,
  likeAdult,
  listAdultMatches,
  reportAdult,
  saveAdultProfile,
  unmatch,
} from './adult'

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

describe('adult API wiring', () => {
  it('reads the gate status on its own safe endpoint', async () => {
    const status = { enabled: false, ageAssurance: 'development_stub', eligible: false, message: 'Off' }
    mockJson(status)

    await expect(getAdultStatus()).resolves.toEqual(status)
    expect(lastCall()[0]).toBe('/api/v1/adult/status')
  })

  it('maps the profile input to snake_case wire keys on PUT /adult/profile', async () => {
    const saved = {
      user_id: 1,
      display_name: 'Mira',
      intent: 'both',
      bio: 'Conventions and quiet talks.',
      boundaries: 'No late messages.',
      visibility: 'opt_in',
      created_at: '2026-11-23T00:00:00Z',
    }
    mockJson(saved)

    await expect(
      saveAdultProfile({
        displayName: 'Mira',
        intent: 'both',
        bio: 'Conventions and quiet talks.',
        boundaries: 'No late messages.',
      }),
    ).resolves.toEqual(saved)
    const [url, init] = lastCall()
    expect(url).toBe('/api/v1/adult/profile')
    expect(init.method).toBe('PUT')
    expect(JSON.parse(init.body as string)).toEqual({
      display_name: 'Mira',
      intent: 'both',
      bio: 'Conventions and quiet talks.',
      boundaries: 'No late messages.',
    })
  })

  it('posts likes, matches and unmatching on the documented paths', async () => {
    mockJson({ actor_id: 1, target_id: 7, status: 'pass', created_at: '2026-11-23T00:00:00Z' })
    await likeAdult(7, 'pass')
    let [url, init] = lastCall()
    expect(url).toBe('/api/v1/adult/like/7')
    expect(init.method).toBe('POST')
    expect(JSON.parse(init.body as string)).toEqual({ status: 'pass' })

    mockJson([])
    await expect(listAdultMatches()).resolves.toEqual([])
    expect(lastCall()[0]).toBe('/api/v1/adult/matches')

    mockJson(undefined, 204)
    await unmatch(9)
    ;[url, init] = lastCall()
    expect(url).toBe('/api/v1/adult/matches/9/unmatch')
    expect(init.method).toBe('POST')
  })

  it('sends adult safety reports with the adult_profile target type', async () => {
    mockJson(undefined, 204)
    await reportAdult({
      targetType: 'adult_profile',
      targetId: 7,
      reason: 'harassment',
      details: 'Unwanted contact after I asked to stop.',
    })
    const [url, init] = lastCall()
    expect(url).toBe('/api/v1/adult/reports')
    expect(init.method).toBe('POST')
    expect(JSON.parse(init.body as string)).toEqual({
      target_type: 'adult_profile',
      target_id: 7,
      reason: 'harassment',
      details: 'Unwanted contact after I asked to stop.',
    })
  })

  it('reads an existing opt-in profile on GET /adult/profile', async () => {
    const profile = {
      user_id: 1,
      display_name: 'Mira',
      intent: 'friendship',
      bio: 'I like conventions.',
      boundaries: 'Take it slow.',
      visibility: 'opt_in',
      created_at: '2026-11-23T00:00:00Z',
    }
    mockJson(profile)
    await expect(getAdultProfile()).resolves.toEqual(profile)
    expect(lastCall()[0]).toBe('/api/v1/adult/profile')
  })
})
