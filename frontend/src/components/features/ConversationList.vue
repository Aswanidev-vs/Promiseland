<script setup lang="ts">
import { computed } from 'vue'
import Avatar from '@/components/ui/Avatar.vue'
import SkeletonList from '@/components/ui/SkeletonList.vue'
import ChatIcon from './ChatIcon.vue'
import PresenceDot from './PresenceDot.vue'
import { conversationLabel, conversationMembers } from './chatMessageUtils'
import { formatWhen } from './format'
import type { Conversation } from '@/types'

const props = withDefaults(
  defineProps<{
    conversations: Conversation[]
    currentUserId?: number
    presence?: Record<number, string>
    loading?: boolean
    error?: string
    forbidden?: boolean
    activeId?: number
  }>(),
  {
    currentUserId: undefined,
    presence: () => ({}),
    loading: false,
    error: '',
    forbidden: false,
    activeId: undefined,
  },
)

const emit = defineEmits<{ select: [id: number]; new: []; retry: [] }>()

const ordered = computed(() =>
  [...(props.conversations || [])].sort((a, b) =>
    String(b.last_message?.created_at || b.created_at || '').localeCompare(
      String(a.last_message?.created_at || a.created_at || ''),
    ),
  ),
)

function labelOf(conversation: Conversation): string {
  return conversationLabel(conversation, props.currentUserId)
}

function avatarOf(conversation: Conversation) {
  return conversationMembers(conversation, props.currentUserId)[0]
}

function statusOf(conversation: Conversation): string {
  const member = avatarOf(conversation)
  return member ? props.presence[member.id] || 'offline' : 'offline'
}
</script>

<template>
  <section class="conversations" aria-labelledby="conversations-title">
    <header class="conversations__head">
      <h2 id="conversations-title" class="conversations__title">Conversations</h2>
      <button type="button" class="conversations__add" aria-label="Start a conversation" title="Start a conversation" @click="emit('new')">
        <ChatIcon name="plus" :size="18" />
      </button>
    </header>

    <SkeletonList v-if="loading" :rows="4" avatar />

    <div v-else-if="forbidden || error" class="conversations__empty" role="alert">
      <p>{{ error || 'Conversations are not available. Your session may have expired.' }}</p>
      <button type="button" class="conversations__retry" @click="emit('retry')">Try again</button>
    </div>

    <p v-else-if="!conversations.length" class="conversations__empty">
      No conversations yet. Use the plus button for a direct message or a small group.
    </p>

    <ul v-else class="conversations__list">
      <li v-for="conversation in ordered" :key="conversation.id">
        <button
          type="button"
          class="conversations__item"
          :class="{ 'conversations__item--active': conversation.id === activeId }"
          :aria-current="conversation.id === activeId ? 'true' : undefined"
          @click="emit('select', conversation.id)"
        >
          <span class="conversations__avatar">
            <Avatar
              v-if="avatarOf(conversation)"
              :name="avatarOf(conversation)?.display_name || avatarOf(conversation)?.username || 'Member'"
              :src="avatarOf(conversation)?.avatar_url"
              size="sm"
            />
            <span v-else class="conversations__group"><ChatIcon name="group" :size="16" /></span>
            <PresenceDot v-if="conversation.type === 'direct'" class="conversations__presence" :status="statusOf(conversation)" />
          </span>
          <span class="conversations__info">
            <span class="conversations__name">{{ labelOf(conversation) }}</span>
            <span class="conversations__preview">
              {{ conversation.last_message?.deleted ? 'Message removed' : conversation.last_message?.body || 'No messages yet' }}
            </span>
          </span>
          <span v-if="conversation.unread_count" class="conversations__unread mono" :aria-label="`${conversation.unread_count} unread`">
            {{ conversation.unread_count }}
          </span>
          <time v-else-if="conversation.last_message?.created_at" class="conversations__time" :datetime="conversation.last_message.created_at">
            {{ formatWhen(conversation.last_message.created_at) }}
          </time>
        </button>
      </li>
    </ul>
  </section>
</template>

<style scoped>
.conversations {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  min-height: 0;
}

.conversations__empty {
  padding: var(--space-3) var(--space-2);
  color: var(--text-secondary);
  font-size: var(--text-sm);
  line-height: 1.5;
}

.conversations__retry {
  display: inline-flex;
  min-height: 44px;
  align-items: center;
  padding: 0;
  border: 0;
  background: none;
  color: var(--tide);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
}

.conversations__retry:hover {
  color: var(--text-primary);
}

.conversations__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
}

.conversations__title {
  font-size: var(--text-lg);
}

.conversations__add {
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

.conversations__add:hover {
  color: var(--text-primary);
}

.conversations__list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  overflow-y: auto;
}

.conversations__item {
  display: flex;
  width: 100%;
  min-height: 56px;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-3);
  border: 0;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--text-primary);
  text-align: left;
  cursor: pointer;
}

.conversations__item:hover {
  background: var(--ink-800);
}

.conversations__item--active {
  background: var(--ink-700);
}

.conversations__avatar {
  position: relative;
  display: inline-flex;
  flex: none;
}

.conversations__group {
  display: inline-flex;
  width: 28px;
  height: 28px;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-pill);
  background: var(--ink-700);
  color: var(--text-secondary);
}

.conversations__presence {
  position: absolute;
  right: -2px;
  bottom: -2px;
  box-shadow: 0 0 0 2px var(--ink-900);
}

.conversations__info {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
}

.conversations__name {
  overflow: hidden;
  font-size: var(--text-sm);
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.conversations__preview {
  overflow: hidden;
  color: var(--text-muted);
  font-size: var(--text-xs);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.conversations__unread {
  min-width: 20px;
  padding: 0 var(--space-1);
  border-radius: var(--radius-pill);
  background: var(--ember);
  color: var(--ink-900);
  font-size: var(--text-xs);
  text-align: center;
}

.conversations__time {
  color: var(--text-muted);
  font-size: var(--text-xs);
}
</style>
