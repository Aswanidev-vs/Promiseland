/** Pure helpers shared by the R3 chat views. No UI primitives live here. */
import type { ChatChannel, ChatMessage, Conversation, UserSummary } from '@/types'
import { formatDay } from './format'

export interface MessageGroup {
  key: string
  day: string
  author: UserSummary
  messages: ChatMessage[]
}

/** Group consecutive messages by calendar day, then by author. Replies stay in order. */
export function groupMessages(
  messages: ChatMessage[],
  dayLabel: (value: string) => string = formatDay,
): MessageGroup[] {
  const groups: MessageGroup[] = []
  for (const message of messages) {
    const day = dayLabel(message.created_at)
    const author = message.author
    const last = groups[groups.length - 1]
    const canJoin =
      !!last &&
      last.day === day &&
      last.author?.id === author?.id &&
      !!message.parent_id === !!last.messages[last.messages.length - 1]?.parent_id

    if (canJoin && last) {
      last.messages.push(message)
      continue
    }

    groups.push({
      key: `${day}-${author?.id ?? 'unknown'}-${message.id}`,
      day,
      author,
      messages: [message],
    })
  }
  return groups
}

/** Append socket deliveries to REST history without duplicating ids. */
export function mergeMessages(current: ChatMessage[], incoming: ChatMessage[]): ChatMessage[] {
  const byId = new Map<number, ChatMessage>()
  for (const item of current) byId.set(item.id, item)
  for (const item of incoming) {
    if (!item || typeof item.id !== 'number') continue
    const existing = byId.get(item.id)
    byId.set(item.id, existing && existing.deleted && !item.deleted ? existing : item)
  }
  return [...byId.values()].sort((a, b) => String(a.created_at).localeCompare(String(b.created_at)))
}

/** The REST layer is authoritative about access; this only drops explicitly hidden channels. */
export function isVisibleChannel(channel: ChatChannel): boolean {
  const candidate = channel as ChatChannel & {
    visible?: boolean
    is_private?: boolean
    private?: boolean
  }
  if (candidate.visible === false || candidate.is_private === true || candidate.private === true) return false
  return channel.type === 'text'
}

/** Show the other person for DMs, the group name for group conversations. */
export function conversationLabel(conversation: Conversation, currentUserId?: number): string {
  if (conversation.type === 'direct') {
    const other = conversation.members?.find((member) => member.id !== currentUserId)
    if (other) return other.display_name || other.username
  }
  return conversation.name || 'Group conversation'
}

export function conversationMembers(conversation: Conversation, currentUserId?: number): UserSummary[] {
  const members = conversation.members || []
  if (conversation.type !== 'direct') return members
  const other = members.filter((member) => member.id !== currentUserId)
  return other.length ? other : members
}

function toArray(value: unknown): unknown[] {
  if (Array.isArray(value)) return value
  if (value && typeof value === 'object') return Object.values(value as Record<string, unknown>)
  return []
}

/** Scope values may be keyed by id, `channel:1`, or `conversation:1` depending on the hub payload. */
function scopedEntries(value: unknown, scopeId: number): unknown[] {
  if (Array.isArray(value)) return value
  if (!value || typeof value !== 'object') return []
  const record = value as Record<string, unknown>
  const direct = record[String(scopeId)] ?? record[`channel:${scopeId}`] ?? record[`conversation:${scopeId}`]
  if (direct !== undefined) return toArray(direct)
  return toArray(record)
}

/** Socket messages are merged by the views; this just filters them to the active scope. */
export function normalizeSocketMessages(value: unknown, scope: { channelId?: number; conversationId?: number }): ChatMessage[] {
  const scopeId = scope.channelId ?? scope.conversationId ?? 0
  return scopedEntries(value, scopeId).filter((entry): entry is ChatMessage => {
    if (!entry || typeof entry !== 'object' || typeof (entry as ChatMessage).id !== 'number') return false
    const message = entry as ChatMessage
    if (scope.channelId && message.channel_id && message.channel_id !== scope.channelId) return false
    if (scope.conversationId && message.conversation_id && message.conversation_id !== scope.conversationId) return false
    return true
  })
}

/** Accept the common typing payload shapes and reduce them to display names. */
export function normalizeTyping(
  value: unknown,
  scope: { channelId?: number; conversationId?: number },
  users: UserSummary[] = [],
): string[] {
  const scopeId = scope.channelId ?? scope.conversationId ?? 0
  const nameById = (id: number) => users.find((user) => user.id === id)?.display_name || users.find((user) => user.id === id)?.username
  const names: string[] = []

  for (const entry of scopedEntries(value, scopeId)) {
    if (entry == null || entry === false) continue
    if (typeof entry === 'number') {
      const name = nameById(entry)
      if (name) names.push(name)
      continue
    }
    if (typeof entry === 'string') {
      names.push(entry)
      continue
    }
    if (typeof entry !== 'object') continue
    const item = entry as Record<string, unknown>
    if (item.isTyping === false || item.typing === false) continue
    const id = Number(item.userId ?? item.user_id ?? item.id)
    const raw = item.displayName ?? item.display_name ?? item.username ?? item.name
    const name = typeof raw === 'string' && raw ? raw : Number.isFinite(id) ? nameById(id) : undefined
    if (name) names.push(name)
  }

  return [...new Set(names)]
}

/** Presence arrives as a map, a list of {userId,status}, or a socket event payload. */
export function normalizePresence(value: unknown): Record<number, string> {
  const result: Record<number, string> = {}
  if (!value || typeof value !== 'object') return result

  if (!Array.isArray(value)) {
    for (const [key, entry] of Object.entries(value as Record<string, unknown>)) {
      const id = Number(key)
      if (Number.isFinite(id) && typeof entry === 'string') result[id] = entry
      else if (entry && typeof entry === 'object') {
        const item = entry as Record<string, unknown>
        const itemId = Number(item.userId ?? item.user_id ?? item.id ?? key)
        const status = item.status ?? item.presence ?? item.state
        if (Number.isFinite(itemId) && typeof status === 'string') result[itemId] = status
      }
    }
    return result
  }

  for (const entry of value) {
    if (!entry || typeof entry !== 'object') continue
    const item = entry as Record<string, unknown>
    const id = Number(item.userId ?? item.user_id ?? item.id)
    const status = item.status ?? item.presence ?? item.state
    if (Number.isFinite(id) && typeof status === 'string') result[id] = status
  }
  return result
}
