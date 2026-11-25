import { computed, getCurrentInstance, onBeforeUnmount, ref } from 'vue'

export type Viewport = 'mobile' | 'tablet' | 'desktop'

const MOBILE = '(max-width: 767px)'
const DESKTOP = '(min-width: 1024px)'

/**
 * Reactive viewport band, matching the breakpoints in docs/design-system.md
 * (mobile under 768, tablet to 1023, desktop from 1024).
 *
 * Reach for this when the structure differs, not when only sizes do: rendering
 * a different number of panes, or wiring a control that only exists on one
 * band. Everything that is purely cosmetic belongs in a media query instead.
 */
export function useBreakpoint() {
  const viewport = ref<Viewport>('desktop')

  if (typeof window !== 'undefined' && typeof window.matchMedia === 'function') {
    const mobile = window.matchMedia(MOBILE)
    const desktop = window.matchMedia(DESKTOP)
    const update = (): void => {
      viewport.value = mobile.matches ? 'mobile' : desktop.matches ? 'desktop' : 'tablet'
    }

    update()
    mobile.addEventListener('change', update)
    desktop.addEventListener('change', update)

    if (getCurrentInstance()) {
      onBeforeUnmount(() => {
        mobile.removeEventListener('change', update)
        desktop.removeEventListener('change', update)
      })
    }
  }

  return {
    viewport,
    isMobile: computed(() => viewport.value === 'mobile'),
    isTablet: computed(() => viewport.value === 'tablet'),
    isDesktop: computed(() => viewport.value === 'desktop'),
  }
}
