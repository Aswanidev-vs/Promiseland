import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Connection, DiscoveryCandidate } from '@/types'

vi.mock('@/api/discovery', () => ({
  listFriends: vi.fn(),
  getDiscoverySettings: vi.fn(),
  updateDiscoverySettings: vi.fn(),
  suppressCandidate: vi.fn(),
  unsuppressCandidate: vi.fn(),
  sendConnection: vi.fn(),
  listConnections: vi.fn(),
  reviewConnection: vi.fn(),
  deleteConnection: vi.fn(),
}))

import * as api from '@/api/discovery'
import { useDiscoveryStore } from './discovery'

function makeCandidate(id: number): DiscoveryCandidate {
  return {
    user: {
      id,
      username: `member${id}`,
      display_name: `Member ${id}`,
      avatar_url: null,
      banner_url: null,
      bio: null,
      location: null,
      visibility: 'public',
      discoverable: true,
      created_at: '2026-11-01T00:00:00Z',
      interests: [],
      anime_interests: [],
      follower_count: 3,
      following_count: 4,
      post_count: 5,
      is_following: false,
      is_blocked: false,
    },
    shared: { anime: ['Frieren'], genres: ['Fantasy'], interests: [], communities: [] },
    shared_count: 1,
    explanation: 'You both follow Frieren',
    connection_status: 'none',
  }
}

function makeConnection(id: number, candidateId: number): Connection {
  return {
    id,
    requester: {
      id: candidateId,
      username: `member${candidateId}`,
      display_name: `Member ${candidateId}`,
      avatar_url: null,
    },
    addressee: { id: 1, username: 'me', display_name: 'Me', avatar_url: null },
    message: 'Hello from the harbour',
    status: 'pending',
    created_at: '2026-11-20T00:00:00Z',
    updated_at: '2026-11-20T00:00:00Z',
  }
}

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
})

describe('discovery store', () => {
  it('removes a suppressed candidate optimistically and restores it on failure', async () => {
    const store = useDiscoveryStore()
    store.candidates = [makeCandidate(2), makeCandidate(3)]
    vi.mocked(api.suppressCandidate).mockRejectedValueOnce(new Error('offline'))

    await expect(store.suppress(2)).resolves.toBe(false)

    expect(store.candidates.map((candidate) => candidate.user.id)).toEqual([2, 3])
    expect(store.error).toContain('offline')

    vi.mocked(api.suppressCandidate).mockResolvedValueOnce(undefined)
    await expect(store.suppress(2)).resolves.toBe(true)
    expect(store.candidates.map((candidate) => candidate.user.id)).toEqual([3])
    expect(store.suppressedCandidates[2]).toBeDefined()

    vi.mocked(api.unsuppressCandidate).mockResolvedValueOnce(undefined)
    await expect(store.unsuppress(2)).resolves.toBe(true)
    expect(store.candidates.map((candidate) => candidate.user.id)).toEqual([2, 3])
    expect(store.suppressedCandidates[2]).toBeUndefined()
  })

  it('reviews a connection optimistically and rolls back on failure', async () => {
    const store = useDiscoveryStore()
    store.candidates = [makeCandidate(2)]
    store.candidates[0].connection_status = 'pending_in'
    store.connections = [makeConnection(7, 2)]
    vi.mocked(api.reviewConnection).mockRejectedValueOnce(new Error('conflict'))

    await expect(store.reviewConnection(7, 'accepted')).resolves.toBeNull()

    expect(store.connections[0].status).toBe('pending')
    expect(store.candidates[0].connection_status).toBe('pending_in')
    expect(store.error).toContain('conflict')

    const confirmed = { ...makeConnection(7, 2), status: 'accepted' as const }
    vi.mocked(api.reviewConnection).mockResolvedValueOnce(confirmed)
    await expect(store.reviewConnection(7, 'accepted')).resolves.toEqual(confirmed)
    expect(store.connections[0].status).toBe('accepted')
    expect(store.candidates[0].connection_status).toBe('accepted')
  })

  it('rolls back a discoverability toggle when the API rejects it', async () => {
    const store = useDiscoveryStore()
    vi.mocked(api.updateDiscoverySettings).mockRejectedValueOnce(new Error('offline'))

    await expect(store.updateSettings(false)).resolves.toBeNull()

    expect(store.settings.discoverable).toBe(true)
    expect(store.error).toContain('offline')

    vi.mocked(api.updateDiscoverySettings).mockResolvedValueOnce({ discoverable: false })
    await expect(store.updateSettings({ discoverable: false })).resolves.toEqual({
      discoverable: false,
    })
    expect(store.settings.discoverable).toBe(false)
  })

  it('keeps stale lists and reports an error when a load fails', async () => {
    const store = useDiscoveryStore()
    store.candidates = [makeCandidate(2)]
    vi.mocked(api.listFriends).mockRejectedValueOnce(new Error('offline'))

    await store.loadCandidates()

    expect(store.candidates).toHaveLength(1)
    expect(store.error).toContain('offline')
    expect(store.loading).toBe(false)

    vi.mocked(api.listConnections).mockResolvedValueOnce({
      items: [makeConnection(7, 2)],
      next_cursor: 'conn-2',
    })
    await store.loadConnections({ status: 'pending', direction: 'incoming' })

    expect(store.connections).toHaveLength(1)
    expect(store.connectionsCursor).toBe('conn-2')
    expect(api.listConnections).toHaveBeenCalledWith({ status: 'pending', direction: 'incoming' })
  })
})
