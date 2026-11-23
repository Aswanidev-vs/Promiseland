import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { errorMessage } from '@/api/client'
import * as api from '@/api/communities'
import type { Community, ListCommunitiesParams } from '@/types'

export const useCommunitiesStore = defineStore('communities', () => {
  const items = ref<Community[]>([])
  const bySlug = ref<Record<string, Community>>({})
  const loading = ref(false)
  const error = ref<string | null>(null)
  const nextCursor = ref('')

  const joined = computed(() => items.value.filter((community) => community.is_member))

  function remember(community: Community): Community {
    bySlug.value = { ...bySlug.value, [community.slug]: community }
    const index = items.value.findIndex((item) => item.slug === community.slug)
    if (index !== -1) items.value.splice(index, 1, community)
    return community
  }

  async function loadList(p: ListCommunitiesParams = {}, append = false): Promise<void> {
    loading.value = true
    error.value = null
    try {
      const page = await api.listCommunities(p)
      items.value = append ? [...items.value, ...page.items] : page.items
      page.items.forEach(remember)
      nextCursor.value = page.next_cursor
    } catch (cause) {
      error.value = errorMessage(cause)
    } finally {
      loading.value = false
    }
  }

  async function load(slug: string): Promise<Community | null> {
    try {
      return remember(await api.getCommunity(slug))
    } catch (cause) {
      error.value = errorMessage(cause)
      return null
    }
  }

  /** Optimistically flips membership and rolls back if the API rejects. */
  async function setMembership(slug: string, join: boolean): Promise<boolean> {
    const current = bySlug.value[slug] ?? items.value.find((item) => item.slug === slug)
    if (!current) return false
    const previousMember = current.is_member
    const previousCount = current.member_count
    current.is_member = join
    current.member_count = Math.max(0, previousCount + (join ? 1 : -1))
    try {
      if (join) await api.joinCommunity(slug)
      else await api.leaveCommunity(slug)
      remember(current)
      return true
    } catch (cause) {
      current.is_member = previousMember
      current.member_count = previousCount
      error.value = errorMessage(cause)
      return false
    }
  }

  return { items, bySlug, joined, loading, error, nextCursor, loadList, load, setMembership }
})
