<script setup lang="ts">
import EmptyState from '@/components/ui/EmptyState.vue'
import SkeletonList from '@/components/ui/SkeletonList.vue'
import ChatIcon from './ChatIcon.vue'
import { plural } from './format'
import type { ChatServer } from '@/types'

withDefaults(
  defineProps<{
    servers: ChatServer[]
    loading?: boolean
    error?: string
    forbidden?: boolean
    activeId?: number
  }>(),
  { loading: false, error: '', forbidden: false, activeId: undefined },
)

const emit = defineEmits<{ select: [id: number]; create: []; retry: [] }>()

function initial(server: ChatServer): string {
  return (server.name || '?').trim().charAt(0).toUpperCase()
}

function statusFor(server: ChatServer): string {
  const role = server.role
  return role === 'owner' ? 'owner' : role === 'moderator' ? 'moderator' : 'member'
}
</script>

<template>
  <section class="servers" aria-labelledby="servers-title">
    <header class="servers__head">
      <h2 id="servers-title" class="servers__title">Servers</h2>
      <button type="button" class="servers__add" aria-label="Create a server" title="Create a server" @click="emit('create')">
        <ChatIcon name="plus" :size="18" />
      </button>
    </header>

    <SkeletonList v-if="loading" :rows="4" />

    <EmptyState
      v-else-if="forbidden"
      title="Servers are not available"
      description="Your session may have expired. Sign in again to see your servers."
      action-label="Try again"
      @action="emit('retry')"
    />

    <EmptyState
      v-else-if="error"
      title="We could not load servers"
      :description="error"
      action-label="Try again"
      @action="emit('retry')"
    />

    <EmptyState
      v-else-if="!servers.length"
      title="No servers yet"
      description="Create a server to gather a channel for your community."
      action-label="Create a server"
      @action="emit('create')"
    />

    <ul v-else class="servers__list">
      <li v-for="server in servers" :key="server.id">
        <button
          type="button"
          class="servers__item"
          :class="{ 'servers__item--active': server.id === activeId }"
          :aria-current="server.id === activeId ? 'true' : undefined"
          @click="emit('select', server.id)"
        >
          <span class="servers__tile" aria-hidden="true">{{ initial(server) }}</span>
          <span class="servers__info">
            <span class="servers__name">{{ server.name }}</span>
            <span class="servers__meta">
              <span class="mono">{{ plural(server.member_count ?? 0, 'member') }}</span>
            </span>
          </span>
          <span class="servers__role">{{ statusFor(server) }}</span>
        </button>
      </li>
    </ul>
  </section>
</template>

<style scoped>
.servers {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  min-height: 0;
}

.servers__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
}

.servers__title {
  font-size: var(--text-lg);
}

.servers__add {
  display: inline-flex;
  width: 44px;
  height: 44px;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-md);
  background: var(--ink-800);
  color: var(--text-secondary);
  cursor: pointer;
}

.servers__add:hover {
  color: var(--text-primary);
}

.servers__list {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  overflow-y: auto;
}

.servers__item {
  display: flex;
  width: 100%;
  min-height: 56px;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2);
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--text-primary);
  text-align: left;
  cursor: pointer;
}

.servers__item:hover {
  background: var(--ink-800);
}

.servers__item--active {
  border-color: var(--border-hairline);
  background: var(--ink-800);
}

.servers__tile {
  display: inline-flex;
  width: 40px;
  height: 40px;
  flex: none;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-md);
  background: var(--ink-700);
  color: var(--text-primary);
  font-family: var(--font-display);
  font-size: var(--text-lg);
}

.servers__info {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
}

.servers__name {
  overflow: hidden;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.servers__meta {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--text-secondary);
  font-size: var(--text-xs);
}

.servers__role {
  color: var(--text-muted);
  font-size: var(--text-xs);
}

@media (max-width: 480px) {
  .servers__role {
    display: none;
  }
}
</style>
