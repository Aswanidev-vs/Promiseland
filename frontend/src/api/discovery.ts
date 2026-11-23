import { buildQuery, request } from './client'
import type {
  Connection,
  ConnectionReviewStatus,
  DiscoveryCandidate,
  DiscoverySettings,
  ListConnectionsParams,
  ListFriendsParams,
  Page,
  SendConnectionInput,
  UpdateDiscoverySettingsInput,
} from '@/types'

/** GET /discovery/friends?limit=&cursor= */
export async function listFriends(p: ListFriendsParams = {}): Promise<Page<DiscoveryCandidate>> {
  return request<Page<DiscoveryCandidate>>(
    `/discovery/friends${buildQuery({ limit: p.limit, cursor: p.cursor })}`,
  )
}

/** GET /discovery/settings */
export async function getDiscoverySettings(): Promise<DiscoverySettings> {
  return request<DiscoverySettings>('/discovery/settings')
}

/** PATCH /discovery/settings — body {discoverable} */
export async function updateDiscoverySettings(
  patch: UpdateDiscoverySettingsInput,
): Promise<DiscoverySettings> {
  return request<DiscoverySettings>('/discovery/settings', {
    method: 'PATCH',
    body: JSON.stringify({ discoverable: patch.discoverable }),
  })
}

/** POST /discovery/{userId}/suppress — hide a suggestion. */
export async function suppressCandidate(userId: number): Promise<void> {
  await request<void>(`/discovery/${userId}/suppress`, { method: 'POST' })
}

/** DELETE /discovery/{userId}/suppress — make the suggestion visible again. */
export async function unsuppressCandidate(userId: number): Promise<void> {
  await request<void>(`/discovery/${userId}/suppress`, { method: 'DELETE' })
}

/** POST /connections — the client's userId is sent as the wire's user_id. */
export async function sendConnection(input: SendConnectionInput): Promise<Connection> {
  return request<Connection>('/connections', {
    method: 'POST',
    body: JSON.stringify({ user_id: input.userId, message: input.message }),
  })
}

/** GET /connections?status=&direction=&limit=&cursor= */
export async function listConnections(p: ListConnectionsParams = {}): Promise<Page<Connection>> {
  return request<Page<Connection>>(
    `/connections${buildQuery({
      status: p.status,
      direction: p.direction,
      limit: p.limit,
      cursor: p.cursor,
    })}`,
  )
}

/** PATCH /connections/{id} — body {status} */
export async function reviewConnection(id: number, status: ConnectionReviewStatus): Promise<Connection> {
  return request<Connection>(`/connections/${id}`, {
    method: 'PATCH',
    body: JSON.stringify({ status }),
  })
}

/** DELETE /connections/{id} */
export async function deleteConnection(id: number): Promise<void> {
  await request<void>(`/connections/${id}`, { method: 'DELETE' })
}
