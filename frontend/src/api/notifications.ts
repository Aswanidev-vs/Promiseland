import { buildQuery, request } from './client'
import type { Notification, Page, PaginationParams } from '@/types'

export async function listNotifications(p: PaginationParams = {}): Promise<Page<Notification>> {
  return request<Page<Notification>>(`/notifications${buildQuery({ ...p })}`)
}

export async function markRead(id: number): Promise<void> {
  await request<void>(`/notifications/${id}/read`, { method: 'PATCH' })
}

/** Bulk read uses its own literal segment so it cannot collide with an id. */
export async function markAllRead(): Promise<void> {
  await request<void>('/notifications/read-all', { method: 'PATCH' })
}
