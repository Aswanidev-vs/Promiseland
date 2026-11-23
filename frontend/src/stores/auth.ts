import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { ApiError, api } from '@/api/client'
import type { User } from '@/types'

/** Statuses that mean "no session here yet" during R0/R1 development. */
const ANONYMOUS_STATUSES = [401, 404, 501]

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const loading = ref(false)
  const initialized = ref(false)

  const isAuthenticated = computed(() => user.value !== null)

  function isNotReady(error: unknown): boolean {
    return error instanceof ApiError && ANONYMOUS_STATUSES.includes(error.status)
  }

  async function fetchMe(): Promise<User | null> {
    if (loading.value) return user.value
    loading.value = true
    try {
      // R0 backend may not expose /auth/me yet — 404/501 is a valid
      // "anonymous for now" answer until R1 ships account endpoints.
      user.value = await api.get<User>('/auth/me')
    } catch (error) {
      user.value = null
      if (!isNotReady(error)) {
        // Network failure is also treated as anonymous: the shell must still
        // render. R1 can surface a retry toast here once the UI store is wired.
      }
    } finally {
      loading.value = false
      initialized.value = true
    }
    return user.value
  }

  async function ensureLoaded(): Promise<void> {
    if (!initialized.value && !loading.value) await fetchMe()
  }

  async function logout(): Promise<void> {
    try {
      await api.post<void>('/auth/logout')
    } catch (error) {
      if (!isNotReady(error)) throw error
    } finally {
      user.value = null
    }
  }

  return { user, loading, initialized, isAuthenticated, fetchMe, ensureLoaded, logout }
})
