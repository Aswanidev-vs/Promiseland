import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  createConversation,
  deleteMessage,
  getPresence,
  listMessages,
  reactToMessage,
  reportMessage,
  sendMessage,
  sendTyping,
} from './chat'

function jsonResponse(status: number, body: unknown): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    statusText: 'OK',
    json: async () => body,
  } as unknown as Response
}

function mockJson(body: unknown, status = 200): void {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse(status, body)))
}

function lastCall(): [string, RequestInit] {
  return vi.mocked(fetch).mock.calls.at(-1) as [string, RequestInit]
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('chat API wiring', () => {
  it('maps the scope to channel/conversation message paths with limit and cursor', async () => {
    mockJson({ items: [], next_cursor: '' })

    await listMessages({ channelId: 7, limit: 20, cursor: 'msg-1' })
    expect(lastCall()[0]).toBe('/api/v1/chat/channels/7/messages?limit=20&cursor=msg-1')

    await listMessages({ conversationId: 3 })
    expect(lastCall()[0]).toBe('/api/v1/chat/conversations/3/messages')
  })

  it('posts a message body with an optional snake_case parent_id', async () => {
    mockJson({ id: 11 })

    await sendMessage({ channelId: 7 }, 'Hello harbour')
    let [url, init] = lastCall()
    expect(url).toBe('/api/v1/chat/channels/7/messages')
    expect(init.method).toBe('POST')
    expect(JSON.parse(init.body as string)).toEqual({ body: 'Hello harbour' })

    await sendMessage({ conversationId: 3 }, 'A reply', 9)
    ;[url, init] = lastCall()
    expect(url).toBe('/api/v1/chat/conversations/3/messages')
    expect(JSON.parse(init.body as string)).toEqual({ body: 'A reply', parent_id: 9 })
  })

  it('translates the typing scope to channel_id/conversation_id and is_typing', async () => {
    mockJson(undefined, 204)

    await sendTyping({ channelId: 7 }, true)
    let [url, init] = lastCall()
    expect(url).toBe('/api/v1/chat/typing')
    expect(init.method).toBe('POST')
    expect(JSON.parse(init.body as string)).toEqual({ channel_id: 7, is_typing: true })

    await sendTyping({ conversationId: 3 }, false)
    ;[url, init] = lastCall()
    expect(url).toBe('/api/v1/chat/typing')
    expect(JSON.parse(init.body as string)).toEqual({ conversation_id: 3, is_typing: false })
  })

  it('requests presence with a literal comma-separated id list', async () => {
    mockJson({ 1: 'online', 2: 'away' })

    await expect(getPresence([1, 2])).resolves.toEqual({ 1: 'online', 2: 'away' })
    expect(lastCall()[0]).toBe('/api/v1/chat/presence?ids=1,2')
  })

  it('maps createConversation memberIds to member_ids and wires reaction/report/delete', async () => {
    mockJson({ id: 3 })
    await createConversation({ memberIds: [4, 5], name: 'Art crew' })
    let [url, init] = lastCall()
    expect(url).toBe('/api/v1/chat/conversations')
    expect(JSON.parse(init.body as string)).toEqual({ member_ids: [4, 5], name: 'Art crew' })

    mockJson({ id: 11 })
    await reactToMessage(11, 'heart')
    expect(lastCall()[0]).toBe('/api/v1/chat/messages/11/reactions')
    expect(JSON.parse(lastCall()[1].body as string)).toEqual({ reaction: 'heart' })

    mockJson(undefined, 204)
    await reportMessage(11, 'spam')
    ;[url, init] = lastCall()
    expect(url).toBe('/api/v1/chat/messages/11/report')
    expect(JSON.parse(init.body as string)).toEqual({ reason: 'spam' })

    await deleteMessage(11)
    ;[url, init] = lastCall()
    expect(url).toBe('/api/v1/chat/messages/11')
    expect(init.method).toBe('DELETE')
  })
})
