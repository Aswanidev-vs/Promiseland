import { createPinia, setActivePinia } from 'pinia'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { register } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'
import RegisterView from './RegisterView.vue'

vi.mock('@/api/auth', () => ({
  login: vi.fn(),
  register: vi.fn(),
  logout: vi.fn(),
  getMe: vi.fn(),
  verifyEmail: vi.fn(),
  requestPasswordReset: vi.fn(),
  confirmPasswordReset: vi.fn(),
}))

const Blank = { template: '<div />' }

async function setup() {
  const pinia = createPinia()
  setActivePinia(pinia)
  vi.spyOn(useAuthStore(pinia), 'fetchMe').mockResolvedValue(null)

  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/register', name: 'register', component: RegisterView },
      { path: '/login', name: 'login', component: Blank },
      { path: '/home', name: 'home', component: Blank },
      { path: '/events', name: 'events', component: Blank },
      { path: '/terms', name: 'terms', component: Blank },
      { path: '/privacy', name: 'privacy', component: Blank },
    ],
  })
  await router.push('/register')
  await router.isReady()

  const wrapper = mount(RegisterView, { global: { plugins: [pinia, router] } })
  return { wrapper, router }
}

async function fill(wrapper: ReturnType<typeof mount>) {
  const fields = wrapper.findAll('input[type="text"], input[type="email"], input[type="password"]')
  await fields[0]!.setValue('mira@example.com')
  await fields[1]!.setValue('mira')
  await fields[2]!.setValue('Mira')
  await fields[3]!.setValue('correct horse battery')
  await fields[4]!.setValue('correct horse battery')
}

describe('RegisterView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(register).mockResolvedValue({ id: 2, username: 'mira' } as never)
  })

  it('will not create an account until both agreements are checked', async () => {
    const { wrapper } = await setup()
    await fill(wrapper)

    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(register).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain(
      'Agree to the terms of use and the privacy policy to create an account.',
    )

    const boxes = wrapper.findAll('input[type="checkbox"]')
    expect(boxes).toHaveLength(2)

    await boxes[0]!.setValue(true)
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(register).not.toHaveBeenCalled()

    await wrapper.findAll('input[type="checkbox"]')[1]!.setValue(true)
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(register).toHaveBeenCalledWith({
      email: 'mira@example.com',
      username: 'mira',
      displayName: 'Mira',
      password: 'correct horse battery',
    })
    expect(wrapper.text()).not.toContain('Agree to the terms of use')
  })

  it('links both agreements to the pages that explain them', async () => {
    const { wrapper } = await setup()
    const links = wrapper.findAll('.consent a')

    expect(links.map((link) => link.attributes('href'))).toEqual(['/terms', '/privacy'])
    expect(links.every((link) => link.attributes('target') === '_blank')).toBe(true)
  })
})
