<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref } from 'vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import ChatIcon from './ChatIcon.vue'

const props = withDefaults(
  defineProps<{
    disabled?: boolean
    busy?: boolean
    placeholder?: string
    replyingTo?: string | null
  }>(),
  {
    disabled: false,
    busy: false,
    placeholder: 'Write a message',
    replyingTo: null,
  },
)

const emit = defineEmits<{
  send: [body: string]
  typing: [isTyping: boolean]
  cancelReply: []
}>()

const draft = ref('')
const textarea = ref<HTMLTextAreaElement | null>(null)
let idleTimer: ReturnType<typeof setTimeout> | null = null
let typingActive = false

function stopTyping(): void {
  if (idleTimer) {
    clearTimeout(idleTimer)
    idleTimer = null
  }
  if (typingActive) {
    typingActive = false
    emit('typing', false)
  }
}

function markTyping(): void {
  if (!typingActive) {
    typingActive = true
    emit('typing', true)
  }
  if (idleTimer) clearTimeout(idleTimer)
  idleTimer = setTimeout(stopTyping, 1500)
}

function submit(): void {
  const body = draft.value.trim()
  if (!body || props.disabled || props.busy) return
  draft.value = ''
  stopTyping()
  emit('send', body)
}

function insertNewline(): void {
  const element = textarea.value
  const fallback = draft.value.length
  const start = element?.selectionStart ?? fallback
  const end = element?.selectionEnd ?? fallback
  draft.value = `${draft.value.slice(0, start)}\n${draft.value.slice(end)}`
  markTyping()
  void nextTick(() => {
    if (textarea.value) {
      textarea.value.selectionStart = start + 1
      textarea.value.selectionEnd = start + 1
    }
  })
}

function onKeydown(event: KeyboardEvent): void {
  if (event.key !== 'Enter' || event.isComposing) return
  if (event.shiftKey) {
    event.preventDefault()
    insertNewline()
    return
  }
  event.preventDefault()
  submit()
}

function focus(): void {
  textarea.value?.focus()
}

defineExpose({ focus })

onBeforeUnmount(stopTyping)
</script>

<template>
  <div class="composer">
    <div v-if="replyingTo" class="composer__reply">
      <ChatIcon name="reply" :size="14" />
      <span>Replying to {{ replyingTo }}</span>
      <button type="button" class="composer__cancel" aria-label="Cancel reply" @click="emit('cancelReply')">
        <ChatIcon name="close" :size="14" />
      </button>
    </div>

    <label class="sr-only" for="chat-composer-input">Message</label>
    <textarea
      id="chat-composer-input"
      ref="textarea"
      v-model="draft"
      class="composer__input"
      rows="2"
      maxlength="4000"
      :placeholder="placeholder"
      :disabled="disabled || busy"
      @input="markTyping"
      @keydown="onKeydown"
      @blur="stopTyping"
    />

    <div class="composer__footer">
      <span class="composer__hint">Enter to send · Shift+Enter for a new line</span>
      <BaseButton variant="primary" size="sm" :disabled="!draft.trim() || disabled || busy" @click="submit">
        <ChatIcon name="send" :size="16" />
        Send
      </BaseButton>
    </div>
  </div>
</template>

<style scoped>
.composer {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-3) var(--space-5) var(--space-4);
  border-top: 1px solid var(--border-hairline);
  background: var(--ink-900);
}

.composer__reply {
  display: inline-flex;
  align-items: center;
  align-self: flex-start;
  gap: var(--space-2);
  padding: var(--space-1) var(--space-2);
  border-left: 2px solid var(--tide);
  border-radius: var(--radius-sm);
  background: var(--ink-850);
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.composer__cancel {
  display: inline-flex;
  min-width: 44px;
  min-height: 44px;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
}

.composer__input {
  width: 100%;
  min-height: 44px;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  background: var(--ink-850);
  color: var(--text-primary);
  font-family: var(--font-body);
  line-height: var(--leading-body);
  resize: vertical;
}

.composer__input:focus {
  border-color: var(--ember);
  outline: 2px solid var(--ember); outline-offset: 2px;
}

.composer__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}

.composer__hint {
  color: var(--text-secondary);
  font-size: var(--text-xs);
}

@media (max-width: 480px) {
  .composer {
    padding-inline: var(--space-3);
  }

  .composer__hint {
    display: none;
  }
}
</style>
