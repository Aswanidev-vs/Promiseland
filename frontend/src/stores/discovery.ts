import { ref } from 'vue'
import { defineStore } from 'pinia'
import { errorMessage } from '@/api/client'
import * as api from '@/api/discovery'
import type {
  CandidateConnectionStatus,
  Connection,
  ConnectionReviewStatus,
  DiscoveryCandidate,
  DiscoverySettings,
  ListConnectionsParams,
  ListFriendsParams,
  SendConnectionInput,
  UpdateDiscoverySettingsInput,
} from '@/types'

interface SuppressedEntry {
  candidate: DiscoveryCandidate
  index: number
}

/**
 * Failure-tolerant cache for the discovery list, connection list and settings.
 * Suppress, unsuppress and review are optimistic: the UI updates immediately
 * and is rolled back if the API rejects the change.
 */
export const useDiscoveryStore = defineStore('discovery', () => {
  const candidates = ref<DiscoveryCandidate[]>([])
  const connections = ref<Connection[]>([])
  const settings = ref<DiscoverySettings>({ discoverable: true })
  const loading = ref(false)
  const error = ref<string | null>(null)
  const nextCursor = ref('')
  const connectionsCursor = ref('')
  /** Hidden suggestions are kept so unsuppress can restore them without a refetch. */
  const suppressedCandidates = ref<Record<number, SuppressedEntry>>({})

  function findCandidate(candidateId: number): DiscoveryCandidate | undefined {
    return candidates.value.find((candidate) => candidate.user.id === candidateId)
  }

  /** One of the two parties is the current user, who is never a candidate. */
  function candidateForConnection(connection: Connection): DiscoveryCandidate | undefined {
    return candidates.value.find(
      (candidate) =>
        candidate.user.id === connection.requester.id ||
        candidate.user.id === connection.addressee.id,
    )
  }

  function statusAfterReview(status: Connection['status']): CandidateConnectionStatus {
    if (status === 'accepted') return 'accepted'
    if (status === 'pending') return 'pending_in'
    return 'none'
  }

  function rememberConnection(connection: Connection): Connection {
    const index = connections.value.findIndex((item) => item.id === connection.id)
    if (index !== -1) connections.value.splice(index, 1, connection)
    else connections.value.push(connection)
    return connection
  }

  async function loadCandidates(p: ListFriendsParams = {}, append = false): Promise<void> {
    loading.value = true
    error.value = null
    try {
      const page = await api.listFriends(p)
      candidates.value = append ? [...candidates.value, ...page.items] : page.items
      nextCursor.value = page.next_cursor
      if (!append) suppressedCandidates.value = {}
    } catch (cause) {
      error.value = errorMessage(cause)
    } finally {
      loading.value = false
    }
  }

  async function loadConnections(p: ListConnectionsParams = {}, append = false): Promise<void> {
    loading.value = true
    error.value = null
    try {
      const page = await api.listConnections(p)
      connections.value = append ? [...connections.value, ...page.items] : page.items
      connectionsCursor.value = page.next_cursor
    } catch (cause) {
      error.value = errorMessage(cause)
    } finally {
      loading.value = false
    }
  }

  async function loadSettings(): Promise<void> {
    loading.value = true
    error.value = null
    try {
      settings.value = await api.getDiscoverySettings()
    } catch (cause) {
      error.value = errorMessage(cause)
    } finally {
      loading.value = false
    }
  }

  /** Accepts either a boolean or the settings patch for caller convenience. */
  async function updateSettings(
    input: UpdateDiscoverySettingsInput | boolean,
  ): Promise<DiscoverySettings | null> {
    const discoverable = typeof input === 'boolean' ? input : input.discoverable
    const previous = settings.value.discoverable
    settings.value = { discoverable }
    try {
      const confirmed = await api.updateDiscoverySettings({ discoverable })
      settings.value = confirmed
      error.value = null
      return confirmed
    } catch (cause) {
      settings.value = { discoverable: previous }
      error.value = errorMessage(cause)
      return null
    }
  }

  async function suppress(userId: number): Promise<boolean> {
    const index = candidates.value.findIndex((candidate) => candidate.user.id === userId)
    const entry: SuppressedEntry | null =
      index === -1 ? null : { candidate: candidates.value[index], index }
    if (entry) {
      candidates.value.splice(entry.index, 1)
      suppressedCandidates.value = { ...suppressedCandidates.value, [userId]: entry }
    }
    try {
      await api.suppressCandidate(userId)
      error.value = null
      return true
    } catch (cause) {
      if (entry) {
        candidates.value.splice(Math.min(entry.index, candidates.value.length), 0, entry.candidate)
        const remaining = { ...suppressedCandidates.value }
        delete remaining[userId]
        suppressedCandidates.value = remaining
      }
      error.value = errorMessage(cause)
      return false
    }
  }

  async function unsuppress(userId: number): Promise<boolean> {
    const entry = suppressedCandidates.value[userId] ?? null
    if (entry) {
      candidates.value.splice(
        Math.min(entry.index, candidates.value.length),
        0,
        entry.candidate,
      )
    }
    try {
      await api.unsuppressCandidate(userId)
      if (entry) {
        const remaining = { ...suppressedCandidates.value }
        delete remaining[userId]
        suppressedCandidates.value = remaining
      }
      error.value = null
      return true
    } catch (cause) {
      if (entry) {
        const index = candidates.value.findIndex((candidate) => candidate.user.id === userId)
        if (index !== -1) candidates.value.splice(index, 1)
      }
      error.value = errorMessage(cause)
      return false
    }
  }

  async function sendConnection(input: SendConnectionInput): Promise<Connection | null> {
    const candidate = findCandidate(input.userId)
    const previousStatus = candidate?.connection_status ?? null
    if (candidate) candidate.connection_status = 'pending_out'
    try {
      const connection = rememberConnection(await api.sendConnection(input))
      error.value = null
      return connection
    } catch (cause) {
      if (candidate && previousStatus !== null) candidate.connection_status = previousStatus
      error.value = errorMessage(cause)
      return null
    }
  }

  async function reviewConnection(
    id: number,
    status: ConnectionReviewStatus,
  ): Promise<Connection | null> {
    const index = connections.value.findIndex((connection) => connection.id === id)
    const previousConnection = index === -1 ? null : connections.value[index]
    const candidate = previousConnection ? candidateForConnection(previousConnection) : undefined
    const previousStatus = candidate?.connection_status ?? null

    if (previousConnection) {
      connections.value.splice(index, 1, { ...previousConnection, status })
    }
    if (candidate) candidate.connection_status = statusAfterReview(status)

    try {
      const confirmed = await api.reviewConnection(id, status)
      if (index !== -1) connections.value.splice(index, 1, confirmed)
      if (candidate) candidate.connection_status = statusAfterReview(confirmed.status)
      error.value = null
      return confirmed
    } catch (cause) {
      if (previousConnection) connections.value.splice(index, 1, previousConnection)
      if (candidate && previousStatus !== null) candidate.connection_status = previousStatus
      error.value = errorMessage(cause)
      return null
    }
  }

  async function deleteConnection(id: number): Promise<boolean> {
    const index = connections.value.findIndex((connection) => connection.id === id)
    const removed = index === -1 ? null : connections.value[index]
    const candidate = removed ? candidateForConnection(removed) : undefined
    const previousStatus = candidate?.connection_status ?? null

    if (index !== -1) connections.value.splice(index, 1)
    if (candidate) candidate.connection_status = 'none'

    try {
      await api.deleteConnection(id)
      error.value = null
      return true
    } catch (cause) {
      if (removed) connections.value.splice(Math.min(index, connections.value.length), 0, removed)
      if (candidate && previousStatus !== null) candidate.connection_status = previousStatus
      error.value = errorMessage(cause)
      return false
    }
  }

  function setError(message: string | null): void {
    error.value = message
  }

  function clear(): void {
    candidates.value = []
    connections.value = []
    settings.value = { discoverable: true }
    nextCursor.value = ''
    connectionsCursor.value = ''
    suppressedCandidates.value = {}
    error.value = null
    loading.value = false
  }

  return {
    candidates,
    connections,
    settings,
    loading,
    error,
    nextCursor,
    connectionsCursor,
    suppressedCandidates,
    loadCandidates,
    loadConnections,
    loadSettings,
    updateSettings,
    suppress,
    unsuppress,
    sendConnection,
    reviewConnection,
    deleteConnection,
    setError,
    clear,
  }
})
