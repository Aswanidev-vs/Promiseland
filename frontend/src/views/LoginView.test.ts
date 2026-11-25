import { createPinia, setActivePinia } from 'pinia'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { login } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'
import LoginView from './LoginView.vue'

vi.mock('@/api/auth', () => ({
  login: vi.fn(),
  register: vi.fn(),
  logout: vi.fn(),
  getMe: vi.fn(),
  verifyEmail: vi.fn(),
  requestPasswordReset: vi.fn(),
  confirmPasswordReset: vi.fn(),
}))

async function setup(redirect?: string) {
  const pinia = createPinia()
  setActivePinia(pinia)
  const auth = useAuthStore(pinia)
  const fetchSpy = vi.spyOn(auth, 'fetchMe').mockResolvedValue(null)
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/login', name: 'login', component: LoginView },
      { path: '/register', name: 'register', component: { template: '<div />' } },
      { path: '/reset-password', name: 'reset-password', component: { template: '<div />' } },
      { path: '/home', name: 'home', component: { template: '<div />' } },
      { path: '/events', name: 'events', component: { template: '<div />' } },
    ],
  })
  await router.push(redirect ? `/login?redirect=${encodeURIComponent(redirect)}` : '/login')
  await router.isReady()
  const wrapper = mount(LoginView, { global: { plugins: [pinia, router] } })
  return { wrapper, router, fetchSpy }
}

function fill(wrapper: ReturnType<typeof mount>) {
  const fields = wrapper.findAll('input')
  return Promise.all([
    fields[0]!.setValue('mira@example.com'),
    fields[1]!.setValue('correct horse battery staple'),
  ])
}

describe('LoginView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('validates both fields before calling the API', async () => {
    const { wrapper } = await setup()
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(wrapper.text()).toContain('Enter your email address.')
    expect(wrapper.text()).toContain('Enter your password.')
    expect(login).not.toHaveBeenCalled()
  })

  it('shows the same generic message whatever the sign-in error', async () => {
    vi.mocked(login).mockRejectedValueOnce(new Error('account not found'))
    const { wrapper } = await setup()
    await fill(wrapper)
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(wrapper.text()).toContain(
      "We couldn't sign you in with that email and password. Check them and try again.",
    )
    expect(wrapper.text()).not.toContain('account not found')
  })

  it('reveals the password only when asked', async () => {
    const { wrapper } = await setup()

    expect(wrapper.findAll('input')[1]!.attributes('type')).toBe('password')
    await wrapper.get('.field__reveal').trigger('click')
    expect(wrapper.findAll('input')[1]!.attributes('type')).toBe('text')
    expect(wrapper.get('.field__reveal').attributes('aria-label')).toBe('Hide password')
  })

  it('refreshes auth state and redirects to a safe local target', async () => {
    vi.mocked(login).mockResolvedValueOnce({ id: 2, username: 'mira' } as never)
    const { wrapper, router, fetchSpy } = await setup('/home')
    await fill(wrapper)
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(login).toHaveBeenCalledWith({
      email: 'mira@example.com',
      password: 'correct horse battery staple',
    })
    expect(fetchSpy).toHaveBeenCalled()
    expect(router.currentRoute.value.path).toBe('/home')
  })

  it('ignores an off-site redirect target', async () => {
    vi.mocked(login).mockResolvedValueOnce({ id: 2, username: 'mira' } as never)
    const { wrapper, router } = await setup('https://example.com/phishing')
    await fill(wrapper)
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(router.currentRoute.value.path).toBe('/home')
  })
})
