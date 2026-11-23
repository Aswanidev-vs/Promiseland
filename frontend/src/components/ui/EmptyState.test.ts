import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import EmptyState from './EmptyState.vue'

describe('EmptyState', () => {
  it('renders the wayfinder, copy, and a working action', async () => {
    const wrapper = mount(EmptyState, {
      props: {
        title: 'No communities yet',
        description: 'Join one to see discussions here.',
        actionLabel: 'Discover communities',
      },
    })

    expect(wrapper.find('svg').exists()).toBe(true)
    expect(wrapper.text()).toContain('No communities yet')
    expect(wrapper.text()).toContain('Join one to see discussions here.')
    expect(wrapper.find('button').text()).toBe('Discover communities')

    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('action')).toHaveLength(1)
  })
})
