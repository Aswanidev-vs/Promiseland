<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseCard from '@/components/ui/BaseCard.vue'
import BaseInput from '@/components/ui/BaseInput.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import SkeletonList from '@/components/ui/SkeletonList.vue'
import TagChip from '@/components/ui/TagChip.vue'
import EventCard from '@/components/features/EventCard.vue'
import { listEvents } from '@/api/events'
import type { Event } from '@/types'
import { errorMessage, isForbidden } from '@/components/features/format'
import { EVENT_TYPES } from '@/components/features/formatR2'

const router = useRouter()

const query = ref('')
const eventType = ref('')
const from = ref('')
const to = ref('')
const city = ref('')
const country = ref('')
const online = ref(false)

const events = ref<Event[]>([])
const cursor = ref('')
const loading = ref(true)
const loadingMore = ref(false)
const error = ref('')
const forbidden = ref(false)
let timer: number | undefined

const hasFilters = computed(
  () => !!(query.value.trim() || eventType.value || from.value || to.value || city.value.trim() || country.value.trim() || online.value),
)

function clearFilters(): void {
  query.value = ''
  eventType.value = ''
  from.value = ''
  to.value = ''
  city.value = ''
  country.value = ''
  online.value = false
  void load(true)
}

async function load(reset = false): Promise<void> {
  if (reset) {
    events.value = []
    cursor.value = ''
    error.value = ''
    forbidden.value = false
  }
  const more = !reset && cursor.value !== ''
  if (more) loadingMore.value = true
  else loading.value = true
  try {
    const page = await listEvents({
      ...(query.value.trim() ? { q: query.value.trim() } : {}),
      ...(eventType.value ? { type: eventType.value } : {}),
      ...(from.value ? { from: from.value } : {}),
      ...(to.value ? { to: to.value } : {}),
      ...(city.value.trim() ? { city: city.value.trim() } : {}),
      ...(country.value.trim() ? { country: country.value.trim() } : {}),
      ...(online.value ? { online: true } : {}),
      limit: 20,
      ...(more ? { cursor: cursor.value } : {}),
    } as Parameters<typeof listEvents>[0])
    events.value = more ? [...events.value, ...page.items] : page.items
    cursor.value = page.next_cursor || ''
  } catch (cause) {
    if (isForbidden(cause)) forbidden.value = true
    else error.value = errorMessage(cause, 'We could not load events. Try again.')
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

function scheduleSearch(): void {
  if (timer !== undefined) window.clearTimeout(timer)
  timer = window.setTimeout(() => void load(true), 300)
}

function setType(value: string): void {
  eventType.value = value
  void load(true)
}

onMounted(() => void load(true))
onBeforeUnmount(() => {
  if (timer !== undefined) window.clearTimeout(timer)
})
</script>

<template>
  <div class="events">
    <header class="events__header">
      <div>
        <h1>Events</h1>
        <p class="events__intro">Find your people, then meet them in the harbour.</p>
      </div>
      <BaseButton variant="primary" to="/events/new">Create event</BaseButton>
    </header>

    <BaseCard class="events__filters">
      <form @submit.prevent="load(true)">
        <BaseInput
          v-model="query"
          type="search"
          label="Search events"
          placeholder="Search by name or description"
          @update:model-value="scheduleSearch"
        />

        <fieldset class="events__types">
          <legend class="events__label">Event type</legend>
          <div class="events__chips">
            <button
              type="button"
              class="events__chip"
              :aria-pressed="!eventType"
              @click="setType('')"
            >
              <TagChip label="All types" :selected="!eventType" />
            </button>
            <button
              v-for="item in EVENT_TYPES"
              :key="item.value"
              type="button"
              class="events__chip"
              :aria-pressed="eventType === item.value"
              @click="setType(item.value)"
            >
              <TagChip :label="item.label" :selected="eventType === item.value" />
            </button>
          </div>
        </fieldset>

        <div class="events__dates">
          <div class="events__field">
            <label class="events__label" for="event-from">From</label>
            <input id="event-from" v-model="from" class="events__control" type="date" @change="load(true)" />
          </div>
          <div class="events__field">
            <label class="events__label" for="event-to">To</label>
            <input id="event-to" v-model="to" class="events__control" type="date" @change="load(true)" />
          </div>
        </div>

        <div class="events__places">
          <BaseInput v-model="city" label="City" placeholder="Any city" @update:model-value="scheduleSearch" />
          <BaseInput v-model="country" label="Country" placeholder="Any country" @update:model-value="scheduleSearch" />
        </div>

        <div class="events__filter-foot">
          <label class="events__check">
            <input v-model="online" type="checkbox" @change="load(true)" />
            Online events only
          </label>
          <div class="events__filter-actions">
            <BaseButton v-if="hasFilters" type="button" size="sm" variant="ghost" @click="clearFilters">
              Clear filters
            </BaseButton>
            <BaseButton type="submit" size="sm">Apply filters</BaseButton>
          </div>
        </div>
      </form>
    </BaseCard>

    <SkeletonList v-if="loading" :rows="4" />

    <EmptyState
      v-else-if="forbidden"
      title="Events are not available"
      description="You may need to sign in again to browse events."
      action-label="Go to home"
      @action="router.push('/home')"
    />

    <EmptyState
      v-else-if="error"
      title="We could not load events"
      :description="error"
      action-label="Try again"
      @action="load(true)"
    />

    <EmptyState
      v-else-if="!events.length"
      :title="hasFilters ? 'No events match these filters' : 'No events yet'"
      :description="hasFilters ? 'Try a wider date range or clear your filters.' : 'Be the first to bring fans together.'"
      :action-label="hasFilters ? 'Clear filters' : 'Create an event'"
      @action="hasFilters ? clearFilters() : router.push('/events/new')"
    />

    <template v-else>
      <ul class="events__list">
        <li v-for="event in events" :key="event.id"><EventCard :event="event" /></li>
      </ul>
      <div v-if="cursor" class="events__more">
        <BaseButton :loading="loadingMore" @click="load()">Load more</BaseButton>
      </div>
      <p v-else class="events__end">That is every event for now.</p>
    </template>
  </div>
</template>

<style scoped>
.events {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.events__header {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}

.events__header h1 {
  font-size: var(--text-4xl);
}

.events__intro {
  margin-top: var(--space-1);
  color: var(--text-secondary);
}

.events__filters form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.events__types {
  margin: 0;
  padding: 0;
  border: 0;
}

.events__label {
  display: block;
  margin-bottom: var(--space-2);
  color: var(--text-primary);
  font-size: var(--text-sm);
  font-weight: 500;
}

.events__chips {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.events__chip {
  min-height: 44px;
  padding: 0;
  border: 0;
  background: none;
  cursor: pointer;
}

.events__chip:focus-within {
  outline: 2px solid var(--ember);
  outline-offset: 2px;
  border-radius: var(--radius-sm);
}

.events__dates,
.events__places {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--space-3);
}

.events__field {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.events__control {
  width: 100%;
  min-height: 44px;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-md);
  background: var(--ink-850);
  color: var(--text-primary);
  transition:
    border-color var(--dur-fast) var(--ease),
    box-shadow var(--dur-fast) var(--ease);
}

.events__control:hover:not(:disabled) {
  border-color: var(--border-strong);
}

.events__control:focus {
  border-color: var(--ember);
  box-shadow: 0 0 0 3px var(--ember-soft);
  outline: none;
}

/* BaseInputs on this screen follow the shared input recipe (ember focus ring + halo). */
.events .field__input {
  border-color: var(--border-hairline);
  transition:
    border-color var(--dur-fast) var(--ease),
    box-shadow var(--dur-fast) var(--ease);
}

.events .field__input:hover:not(:disabled) {
  border-color: var(--border-strong);
}

.events .field__input:focus {
  border-color: var(--ember);
  box-shadow: 0 0 0 3px var(--ember-soft);
  outline: none;
}

.events__filter-foot {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding-top: var(--space-2);
  border-top: 1px solid var(--border-hairline);
}

.events__check {
  display: inline-flex;
  min-height: 44px;
  align-items: center;
  gap: var(--space-2);
  color: var(--text-primary);
  cursor: pointer;
}

.events__check input {
  width: 18px;
  height: 18px;
  accent-color: var(--ember);
}

.events__filter-actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.events__list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(100%, 22rem), 1fr));
  gap: var(--space-5);
}

.events__more {
  display: flex;
  justify-content: center;
}

.events__end {
  color: var(--text-secondary);
  font-size: var(--text-sm);
  text-align: center;
}

@media (max-width: 480px) {
  .events__dates,
  .events__places {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
