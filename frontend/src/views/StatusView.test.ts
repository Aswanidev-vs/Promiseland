import { createPinia } from 'pinia'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import StatusView from './StatusView.vue'

const Blank = { template: '<div />' }

function jsonResponse(status: number, body: unknown): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
  } as unknown as Response
}

async function mountStatus() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/status', name: 'status', component: StatusView },
      { path: '/', name: 'landing', component: Blank },
      { path: '/events', name: 'events', component: Blank },
      { path: '/creators', name: 'creators', component: Blank },
      { path: '/login', name: 'login', component: Blank },
      { path: '/register', name: 'register', component: Blank },
      { path: '/terms', name: 'terms', component: Blank },
      { path: '/privacy', name: 'privacy', component: Blank },
      { path: '/home', name: 'home', component: Blank },
      { path: '/settings', name: 'settings', component: Blank },
    ],
  })
  await router.push('/status')
  await router.isReady()

  const wrapper = mount(StatusView, { global: { plugins: [createPinia(), router] } })
  await flushPromises()
  return wrapper
}

describe('StatusView', () => {
  let fetchMock: ReturnType<typeof vi.fn>

  beforeEach(() => {
    fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('reports both endpoints as operational', async () => {
    fetchMock.mockImplementation(async (input: RequestInfo | URL) => {
      const path = String(input)
      if (path === '/healthz') return jsonResponse(200, { status: 'ok' })
      if (path === '/readyz') return jsonResponse(200, { status: 'ready' })
      throw new Error(`unexpected request: ${path}`)
    })

    const wrapper = await mountStatus()

    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(wrapper.get('h1').text()).toContain('All systems operational')
    expect(wrapper.findAll('.check')).toHaveLength(2)
    expect(wrapper.text()).toContain('1 of 1 check was fully healthy')
    wrapper.unmount()
  })

  it('reports a partial outage with the API message', async () => {
    fetchMock.mockImplementation(async (input: RequestInfo | URL) => {
      const path = String(input)
      if (path === '/healthz') return jsonResponse(200, { status: 'ok' })
      return jsonResponse(503, {
        error: { code: 'database_unavailable', message: 'Database is not ready.' },
      })
    })

    const wrapper = await mountStatus()

    expect(wrapper.get('h1').text()).toContain('Partially available')
    expect(wrapper.text()).toContain('Database is not ready.')
    expect(wrapper.text()).toContain('HTTP 503')
    wrapper.unmount()
  })

  it('reports an unreachable service when the requests fail', async () => {
    fetchMock.mockRejectedValue(new TypeError('Failed to fetch'))

    const wrapper = await mountStatus()

    expect(wrapper.get('h1').text()).toContain('PromisedLand is unreachable from this browser')
    expect(wrapper.text()).toContain('The service could not be reached.')
    expect(wrapper.text()).toContain('0 of 1 check was fully healthy')
    wrapper.unmount()
  })

  it('does not mistake a non-JSON answer for a healthy service', async () => {
    fetchMock.mockImplementation(
      async () =>
        ({
          ok: true,
          status: 200,
          json: async () => {
            throw new SyntaxError('Unexpected token <')
          },
        }) as unknown as Response,
    )

    const wrapper = await mountStatus()

    expect(wrapper.get('h1').text()).not.toContain('operational')
    expect(wrapper.text()).toContain('The endpoint answered with something other than JSON.')
    wrapper.unmount()
  })

  it('checks again on request and records each round', async () => {
    fetchMock.mockImplementation(async () => jsonResponse(200, { status: 'ok' }))

    const wrapper = await mountStatus()
    expect(fetchMock).toHaveBeenCalledTimes(2)

    await wrapper.findAll('button').find((button) => button.text() === 'Check again')!.trigger('click')
    await flushPromises()

    expect(fetchMock).toHaveBeenCalledTimes(4)
    expect(wrapper.text()).toContain('2 of 2 checks were fully healthy')
    wrapper.unmount()
  })
})
