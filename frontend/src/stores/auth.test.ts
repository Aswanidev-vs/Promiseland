import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { User } from '@/types'

vi.mock('@/api/auth', () => ({
  getMe: vi.fn(),
  login: vi.fn(),
  register: vi.fn(),
  logout: vi.fn(),
}))

vi.mock('@/api/users', () => ({
  updateMe: vi.fn(),
  updateInterests: vi.fn(),
  updateAnimeInterests: vi.fn(),
}))

import * as authApi from '@/api/auth'
import { useAuthStore } from './auth'

const user: User = {
  id: 1,
  username: 'kai',
  display_name: 'Kai',
  avatar_url: null,
  banner_url: null,
  bio: null,
  location: null,
  visibility: 'public',
  discoverable: true,
  created_at: '2026-01-01T00:00:00Z',
}

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
})

describe('auth store', () => {
  it('treats a failed session lookup as anonymous and marks itself ready', async () => {
    vi.mocked(authApi.getMe).mockRejectedValueOnce(new Error('offline'))
    const auth = useAuthStore()

    await expect(auth.fetchMe()).resolves.toBeNull()

    expect(auth.initialized).toBe(true)
    expect(auth.isReady).toBe(true)
    expect(auth.isAuthenticated).toBe(false)
  })

  it('shares one in-flight session lookup between callers', async () => {
    let resolveLookup!: (value: User) => void
    vi.mocked(authApi.getMe).mockReturnValueOnce(new Promise((resolve) => (resolveLookup = resolve)))
    const auth = useAuthStore()

    const first = auth.fetchMe()
    const second = auth.fetchMe()
    resolveLookup(user)
    await Promise.all([first, second])

    expect(authApi.getMe).toHaveBeenCalledTimes(1)
    expect(auth.isAuthenticated).toBe(true)
  })

  it('adopts the signed-in user and clears it on sign out', async () => {
    vi.mocked(authApi.login).mockResolvedValueOnce(user)
    vi.mocked(authApi.logout).mockRejectedValueOnce(new Error('offline'))
    const auth = useAuthStore()

    await auth.login({ email: 'kai@example.com', password: 'secret' })
    expect(auth.isAuthenticated).toBe(true)
    await expect(auth.logout()).resolves.toBeUndefined()
    expect(auth.isAuthenticated).toBe(false)
  })
})
