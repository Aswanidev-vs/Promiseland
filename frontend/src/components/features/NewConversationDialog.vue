<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseInput from '@/components/ui/BaseInput.vue'
import Avatar from '@/components/ui/Avatar.vue'
import SkeletonList from '@/components/ui/SkeletonList.vue'
import ChatIcon from './ChatIcon.vue'
import { search } from '@/api/search'
import { createConversation } from '@/api/chat'
import { errorMessage, isForbidden } from './format'
import type { Conversation, UserSummary } from '@/types'

const emit = defineEmits<{
  close: []
  created: [conversation: Conversation]
}>()

const query = ref('')
const groupName = ref('')
const results = ref<UserSummary[]>([])
const selected = ref<UserSummary[]>([])
const loading = ref(false)
const searched = ref(false)
const error = ref('')
const forbidden = ref(false)
const submitting = ref(false)
const validation = ref('')
const submitError = ref('')

function onKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape') emit('close')
}

onMounted(() => window.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))

async function runSearch(): Promise<void> {
  const term = query.value.trim()
  if (!term) {
    results.value = []
    searched.value = false
    return
  }
  loading.value = true
  searched.value = true
  error.value = ''
  forbidden.value = false
  try {
    const found = await search({ q: term, type: 'users', limit: 20 })
    results.value = found.users || []
  } catch (cause) {
    results.value = []
    if (isForbidden(cause)) forbidden.value = true
    else error.value = errorMessage(cause, 'We could not find people. Try again.')
  } finally {
    loading.value = false
  }
}

function toggle(user: UserSummary): void {
  const index = selected.value.findIndex((item) => item.id === user.id)
  if (index >= 0) selected.value.splice(index, 1)
  else selected.value.push(user)
  validation.value = ''
}

function isSelected(user: UserSummary): boolean {
  return selected.value.some((item) => item.id === user.id)
}

async function submit(): Promise<void> {
  if (!selected.value.length) {
    validation.value = 'Choose at least one person.'
    return
  }
  if (selected.value.length > 1 && !groupName.value.trim()) {
    validation.value = 'Give this group a name.'
    return
  }
  submitting.value = true
  validation.value = ''
  submitError.value = ''
  try {
    const conversation = await createConversation({
      memberIds: selected.value.map((item) => item.id),
      name: selected.value.length > 1 ? groupName.value.trim() : undefined,
    })
    emit('created', conversation)
    emit('close')
  } catch (cause) {
    submitError.value = errorMessage(cause, 'We could not start this conversation. Try again.')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="backdrop" @click.self="emit('close')">
    <section class="dialog" role="dialog" aria-modal="true" aria-labelledby="new-conversation-title">
      <header class="dialog__head">
        <h2 id="new-conversation-title" class="dialog__title">New conversation</h2>
        <button type="button" class="dialog__close" aria-label="Close" @click="emit('close')">
          <ChatIcon name="close" :size="18" />
        </button>
      </header>

      <form role="search" class="dialog__search" @submit.prevent="runSearch">
        <BaseInput v-model="query" type="search" label="Find people" placeholder="Search by name or username" />
        <BaseButton type="submit" size="sm" :loading="loading" :disabled="!query.trim()">Search</BaseButton>
      </form>

      <div v-if="selected.length" class="dialog__selected" aria-label="Selected people">
        <span v-for="user in selected" :key="user.id" class="dialog__chip">
          <Avatar :name="user.display_name || user.username" :src="user.avatar_url" size="sm" />
          {{ user.display_name || user.username }}
          <button type="button" :aria-label="`Remove ${user.display_name || user.username}`" @click="toggle(user)">
            <ChatIcon name="close" :size="12" />
          </button>
        </span>
      </div>

      <div v-if="selected.length > 1" class="dialog__name">
        <BaseInput v-model="groupName" label="Group name" placeholder="What should this group be called?" />
      </div>

      <div class="dialog__results" aria-live="polite">
        <SkeletonList v-if="loading" :rows="3" avatar />

        <p v-else-if="forbidden" class="dialog__notice" role="alert">
          Search is not available for this session. Sign in again and try.
        </p>
        <p v-else-if="error" class="dialog__notice dialog__notice--error" role="alert">
          {{ error }} <button type="button" class="dialog__retry" @click="runSearch">Try again</button>
        </p>
        <p v-else-if="!searched" class="dialog__notice">Search for people to start a direct message.</p>
        <p v-else-if="!results.length" class="dialog__notice">No people found for that name. Try a different term.</p>

        <ul v-else class="dialog__people">
          <li v-for="user in results" :key="user.id">
            <button
              type="button"
              class="dialog__person"
              :aria-pressed="isSelected(user)"
              @click="toggle(user)"
            >
              <Avatar :name="user.display_name || user.username" :src="user.avatar_url" size="sm" />
              <span class="dialog__person-info">
                <span class="dialog__person-name">{{ user.display_name || user.username }}</span>
                <span class="dialog__person-handle">@{{ user.username }}</span>
              </span>
              <ChatIcon :name="isSelected(user) ? 'check' : 'plus'" :size="18" />
            </button>
          </li>
        </ul>
      </div>

      <p v-if="validation" class="dialog__error" role="alert">{{ validation }}</p>
      <p v-else-if="submitError" class="dialog__error" role="alert">{{ submitError }}</p>

      <footer class="dialog__actions">
        <BaseButton variant="ghost" @click="emit('close')">Cancel</BaseButton>
        <BaseButton variant="primary" :loading="submitting" :disabled="!selected.length" @click="submit">
          Start conversation
        </BaseButton>
      </footer>
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
  background: var(--scrim);
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

.dialog__search {
  display: flex;
  align-items: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-5);
}

.dialog__search > :first-child {
  flex: 1;
  min-width: 0;
}

.dialog__selected,
.dialog__name {
  margin-top: var(--space-4);
}

.dialog__selected {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.dialog__chip {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-1) var(--space-2);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-pill);
  background: var(--ink-850);
  font-size: var(--text-sm);
}

.dialog__chip button {
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

.dialog__results {
  min-height: 120px;
  max-height: 280px;
  margin-top: var(--space-4);
  overflow-y: auto;
}

.dialog__notice {
  padding: var(--space-4);
  border: 1px dashed var(--border-hairline);
  border-radius: var(--radius-md);
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.dialog__notice--error {
  color: var(--alert);
}

.dialog__retry {
  min-height: 44px;
  padding: 0 var(--space-2);
  border: 0;
  background: transparent;
  color: var(--tide);
  cursor: pointer;
}

.dialog__people {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.dialog__person {
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

.dialog__person:hover {
  background: var(--ink-700);
}

.dialog__person[aria-pressed='true'] {
  border-color: var(--border-strong);
  background: var(--ink-700);
  color: var(--tide);
}

.dialog__person-info {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
}

.dialog__person-name {
  font-size: var(--text-sm);
  font-weight: 600;
}

.dialog__person-handle {
  color: var(--text-secondary);
  font-family: var(--font-mono);
  font-size: var(--text-xs);
}

.dialog__error {
  margin-top: var(--space-3);
  color: var(--alert);
  font-size: var(--text-sm);
}

.dialog__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-5);
}

@media (max-width: 480px) {
  .dialog {
    padding: var(--space-4);
  }
}
</style>
