import { request } from './client'
import type {
  AnimeInterestInput,
  Interest,
  UpdateSettingsInput,
  User,
  UserProfile,
} from '@/types'

export async function getUser(username: string): Promise<UserProfile> {
  return request<UserProfile>(`/users/${encodeURIComponent(username)}`)
}

export async function updateMe(patch: Partial<User>): Promise<User> {
  return request<User>('/users/me', { method: 'PATCH', body: JSON.stringify(patch) })
}

export async function listInterests(): Promise<Interest[]> {
  return request<Interest[]>('/users/me/interests')
}

export async function updateInterests(ids: number[]): Promise<UserProfile> {
  return request<UserProfile>('/users/me/interests', {
    method: 'PUT',
    body: JSON.stringify({ ids }),
  })
}

export async function updateAnimeInterests(items: AnimeInterestInput[]): Promise<UserProfile> {
  return request<UserProfile>('/users/me/anime-interests', {
    method: 'PUT',
    body: JSON.stringify({
      items: items.map((item) => ({
        anime_id: item.animeId,
        interest_type: item.interestType,
      })),
    }),
  })
}

export async function followUser(id: number): Promise<void> {
  await request<void>(`/users/${id}/follow`, { method: 'POST' })
}

export async function unfollowUser(id: number): Promise<void> {
  await request<void>(`/users/${id}/follow`, { method: 'DELETE' })
}

export async function blockUser(id: number): Promise<void> {
  await request<void>(`/users/${id}/block`, { method: 'POST' })
}

export async function unblockUser(id: number): Promise<void> {
  await request<void>(`/users/${id}/block`, { method: 'DELETE' })
}

export async function updateSettings(patch: UpdateSettingsInput): Promise<User> {
  return request<User>('/users/me', { method: 'PATCH', body: JSON.stringify(patch) })
}
