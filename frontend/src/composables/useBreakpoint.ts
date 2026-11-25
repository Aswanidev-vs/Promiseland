import { computed, getCurrentInstance, onBeforeUnmount, ref } from 'vue'

export type Viewport = 'mobile' | 'tablet' | 'desktop'

const MOBILE = '(max-width: 767px)'
const DESKTOP = '(min-width: 1024px)'

/**
 * Reactive boolean for one media query, cleaned up with the component.
 */
export function useMediaQuery(query: string) {
  const matches = ref(false)

  if (typeof window !== 'undefined' && typeof window.matchMedia === 'function') {
    const list = window.matchMedia(query)
    matches.value = list.matches
    const update = (): void => {
      matches.value = list.matches
    }

    list.addEventListener('change', update)
    if (getCurrentInstance()) {
      onBeforeUnmount(() => list.removeEventListener('change', update))
    }
  }

  return matches
}

/**
 * Reactive viewport band, matching the breakpoints in docs/design-system.md
 * (mobile under 768, tablet to 1023, desktop from 1024).
 *
 * Reach for this when the structure differs, not when only sizes do: rendering
 * a different number of panes, or wiring a control that only exists on one
 * band. Everything that is purely cosmetic belongs in a media query instead.
 */
export function useBreakpoint() {
  const mobile = useMediaQuery(MOBILE)
  const desktop = useMediaQuery(DESKTOP)

  const viewport = computed<Viewport>(() => {
    if (mobile.value) return 'mobile'
    return desktop.value ? 'desktop' : 'tablet'
  })

  return {
    viewport,
    isMobile: computed(() => viewport.value === 'mobile'),
    isTablet: computed(() => viewport.value === 'tablet'),
    isDesktop: computed(() => viewport.value === 'desktop'),
  }
}
