<script setup lang="ts">
import { computed } from 'vue'
import ChatIcon, { type ChatIconName } from './ChatIcon.vue'
import type { ChatMessage } from '@/types'

const props = withDefaults(
  defineProps<{
    message: ChatMessage
    isOwned?: boolean
    busy?: boolean
  }>(),
  { isOwned: false, busy: false },
)

const emit = defineEmits<{
  react: [code: string]
  reply: []
  report: []
  delete: []
}>()

const reactions: { code: string; label: string; icon: ChatIconName }[] = [
  { code: 'like', label: 'Like', icon: 'like' },
  { code: 'heart', label: 'Heart', icon: 'heart' },
  { code: 'star', label: 'Star', icon: 'star' },
  { code: 'laugh', label: 'Laugh', icon: 'laugh' },
]

function reactionFor(code: string) {
  return (props.message.reactions || []).find((item) => item.reaction === code)
}

const hasReactions = computed(() => (props.message.reactions || []).some((item) => item.count > 0))
</script>

<template>
  <div class="actions" :class="{ 'actions--with-reactions': hasReactions }">
    <ul class="actions__reactions" aria-label="Message reactions">
      <li v-for="item in reactions" :key="item.code">
        <button
          type="button"
          class="actions__button actions__reaction"
          :class="{ 'actions__reaction--on': reactionFor(item.code)?.me }"
          :aria-label="`${item.label} reaction`"
          :aria-pressed="!!reactionFor(item.code)?.me"
          :disabled="busy"
          @click="emit('react', item.code)"
        >
          <ChatIcon :name="item.icon" :size="16" />
          <span v-if="reactionFor(item.code)?.count" class="actions__count mono">{{ reactionFor(item.code)?.count }}</span>
        </button>
      </li>
    </ul>

    <span class="actions__divider" aria-hidden="true" />

    <button type="button" class="actions__button" aria-label="Reply to message" :disabled="busy" @click="emit('reply')">
      <ChatIcon name="reply" :size="16" />
      <span class="actions__label">Reply</span>
    </button>

    <button type="button" class="actions__button" aria-label="Report message" :disabled="busy" @click="emit('report')">
      <ChatIcon name="report" :size="16" />
      <span class="actions__label">Report</span>
    </button>

    <button
      v-if="isOwned"
      type="button"
      class="actions__button actions__button--danger"
      aria-label="Delete message"
      :disabled="busy"
      @click="emit('delete')"
    >
      <ChatIcon name="trash" :size="16" />
      <span class="actions__label">Delete</span>
    </button>
  </div>
</template>

<style scoped>
.actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-1);
  margin-top: var(--space-1);
}

.actions__reactions {
  display: flex;
  align-items: center;
  gap: 2px;
}

.actions__button {
  display: inline-flex;
  min-width: 44px;
  min-height: 44px;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  padding: 0 var(--space-2);
  border: 0;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  transition: background-color var(--dur-fast) var(--ease), color var(--dur-fast) var(--ease);
}

.actions__button:hover:not(:disabled) {
  background: var(--ink-700);
  color: var(--text-primary);
}

.actions__button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.actions__reaction--on {
  color: var(--tide);
}

.actions__count {
  font-size: var(--text-xs);
}

.actions__label {
  font-size: var(--text-sm);
}

.actions__divider {
  width: 1px;
  height: 20px;
  margin: 0 var(--space-1);
  background: var(--border-hairline);
}

.actions__button--danger {
  color: var(--alert);
}

@media (max-width: 480px) {
  .actions__label {
    display: none;
  }
}
</style>
