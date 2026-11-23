import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import ChatMessageList from './ChatMessageList.vue'
import type { ChatMessage, UserSummary } from '@/types'

const aria: UserSummary = { id: 1, username: 'aria', display_name: 'Aria', avatar_url: null }
const ben: UserSummary = { id: 2, username: 'ben', display_name: 'Ben', avatar_url: null }

function message(
  id: number,
  author: UserSummary,
  body: string,
  extra: Partial<ChatMessage> = {},
): ChatMessage {
  return {
    id,
    channel_id: 7,
    conversation_id: null,
    author,
    parent_id: null,
    body,
    reactions: [],
    created_at: new Date().toISOString(),
    edited_at: '',
    deleted: false,
    ...extra,
  }
}

function mountList(messages: ChatMessage[]) {
  return mount(ChatMessageList, { props: { messages, currentUserId: 1 } })
}

describe('ChatMessageList', () => {
  it('groups consecutive messages by day and author, indenting replies', () => {
    const wrapper = mountList([
      message(1, aria, 'first'),
      message(2, aria, 'second'),
      message(3, ben, 'third'),
      message(4, aria, 'a reply', { parent_id: 1 }),
    ])

    expect(wrapper.findAll('.messages__day')).toHaveLength(1)
    expect(wrapper.findAll('.messages__group')).toHaveLength(3)
    expect(wrapper.findAll('.message')).toHaveLength(4)
    expect(wrapper.findAll('.message--reply')).toHaveLength(1)
    expect(wrapper.findAll('.message')[1]!.text()).toContain('second')
    expect(wrapper.findAll('.message')[3]!.text()).toContain('a reply')
  })

  it('renders a deleted state without message actions', () => {
    const wrapper = mountList([message(1, aria, 'removed text', { deleted: true })])

    expect(wrapper.findAll('.message__deleted')).toHaveLength(1)
    expect(wrapper.text()).toContain('This message was removed.')
    expect(wrapper.text()).not.toContain('removed text')
    expect(wrapper.find('.actions').exists()).toBe(false)
  })

  it('offers report and owner-only delete actions on live messages', () => {
    const wrapper = mountList([
      message(1, aria, 'mine'),
      message(2, ben, 'theirs'),
    ])

    const articles = wrapper.findAll('.message')
    expect(articles[0]!.find('[aria-label="Delete message"]').exists()).toBe(true)
    expect(articles[1]!.find('[aria-label="Delete message"]').exists()).toBe(false)
    expect(wrapper.findAll('[aria-label="Report message"]')).toHaveLength(2)
  })
})
