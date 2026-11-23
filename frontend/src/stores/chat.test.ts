import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { ChatServer } from '@/types'

vi.mock('@/api/chat', () => ({
  listServers: vi.fn(),
  listConversations: vi.fn(),
  listChannels: vi.fn(),
  createServer: vi.fn(),
  joinServer: vi.fn(),
  createChannel: vi.fn(),
  createConversation: vi.fn(),
}))

import * as api from '@/api/chat'
import { useChatStore } from './chat'

function makeServer(id: number): ChatServer {
  return {
    id,
    owner: { id: 1, username: 'kai', display_name: 'Kai', avatar_url: null },
    name: 'Harbour crew',
    slug: 'harbour-crew',
    description: 'After-hours chat',
    icon_url: null,
    member_count: 12,
    channels: [],
    role: 'member',
    created_at: '2026-11-01T00:00:00Z',
  }
}

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
})

describe('chat store', () => {
  it('caches the server list and the active scope', async () => {
    vi.mocked(api.listServers).mockResolvedValueOnce([makeServer(1)])
    const store = useChatStore()

    await expect(store.loadServers()).resolves.toHaveLength(1)
    store.setActiveScope({ channelId: 5 })
    expect(store.servers[0]?.name).toBe('Harbour crew')
    expect(store.activeScope).toEqual({ channelId: 5 })
    expect(store.error).toBeNull()
  })

  it('stays usable when the API fails', async () => {
    vi.mocked(api.listServers).mockRejectedValueOnce(new Error('offline'))
    const store = useChatStore()

    await expect(store.loadServers()).resolves.toEqual([])
    expect(store.loading).toBe(false)
    expect(store.error).toContain('offline')
  })
})
