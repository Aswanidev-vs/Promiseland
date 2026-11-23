import { ref } from 'vue'
import { defineStore } from 'pinia'
import { errorMessage } from '@/api/client'
import * as api from '@/api/chat'
import type {
  ChatChannel,
  ChatScope,
  ChatServer,
  Conversation,
  CreateChannelInput,
  CreateConversationInput,
  CreateServerInput,
} from '@/types'

/**
 * Failure-tolerant list cache for the chat directory. Chat views own their
 * message lists (and the socket buffer); this store only keeps the navigation
 * data and the scope the user is currently looking at.
 */
export const useChatStore = defineStore('chat', () => {
  const servers = ref<ChatServer[]>([])
  const conversations = ref<Conversation[]>([])
  const channels = ref<Record<number, ChatChannel[]>>({})
  const activeScope = ref<ChatScope>({})
  const loading = ref(false)
  const error = ref<string | null>(null)

  function rememberServer(server: ChatServer): ChatServer {
    const index = servers.value.findIndex((item) => item.id === server.id)
    if (index !== -1) servers.value.splice(index, 1, server)
    else servers.value.push(server)
    return server
  }

  function rememberConversation(conversation: Conversation): Conversation {
    const index = conversations.value.findIndex((item) => item.id === conversation.id)
    if (index !== -1) conversations.value.splice(index, 1, conversation)
    else conversations.value.push(conversation)
    return conversation
  }

  function rememberChannel(serverId: number, channel: ChatChannel): ChatChannel {
    const list = channels.value[serverId] ?? []
    const index = list.findIndex((item) => item.id === channel.id)
    if (index !== -1) list.splice(index, 1, channel)
    else list.push(channel)
    channels.value = { ...channels.value, [serverId]: list }
    return channel
  }

  async function loadServers(): Promise<ChatServer[]> {
    loading.value = true
    error.value = null
    try {
      const result = await api.listServers()
      servers.value = result
      return result
    } catch (cause) {
      error.value = errorMessage(cause)
      return servers.value
    } finally {
      loading.value = false
    }
  }

  async function loadConversations(): Promise<Conversation[]> {
    loading.value = true
    error.value = null
    try {
      const result = await api.listConversations()
      conversations.value = result
      return result
    } catch (cause) {
      error.value = errorMessage(cause)
      return conversations.value
    } finally {
      loading.value = false
    }
  }

  async function loadChannels(serverId: number): Promise<ChatChannel[]> {
    loading.value = true
    error.value = null
    try {
      const result = await api.listChannels(serverId)
      channels.value = { ...channels.value, [serverId]: result }
      return result
    } catch (cause) {
      error.value = errorMessage(cause)
      return channels.value[serverId] ?? []
    } finally {
      loading.value = false
    }
  }

  async function addServer(input: CreateServerInput): Promise<ChatServer | null> {
    try {
      return rememberServer(await api.createServer(input))
    } catch (cause) {
      error.value = errorMessage(cause)
      return null
    }
  }

  async function join(id: number): Promise<ChatServer | null> {
    try {
      return rememberServer(await api.joinServer(id))
    } catch (cause) {
      error.value = errorMessage(cause)
      return null
    }
  }

  async function addChannel(serverId: number, input: CreateChannelInput): Promise<ChatChannel | null> {
    try {
      return rememberChannel(serverId, await api.createChannel(serverId, input))
    } catch (cause) {
      error.value = errorMessage(cause)
      return null
    }
  }

  async function addConversation(input: CreateConversationInput): Promise<Conversation | null> {
    try {
      return rememberConversation(await api.createConversation(input))
    } catch (cause) {
      error.value = errorMessage(cause)
      return null
    }
  }

  function setActiveScope(scope: ChatScope): void {
    activeScope.value = { ...scope }
  }

  function clear(): void {
    servers.value = []
    conversations.value = []
    channels.value = {}
    activeScope.value = {}
    error.value = null
  }

  return {
    servers,
    conversations,
    channels,
    activeScope,
    loading,
    error,
    loadServers,
    loadConversations,
    loadChannels,
    addServer,
    join,
    addChannel,
    addConversation,
    setActiveScope,
    clear,
  }
})
