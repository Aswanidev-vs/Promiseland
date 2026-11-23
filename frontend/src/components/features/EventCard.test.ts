import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { Event } from '@/types'
import EventCard from './EventCard.vue'

const RouterLinkStub = defineComponent({
  props: { to: { type: [String, Object], default: '' } },
  setup(props, { slots }) {
    return () => h('a', { href: String(props.to) }, slots.default?.())
  },
})

function makeEvent(overrides: Partial<Event> = {}): Event {
  return {
    id: 9,
    organizer: { id: 2, username: 'mira', display_name: 'Mira', avatar_url: null },
    title: 'Winter photoshoot',
    description: 'Bring your camera',
    event_type: 'photoshoot',
    starts_at: '2026-11-21T12:00:00Z',
    ends_at: '2026-11-21T15:00:00Z',
    timezone: 'UTC',
    venue_text: 'Riverside studio',
    city: 'Kyoto',
    country: 'Japan',
    is_online: false,
    capacity: 20,
    rsvp_count: 5,
    rsvp_status: null,
    cover_url: null,
    status: 'scheduled',
    created_at: '2026-11-01T10:00:00Z',
    ...overrides,
  } as Event
}

function mountCard(event: Event) {
  return mount(EventCard, {
    props: { event },
    global: { stubs: { RouterLink: RouterLinkStub } },
  })
}

describe('EventCard', () => {
  it('formats the date, type, place, and response count', () => {
    const event = makeEvent()
    const wrapper = mountCard(event)
    const start = new Date(event.starts_at)

    expect(wrapper.get('.event-card__title').text()).toBe('Winter photoshoot')
    expect(wrapper.text()).toContain('Photoshoot')
    expect(wrapper.text()).toContain('Riverside studio, Kyoto, Japan')
    expect(wrapper.get('.event-card__count').text()).toBe('5 of 20 going')
    expect(wrapper.get('.event-card__day').text()).toBe(String(start.getDate()).padStart(2, '0'))
    expect(wrapper.get('.event-card__month').text()).toBe(
      new Intl.DateTimeFormat(undefined, { month: 'short' }).format(start),
    )
  })

  it('labels online events and marks a full event', () => {
    const wrapper = mountCard(makeEvent({ is_online: true, capacity: 10, rsvp_count: 10 }))

    expect(wrapper.text()).toContain('Online event')
    expect(wrapper.text()).toContain('At capacity')
    expect(wrapper.get('.event-card__count').text()).toBe('10 of 10 going')
  })

  it('links to the event detail page', () => {
    const wrapper = mountCard(makeEvent())
    expect(wrapper.get('.event-card__body').attributes('href')).toBe('/events/9')
  })
})
