<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import Avatar from '@/components/ui/Avatar.vue'
import type { UserSummary } from '@/types'

const props = defineProps<{
  user: UserSummary
  busy?: boolean
  error?: string
}>()

const emit = defineEmits<{ close: []; submit: [message: string] }>()

const LIMIT = 280
const message = ref('')
const localError = ref('')
const textarea = ref<HTMLTextAreaElement | null>(null)

function onKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape') emit('close')
}

onMounted(() => {
  window.addEventListener('keydown', onKeydown)
  textarea.value?.focus()
})
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))

function send(): void {
  const value = message.value.trim()
  if (value.length > LIMIT) {
    localError.value = `Keep your message under ${LIMIT} characters.`
    return
  }
  localError.value = ''
  emit('submit', value)
}
</script>

<template>
  <div class="connect-backdrop" @click.self="emit('close')">
    <section class="connect-dialog" role="dialog" aria-modal="true" aria-labelledby="connect-title">
      <h2 id="connect-title" class="connect-dialog__title">Send a connection request</h2>

      <div class="connect-dialog__person">
        <Avatar :name="props.user.display_name || props.user.username" :src="props.user.avatar_url" size="sm" />
        <div>
          <p class="connect-dialog__name">{{ props.user.display_name || props.user.username }}</p>
          <p class="connect-dialog__handle mono">@{{ props.user.username }}</p>
        </div>
      </div>

      <form novalidate @submit.prevent="send">
        <label class="connect-dialog__label" for="connect-message">A note with your request</label>
        <p class="connect-dialog__hint">Optional. A sentence about what you have in common goes a long way.</p>
        <textarea
          id="connect-message"
          ref="textarea"
          v-model="message"
          class="connect-dialog__control"
          rows="4"
          :maxlength="LIMIT + 20"
          placeholder="Say hello, or mention a series you both follow."
        />
        <div class="connect-dialog__meta">
          <span class="connect-dialog__count mono">{{ message.length }} / {{ LIMIT }}</span>
        </div>

        <p v-if="localError || props.error" class="connect-dialog__error" role="alert">
          {{ localError || props.error }}
        </p>

        <div class="connect-dialog__actions">
          <BaseButton type="button" variant="ghost" @click="emit('close')">Cancel</BaseButton>
          <BaseButton type="submit" variant="primary" :loading="props.busy">Send request</BaseButton>
        </div>
      </form>
    </section>
  </div>
</template>

<style scoped>
.connect-backdrop {
  position: fixed;
  inset: 0;
  z-index: 60;
  display: grid;
  overflow-y: auto;
  padding: var(--space-4);
  place-items: center;
  background: rgba(0, 0, 0, 0.68);
}

.connect-dialog {
  width: min(100%, 440px);
  padding: var(--space-6);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-lg);
  background: var(--ink-800);
  box-shadow: var(--shadow-2);
}

.connect-dialog__title {
  font-size: var(--text-xl);
}

.connect-dialog__person {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  margin-top: var(--space-4);
  padding: var(--space-3);
  border-radius: var(--radius-md);
  background: var(--ink-850);
}

.connect-dialog__name {
  font-weight: 500;
}

.connect-dialog__handle {
  color: var(--text-secondary);
  font-size: var(--text-xs);
}

.connect-dialog__label {
  display: block;
  margin-top: var(--space-5);
  font-size: var(--text-sm);
  font-weight: 500;
}

.connect-dialog__hint {
  margin-top: var(--space-1);
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.connect-dialog__control {
  width: 100%;
  min-height: 96px;
  margin-top: var(--space-3);
  padding: var(--space-3);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  background: var(--ink-850);
  color: var(--text-primary);
  font-family: var(--font-body);
  line-height: var(--leading-body);
  resize: vertical;
}

.connect-dialog__meta {
  display: flex;
  justify-content: flex-end;
  margin-top: var(--space-1);
}

.connect-dialog__count {
  color: var(--text-muted);
  font-size: var(--text-xs);
}

.connect-dialog__error {
  margin-top: var(--space-2);
  color: var(--alert);
  font-size: var(--text-sm);
}

.connect-dialog__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-5);
}

@media (max-width: 480px) {
  .connect-dialog {
    padding: var(--space-4);
  }

  .connect-dialog__actions {
    flex-direction: column-reverse;
  }

  .connect-dialog__actions > * {
    width: 100%;
  }
}
</style>
