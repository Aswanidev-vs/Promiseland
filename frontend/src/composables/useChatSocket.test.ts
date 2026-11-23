import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { scopeKey, useChatSocket } from './useChatSocket'
import type { ChatMessage } from '@/types'

class MockSocket {
  static instances: MockSocket[] = []
  static readonly CONNECTING = 0
  static readonly OPEN = 1
  static readonly CLOSING = 2
  static readonly CLOSED = 3

  readyState = MockSocket.CONNECTING
  url: string
  options: unknown
  sent: string[] = []
  onopen: ((event: Event) => void) | null = null
  onmessage: ((event: MessageEvent) => void) | null = null
  onerror: ((event: Event) => void) | null = null
  onclose: ((event: CloseEvent) => void) | null = null

  constructor(url: string, options?: unknown) {
    this.url = url
    this.options = options
    MockSocket.instances.push(this)
  }

  send(data: string): void {
    this.sent.push(data)
  }

  close(): void {
    this.readyState = MockSocket.CLOSED
  }

  emit(type: string, payload: object, extra: Record<string, unknown> = {}): void {
    this.onmessage?.({ data: JSON.stringify({ type, payload, ...extra }) } as MessageEvent)
  }

  open(): void {
    this.readyState = MockSocket.OPEN
    this.onopen?.(new Event('open'))
  }
}

function makeMessage(id: number, body = 'A message'): ChatMessage {
  return {
    id,
    channel_id: 7,
    conversation_id: null,
    author: { id: 4, username: 'mira', display_name: 'Mira', avatar_url: null },
    parent_id: null,
    body,
    reactions: [],
    created_at: '2026-11-23T10:00:00Z',
    edited_at: null,
    deleted: false,
  }
}

function currentSocket(): MockSocket {
  const socket = MockSocket.instances.at(-1)
  if (!socket) throw new Error('No socket was created')
  return socket
}

beforeEach(() => {
  MockSocket.instances = []
  vi.stubGlobal('WebSocket', MockSocket)
  vi.stubGlobal(
    'fetch',
    vi.fn().mockResolvedValue({
      ok: true,
      status: 204,
      json: async () => undefined,
    }),
  )
})

afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

describe('useChatSocket', () => {
  it('opens /ws with same-origin credentials and replays subscribe frames on open', () => {
    const chat = useChatSocket()
    chat.connect()
    const socket = currentSocket()

    expect(socket.url).toBe(
      `${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}/ws`,
    )
    expect(socket.options).toEqual({ credentials: 'same-origin' })

    expect(scopeKey({ channelId: 7 })).toBe('channel:7')
    expect(scopeKey({ conversationId: 3 })).toBe('conversation:3')
    chat.subscribe({ channelId: 7 })
    expect(socket.sent).toEqual([])

    socket.open()
    expect(chat.connected.value).toBe(true)
    expect(socket.sent).toEqual(['{"type":"subscribe","channel_id":7}'])
    chat.disconnect()
  })

  it('appends message.new events once and records the last event', () => {
    const chat = useChatSocket()
    chat.connect()
    const socket = currentSocket()
    socket.open()
    chat.subscribe({ channelId: 7 })

    socket.emit('message.new', makeMessage(1, 'First'), { channel_id: 7 })
    socket.emit('message.new', makeMessage(1, 'First'), { channel_id: 7 })
    socket.emit('message.new', makeMessage(2, 'Second'), { channel_id: 8 })

    expect(chat.messages.value.map((message) => message.id)).toEqual([1])
    expect(chat.lastEvent.value?.type).toBe('message.new')
    chat.disconnect()
  })

  it('applies reaction.updated and removes message.deleted targets', () => {
    const chat = useChatSocket()
    chat.connect()
    const socket = currentSocket()
    socket.open()
    chat.subscribe({ conversationId: 3 })

    const original = { ...makeMessage(5), channel_id: null, conversation_id: 3 }
    socket.emit('message.new', original, { conversation_id: 3 })
    expect(chat.messages.value[0]?.reactions).toEqual([])

    const liked = { ...original, reactions: [{ reaction: 'heart', count: 1, me: true }] }
    socket.emit('reaction.updated', liked, { conversation_id: 3 })
    expect(chat.messages.value[0]?.reactions).toEqual([
      { reaction: 'heart', count: 1, me: true },
    ])

    socket.emit('message.deleted', { id: 5 }, { conversation_id: 3 })
    expect(chat.messages.value).toEqual([])
    chat.disconnect()
  })

  it('tracks typing and presence events by user id', () => {
    const chat = useChatSocket()
    chat.connect()
    const socket = currentSocket()
    socket.open()
    chat.subscribe({ channelId: 7 })

    socket.emit('typing', { user_id: 9, is_typing: true }, { channel_id: 7, user_ids: [9] })
    expect(chat.typing.value['channel:7']).toEqual([9])

    socket.emit('typing', { user_id: 9, is_typing: false }, { channel_id: 7, user_ids: [9] })
    expect(chat.typing.value['channel:7']).toEqual([])

    socket.emit('presence', { 9: 'online', 10: 'away' })
    expect(chat.presence.value).toEqual({ 9: 'online', 10: 'away' })
    chat.disconnect()
  })

  it('throttles typing sends to one per two seconds and swallows API failures', async () => {
    const chat = useChatSocket()
    const now = vi.spyOn(Date, 'now').mockReturnValue(1_000)

    await chat.sendTyping({ channelId: 7 }, true)
    await chat.sendTyping({ channelId: 7 }, true)
    expect(fetch).toHaveBeenCalledTimes(1)

    now.mockReturnValue(3_000)
    await chat.sendTyping({ channelId: 7 }, true)
    expect(fetch).toHaveBeenCalledTimes(2)
    await chat.sendTyping({ channelId: 7 }, false)
    expect(fetch).toHaveBeenCalledTimes(3)

    const [url, init] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/chat/typing')
    expect(JSON.parse(init.body as string)).toEqual({ channel_id: 7, is_typing: true })

    vi.mocked(fetch).mockRejectedValueOnce(new Error('offline'))
    await expect(chat.sendTyping({ channelId: 7 }, false)).resolves.toBeUndefined()
    expect(chat.error.value).toContain('offline')
  })

  it('reconnects with capped backoff and reports socket failures without throwing', () => {
    vi.useFakeTimers()
    const chat = useChatSocket()
    chat.connect()
    const first = currentSocket()

    first.onclose?.(new Event('close') as CloseEvent)
    expect(chat.connected.value).toBe(false)
    vi.advanceTimersByTime(500)
    expect(MockSocket.instances).toHaveLength(2)

    chat.disconnect()
    expect(() => chat.connect()).not.toThrow()
  })
})
