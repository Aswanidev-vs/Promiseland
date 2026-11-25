import { createPinia } from 'pinia'
import { mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { describe, expect, it } from 'vitest'
import ErrorView from './ErrorView.vue'

const Blank = { template: '<div />' }

async function mountError(code?: string | number) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', name: 'landing', component: Blank },
      { path: '/login', name: 'login', component: Blank },
      { path: '/events', name: 'events', component: Blank },
      { path: '/creators', name: 'creators', component: Blank },
      { path: '/terms', name: 'terms', component: Blank },
      { path: '/privacy', name: 'privacy', component: Blank },
    ],
  })
  await router.push('/')
  await router.isReady()

  return mount(ErrorView, {
    props: code === undefined ? {} : { code },
    global: { plugins: [createPinia(), router] },
  })
}

describe('ErrorView', () => {
  it('explains each common failure in its own words', async () => {
    const cases: Array<[number, string]> = [
      [403, 'This page is not yours to open'],
      [404, 'This bearing leads nowhere'],
      [429, 'Slow down a moment'],
      [500, 'Something gave way on our side'],
      [503, 'The harbour is closed for a moment'],
    ]

    for (const [code, heading] of cases) {
      const wrapper = await mountError(code)
      expect(wrapper.get('h1').text()).toBe(heading)
      expect(wrapper.get('.error-page__code').text()).toBe(String(code))
    }
  })

  it('falls back to a 404 for a code it does not know', async () => {
    const wrapper = await mountError('999')

    expect(wrapper.get('h1').text()).toBe('This bearing leads nowhere')
    expect(wrapper.get('.error-page__code').text()).toBe('404')
  })

  it('offers retry for server-side failures and links for the rest', async () => {
    const retryable = await mountError(503)
    expect(retryable.findAll('button').some((button) => button.text() === 'Try again')).toBe(true)

    const forbidden = await mountError(403)
    expect(forbidden.text()).toContain('Sign in')
    expect(forbidden.findAll('button').some((button) => button.text() === 'Try again')).toBe(false)
  })

  it('offers the public pages as a way onward', async () => {
    const wrapper = await mountError(404)
    const hrefs = wrapper.findAll('.error-page__routes a').map((link) => link.attributes('href'))

    expect(hrefs).toEqual(['/events', '/creators'])
  })
})
