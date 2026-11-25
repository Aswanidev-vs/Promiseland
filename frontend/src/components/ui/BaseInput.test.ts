import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { defineComponent, h } from 'vue'
import BaseInput from './BaseInput.vue'

describe('BaseInput', () => {
  it('gives each instance a unique id that its label points at', () => {
    const Form = defineComponent({
      render: () => h('div', [
        h(BaseInput, { label: 'Name' }),
        h(BaseInput, { label: 'Description' }),
      ]),
    })

    const wrapper = mount(Form)
    const inputs = wrapper.findAll('input')
    const labels = wrapper.findAll('label')

    expect(inputs[0]!.attributes('id')).not.toBe(inputs[1]!.attributes('id'))
    expect(labels[0]!.attributes('for')).toBe(inputs[0]!.attributes('id'))
    expect(labels[1]!.attributes('for')).toBe(inputs[1]!.attributes('id'))
  })

  it('reveals a password only when asked', async () => {
    const wrapper = mount(BaseInput, {
      props: { label: 'Password', type: 'password', revealable: true },
    })

    expect(wrapper.get('input').attributes('type')).toBe('password')
    await wrapper.get('.field__reveal').trigger('click')
    expect(wrapper.get('input').attributes('type')).toBe('text')
  })
})
