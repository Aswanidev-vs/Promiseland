<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import BaseButton from '@/components/ui/BaseButton.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import SkeletonList from '@/components/ui/SkeletonList.vue'
import FeedTabs from '@/components/features/FeedTabs.vue'
import ConnectionRow from '@/components/features/ConnectionRow.vue'
import { deleteConnection, listConnections, reviewConnection } from '@/api/discovery'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import type { Connection } from '@/types'
import { errorMessage, isForbidden } from '@/components/features/format'

type ConnectionTab = 'incoming' | 'outgoing' | 'accepted' | 'all'

const auth = useAuthStore()
const ui = useUiStore()
const router = useRouter()

const tab = ref<ConnectionTab>('incoming')
const tabs = [
  { value: 'incoming', label: 'Incoming' },
  { value: 'outgoing', label: 'Outgoing' },
  { value: 'accepted', label: 'Accepted' },
  { value: 'all', label: 'All' },
]

const connections = ref<Connection[]>([])
const cursor = ref('')
const loading = ref(true)
const loadingMore = ref(false)
const error = ref('')
const forbidden = ref(false)
const busyId = ref<number | null>(null)

const currentUserId = computed(() => Number(auth.user?.id ?? 0))

const emptyCopy = computed(() => {
  if (tab.value === 'incoming') {
    return {
      title: 'No pending requests',
      description: 'When someone wants to connect, their request will show up here.',
    }
  }
  if (tab.value === 'outgoing') {
    return {
      title: 'No sent requests',
      description: 'Requests you send will appear here while you wait for a reply.',
    }
  }
  if (tab.value === 'accepted') {
    return {
      title: 'No connections yet',
      description: 'Connect with someone who shares your interests to see them here.',
    }
  }
  return {
    title: 'No connections yet',
    description: 'Find someone with shared interests and send a request to get started.',
  }
})

function listParams() {
  if (tab.value === 'incoming') {
    return { direction: 'incoming' as const, status: 'pending' as const }
  }
  if (tab.value === 'outgoing') {
    return { direction: 'outgoing' as const, status: 'pending' as const }
  }
  if (tab.value === 'accepted') return { status: 'accepted' as const }
  return { direction: 'all' as const }
}

async function load(reset = false): Promise<void> {
  if (reset) {
    connections.value = []
    cursor.value = ''
    error.value = ''
    forbidden.value = false
  }
  const more = !reset && cursor.value !== ''
  if (more) loadingMore.value = true
  else loading.value = true
  try {
    const page = await listConnections({
      ...listParams(),
      limit: 20,
      ...(more ? { cursor: cursor.value } : {}),
    })
    connections.value = more ? [...connections.value, ...page.items] : [...page.items]
    cursor.value = page.next_cursor || ''
  } catch (cause) {
    if (isForbidden(cause)) forbidden.value = true
    else error.value = errorMessage(cause, 'We could not load connections. Try again.')
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

function changeTab(value: string): void {
  const next = value as ConnectionTab
  if (next === tab.value) return
  tab.value = next
  void load(true)
}

function applyUpdate(updated: Connection): void {
  const index = connections.value.findIndex((item) => item.id === updated.id)
  if (index === -1) return
  const leavingCurrentTab =
    (tab.value === 'incoming' || tab.value === 'outgoing') && updated.status !== 'pending'
  if (leavingCurrentTab) connections.value.splice(index, 1)
  else connections.value[index] = updated
}

async function review(connection: Connection, status: 'accepted' | 'rejected' | 'ignored'): Promise<void> {
  if (busyId.value !== null) return
  busyId.value = connection.id
  try {
    const updated = await reviewConnection(connection.id, status)
    applyUpdate(updated)
    const titles = {
      accepted: 'Request accepted',
      rejected: 'Request declined',
      ignored: 'Request ignored',
    }
    ui.pushToast({ title: titles[status], tone: 'success' })
  } catch (cause) {
    ui.pushToast({
      title: 'Could not update request',
      message: errorMessage(cause, 'Try again in a moment.'),
      tone: 'error',
    })
  } finally {
    busyId.value = null
  }
}

async function cancel(connection: Connection): Promise<void> {
  if (busyId.value !== null) return
  busyId.value = connection.id
  try {
    await deleteConnection(connection.id)
    connections.value = connections.value.filter((item) => item.id !== connection.id)
    ui.pushToast({ title: 'Request cancelled', tone: 'success' })
  } catch (cause) {
    ui.pushToast({
      title: 'Could not cancel request',
      message: errorMessage(cause, 'Try again in a moment.'),
      tone: 'error',
    })
  } finally {
    busyId.value = null
  }
}

onMounted(() => void load(true))
</script>

<template>
  <div class="connections">
    <header class="connections__header">
      <h1 class="connections__title">Connections</h1>
      <p class="connections__intro">Keep track of the people you want to meet through fandom.</p>
    </header>

    <FeedTabs
      :model-value="tab"
      :tabs="tabs"
      label="Connection sections"
      @update:model-value="changeTab"
    />

    <SkeletonList v-if="loading" :rows="3" avatar />

    <EmptyState
      v-else-if="forbidden"
      title="Connections are not available"
      description="This needs an active account. Sign in again to view your requests."
      action-label="Go to home"
      @action="router.push('/home')"
    />

    <EmptyState
      v-else-if="error"
      title="We could not load connections"
      :description="error"
      action-label="Try again"
      @action="load(true)"
    />

    <EmptyState
      v-else-if="!connections.length"
      :title="emptyCopy.title"
      :description="emptyCopy.description"
      action-label="Find people"
      @action="router.push('/discover')"
    />

    <template v-else>
      <ul class="connections__list">
        <li v-for="connection in connections" :key="connection.id">
          <ConnectionRow
            :connection="connection"
            :current-user-id="currentUserId"
            :direction="tab === 'all' || tab === 'accepted' ? 'all' : tab"
            :busy="busyId === connection.id"
            @accept="review(connection, 'accepted')"
            @reject="review(connection, 'rejected')"
            @ignore="review(connection, 'ignored')"
            @cancel="cancel(connection)"
          />
        </li>
      </ul>
      <div v-if="cursor" class="connections__more">
        <BaseButton :loading="loadingMore" @click="load()">Load more</BaseButton>
      </div>
    </template>
  </div>
</template>

<style scoped>
.connections {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.connections__header {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.connections__title {
  font-family: var(--font-display);
  font-weight: 560;
  letter-spacing: -0.02em;
}

.connections__intro {
  color: var(--text-secondary);
}

.connections__list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.connections__more {
  display: flex;
  justify-content: center;
}
</style>
