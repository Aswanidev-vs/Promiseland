import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import CreatorCategoryPicker from './CreatorCategoryPicker.vue'

function optionByLabel(wrapper: ReturnType<typeof mount>, label: string) {
  const button = wrapper.findAll('button').find((item) => item.text() === label)
  if (!button) throw new Error(`Option ${label} not found`)
  return button
}

describe('CreatorCategoryPicker', () => {
  it('emits a single category and clears it when picked again', async () => {
    const wrapper = mount(CreatorCategoryPicker, { props: { modelValue: '' } })

    await optionByLabel(wrapper, 'Cosplay').trigger('click')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual(['cosplay'])

    await wrapper.setProps({ modelValue: 'cosplay' })
    await optionByLabel(wrapper, 'Cosplay').trigger('click')
    expect(wrapper.emitted('update:modelValue')?.[1]).toEqual([''])
  })

  it('accumulates categories in multiple mode', async () => {
    const wrapper = mount(CreatorCategoryPicker, {
      props: { modelValue: [], multiple: true, includeNone: false },
    })

    await optionByLabel(wrapper, 'Cosplay').trigger('click')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([['cosplay']])

    await wrapper.setProps({ modelValue: ['cosplay'] })
    await optionByLabel(wrapper, 'Edit').trigger('click')
    expect(wrapper.emitted('update:modelValue')?.[1]).toEqual([['cosplay', 'edit']])
  })
})
