import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { ApiError } from '@/api/client'
import * as authApi from '@/api/auth'
import * as usersApi from '@/api/users'
import type {
  AnimeInterestInput,
  LoginInput,
  RegisterInput,
  User,
  UserProfile,
} from '@/types'

/** Statuses that mean "no session here" (R0 backends may not expose auth yet). */
const ANONYMOUS_STATUSES = [401, 404, 501]

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const loading = ref(false)
  const initialized = ref(false)
  let inFlight: Promise<User | null> | null = null

  const isAuthenticated = computed(() => user.value !== null)
  /** True once the session lookup has settled — safe for auth-aware chrome. */
  const isReady = computed(() => initialized.value && !loading.value)

  function isNotReady(error: unknown): boolean {
    return error instanceof ApiError && ANONYMOUS_STATUSES.includes(error.status)
  }

  /** Never rejects: a failed lookup is simply an anonymous session. */
  async function fetchMe(): Promise<User | null> {
    if (inFlight) return inFlight
    inFlight = (async () => {
      loading.value = true
      try {
        user.value = await authApi.getMe()
      } catch {
        user.value = null
      } finally {
        loading.value = false
        initialized.value = true
      }
      return user.value
    })()
    try {
      return await inFlight
    } finally {
      inFlight = null
    }
  }

  async function ensureLoaded(): Promise<void> {
    if (!initialized.value) await fetchMe()
  }

  function adopt(profile: User): User {
    user.value = profile
    initialized.value = true
    return profile
  }

  async function register(input: RegisterInput): Promise<User> {
    return adopt(await authApi.register(input))
  }

  async function login(input: LoginInput): Promise<User> {
    return adopt(await authApi.login(input))
  }

  async function logout(): Promise<void> {
    try {
      await authApi.logout()
    } catch (error) {
      if (!isNotReady(error)) {
        // A dropped connection must not leave the UI claiming a live session.
      }
    } finally {
      user.value = null
      initialized.value = true
    }
  }

  async function updateProfile(patch: Partial<User>): Promise<User> {
    return adopt(await usersApi.updateMe(patch))
  }

  async function setInterests(ids: number[]): Promise<UserProfile> {
    const profile = await usersApi.updateInterests(ids)
    adopt(profile)
    return profile
  }

  async function setAnimeInterests(items: AnimeInterestInput[]): Promise<UserProfile> {
    const profile = await usersApi.updateAnimeInterests(items)
    adopt(profile)
    return profile
  }

  return {
    user,
    loading,
    initialized,
    isAuthenticated,
    isReady,
    fetchMe,
    ensureLoaded,
    register,
    login,
    logout,
    updateProfile,
    setInterests,
    setAnimeInterests,
  }
})
