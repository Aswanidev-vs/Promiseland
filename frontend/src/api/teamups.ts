import { buildQuery, request } from './client'
import type {
  ApplicationStatus,
  CreateTeamUpInput,
  ListTeamUpsParams,
  Page,
  TeamUp,
  TeamUpApplication,
} from '@/types'

export async function listTeamUps(p: ListTeamUpsParams = {}): Promise<Page<TeamUp>> {
  return request<Page<TeamUp>>(
    `/team-ups${buildQuery({ status: p.status, category: p.category, q: p.q, limit: p.limit, cursor: p.cursor })}`,
  )
}

export async function getTeamUp(id: number): Promise<TeamUp> {
  return request<TeamUp>(`/team-ups/${id}`)
}

export async function createTeamUp(input: CreateTeamUpInput): Promise<TeamUp> {
  return request<TeamUp>('/team-ups', {
    method: 'POST',
    body: JSON.stringify({
      title: input.title,
      description: input.description,
      event_id: input.eventId,
      categories: input.categories,
    }),
  })
}

export async function updateTeamUp(id: number, patch: Partial<TeamUp>): Promise<TeamUp> {
  return request<TeamUp>(`/team-ups/${id}`, { method: 'PATCH', body: JSON.stringify(patch) })
}

export async function deleteTeamUp(id: number): Promise<void> {
  await request<void>(`/team-ups/${id}`, { method: 'DELETE' })
}

export async function applyToTeamUp(id: number, message: string): Promise<TeamUpApplication> {
  return request<TeamUpApplication>(`/team-ups/${id}/apply`, {
    method: 'POST',
    body: JSON.stringify({ message }),
  })
}

export async function listApplications(id: number): Promise<TeamUpApplication[]> {
  return request<TeamUpApplication[]>(`/team-ups/${id}/applications`)
}

export async function reviewApplication(
  id: number,
  applicationId: number,
  status: Exclude<ApplicationStatus, 'pending'>,
): Promise<TeamUpApplication> {
  return request<TeamUpApplication>(`/team-ups/${id}/applications/${applicationId}`, {
    method: 'PATCH',
    body: JSON.stringify({ status }),
  })
}
