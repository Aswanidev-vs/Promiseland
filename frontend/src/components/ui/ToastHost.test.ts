import { createPinia, setActivePinia } from 'pinia'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { describe, expect, it } from 'vitest'
import { useUiStore } from '@/stores/ui'
import ToastHost from './ToastHost.vue'

function mountHost() {
  const pinia = createPinia()
  setActivePinia(pinia)
  const ui = useUiStore(pinia)
  const wrapper = mount(ToastHost, { global: { plugins: [pinia] } })
  return { wrapper, ui }
}

describe('ToastHost', () => {
  it('renders a tone icon, the copy and a 44px dismiss control', async () => {
    const { wrapper, ui } = mountHost()

    ui.pushToast({ title: 'Profile saved', tone: 'success' })
    await nextTick()

    const toast = wrapper.get('.toast')
    expect(toast.attributes('role')).toBe('status')
    expect(toast.classes()).toContain('toast--success')
    expect(toast.find('.toast__icon svg').exists()).toBe(true)
    expect(toast.text()).toContain('Profile saved')
    expect(wrapper.get('.toast__close').attributes('aria-label')).toContain('Profile saved')
  })

  it('marks a failure with the alert icon', async () => {
    const { wrapper, ui } = mountHost()

    ui.pushToast({ title: 'Upload failed', message: 'Try again.', tone: 'error' })
    await nextTick()

    expect(wrapper.get('.toast').classes()).toContain('toast--error')
    expect(wrapper.get('.toast').text()).toContain('Try again.')
  })

  it('dismisses the toast it closes', async () => {
    const { wrapper, ui } = mountHost()

    ui.pushToast({ title: 'One' })
    ui.pushToast({ title: 'Two' })
    await nextTick()
    expect(wrapper.findAll('.toast')).toHaveLength(2)

    await wrapper.findAll('.toast__close')[0]!.trigger('click')

    expect(ui.toasts).toHaveLength(1)
    expect(ui.toasts[0]!.title).toBe('Two')
  })
})
