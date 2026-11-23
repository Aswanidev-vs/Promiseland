import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { DiscoveryCandidate } from '@/types'
import DiscoveryCard from './DiscoveryCard.vue'

const RouterLinkStub = defineComponent({
  props: { to: { type: [String, Object], default: '' } },
  setup(props, { slots }) {
    return () => h('a', { href: String(props.to) }, slots.default?.())
  },
})

function makeCandidate(overrides: Partial<DiscoveryCandidate> = {}): DiscoveryCandidate {
  return {
    user: { id: 12, username: 'mira', display_name: 'Mira', avatar_url: null },
    shared: {
      anime: ['Frieren', 'Vinland Saga'],
      genres: ['Isekai'],
      interests: ['Cosplay'],
      communities: ['Harbour Club'],
    },
    shared_count: 5,
    explanation: 'You both follow 8 of the same anime and 3 of the same communities',
    connection_status: 'none',
    ...overrides,
  } as DiscoveryCandidate
}

function mountCard(candidate: DiscoveryCandidate) {
  return mount(DiscoveryCard, {
    props: { candidate },
    global: { stubs: { RouterLink: RouterLinkStub } },
  })
}

describe('DiscoveryCard', () => {
  it('renders the compatibility explanation as the memorable content moment', () => {
    const wrapper = mountCard(makeCandidate())

    expect(wrapper.get('.compat__explanation').text()).toBe(
      'You both follow 8 of the same anime and 3 of the same communities',
    )
    expect(wrapper.get('.compat__count').text()).toBe('5 things')
  })

  it('renders shared anime, genres, interests and communities as TagChips', () => {
    const wrapper = mountCard(makeCandidate())

    for (const label of ['Anime', 'Genres', 'Interests', 'Communities']) {
      expect(wrapper.text()).toContain(label)
    }
    for (const tag of ['Frieren', 'Vinland Saga', 'Isekai', 'Cosplay', 'Harbour Club']) {
      expect(wrapper.text()).toContain(tag)
    }
    expect(wrapper.findAll('.chip')).toHaveLength(5)
  })

  it('emits connect and not-interested from its actions', async () => {
    const wrapper = mountCard(makeCandidate())
    const buttons = wrapper.findAll('button')
    const connect = buttons.find((button) => button.text() === 'Connect')
    const notInterested = buttons.find((button) => button.text() === 'Not interested')

    await connect!.trigger('click')
    await notInterested!.trigger('click')

    expect(wrapper.emitted('connect')).toHaveLength(1)
    expect(wrapper.emitted('notInterested')).toHaveLength(1)
  })

  it('replaces Connect with a review link for an incoming request', () => {
    const wrapper = mountCard(makeCandidate({ connection_status: 'pending_in' }))
    expect(wrapper.text()).toContain('Wants to connect')
    expect(wrapper.find('.discovery-card__link').exists()).toBe(true)
    expect(wrapper.findAll('button').some((button) => button.text() === 'Connect')).toBe(false)
  })
})
