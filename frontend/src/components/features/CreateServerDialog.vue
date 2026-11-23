<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseInput from '@/components/ui/BaseInput.vue'
import ChatIcon from './ChatIcon.vue'
import { createServer } from '@/api/chat'
import { errorMessage } from './format'
import type { ChatServer } from '@/types'

const emit = defineEmits<{
  close: []
  created: [server: ChatServer]
}>()

const name = ref('')
const description = ref('')
const submitting = ref(false)
const error = ref('')
const validation = ref('')

function onKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape') emit('close')
}

onMounted(() => window.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))

async function submit(): Promise<void> {
  if (!name.value.trim()) {
    validation.value = 'Give your server a name.'
    return
  }
  submitting.value = true
  validation.value = ''
  error.value = ''
  try {
    const server = await createServer({ name: name.value.trim(), description: description.value.trim() })
    emit('created', server)
    emit('close')
  } catch (cause) {
    error.value = errorMessage(cause, 'We could not create this server. Try again.')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="backdrop" @click.self="emit('close')">
    <section class="dialog" role="dialog" aria-modal="true" aria-labelledby="create-server-title">
      <header class="dialog__head">
        <h2 id="create-server-title" class="dialog__title">Create a server</h2>
        <button type="button" class="dialog__close" aria-label="Close" @click="emit('close')">
          <ChatIcon name="close" :size="18" />
        </button>
      </header>

      <p class="dialog__intro">A server is a place for your community's channels.</p>

      <form class="dialog__form" @submit.prevent="submit">
        <BaseInput v-model="name" label="Server name" placeholder="e.g. Convention after hours" :error="validation" />

        <label class="dialog__label" for="server-description">Description</label>
        <textarea
          id="server-description"
          v-model="description"
          class="dialog__control"
          rows="3"
          maxlength="500"
          placeholder="What brings this community together?"
        />

        <p v-if="error" class="dialog__error" role="alert">{{ error }}</p>

        <footer class="dialog__actions">
          <BaseButton type="button" variant="ghost" @click="emit('close')">Cancel</BaseButton>
          <BaseButton type="submit" variant="primary" :loading="submitting" :disabled="!name.trim()">
            Create server
          </BaseButton>
        </footer>
      </form>
    </section>
  </div>
</template>

<style scoped>
.backdrop {
  position: fixed;
  inset: 0;
  z-index: 60;
  display: grid;
  overflow-y: auto;
  padding: var(--space-4);
  place-items: center;
  background: rgba(0, 0, 0, 0.68);
}

.dialog {
  width: min(100%, 480px);
  padding: var(--space-6);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-lg);
  background: var(--ink-800);
  box-shadow: var(--shadow-2);
}

.dialog__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}

.dialog__title {
  font-size: var(--text-xl);
}

.dialog__close {
  display: inline-flex;
  width: 44px;
  height: 44px;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
}

.dialog__intro {
  margin-top: var(--space-2);
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.dialog__form {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  margin-top: var(--space-5);
}

.dialog__label {
  color: var(--text-primary);
  font-size: var(--text-sm);
  font-weight: 500;
}

.dialog__control {
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

.dialog__error {
  color: var(--alert);
  font-size: var(--text-sm);
}

.dialog__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-2);
}

@media (max-width: 480px) {
  .dialog {
    padding: var(--space-4);
  }
}
</style>
