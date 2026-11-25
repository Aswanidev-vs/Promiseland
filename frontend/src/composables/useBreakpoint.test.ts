import { defineComponent, h, nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { useBreakpoint } from './useBreakpoint'

type Handler = (event: MediaQueryListEvent) => void

function stubMatchMedia(initial: Record<string, boolean>) {
  const matches = { ...initial }
  const listeners = new Map<string, Set<Handler>>()

  vi.stubGlobal(
    'matchMedia',
    vi.fn((query: string) => {
      const set = listeners.get(query) ?? new Set<Handler>()
      listeners.set(query, set)
      return {
        // A getter, so emit() can change the queried state under the composable.
        get matches() {
          return matches[query] ?? false
        },
        media: query,
        addEventListener: (_: string, handler: Handler) => set.add(handler),
        removeEventListener: (_: string, handler: Handler) => set.delete(handler),
      }
    }),
  )

  return {
    emit(query: string, value: boolean) {
      matches[query] = value
      for (const handler of listeners.get(query) ?? []) handler({ matches: value } as MediaQueryListEvent)
    },
    listenerCount(query: string) {
      return listeners.get(query)?.size ?? 0
    },
  }
}

const MOBILE = '(max-width: 767px)'
const DESKTOP = '(min-width: 1024px)'

function mountProbe() {
  const Probe = defineComponent({
    setup() {
      const { viewport, isMobile, isDesktop } = useBreakpoint()
      return () => h('p', `${viewport.value}:${isMobile.value}:${isDesktop.value}`)
    },
  })
  return mount(Probe)
}

describe('useBreakpoint', () => {
  it('reports the band for the current width', () => {
    stubMatchMedia({ [DESKTOP]: true })
    expect(mountProbe().text()).toBe('desktop:false:true')
    vi.unstubAllGlobals()

    stubMatchMedia({ [MOBILE]: true })
    expect(mountProbe().text()).toBe('mobile:true:false')
    vi.unstubAllGlobals()

    stubMatchMedia({})
    expect(mountProbe().text()).toBe('tablet:false:false')
    vi.unstubAllGlobals()
  })

  it('follows a resize across bands', async () => {
    const media = stubMatchMedia({ [DESKTOP]: true })
    const wrapper = mountProbe()

    media.emit(MOBILE, true)
    media.emit(DESKTOP, false)
    await nextTick()

    expect(wrapper.text()).toBe('mobile:true:false')
    vi.unstubAllGlobals()
  })

  it('removes its listeners when the component unmounts', () => {
    const media = stubMatchMedia({ [DESKTOP]: true })
    const wrapper = mountProbe()
    expect(media.listenerCount(DESKTOP)).toBe(1)

    wrapper.unmount()

    expect(media.listenerCount(DESKTOP)).toBe(0)
    expect(media.listenerCount(MOBILE)).toBe(0)
    vi.unstubAllGlobals()
  })
})
