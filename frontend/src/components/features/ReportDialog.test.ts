import { createPinia } from 'pinia'
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { report } from '@/api/moderation'
import ReportDialog from './ReportDialog.vue'

vi.mock('@/api/moderation', () => ({ report: vi.fn(async () => undefined) }))

function mountDialog() {
  return mount(ReportDialog, {
    props: { targetType: 'post', targetId: 12 },
    global: { plugins: [createPinia()] },
  })
}

describe('ReportDialog', () => {
  it('lists the PRD report reasons and sends the chosen reason with details', async () => {
    const wrapper = mountDialog()
    const options = wrapper.findAll('#report-reason option').map((option) => option.text())

    expect(options).toContain('Spam')
    expect(options).toContain('Spoiler abuse')
    expect(options).toContain('Something else')

    await wrapper.get('#report-reason').setValue('spoiler_abuse')
    await wrapper.get('#report-details').setValue('Finale twist was not marked.')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(report).toHaveBeenCalledWith({
      targetType: 'post',
      targetId: 12,
      reason: 'spoiler_abuse',
      details: 'Finale twist was not marked.',
    })
    expect(wrapper.emitted('submitted')).toHaveLength(1)
    expect(wrapper.emitted('close')).toHaveLength(1)
  })

  it('requires a reason before sending', async () => {
    const wrapper = mountDialog()
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(report).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('Choose a reason for this report.')
  })
})
