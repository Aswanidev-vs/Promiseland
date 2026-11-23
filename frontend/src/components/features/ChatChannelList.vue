<script setup lang="ts">
import EmptyState from '@/components/ui/EmptyState.vue'
import SkeletonList from '@/components/ui/SkeletonList.vue'
import ChatIcon from './ChatIcon.vue'
import { formatWhen } from './format'
import type { ChatChannel } from '@/types'

withDefaults(
  defineProps<{
    channels: ChatChannel[]
    loading?: boolean
    error?: string
    forbidden?: boolean
    activeId?: number
    canCreate?: boolean
  }>(),
  { loading: false, error: '', forbidden: false, activeId: undefined, canCreate: false },
)

const emit = defineEmits<{ select: [id: number]; create: []; retry: [] }>()
</script>

<template>
  <section class="channels" aria-labelledby="channels-title">
    <header class="channels__head">
      <h2 id="channels-title" class="channels__title">Channels</h2>
      <button
        v-if="canCreate"
        type="button"
        class="channels__add"
        aria-label="Create a channel"
        title="Create a channel"
        @click="emit('create')"
      >
        <ChatIcon name="plus" :size="18" />
      </button>
    </header>

    <SkeletonList v-if="loading" :rows="4" />

    <EmptyState
      v-else-if="forbidden"
      title="You do not have access to these channels"
      description="Ask a server owner for access."
      action-label="Try again"
      @action="emit('retry')"
    />

    <EmptyState
      v-else-if="error"
      title="We could not load channels"
      :description="error"
      action-label="Try again"
      @action="emit('retry')"
    />

    <EmptyState
      v-else-if="!channels.length"
      title="No channels yet"
      description="Channels keep conversations organized by topic."
      :action-label="canCreate ? 'Create a channel' : ''"
      @action="emit('create')"
    />

    <ul v-else class="channels__list">
      <li v-for="channel in channels" :key="channel.id">
        <button
          type="button"
          class="channels__item"
          :class="{ 'channels__item--active': channel.id === activeId }"
          :aria-current="channel.id === activeId ? 'true' : undefined"
          @click="emit('select', channel.id)"
        >
          <ChatIcon name="hash" :size="16" />
          <span class="channels__info">
            <span class="channels__name">{{ channel.name }}</span>
            <span v-if="channel.last_message" class="channels__preview">
              {{ channel.last_message.deleted ? 'Message removed' : channel.last_message.body }}
            </span>
            <span v-else-if="channel.description" class="channels__preview">{{ channel.description }}</span>
          </span>
          <span v-if="channel.unread_count" class="channels__unread mono" :aria-label="`${channel.unread_count} unread`">
            {{ channel.unread_count }}
          </span>
          <time v-else-if="channel.last_message?.created_at" class="channels__time" :datetime="channel.last_message.created_at">
            {{ formatWhen(channel.last_message.created_at) }}
          </time>
        </button>
      </li>
    </ul>
  </section>
</template>

<style scoped>
.channels {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  min-height: 0;
}

.channels__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
}

.channels__title {
  font-size: var(--text-lg);
}

.channels__add {
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

.channels__add:hover {
  color: var(--text-primary);
}

.channels__list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  overflow-y: auto;
}

.channels__item {
  display: flex;
  width: 100%;
  min-height: 44px;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  border: 0;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--text-secondary);
  text-align: left;
  cursor: pointer;
}

.channels__item:hover {
  background: var(--ink-800);
  color: var(--text-primary);
}

.channels__item--active {
  background: var(--ink-700);
  color: var(--text-primary);
}

.channels__info {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
}

.channels__name {
  overflow: hidden;
  font-size: var(--text-sm);
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.channels__preview {
  overflow: hidden;
  color: var(--text-muted);
  font-size: var(--text-xs);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.channels__unread {
  min-width: 20px;
  padding: 0 var(--space-1);
  border-radius: var(--radius-pill);
  background: var(--ember);
  color: var(--ink-900);
  font-size: var(--text-xs);
  text-align: center;
}

.channels__time {
  color: var(--text-muted);
  font-size: var(--text-xs);
}
</style>
