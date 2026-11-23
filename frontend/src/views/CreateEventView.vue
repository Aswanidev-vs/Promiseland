<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseInput from '@/components/ui/BaseInput.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import { createEvent } from '@/api/events'
import { useUiStore } from '@/stores/ui'
import { errorMessage, isForbidden } from '@/components/features/format'
import { EVENT_TYPES, toRfc3339 } from '@/components/features/formatR2'

const router = useRouter()
const ui = useUiStore()

const title = ref('')
const description = ref('')
const eventType = ref<string>('meetup')
const startsAt = ref('')
const endsAt = ref('')
const timezone = ref(Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC')
const venueText = ref('')
const city = ref('')
const country = ref('')
const isOnline = ref(false)
const capacity = ref('')
const coverMediaId = ref('')
const publishing = ref(false)
const error = ref('')
const forbidden = ref(false)
const fieldErrors = ref<Record<string, string>>({})

const canPublish = computed(() => !publishing.value)

function validate(): boolean {
  const next: Record<string, string> = {}
  if (!title.value.trim()) next.title = 'Add a title.'
  if (!description.value.trim()) next.description = 'Add a description.'
  if (!startsAt.value) next.startsAt = 'Add a start time.'
  if (!endsAt.value) next.endsAt = 'Add an end time.'
  if (startsAt.value && endsAt.value && new Date(endsAt.value) <= new Date(startsAt.value)) {
    next.endsAt = 'The end time must be after the start time.'
  }
  if (!timezone.value.trim()) next.timezone = 'Add a timezone, for example Europe/London.'
  if (capacity.value && (!Number.isInteger(Number(capacity.value)) || Number(capacity.value) < 0)) {
    next.capacity = 'Use a whole number of 0 or more. Leave empty for no limit.'
  }
  if (coverMediaId.value && (!Number.isInteger(Number(coverMediaId.value)) || Number(coverMediaId.value) <= 0)) {
    next.coverMediaId = 'Use the numeric ID of an uploaded image.'
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
    const created = await createEvent({
      title: title.value.trim(),
      description: description.value.trim(),
      eventType: eventType.value,
      startsAt: toRfc3339(startsAt.value),
      endsAt: toRfc3339(endsAt.value),
      timezone: timezone.value.trim(),
      venueText: venueText.value.trim(),
      city: city.value.trim(),
      country: country.value.trim(),
      isOnline: isOnline.value,
      ...(capacity.value ? { capacity: Number(capacity.value) } : {}),
      ...(coverMediaId.value ? { coverMediaId: Number(coverMediaId.value) } : {}),
    } as unknown as Parameters<typeof createEvent>[0])
    ui.pushToast({ title: 'Event created', tone: 'success' })
    await router.push(`/events/${created.id}`)
  } catch (cause) {
    if (isForbidden(cause)) forbidden.value = true
    else error.value = errorMessage(cause, 'We could not create your event. Try again.')
  } finally {
    publishing.value = false
  }
}
</script>

<template>
  <div class="create-event">
    <h1 class="create-event__title">Create an event</h1>
    <p class="create-event__intro">Give fans the details they need to show up.</p>

    <EmptyState
      v-if="forbidden"
      title="You cannot create events"
      description="Your account does not have permission to publish events."
      action-label="Go to home"
      @action="router.push('/home')"
    />

    <form v-else class="create-event__form" novalidate @submit.prevent="publish">
      <BaseInput v-model="title" label="Title" :error="fieldErrors.title" placeholder="What are you gathering for?" />

      <div class="create-event__field">
        <label class="create-event__label" for="event-description">Description</label>
        <textarea
          id="event-description"
          v-model="description"
          class="create-event__control create-event__textarea"
          rows="6"
          maxlength="10000"
          placeholder="What will happen, and who is it for?"
        />
        <p v-if="fieldErrors.description" class="create-event__error" role="alert">{{ fieldErrors.description }}</p>
      </div>

      <div class="create-event__field">
        <label class="create-event__label" for="event-type">Event type</label>
        <select id="event-type" v-model="eventType" class="create-event__control">
          <option v-for="item in EVENT_TYPES" :key="item.value" :value="item.value">{{ item.label }}</option>
        </select>
      </div>

      <div class="create-event__grid">
        <div class="create-event__field">
          <label class="create-event__label" for="event-starts">Starts</label>
          <input id="event-starts" v-model="startsAt" class="create-event__control" type="datetime-local" />
          <p v-if="fieldErrors.startsAt" class="create-event__error" role="alert">{{ fieldErrors.startsAt }}</p>
        </div>
        <div class="create-event__field">
          <label class="create-event__label" for="event-ends">Ends</label>
          <input id="event-ends" v-model="endsAt" class="create-event__control" type="datetime-local" />
          <p v-if="fieldErrors.endsAt" class="create-event__error" role="alert">{{ fieldErrors.endsAt }}</p>
        </div>
      </div>

      <BaseInput
        v-model="timezone"
        label="Timezone"
        hint="An IANA timezone, such as Asia/Tokyo. Times above are entered in your local time."
        :error="fieldErrors.timezone"
      />

      <div class="create-event__grid">
        <BaseInput v-model="venueText" label="Venue" hint="Only share the details you want made public." />
        <BaseInput v-model="city" label="City" />
      </div>
      <BaseInput v-model="country" label="Country" />

      <label class="create-event__check">
        <input v-model="isOnline" type="checkbox" />
        This is an online event
      </label>

      <div class="create-event__grid">
        <BaseInput
          v-model="capacity"
          label="Capacity"
          type="number"
          min="0"
          hint="Leave empty for no limit."
          :error="fieldErrors.capacity"
        />
        <BaseInput
          v-model="coverMediaId"
          label="Cover media ID"
          type="number"
          min="1"
          hint="The ID of an uploaded image, if you have one."
          :error="fieldErrors.coverMediaId"
        />
      </div>

      <p v-if="error" class="create-event__error" role="alert">{{ error }}</p>

      <div class="create-event__actions">
        <BaseButton variant="ghost" @click="router.push('/events')">Cancel</BaseButton>
        <BaseButton type="submit" variant="primary" :loading="publishing" :disabled="!canPublish">
          Create event
        </BaseButton>
      </div>
    </form>
  </div>
</template>

<style scoped>
.create-event {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.create-event__title {
  font-size: var(--text-2xl);
}

.create-event__intro {
  margin-top: calc(var(--space-4) * -1);
  color: var(--text-secondary);
}

.create-event__form {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.create-event__field {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.create-event__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--space-3);
  align-items: start;
}

.create-event__label {
  color: var(--text-primary);
  font-size: var(--text-sm);
  font-weight: 500;
}

.create-event__control {
  width: 100%;
  min-height: 44px;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  background: var(--ink-850);
  color: var(--text-primary);
  font-family: var(--font-body);
}

.create-event__textarea {
  line-height: var(--leading-body);
  resize: vertical;
}

.create-event__check {
  display: inline-flex;
  min-height: 44px;
  align-items: center;
  gap: var(--space-2);
  color: var(--text-primary);
  cursor: pointer;
}

.create-event__check input {
  width: 18px;
  height: 18px;
  accent-color: var(--ember);
}

.create-event__error {
  color: var(--alert);
  font-size: var(--text-sm);
}

.create-event__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}

@media (max-width: 480px) {
  .create-event__grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
