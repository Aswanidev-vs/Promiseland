import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import FeedTabs from './FeedTabs.vue'

describe('FeedTabs', () => {
  it('exposes accessible tabs and emits only the selected tab', async () => {
    const wrapper = mount(FeedTabs, {
      props: {
        modelValue: 'for-you',
        tabs: [
          { value: 'for-you', label: 'For you' },
          { value: 'following', label: 'Following' },
        ],
      },
    })

    const tabs = wrapper.findAll('[role="tab"]')
    expect(tabs).toHaveLength(2)
    expect(tabs[0]!.attributes('aria-selected')).toBe('true')

    await tabs[1]!.trigger('click')
    expect(wrapper.emitted('update:modelValue')).toEqual([['following']])

    await wrapper.setProps({ modelValue: 'following' })
    await tabs[1]!.trigger('click')
    expect(wrapper.emitted('update:modelValue')).toHaveLength(1)
  })
})
