import { getCurrentInstance, onMounted, onUnmounted, ref } from 'vue'
import { sendTyping as apiSendTyping } from '@/api/chat'
import type { ChatEvent, ChatMessage, ChatReaction, ChatScope, PresenceMap } from '@/types'

const CONNECTING = 0
const OPEN = 1
const INITIAL_BACKOFF_MS = 500
const MAX_BACKOFF_MS = 10_000
const TYPING_THROTTLE_MS = 2_000

/** Stable key for a chat target, used to ignore events from other scopes. */
export function scopeKey(scope: ChatScope): string {
  if (scope.channelId !== undefined) return `channel:${scope.channelId}`
  if (scope.conversationId !== undefined) return `conversation:${scope.conversationId}`
  return ''
}

/**
 * `credentials: 'same-origin'` is the fetch-era spelling of what a same-origin
 * WebSocket already does (the browser attaches pl_session automatically).
 * WebIDL ignores extra constructor arguments, so passing it is harmless and
 * keeps the intent explicit.
 */
type WebSocketWithOptions = new (
  url: string,
  options?: { credentials?: RequestCredentials },
) => WebSocket

/**
 * Shared realtime socket for chat. Safe to call from any chat view: connect()
 * is idempotent, subscriptions are replayed after a reconnect, and connection
 * problems are surfaced through `error` instead of being thrown.
 *
 * `typing` is keyed by scope (`channel:N` / `conversation:N`) with arrays of
 * user ids, which the chat views' normalizeTyping helper consumes directly.
 * `presence` is keyed by user id → status string.
 */
export function useChatSocket() {
  const connected = ref(false)
  const messages = ref<ChatMessage[]>([])
  const typing = ref<Record<string, number[]>>({})
  const presence = ref<PresenceMap>({})
  const lastEvent = ref<ChatEvent | null>(null)
  const error = ref<string | null>(null)

  let socket: WebSocket | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let backoffMs = INITIAL_BACKOFF_MS
  let stopped = false
  let visibilityBound = false
  let activeScopeKey = ''
  const subscriptions = new Set<string>()
  const lastTypingSent = new Map<string, number>()

  function clearReconnectTimer(): void {
    if (reconnectTimer !== null) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
  }

  function closeSocket(): void {
    const current = socket
    socket = null
    if (!current) return
    current.onopen = null
    current.onmessage = null
    current.onerror = null
    current.onclose = null
    try {
      current.close()
    } catch {
      // A half-open mock/proxy must not break teardown.
    }
  }

  function scheduleReconnect(): void {
    if (stopped || reconnectTimer !== null) return
    // Hidden tabs do not hammer the server; visibilitychange resumes them.
    if (typeof document !== 'undefined' && document.hidden) return
    const delay = backoffMs
    backoffMs = Math.min(backoffMs * 2, MAX_BACKOFF_MS)
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null
      connect()
    }, delay)
  }

  function onVisibilityChange(): void {
    if (typeof document === 'undefined' || document.hidden) return
    clearReconnectTimer()
    backoffMs = INITIAL_BACKOFF_MS
    if (!socket || socket.readyState !== OPEN) connect()
  }

  function bindVisibility(): void {
    if (visibilityBound || typeof document === 'undefined') return
    document.addEventListener('visibilitychange', onVisibilityChange)
    visibilityBound = true
  }

  function connect(): void {
    if (typeof WebSocket === 'undefined') {
      error.value = 'Realtime chat is not available in this browser.'
      return
    }
    if (socket && (socket.readyState === OPEN || socket.readyState === CONNECTING)) return

    stopped = false
    clearReconnectTimer()
    bindVisibility()

    const protocol =
      typeof location !== 'undefined' && location.protocol === 'https:' ? 'wss' : 'ws'
    const host = typeof location !== 'undefined' ? location.host : 'localhost'
    const url = `${protocol}://${host}/ws`

    try {
      const SocketCtor = WebSocket as unknown as WebSocketWithOptions
      const next = new SocketCtor(url, { credentials: 'same-origin' })
      socket = next

      next.onopen = () => {
        if (socket !== next) return
        connected.value = true
        error.value = null
        backoffMs = INITIAL_BACKOFF_MS
        subscriptions.forEach((frame) => next.send(frame))
      }
      next.onmessage = (event: MessageEvent) => handleMessage(event)
      next.onerror = () => {
        error.value = 'Chat connection dropped. Trying again…'
      }
      next.onclose = () => {
        if (socket !== next) return
        connected.value = false
        socket = null
        if (!stopped) scheduleReconnect()
      }
    } catch (cause) {
      socket = null
      error.value = cause instanceof Error ? cause.message : 'Could not open the chat connection.'
      scheduleReconnect()
    }
  }

  function disconnect(): void {
    stopped = true
    clearReconnectTimer()
    closeSocket()
    connected.value = false
  }

  function subscribe(scope: ChatScope): void {
    let frame: string | null = null
    if (scope.channelId !== undefined) {
      frame = JSON.stringify({ type: 'subscribe', channel_id: scope.channelId })
    } else if (scope.conversationId !== undefined) {
      frame = JSON.stringify({ type: 'subscribe', conversation_id: scope.conversationId })
    }
    if (!frame) return

    const key = scopeKey(scope)
    if (key !== activeScopeKey) {
      activeScopeKey = key
      typing.value = {}
    }
    subscriptions.add(frame)
    if (socket && socket.readyState === OPEN) socket.send(frame)
  }

  /** REST-backed typing heartbeat; repeated "typing" sends are throttled to 2s. */
  async function sendTyping(scope: ChatScope, isTyping: boolean): Promise<void> {
    const key = scopeKey(scope) || 'global'
    if (isTyping) {
      const now = Date.now()
      const previous = lastTypingSent.get(key)
      if (previous !== undefined && now - previous < TYPING_THROTTLE_MS) return
      lastTypingSent.set(key, now)
    } else {
      // "Stopped typing" must always go through immediately.
      lastTypingSent.delete(key)
    }
    try {
      await apiSendTyping(scope, isTyping)
    } catch (cause) {
      error.value = cause instanceof Error ? cause.message : 'Could not send typing status.'
    }
  }

  function scopeMatches(event: ChatEvent): boolean {
    if (!activeScopeKey) return true
    if (activeScopeKey.startsWith('channel:') && event.channel_id !== null && event.channel_id !== undefined) {
      return activeScopeKey === `channel:${event.channel_id}`
    }
    if (
      activeScopeKey.startsWith('conversation:') &&
      event.conversation_id !== null &&
      event.conversation_id !== undefined
    ) {
      return activeScopeKey === `conversation:${event.conversation_id}`
    }
    return true
  }

  function messageFrom(payload: Record<string, unknown>): ChatMessage | null {
    const candidate = (payload.message ?? payload) as Partial<ChatMessage> | null
    return candidate && typeof candidate.id === 'number' ? (candidate as ChatMessage) : null
  }

  function applyNewMessage(event: ChatEvent): void {
    const message = messageFrom(event.payload ?? {})
    if (!message || !scopeMatches(event)) return
    if (!messages.value.some((item) => item.id === message.id)) {
      messages.value = [...messages.value, message]
    }
  }

  function applyDeletedMessage(event: ChatEvent): void {
    const payload = event.payload ?? {}
    const id =
      typeof payload.id === 'number'
        ? payload.id
        : typeof payload.message_id === 'number'
          ? payload.message_id
          : messageFrom(payload)?.id
    if (id === undefined || !scopeMatches(event)) return
    messages.value = messages.value.filter((item) => item.id !== id)
  }

  function applyReactionUpdate(event: ChatEvent): void {
    const payload = event.payload ?? {}
    const updated = messageFrom(payload)
    if (!scopeMatches(event)) return
    if (Array.isArray(payload.reactions) && typeof payload.message_id === 'number') {
      const reactions = payload.reactions as ChatReaction[]
      messages.value = messages.value.map((item) =>
        item.id === payload.message_id ? { ...item, reactions } : item,
      )
      return
    }
    if (!updated) return
    messages.value = messages.value.map((item) => (item.id === updated.id ? updated : item))
  }

  function applyTyping(event: ChatEvent): void {
    const payload = event.payload ?? {}
    const isTyping =
      typeof payload.is_typing === 'boolean'
        ? payload.is_typing
        : typeof payload.isTyping === 'boolean'
          ? payload.isTyping
          : typeof payload.typing === 'boolean'
            ? payload.typing
            : true
    const single =
      typeof payload.user_id === 'number'
        ? payload.user_id
        : typeof payload.userId === 'number'
          ? payload.userId
          : null
    const targets = single !== null ? [single] : Array.isArray(event.user_ids) ? event.user_ids : []
    if (!targets.length) return

    const key =
      event.channel_id !== null && event.channel_id !== undefined
        ? `channel:${event.channel_id}`
        : event.conversation_id !== null && event.conversation_id !== undefined
          ? `conversation:${event.conversation_id}`
          : activeScopeKey || 'global'
    const ids = new Set(typing.value[key] ?? [])
    targets.forEach((id) => {
      if (isTyping) ids.add(id)
      else ids.delete(id)
    })
    typing.value = { ...typing.value, [key]: [...ids] }
  }

  function applyPresence(event: ChatEvent): void {
    const payload = event.payload ?? {}
    const statusOf = (value: unknown): string | null => (typeof value === 'string' ? value : null)
    const next = { ...presence.value }

    if (typeof payload.user_id === 'number') {
      const status = statusOf(payload.status) ?? statusOf(payload.presence)
      if (status) next[payload.user_id] = status
      else delete next[payload.user_id]
    } else {
      Object.entries(payload).forEach(([key, value]) => {
        const id = Number(key)
        const status = statusOf(value)
        if (Number.isFinite(id) && status) next[id] = status
      })
      if (Array.isArray(event.user_ids)) {
        const status = statusOf(payload.status) ?? 'offline'
        event.user_ids.forEach((id) => {
          next[id] = status
        })
      }
    }
    presence.value = next
  }

  function handleMessage(event: MessageEvent): void {
    let incoming: ChatEvent
    try {
      const raw = typeof event.data === 'string' ? event.data : String(event.data)
      incoming = JSON.parse(raw) as ChatEvent
    } catch {
      return
    }
    if (!incoming || typeof incoming.type !== 'string') return
    lastEvent.value = incoming
    try {
      switch (incoming.type) {
        case 'message.new':
          applyNewMessage(incoming)
          break
        case 'message.deleted':
          applyDeletedMessage(incoming)
          break
        case 'reaction.updated':
          applyReactionUpdate(incoming)
          break
        case 'typing':
          applyTyping(incoming)
          break
        case 'presence':
          applyPresence(incoming)
          break
      }
    } catch {
      // One malformed payload must never tear down the socket.
    }
  }

  if (getCurrentInstance()) {
    onMounted(connect)
    onUnmounted(() => {
      disconnect()
      if (visibilityBound && typeof document !== 'undefined') {
        document.removeEventListener('visibilitychange', onVisibilityChange)
        visibilityBound = false
      }
    })
  }

  return {
    connected,
    messages,
    typing,
    presence,
    lastEvent,
    error,
    connect,
    disconnect,
    subscribe,
    sendTyping,
  }
}
