import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { errorMessage } from '@/api/client'
import * as api from '@/api/notifications'
import type { Notification, PaginationParams } from '@/types'

export const useNotificationsStore = defineStore('notifications', () => {
  const items = ref<Notification[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  const unreadCount = computed(() => items.value.filter((item) => item.read_at === null).length)

  async function load(p: PaginationParams = { limit: 20 }): Promise<void> {
    loading.value = true
    error.value = null
    try {
      items.value = (await api.listNotifications(p)).items
    } catch (cause) {
      error.value = errorMessage(cause)
    } finally {
      loading.value = false
    }
  }

  async function markRead(id: number): Promise<void> {
    const item = items.value.find((entry) => entry.id === id)
    const previous = item?.read_at ?? null
    if (item) item.read_at = new Date().toISOString()
    try {
      await api.markRead(id)
    } catch (cause) {
      if (item) item.read_at = previous
      error.value = errorMessage(cause)
    }
  }

  async function markAllRead(): Promise<void> {
    const previous = items.value.map((item) => item.read_at)
    const now = new Date().toISOString()
    items.value.forEach((item) => (item.read_at = now))
    try {
      await api.markAllRead()
    } catch (cause) {
      items.value.forEach((item, index) => (item.read_at = previous[index] ?? null))
      error.value = errorMessage(cause)
    }
  }

  return { items, loading, error, unreadCount, load, markRead, markAllRead }
})
