<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseInput from '@/components/ui/BaseInput.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import CreatorCategoryPicker from '@/components/features/CreatorCategoryPicker.vue'
import { createTeamUp } from '@/api/teamups'
import { useUiStore } from '@/stores/ui'
import { errorMessage, isForbidden } from '@/components/features/format'

const route = useRoute()
const router = useRouter()
const ui = useUiStore()

const title = ref('')
const description = ref('')
const categories = ref<string[]>([])
const eventId = ref(typeof route.query.event === 'string' ? route.query.event : '')
const publishing = ref(false)
const error = ref('')
const forbidden = ref(false)
const fieldErrors = ref<Record<string, string>>({})

const canPublish = computed(() => !publishing.value)

function setCategories(value: string | string[]): void {
  categories.value = Array.isArray(value) ? value : value ? [value] : []
}

function validate(): boolean {
  const next: Record<string, string> = {}
  if (!title.value.trim()) next.title = 'Add a title.'
  if (!description.value.trim()) next.description = 'Describe who you are looking for.'
  if (!categories.value.length) next.categories = 'Choose at least one category.'
  if (eventId.value && (!Number.isInteger(Number(eventId.value)) || Number(eventId.value) <= 0)) {
    next.eventId = 'Use the numeric ID of an event.'
  }
  fieldErrors.value = next
  return Object.keys(next).length === 0
}

async function publish(): Promise<void> {
  if (!validate() || !canPublish.value) return
  publishing.value = true
  error.value = ''
  forbidden.value = false
  try {
    const created = await createTeamUp({
      title: title.value.trim(),
      description: description.value.trim(),
      categories: categories.value,
      ...(eventId.value ? { eventId: Number(eventId.value) } : {}),
    } as unknown as Parameters<typeof createTeamUp>[0])
    ui.pushToast({ title: 'Team up published', tone: 'success' })
    await router.push(`/team-ups/${created.id}`)
  } catch (cause) {
    if (isForbidden(cause)) forbidden.value = true
    else error.value = errorMessage(cause, 'We could not publish your request. Try again.')
  } finally {
    publishing.value = false
  }
}
</script>

<template>
  <div class="create-team-up">
    <h1 class="create-team-up__title">Start a team up</h1>
    <p class="create-team-up__intro">Tell people what you are putting together and who can join.</p>

    <EmptyState
      v-if="forbidden"
      title="You cannot publish requests"
      description="Your account does not have permission to create team ups."
      action-label="Go to home"
      @action="router.push('/home')"
    />

    <form v-else class="create-team-up__form" novalidate @submit.prevent="publish">
      <BaseInput v-model="title" label="Title" :error="fieldErrors.title" placeholder="What are you forming?" />

      <div class="create-team-up__field">
        <label class="create-team-up__label" for="team-up-description">Description</label>
        <textarea
          id="team-up-description"
          v-model="description"
          class="create-team-up__control"
          rows="6"
          maxlength="10000"
          placeholder="Who are you looking for, and what will you do together?"
        />
        <p v-if="fieldErrors.description" class="create-team-up__error" role="alert">{{ fieldErrors.description }}</p>
      </div>

      <div class="create-team-up__field">
        <CreatorCategoryPicker
          :model-value="categories"
          label="Categories"
          hint="Choose all that fit. Optional details like dates and location can go in the description."
          :multiple="true"
          :include-none="false"
          @update:model-value="setCategories"
        />
        <p v-if="fieldErrors.categories" class="create-team-up__error" role="alert">{{ fieldErrors.categories }}</p>
      </div>

      <BaseInput
        v-model="eventId"
        label="Event ID"
        type="number"
        min="1"
        hint="Optional. Link this request to an existing event."
        :error="fieldErrors.eventId"
      />

      <p v-if="error" class="create-team-up__error" role="alert">{{ error }}</p>

      <div class="create-team-up__actions">
        <BaseButton variant="ghost" @click="router.push('/team-ups')">Cancel</BaseButton>
        <BaseButton type="submit" variant="primary" :loading="publishing" :disabled="!canPublish">
          Publish request
        </BaseButton>
      </div>
    </form>
  </div>
</template>

<style scoped>
.create-team-up {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.create-team-up__title {
  font-size: var(--text-2xl);
}

.create-team-up__intro {
  margin-top: calc(var(--space-4) * -1);
  color: var(--text-secondary);
}

.create-team-up__form {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.create-team-up__field {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.create-team-up__label {
  color: var(--text-primary);
  font-size: var(--text-sm);
  font-weight: 500;
}

.create-team-up__control {
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

.create-team-up__error {
  color: var(--alert);
  font-size: var(--text-sm);
}

.create-team-up__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}
</style>
