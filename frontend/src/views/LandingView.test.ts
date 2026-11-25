import { createPinia } from 'pinia'
import { mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { describe, expect, it } from 'vitest'
import LandingView from './LandingView.vue'

const Blank = { template: '<div />' }

async function mountLanding() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', name: 'landing', component: LandingView },
      { path: '/register', name: 'register', component: Blank },
      { path: '/login', name: 'login', component: Blank },
      { path: '/home', name: 'home', component: Blank },
      { path: '/events', name: 'events', component: Blank },
      { path: '/creators', name: 'creators', component: Blank },
      { path: '/terms', name: 'terms', component: Blank },
      { path: '/privacy', name: 'privacy', component: Blank },
    ],
  })
  router.push('/')
  await router.isReady()

  return mount(LandingView, {
    global: { plugins: [createPinia(), router] },
  })
}

describe('LandingView', () => {
  it('states the promise and points guests at account creation', async () => {
    const wrapper = await mountLanding()
    const text = wrapper.text()

    expect(text).toContain('Your fandom.')
    expect(text).toContain('Your people.')
    expect(text).toContain('Your place.')
    expect(text).toContain('One interest list, four ways to use it')
    expect(text).toContain('Discuss')
    expect(text).toContain('Later release')
    expect(text).toContain('Sample event, not live data.')

    const primary = wrapper.get('.btn--primary')
    expect(primary.attributes('href')).toBe('/register')
    expect(primary.text()).toBe('Create account')
  })

  it('plots interests on the map and clears them again', async () => {
    const wrapper = await mountLanding()

    const chips = wrapper.findAll('.map__chip')
    expect(chips).toHaveLength(6)
    expect(wrapper.text()).toContain('Choose an interest to plot it on the map.')

    await chips[0]!.trigger('click')
    expect(wrapper.findAll('.map__chip')[0]!.attributes('aria-pressed')).toBe('true')
    expect(wrapper.text()).toContain('Frieren is plotted. Its community sits under the star.')
    expect(wrapper.find('.map__link').exists()).toBe(false)

    await wrapper.findAll('.map__chip')[1]!.trigger('click')
    expect(wrapper.find('.map__link').exists()).toBe(true)
    expect(wrapper.text()).toContain('Frieren and Dungeon Meshi are plotted.')

    await wrapper.findAll('.map__chip')[0]!.trigger('click')
    expect(wrapper.findAll('.map__chip')[0]!.attributes('aria-pressed')).toBe('false')
    expect(wrapper.text()).toContain('Dungeon Meshi is plotted.')
  })
})
