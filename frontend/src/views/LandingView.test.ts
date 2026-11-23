import { createPinia } from 'pinia'
import { mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { describe, expect, it } from 'vitest'
import LandingView from './LandingView.vue'

describe('LandingView', () => {
  it('renders the product promise, wayfinder, feature blocks and primary action', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/', name: 'landing', component: LandingView }],
    })
    router.push('/')
    await router.isReady()

    const wrapper = mount(LandingView, {
      global: { plugins: [createPinia(), router] },
    })

    const text = wrapper.text()
    expect(text).toContain('Your fandom.')
    expect(text).toContain('Your people.')
    expect(text).toContain('Your place.')
    expect(text).toContain('Find your people')
    expect(text).toContain('Share what you make')
    expect(text).toContain('Meet beyond the thread')
    expect(wrapper.findAll('svg').length).toBeGreaterThan(2)
    expect(wrapper.get('a').attributes('href')).toBe('/')
    expect(wrapper.text()).toContain('Enter PromisedLand')
  })
})
