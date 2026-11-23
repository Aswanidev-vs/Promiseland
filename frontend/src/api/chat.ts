import { buildQuery, request } from './client'
import type {
  ChatChannel,
  ChatMessage,
  ChatScope,
  ChatServer,
  Conversation,
  CreateChannelInput,
  CreateConversationInput,
  CreateServerInput,
  ListMessagesParams,
  Page,
  PresenceMap,
} from '@/types'

/**
 * Message collections live under channel or conversation subresources.
 * The scope object uses camelCase on the client; the wire path carries the id.
 */
function messagesPath(scope: ChatScope): string {
  if (scope.channelId !== undefined) return `/chat/channels/${scope.channelId}/messages`
  if (scope.conversationId !== undefined) {
    return `/chat/conversations/${scope.conversationId}/messages`
  }
  throw new Error('A channelId or conversationId is required to address messages')
}

export async function listServers(): Promise<ChatServer[]> {
  return request<ChatServer[]>('/chat/servers')
}

export async function createServer(input: CreateServerInput): Promise<ChatServer> {
  return request<ChatServer>('/chat/servers', {
    method: 'POST',
    body: JSON.stringify({ name: input.name, description: input.description }),
  })
}

export async function joinServer(id: number): Promise<ChatServer> {
  return request<ChatServer>(`/chat/servers/${id}/join`, { method: 'POST' })
}

export async function listChannels(serverId: number): Promise<ChatChannel[]> {
  return request<ChatChannel[]>(`/chat/servers/${serverId}/channels`)
}

export async function createChannel(
  serverId: number,
  input: CreateChannelInput,
): Promise<ChatChannel> {
  return request<ChatChannel>(`/chat/servers/${serverId}/channels`, {
    method: 'POST',
    body: JSON.stringify({ name: input.name, description: input.description }),
  })
}

export async function listConversations(): Promise<Conversation[]> {
  return request<Conversation[]>('/chat/conversations')
}

export async function createConversation(input: CreateConversationInput): Promise<Conversation> {
  return request<Conversation>('/chat/conversations', {
    method: 'POST',
    body: JSON.stringify({ member_ids: input.memberIds, name: input.name }),
  })
}

export async function listMessages(p: ListMessagesParams = {}): Promise<Page<ChatMessage>> {
  return request<Page<ChatMessage>>(
    `${messagesPath(p)}${buildQuery({ limit: p.limit, cursor: p.cursor })}`,
  )
}

export async function sendMessage(
  scope: ChatScope,
  body: string,
  parentId?: number,
): Promise<ChatMessage> {
  return request<ChatMessage>(messagesPath(scope), {
    method: 'POST',
    body: JSON.stringify({ body, parent_id: parentId }),
  })
}

export async function reactToMessage(id: number, reaction: string): Promise<ChatMessage> {
  return request<ChatMessage>(`/chat/messages/${id}/reactions`, {
    method: 'POST',
    body: JSON.stringify({ reaction }),
  })
}

export async function deleteMessage(id: number): Promise<void> {
  await request<void>(`/chat/messages/${id}`, { method: 'DELETE' })
}

export async function reportMessage(id: number, reason: string): Promise<void> {
  await request<void>(`/chat/messages/${id}/report`, {
    method: 'POST',
    body: JSON.stringify({ reason }),
  })
}

/** The scope's ids are translated to snake_case wire keys before posting. */
export async function sendTyping(scope: ChatScope, isTyping: boolean): Promise<void> {
  await request<void>('/chat/typing', {
    method: 'POST',
    body: JSON.stringify({
      channel_id: scope.channelId,
      conversation_id: scope.conversationId,
      is_typing: isTyping,
    }),
  })
}

/** GET /chat/presence?ids=1,2 — a map of user id → status string. */
export async function getPresence(ids: number[]): Promise<PresenceMap> {
  const query = ids.length ? `?ids=${ids.join(',')}` : ''
  return request<PresenceMap>(`/chat/presence${query}`)
}
