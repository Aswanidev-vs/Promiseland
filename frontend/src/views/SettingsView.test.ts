import { createPinia, setActivePinia } from 'pinia'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { getMe } from '@/api/auth'
import { uploadMedia } from '@/api/media'
import { updateMe } from '@/api/users'
import SettingsView from './SettingsView.vue'

vi.mock('@/api/auth', () => ({
  getMe: vi.fn(),
  login: vi.fn(),
  register: vi.fn(),
  logout: vi.fn(),
  verifyEmail: vi.fn(),
  requestPasswordReset: vi.fn(),
  confirmPasswordReset: vi.fn(),
}))

vi.mock('@/api/users', () => ({
  updateMe: vi.fn(),
  updateSettings: vi.fn(),
  listInterests: vi.fn(),
  addInterest: vi.fn(),
  removeInterest: vi.fn(),
}))

vi.mock('@/api/media', () => ({ uploadMedia: vi.fn() }))

const Blank = { template: '<div />' }

const me = {
  id: 2,
  username: 'mira',
  display_name: 'Mira',
  bio: '',
  location: '',
  avatar_url: null,
  banner_url: null,
  visibility: 'public',
  discoverable: true,
  email: 'mira@example.com',
} as never

async function setup() {
  const pinia = createPinia()
  setActivePinia(pinia)
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/settings', name: 'settings', component: SettingsView },
      { path: '/login', name: 'login', component: Blank },
      { path: '/home', name: 'home', component: Blank },
    ],
  })
  await router.push('/settings')
  await router.isReady()

  const wrapper = mount(SettingsView, { global: { plugins: [pinia, router] } })
  await flushPromises()
  return wrapper
}

function chooseFile(wrapper: Awaited<ReturnType<typeof setup>>, file: File) {
  const input = wrapper.get('input[type="file"]')
  Object.defineProperty(input.element, 'files', { value: [file], configurable: true })
  return input.trigger('change')
}

describe('SettingsView avatar upload', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(getMe).mockResolvedValue(me)
    vi.mocked(updateMe).mockResolvedValue(me)
    vi.mocked(uploadMedia).mockResolvedValue({
      id: 9,
      url: '/media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.png',
      mime_type: 'image/png',
      size_bytes: 120,
      width: 4,
      height: 3,
    })
  })

  it('uploads a chosen photo and sends it with the profile save', async () => {
    const wrapper = await setup()

    await chooseFile(wrapper, new File(['image'], 'me.png', { type: 'image/png' }))
    await flushPromises()

    expect(uploadMedia).toHaveBeenCalledTimes(1)
    expect(wrapper.get('.uploader__preview img').attributes('src')).toBe(
      '/media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.png',
    )
    expect(wrapper.text()).toContain('me.png · 4 × 3')

    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(updateMe).toHaveBeenCalledWith(
      expect.objectContaining({
        avatar_url: '/media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.png',
      }),
    )
  })

  it('refuses a file that is not an allowed image', async () => {
    const wrapper = await setup()

    await chooseFile(wrapper, new File(['%PDF'], 'notes.pdf', { type: 'application/pdf' }))
    await flushPromises()

    expect(uploadMedia).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('Choose a JPEG, PNG, GIF or WebP image.')
  })

  it('refuses an image over 5 MB before uploading it', async () => {
    const wrapper = await setup()
    const big = new File([new Uint8Array(5 * 1024 * 1024 + 1)], 'big.png', { type: 'image/png' })

    await chooseFile(wrapper, big)
    await flushPromises()

    expect(uploadMedia).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('Images must be 5 MB or smaller.')
  })

  it('removes the photo on the server as soon as Remove is pressed', async () => {
    const wrapper = await setup()

    await chooseFile(wrapper, new File(['image'], 'me.png', { type: 'image/png' }))
    await flushPromises()

    await wrapper.findAll('button').find((button) => button.text() === 'Remove')!.trigger('click')
    await flushPromises()

    expect(wrapper.find('.uploader__preview img').exists()).toBe(false)
    expect(updateMe).toHaveBeenCalledWith({ avatar_url: '' })
  })
})
