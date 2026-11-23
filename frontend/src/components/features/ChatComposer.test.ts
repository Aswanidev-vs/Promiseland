import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import ChatComposer from './ChatComposer.vue'

function mountComposer() {
  return mount(ChatComposer)
}

describe('ChatComposer', () => {
  it('sends on Enter, clears the draft, and stops typing', async () => {
    const wrapper = mountComposer()
    const textarea = wrapper.get('textarea')

    await textarea.setValue('Hello there')
    expect(wrapper.emitted('typing')).toEqual([[true]])

    await textarea.trigger('keydown', { key: 'Enter' })

    expect(wrapper.emitted('send')).toEqual([['Hello there']])
    expect(wrapper.emitted('typing')).toContainEqual([false])
    expect((textarea.element as HTMLTextAreaElement).value).toBe('')
  })

  it('inserts a newline on Shift+Enter instead of sending', async () => {
    const wrapper = mountComposer()
    const textarea = wrapper.get('textarea')

    await textarea.setValue('first line')
    await textarea.trigger('keydown', { key: 'Enter', shiftKey: true })

    expect(wrapper.emitted('send')).toBeUndefined()
    expect((textarea.element as HTMLTextAreaElement).value).toContain('\n')
    expect(wrapper.emitted('typing')).toEqual([[true]])
  })

  it('emits a single typing start and stops after the idle timeout', async () => {
    vi.useFakeTimers()
    try {
      const wrapper = mountComposer()
      const textarea = wrapper.get('textarea')

      await textarea.setValue('d')
      await textarea.setValue('dr')
      expect(wrapper.emitted('typing')).toEqual([[true]])

      vi.advanceTimersByTime(1600)
      expect(wrapper.emitted('typing')).toContainEqual([false])
    } finally {
      vi.useRealTimers()
    }
  })

  it('does not send an empty or whitespace-only message', async () => {
    const wrapper = mountComposer()
    const textarea = wrapper.get('textarea')

    await textarea.setValue('   ')
    await textarea.trigger('keydown', { key: 'Enter' })

    expect(wrapper.emitted('send')).toBeUndefined()
  })
})
