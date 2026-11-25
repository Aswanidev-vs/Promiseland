import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { useAuthStore } from './auth'

const STORAGE_KEY = 'pl-guest'

function readInitial(): boolean {
  if (typeof window === 'undefined') return false
  try {
    return window.localStorage.getItem(STORAGE_KEY) === '1'
  } catch {
    return false
  }
}

/**
 * Guest browsing (PRD §6): public pages stay readable without an account and
 * every write action asks for one. The flag records intent only; it stops
 * counting the moment a session exists.
 */
export const useGuestStore = defineStore('guest', () => {
  const auth = useAuthStore()
  const active = ref(readInitial())
  const isGuest = computed(() => active.value && !auth.isAuthenticated)

  function persist(): void {
    if (typeof window === 'undefined') return
    try {
      if (active.value) window.localStorage.setItem(STORAGE_KEY, '1')
      else window.localStorage.removeItem(STORAGE_KEY)
    } catch {
      // Storage can be unavailable; the flag still holds for this session.
    }
  }

  function enter(): void {
    active.value = true
    persist()
  }

  function leave(): void {
    active.value = false
    persist()
  }

  return { isGuest, enter, leave }
})
