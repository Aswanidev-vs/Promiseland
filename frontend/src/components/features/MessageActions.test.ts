import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import MessageActions from './MessageActions.vue'
import type { ChatMessage } from '@/types'

const message: ChatMessage = {
  id: 12,
  channel_id: 7,
  conversation_id: null,
  author: { id: 1, username: 'aria', display_name: 'Aria', avatar_url: null },
  parent_id: null,
  body: 'A message',
  reactions: [{ reaction: 'like', count: 2, me: false }],
  created_at: new Date().toISOString(),
  edited_at: '',
  deleted: false,
}

function mountActions(isOwned: boolean) {
  return mount(MessageActions, { props: { message, isOwned } })
}

describe('MessageActions', () => {
  it('shows delete only for messages the current user owns', async () => {
    const other = mountActions(false)
    expect(other.find('[aria-label="Delete message"]').exists()).toBe(false)

    const own = mountActions(true)
    const deleteButton = own.get('[aria-label="Delete message"]')
    expect(own.findAll('[aria-label="Delete message"]')).toHaveLength(1)

    await deleteButton.trigger('click')
    expect(own.emitted('delete')).toHaveLength(1)
  })

  it('emits the reaction code using icon glyphs, never emoji', async () => {
    const wrapper = mountActions(true)
    const labels = wrapper.findAll('.actions__reaction').map((button) => button.attributes('aria-label'))

    expect(labels).toEqual(['Like reaction', 'Heart reaction', 'Star reaction', 'Laugh reaction'])
    expect(wrapper.text()).not.toMatch(/[\p{Emoji_Presentation}]/u)

    await wrapper.get('[aria-label="Like reaction"]').trigger('click')
    expect(wrapper.emitted('react')).toEqual([['like']])
  })

  it('always offers reply and report', async () => {
    const wrapper = mountActions(false)

    await wrapper.get('[aria-label="Reply to message"]').trigger('click')
    await wrapper.get('[aria-label="Report message"]').trigger('click')

    expect(wrapper.emitted('reply')).toHaveLength(1)
    expect(wrapper.emitted('report')).toHaveLength(1)
  })
})
