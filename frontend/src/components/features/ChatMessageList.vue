<script setup lang="ts">
import { computed } from 'vue'
import Avatar from '@/components/ui/Avatar.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import SkeletonList from '@/components/ui/SkeletonList.vue'
import ChatIcon from './ChatIcon.vue'
import MessageActions from './MessageActions.vue'
import PresenceDot from './PresenceDot.vue'
import { groupMessages } from './chatMessageUtils'
import { formatWhen } from './format'
import type { ChatMessage } from '@/types'

const props = withDefaults(
  defineProps<{
    messages: ChatMessage[]
    currentUserId?: number
    presence?: Record<number, string>
    loading?: boolean
    error?: string
    forbidden?: boolean
  }>(),
  {
    currentUserId: undefined,
    presence: () => ({}),
    loading: false,
    error: '',
    forbidden: false,
  },
)

const emit = defineEmits<{
  react: [message: ChatMessage, code: string]
  reply: [message: ChatMessage]
  report: [message: ChatMessage]
  delete: [message: ChatMessage]
  focusComposer: []
  retry: []
}>()

const groups = computed(() => groupMessages(props.messages || []))

function nameOf(message: ChatMessage): string {
  return message.author?.display_name || message.author?.username || 'Member'
}

function isOwned(message: ChatMessage): boolean {
  return !!props.currentUserId && message.author?.id === props.currentUserId
}
</script>

<template>
  <div class="messages" role="log" aria-live="polite" aria-relevant="additions" aria-label="Messages">
    <SkeletonList v-if="loading" :rows="4" avatar />

    <EmptyState
      v-else-if="forbidden"
      title="You do not have access to these messages"
      description="Ask a server owner or moderator for access."
      action-label="Back to chat"
      @action="emit('focusComposer')"
    />

    <EmptyState
      v-else-if="error"
      title="We could not load messages"
      :description="error"
      action-label="Try again"
      @action="emit('retry')"
    />

    <EmptyState
      v-else-if="!messages.length"
      title="No messages yet"
      description="This is the start of the conversation. Say something to get it going."
      action-label="Write the first message"
      @action="emit('focusComposer')"
    >
      <template #default>
        <span class="messages__hint">Messages sent here are visible to everyone with access.</span>
      </template>
    </EmptyState>

    <template v-else>
      <section v-for="(group, index) in groups" :key="group.key" class="messages__group">
        <h3 v-if="index === 0 || groups[index - 1]?.day !== group.day" class="messages__day">
          <span class="messages__day-rule" aria-hidden="true" />
          <span>{{ group.day }}</span>
          <span class="messages__day-rule" aria-hidden="true" />
        </h3>

        <article
          v-for="message in group.messages"
          :key="message.id"
          class="message"
          :class="{ 'message--reply': !!message.parent_id, 'message--mine': isOwned(message) }"
        >
          <Avatar :name="nameOf(message)" :src="message.author?.avatar_url" size="sm" />

          <div class="message__body">
            <header class="message__head">
              <span class="message__author">{{ nameOf(message) }}</span>
              <PresenceDot :status="presence[message.author?.id ?? -1]" />
              <span v-if="message.parent_id" class="message__reply-tag">
                <ChatIcon name="reply" :size="12" />
                Reply
              </span>
              <time v-if="message.created_at" :datetime="message.created_at" class="message__time">
                {{ formatWhen(message.created_at) }}
              </time>
              <span v-if="message.edited_at" class="message__edited">edited</span>
            </header>

            <p v-if="message.deleted" class="message__deleted">This message was removed.</p>
            <p v-else class="message__text">{{ message.body }}</p>

            <MessageActions
              v-if="!message.deleted"
              :message="message"
              :is-owned="isOwned(message)"
              @react="(code) => emit('react', message, code)"
              @reply="emit('reply', message)"
              @report="emit('report', message)"
              @delete="emit('delete', message)"
            />
          </div>
        </article>
      </section>
    </template>
  </div>
</template>

<style scoped>
.messages {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
  gap: var(--space-5);
  overflow-y: auto;
  overscroll-behavior: contain;
  padding: var(--space-5) var(--space-5) var(--space-3);
}

.messages__hint {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.messages__group {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.messages__day {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  margin-bottom: var(--space-2);
  color: var(--text-secondary);
  font-family: var(--font-body);
  font-size: var(--text-xs);
  font-weight: 500;
  letter-spacing: 0;
}

.messages__day-rule {
  flex: 1;
  height: 1px;
  background: var(--border-hairline);
}

.message {
  display: flex;
  gap: var(--space-3);
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-md);
}

.message--mine {
  background: color-mix(in srgb, var(--ember) 4%, transparent);
}

.message--reply {
  margin-left: var(--space-8);
  padding-left: var(--space-4);
  border-left: 2px solid var(--border-strong);
}

.message__body {
  min-width: 0;
  flex: 1;
}

.message__head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-2);
}

.message__author {
  color: var(--text-primary);
  font-size: var(--text-sm);
  font-weight: 600;
}

.message__reply-tag {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  color: var(--tide);
  font-size: var(--text-xs);
}

.message__time,
.message__edited {
  color: var(--text-secondary);
  font-size: var(--text-xs);
}

.message__text {
  overflow-wrap: anywhere;
  font-family: var(--font-body);
  font-size: var(--text-base);
  line-height: var(--leading-body);
  white-space: pre-wrap;
}

.message__deleted {
  color: var(--text-secondary);
  font-size: var(--text-base);
  font-style: italic;
}

@media (max-width: 480px) {
  .messages {
    padding-inline: var(--space-3);
  }

  .message--reply {
    margin-left: var(--space-4);
    padding-left: var(--space-3);
  }
}
</style>
