<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Avatar from '@/components/ui/Avatar.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseCard from '@/components/ui/BaseCard.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import SkeletonList from '@/components/ui/SkeletonList.vue'
import TagChip from '@/components/ui/TagChip.vue'
import WayfinderMark from '@/components/ui/WayfinderMark.vue'
import EventCommentThread from '@/components/features/EventCommentThread.vue'
import RsvpControl from '@/components/features/RsvpControl.vue'
import TeamUpCard from '@/components/features/TeamUpCard.vue'
import {
  addEventUpdate,
  deleteEvent,
  getEvent,
  listEventComments,
  listEventUpdates,
  updateEvent,
} from '@/api/events'
import { listTeamUps } from '@/api/teamups'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import type { Event, EventComment, EventUpdate, TeamUp } from '@/types'
import { errorMessage, formatWhen, isForbidden, isNotFound } from '@/components/features/format'
import {
  eventDateParts,
  eventTypeLabel,
  formatCapacity,
  fromRfc3339,
  sentenceCase,
  toRfc3339,
} from '@/components/features/formatR2'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const ui = useUiStore()
const id = computed(() => Number(route.params.id))

const event = ref<Event | null>(null)
const updates = ref<EventUpdate[]>([])
const comments = ref<EventComment[]>([])
const teamUps = ref<TeamUp[]>([])
const loading = ref(true)
const loadingComments = ref(false)
const busy = ref(false)
const error = ref('')
const notFound = ref(false)
const forbidden = ref(false)
const commentsError = ref('')

const updateDraft = ref('')
const updateError = ref('')
const editing = ref(false)
const confirmDelete = ref(false)
const edit = ref({
  title: '',
  description: '',
  startsAt: '',
  endsAt: '',
  venueText: '',
  city: '',
  country: '',
  capacity: '',
})
const fieldErrors = ref<Record<string, string>>({})

const isOrganizer = computed(
  () => !!event.value && !!auth.user && event.value.organizer?.id === auth.user.id,
)
const parts = computed(() =>
  event.value ? eventDateParts(event.value.starts_at, event.value.ends_at) : null,
)
const capacityNote = computed(() =>
  event.value ? formatCapacity(event.value.rsvp_count, event.value.capacity) : '',
)
const place = computed(() => {
  if (!event.value) return ''
  if (event.value.is_online) return 'Online event'
  return [event.value.venue_text, event.value.city, event.value.country].filter(Boolean).join(', ') || 'Location to be confirmed'
})

function startEdit(): void {
  if (!event.value) return
  edit.value = {
    title: event.value.title,
    description: event.value.description,
    startsAt: fromRfc3339(event.value.starts_at),
    endsAt: fromRfc3339(event.value.ends_at),
    venueText: event.value.venue_text,
    city: event.value.city,
    country: event.value.country,
    capacity: event.value.capacity > 0 ? String(event.value.capacity) : '',
  }
  fieldErrors.value = {}
  editing.value = true
}

async function loadUpdates(): Promise<void> {
  try {
    updates.value = await listEventUpdates(id.value)
  } catch {
    updates.value = []
  }
}

async function loadComments(): Promise<void> {
  loadingComments.value = true
  commentsError.value = ''
  try {
    const page = await listEventComments(id.value, { limit: 50 })
    comments.value = page.items
  } catch {
    comments.value = []
    commentsError.value = 'We could not load comments. Try again.'
  } finally {
    loadingComments.value = false
  }
}

async function loadLinkedTeamUps(): Promise<void> {
  try {
    const page = await listTeamUps({ limit: 100 } as Parameters<typeof listTeamUps>[0])
    teamUps.value = page.items.filter((item) => item.event?.id === id.value).slice(0, 3)
  } catch {
    teamUps.value = []
  }
}

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  notFound.value = false
  forbidden.value = false
  editing.value = false
  try {
    event.value = await getEvent(id.value)
    await Promise.all([loadUpdates(), loadComments(), loadLinkedTeamUps()])
  } catch (cause) {
    if (isForbidden(cause)) forbidden.value = true
    else if (isNotFound(cause)) notFound.value = true
    else error.value = errorMessage(cause, 'We could not load this event. Try again.')
  } finally {
    loading.value = false
  }
}

async function publishUpdate(): Promise<void> {
  if (!updateDraft.value.trim() || busy.value) return
  busy.value = true
  updateError.value = ''
  try {
    const created = await addEventUpdate(id.value, updateDraft.value.trim())
    updates.value = [created, ...updates.value]
    updateDraft.value = ''
    ui.pushToast({ title: 'Update published', tone: 'success' })
  } catch (cause) {
    updateError.value = errorMessage(cause, 'We could not publish your update. Try again.')
  } finally {
    busy.value = false
  }
}

async function saveEdit(): Promise<void> {
  if (!event.value || busy.value) return
  const next: Record<string, string> = {}
  if (!edit.value.title.trim()) next.title = 'Add a title.'
  if (!edit.value.startsAt || !edit.value.endsAt) next.endsAt = 'Add both a start and an end time.'
  else if (new Date(edit.value.endsAt) <= new Date(edit.value.startsAt)) {
    next.endsAt = 'The end time must be after the start time.'
  }
  if (edit.value.capacity && (!Number.isInteger(Number(edit.value.capacity)) || Number(edit.value.capacity) < 0)) {
    next.capacity = 'Use a whole number of 0 or more.'
  }
  fieldErrors.value = next
  if (Object.keys(next).length) return

  busy.value = true
  try {
    const updated = await updateEvent(event.value.id, {
      title: edit.value.title.trim(),
      description: edit.value.description.trim(),
      starts_at: toRfc3339(edit.value.startsAt),
      ends_at: toRfc3339(edit.value.endsAt),
      venue_text: edit.value.venueText.trim(),
      city: edit.value.city.trim(),
      country: edit.value.country.trim(),
      capacity: edit.value.capacity ? Number(edit.value.capacity) : 0,
    } as unknown as Parameters<typeof updateEvent>[1])
    event.value = updated
    editing.value = false
    ui.pushToast({ title: 'Event updated', tone: 'success' })
  } catch (cause) {
    ui.pushToast({ title: 'Event not updated', message: errorMessage(cause), tone: 'error' })
  } finally {
    busy.value = false
  }
}

async function cancelEvent(): Promise<void> {
  if (!event.value || busy.value) return
  busy.value = true
  try {
    await deleteEvent(event.value.id)
    ui.pushToast({ title: 'Event cancelled', tone: 'success' })
    await router.push('/events')
  } catch (cause) {
    ui.pushToast({ title: 'Event not cancelled', message: errorMessage(cause), tone: 'error' })
    confirmDelete.value = false
  } finally {
    busy.value = false
  }
}

onMounted(load)
watch(id, load)
</script>

<template>
  <div class="event-view">
    <BaseButton variant="ghost" @click="router.back()">Back</BaseButton>

    <SkeletonList v-if="loading" :rows="5" />

    <EmptyState
      v-else-if="notFound"
      title="This event is not available"
      description="It may have been cancelled, or the link may be wrong."
      action-label="Browse events"
      @action="router.push('/events')"
    />

    <EmptyState
      v-else-if="forbidden"
      title="You do not have access to this event"
      description="Check the event's visibility or sign in again."
      action-label="Browse events"
      @action="router.push('/events')"
    />

    <EmptyState
      v-else-if="error || !event || !parts"
      title="We could not load this event"
      :description="error || 'The event is unavailable right now.'"
      action-label="Try again"
      @action="load"
    />

    <template v-else>
      <!-- The moment: a harbour ticket with the date as the hero, not the chrome. -->
      <article class="plaque">
        <div class="plaque__media">
          <img v-if="event.cover_url" :src="event.cover_url" :alt="event.title" />
          <WayfinderMark v-else detailed size="200px" />
        </div>

        <div class="plaque__main">
          <div class="plaque__date">
            <span class="plaque__month mono">{{ parts.month }}</span>
            <span class="plaque__day">{{ parts.day }}</span>
            <span class="plaque__weekday mono">{{ parts.weekday }}</span>
            <span class="plaque__time mono">{{ parts.time }}</span>
          </div>

          <div class="plaque__info">
            <div class="plaque__tags">
              <TagChip :label="eventTypeLabel(event.event_type)" tone="ember" />
              <TagChip v-if="event.is_online" label="Online" tone="tide" />
              <TagChip v-if="event.status && event.status !== 'scheduled'" :label="sentenceCase(event.status)" />
            </div>
            <h1 class="plaque__title">{{ event.title }}</h1>
            <p v-if="event.description" class="plaque__description u-measure">{{ event.description }}</p>

            <dl class="plaque__details">
              <div>
                <dt>When</dt>
                <dd class="plaque__when">
                  <span class="mono">{{ parts.time }}</span>
                  <span class="plaque__zone mono">{{ event.timezone }}</span>
                </dd>
              </div>
              <div>
                <dt>Where</dt>
                <dd>{{ place }}</dd>
              </div>
              <div>
                <dt>Organizer</dt>
                <dd>
                  <RouterLink class="plaque__organizer" :to="`/u/${event.organizer?.username || ''}`">
                    {{ event.organizer?.display_name || event.organizer?.username || 'Member' }}
                  </RouterLink>
                </dd>
              </div>
              <div>
                <dt>Responses</dt>
                <dd class="mono plaque__capacity">{{ capacityNote }}</dd>
              </div>
            </dl>
          </div>
        </div>

        <div class="plaque__stub">
          <WayfinderMark size="36px" />
          <span class="plaque__stub-text mono">Harbour gathering</span>
          <span class="plaque__stub-id mono">#{{ event.id }}</span>
        </div>
      </article>

      <div class="event-view__columns">
        <div class="event-view__main">
          <RsvpControl
            v-if="!isOrganizer"
            :event="event"
            @updated="event = $event"
          />

          <BaseCard v-else class="organizer">
            <h2 class="organizer__title">Organizer tools</h2>

            <template v-if="!editing">
              <div class="organizer__actions">
                <BaseButton size="sm" @click="startEdit">Edit details</BaseButton>
                <BaseButton
                  v-if="!confirmDelete"
                  size="sm"
                  variant="ghost"
                  @click="confirmDelete = true"
                >
                  Cancel event
                </BaseButton>
                <BaseButton
                  v-else
                  size="sm"
                  variant="danger"
                  :loading="busy"
                  @click="cancelEvent"
                >
                  Confirm cancellation
                </BaseButton>
              </div>

              <form class="organizer__update" @submit.prevent="publishUpdate">
                <label class="organizer__label" for="event-update">Post an update</label>
                <textarea
                  id="event-update"
                  v-model="updateDraft"
                  class="organizer__control"
                  rows="3"
                  maxlength="2000"
                  placeholder="Share a change or a reminder with attendees."
                />
                <div class="organizer__update-actions">
                  <BaseButton type="submit" variant="primary" size="sm" :loading="busy" :disabled="!updateDraft.trim()">
                    Publish update
                  </BaseButton>
                </div>
                <p v-if="updateError" class="organizer__error" role="alert">{{ updateError }}</p>
              </form>
            </template>

            <form v-else class="organizer__edit" @submit.prevent="saveEdit">
              <h3 class="organizer__edit-title">Edit details</h3>
              <BaseInput v-model="edit.title" label="Title" :error="fieldErrors.title" />
              <div class="organizer__field">
                <label class="organizer__label" for="edit-event-description">Description</label>
                <textarea id="edit-event-description" v-model="edit.description" class="organizer__control" rows="5" />
              </div>
              <div class="organizer__grid">
                <div class="organizer__field">
                  <label class="organizer__label" for="edit-event-starts">Starts</label>
                  <input id="edit-event-starts" v-model="edit.startsAt" class="organizer__control" type="datetime-local" />
                </div>
                <div class="organizer__field">
                  <label class="organizer__label" for="edit-event-ends">Ends</label>
                  <input id="edit-event-ends" v-model="edit.endsAt" class="organizer__control" type="datetime-local" />
                  <p v-if="fieldErrors.endsAt" class="organizer__error" role="alert">{{ fieldErrors.endsAt }}</p>
                </div>
              </div>
              <div class="organizer__grid">
                <BaseInput v-model="edit.venueText" label="Venue" />
                <BaseInput v-model="edit.city" label="City" />
              </div>
              <BaseInput v-model="edit.country" label="Country" />
              <BaseInput v-model="edit.capacity" label="Capacity" type="number" min="0" :error="fieldErrors.capacity" />
              <div class="organizer__edit-actions">
                <BaseButton type="button" size="sm" variant="ghost" @click="editing = false">Discard</BaseButton>
                <BaseButton type="submit" size="sm" :loading="busy">Save changes</BaseButton>
              </div>
            </form>
          </BaseCard>

          <section class="event-view__section" aria-labelledby="updates-title">
            <h2 id="updates-title" class="event-view__section-title">Updates</h2>
            <p v-if="!updates.length" class="event-view__note">No updates yet.</p>
            <ol v-else class="updates">
              <li v-for="update in updates" :key="update.id" class="update">
                <Avatar
                  :name="update.author?.display_name || update.author?.username || 'Member'"
                  :src="update.author?.avatar_url"
                  size="sm"
                />
                <div class="update__body">
                  <div class="update__head">
                    <span>{{ update.author?.display_name || update.author?.username || 'Member' }}</span>
                    <time :datetime="update.created_at">{{ formatWhen(update.created_at) }}</time>
                  </div>
                  <p class="update__text">{{ update.body }}</p>
                </div>
              </li>
            </ol>
          </section>

          <section v-if="teamUps.length" class="event-view__section" aria-labelledby="event-teamups-title">
            <h2 id="event-teamups-title" class="event-view__section-title">Team ups for this event</h2>
            <ul class="event-view__teamups">
              <li v-for="teamUp in teamUps" :key="teamUp.id"><TeamUpCard :team-up="teamUp" /></li>
            </ul>
          </section>

          <section class="event-view__section" aria-labelledby="event-comments-title">
            <h2 id="event-comments-title" class="sr-only">Event discussion</h2>
            <SkeletonList v-if="loadingComments" :rows="2" />
            <template v-else>
              <p v-if="commentsError" class="event-view__error" role="alert">
                {{ commentsError }}
                <button type="button" @click="loadComments">Retry</button>
              </p>
              <EventCommentThread v-else :comments="comments" :event-id="event.id" @refresh="loadComments" />
            </template>
          </section>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.event-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

/* ── The date treatment: a harbour ticket, quiet chrome around a rich centre ── */
.plaque {
  display: flex;
  overflow: hidden;
  flex-direction: column;
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-lg);
  background: var(--ink-800);
  box-shadow: var(--shadow-1);
}

.plaque__media {
  display: flex;
  min-height: 160px;
  max-height: 240px;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border-bottom: 1px dashed var(--border-strong);
  background: var(--ink-900);
}

.plaque__media img {
  width: 100%;
  max-height: 240px;
  object-fit: cover;
}

.plaque__main {
  display: flex;
  gap: var(--space-5);
  padding: var(--space-6);
}

.plaque__date {
  display: flex;
  width: 108px;
  flex: none;
  flex-direction: column;
  align-items: center;
  gap: var(--space-1);
  padding-right: var(--space-5);
  border-right: 1px dashed var(--border-strong);
}

.plaque__weekday {
  color: var(--text-secondary);
  font-size: var(--text-xs);
  letter-spacing: 0.08em;
}

.plaque__day {
  color: var(--text-primary);
  font-family: var(--font-display);
  font-size: var(--text-4xl);
  font-weight: 560;
  font-variant-numeric: tabular-nums;
  line-height: 1;
}

.plaque__month {
  color: var(--text-secondary);
}

.plaque__time {
  margin-top: var(--space-2);
  color: var(--text-secondary);
  font-size: var(--text-xs);
  text-align: center;
}

.plaque__info {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: var(--space-3);
}

.plaque__tags {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.plaque__title {
  font-size: var(--text-4xl);
}

.plaque__description {
  color: var(--text-secondary);
  white-space: pre-wrap;
}

.plaque__details {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--space-4);
  margin: var(--space-2) 0 0;
  padding-top: var(--space-4);
  border-top: 1px solid var(--border-hairline);
}

.plaque__details dt {
  color: var(--text-secondary);
  font-size: var(--text-xs);
  letter-spacing: 0.04em;
}

.plaque__details dd {
  margin: var(--space-1) 0 0;
  color: var(--text-primary);
  font-size: var(--text-sm);
}

.plaque__zone {
  color: var(--text-secondary);
  font-size: var(--text-xs);
}

.plaque__when {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  color: var(--text-primary);
}

.plaque__capacity {
  font-variant-numeric: tabular-nums;
}

.plaque__organizer {
  color: var(--tide);
}

.plaque__stub {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-6);
  border-top: 1px dashed var(--border-strong);
  background: var(--ink-850);
  color: var(--text-secondary);
}

.plaque__stub-text {
  font-size: var(--text-xs);
  letter-spacing: 0.08em;
}

.plaque__stub-id {
  margin-left: auto;
  font-size: var(--text-xs);
}

.event-view__columns {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
}

.event-view__main {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.event-view__section {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.event-view__section-title {
  font-size: var(--text-xl);
}

.event-view__note {
  color: var(--text-secondary);
}

.event-view__error {
  color: var(--alert);
  font-size: var(--text-sm);
}

.event-view__error button {
  padding: 0;
  border: 0;
  background: none;
  color: var(--alert);
  text-decoration: underline;
  cursor: pointer;
}

.updates {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.update {
  display: flex;
  gap: var(--space-3);
}

.update__body {
  min-width: 0;
  flex: 1;
}

.update__head {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  color: var(--text-primary);
  font-size: var(--text-sm);
  font-weight: 500;
}

.update__head time {
  color: var(--text-secondary);
  font-weight: 400;
}

.update__text {
  margin-top: var(--space-1);
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}

.event-view__teamups {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.organizer {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.organizer__title {
  font-size: var(--text-lg);
}

.organizer__actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  padding-top: var(--space-4);
  border-top: 1px solid var(--border-hairline);
}

.organizer__update,
.organizer__edit {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding-top: var(--space-4);
  border-top: 1px solid var(--border-hairline);
}

.organizer__edit-title {
  font-size: var(--text-lg);
}

.organizer__field {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.organizer__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--space-3);
  align-items: start;
}

.organizer__label {
  color: var(--text-primary);
  font-size: var(--text-sm);
  font-weight: 500;
}

.organizer__control {
  width: 100%;
  min-height: 44px;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-md);
  background: var(--ink-850);
  color: var(--text-primary);
  font-family: var(--font-body);
  line-height: var(--leading-body);
  resize: vertical;
  transition:
    border-color var(--dur-fast) var(--ease),
    box-shadow var(--dur-fast) var(--ease);
}

.organizer__control:hover:not(:disabled) {
  border-color: var(--border-strong);
}

.organizer__control:focus {
  border-color: var(--ember);
  box-shadow: 0 0 0 3px var(--ember-soft);
  outline: none;
}

.organizer__update-actions,
.organizer__edit-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}

.organizer__error {
  color: var(--alert);
  font-size: var(--text-sm);
}

@media (max-width: 767px) {
  .plaque__main {
    flex-direction: column;
    gap: var(--space-4);
    padding: var(--space-4);
  }

  .plaque__date {
    width: 100%;
    flex-direction: row;
    align-items: baseline;
    justify-content: flex-start;
    gap: var(--space-2);
    padding: 0 0 var(--space-3);
    border-right: 0;
    border-bottom: 1px dashed var(--border-strong);
  }

  .plaque__time {
    margin-top: 0;
    margin-left: auto;
  }

  .plaque__details {
    grid-template-columns: minmax(0, 1fr);
  }

  .organizer__grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
