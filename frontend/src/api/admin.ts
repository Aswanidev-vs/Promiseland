import { buildQuery, request } from './client'
import type { Page, PaginationParams } from '@/types'

export interface AdminCounts {
  users: number
  active_users: number
  suspended_users: number
  active_sessions: number
  posts: number
  comments: number
  communities: number
  events: number
  media: number
  media_bytes: number
  open_reports: number
  pending_memberships: number
}

export interface AdminUser {
  id: number
  username: string
  display_name: string
  email: string
  is_active: boolean
  is_staff: boolean
  post_count: number
  active_sessions: number
  created_at: string
}

export interface AdminPost {
  id: number
  title: string
  type: string
  author: string
  community: string
  comment_count: number
  vote_score: number
  deleted_at: string | null
  created_at: string
}

export interface AdminCommunity {
  id: number
  name: string
  slug: string
  visibility: string
  owner: string
  member_count: number
  post_count: number
  created_at: string
}

export interface AdminEvent {
  id: number
  title: string
  event_type: string
  organizer: string
  city: string
  starts_at: string
  status: string
  is_online: boolean
  going_count: number
}

export interface AdminOverview {
  counts: AdminCounts
  recent_users: Array<Pick<AdminUser, 'id' | 'username' | 'display_name' | 'created_at'>>
  recent_posts: Array<{ id: number; title: string; author: string; community: string; created_at: string }>
  open_reports: Array<{
    id: number
    target_type: string
    target_id: number
    reason: string
    created_at: string
  }>
}

export async function getAdminMe(): Promise<{ is_staff: boolean }> {
  return request<{ is_staff: boolean }>('/admin/me')
}

export async function getAdminOverview(): Promise<AdminOverview> {
  return request<AdminOverview>('/admin/overview')
}

export function listAdminUsers(p: PaginationParams & { q?: string } = {}): Promise<Page<AdminUser>> {
  return request<Page<AdminUser>>(`/admin/users${buildQuery({ ...p })}`)
}

export async function setAdminUserActive(id: number, isActive: boolean): Promise<{ id: number; is_active: boolean }> {
  return request<{ id: number; is_active: boolean }>(`/admin/users/${id}`, {
    method: 'PATCH',
    body: JSON.stringify({ is_active: isActive }),
  })
}

export function listAdminPosts(p: PaginationParams & { q?: string } = {}): Promise<Page<AdminPost>> {
  return request<Page<AdminPost>>(`/admin/posts${buildQuery({ ...p })}`)
}

export function listAdminCommunities(p: PaginationParams & { q?: string } = {}): Promise<Page<AdminCommunity>> {
  return request<Page<AdminCommunity>>(`/admin/communities${buildQuery({ ...p })}`)
}

export function listAdminEvents(p: PaginationParams & { q?: string } = {}): Promise<Page<AdminEvent>> {
  return request<Page<AdminEvent>>(`/admin/events${buildQuery({ ...p })}`)
}
