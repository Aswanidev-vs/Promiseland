import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { UserSummary } from '@/types'
import ConnectDialog from './ConnectDialog.vue'

const user: UserSummary = {
  id: 12,
  username: 'mira',
  display_name: 'Mira',
  avatar_url: null,
}

function mountDialog(props: Partial<{ busy: boolean; error: string }> = {}) {
  return mount(ConnectDialog, { props: { user, ...props } })
}

describe('ConnectDialog', () => {
  it('emits the typed message when the request is submitted', async () => {
    const wrapper = mountDialog()
    const message = 'We both follow Frieren — want to swap recommendations?'

    await wrapper.get('#connect-message').setValue(message)
    await wrapper.get('form').trigger('submit')

    expect(wrapper.emitted('submit')).toEqual([[message]])
  })

  it('allows an empty note but blocks messages over the limit', async () => {
    const wrapper = mountDialog()

    await wrapper.get('form').trigger('submit')
    expect(wrapper.emitted('submit')).toEqual([['']])

    await wrapper.get('#connect-message').setValue('x'.repeat(281))
    await wrapper.get('form').trigger('submit')
    expect(wrapper.emitted('submit')).toHaveLength(1)
    expect(wrapper.text()).toContain('Keep your message under 280 characters.')
  })

  it('shows an error passed from the view and closes on cancel', async () => {
    const wrapper = mountDialog({ error: 'We could not send your request.' })
    expect(wrapper.text()).toContain('We could not send your request.')

    const cancel = wrapper.findAll('button').find((button) => button.text() === 'Cancel')
    await cancel!.trigger('click')
    expect(wrapper.emitted('close')).toHaveLength(1)
  })
})
