<script setup lang="ts">
import { onMounted, ref } from 'vue'
import Avatar from '@/components/ui/Avatar.vue'
import { listCreators } from '@/api/creators'
import { listEvents } from '@/api/events'
import type { Creator, Event } from '@/types'

const events = ref<Event[]>([])
const creators = ref<Creator[]>([])
const eventsLoading = ref(true)
const creatorsLoading = ref(true)

function eventDate(value: string): string {
  return new Intl.DateTimeFormat(undefined, { day: 'numeric', month: 'short' }).format(new Date(value))
}

function eventPlace(event: Event): string {
  if (event.city) return event.city
  return event.is_online ? 'Online' : 'Location to be announced'
}

function creatorName(creator: Creator): string {
  return creator.user?.display_name || creator.user?.username || 'Creator'
}

async function loadEvents(): Promise<void> {
  try {
    events.value = (await listEvents({ limit: 3, from: new Date().toISOString() })).items
  } catch {
    // The rail is secondary: a failed section disappears rather than shouting.
    events.value = []
  } finally {
    eventsLoading.value = false
  }
}

async function loadCreators(): Promise<void> {
  try {
    creators.value = (await listCreators({ limit: 3 })).items
  } catch {
    creators.value = []
  } finally {
    creatorsLoading.value = false
  }
}

onMounted(() => {
  void loadEvents()
  void loadCreators()
})
</script>

<template>
  <div class="rail">
    <section class="rail__section" aria-labelledby="rail-events-title">
      <h2 id="rail-events-title" class="rail__title">Upcoming events</h2>

      <div v-if="eventsLoading" class="rail__loading" aria-hidden="true">
        <span class="skeleton-bar" />
        <span class="skeleton-bar" />
        <span class="skeleton-bar" />
      </div>

      <ul v-else-if="events.length" class="rail__list">
        <li v-for="event in events" :key="event.id">
          <RouterLink class="rail__row" :to="`/events/${event.id}`">
            <span class="rail__date mono">{{ eventDate(event.starts_at) }}</span>
            <span class="rail__text">
              <span class="rail__name">{{ event.title }}</span>
              <span class="rail__meta">{{ eventPlace(event) }}</span>
            </span>
          </RouterLink>
        </li>
      </ul>

      <p v-else class="rail__empty">No upcoming events yet.</p>

      <RouterLink class="rail__more" to="/events">Browse events</RouterLink>
    </section>

    <section class="rail__section" aria-labelledby="rail-creators-title">
      <h2 id="rail-creators-title" class="rail__title">Creators</h2>

      <div v-if="creatorsLoading" class="rail__loading" aria-hidden="true">
        <span class="skeleton-bar" />
        <span class="skeleton-bar" />
      </div>

      <ul v-else-if="creators.length" class="rail__list">
        <li v-for="creator in creators" :key="creator.user?.id">
          <RouterLink class="rail__row" :to="`/u/${creator.user?.username}`">
            <Avatar :name="creatorName(creator)" :src="creator.user?.avatar_url" size="sm" />
            <span class="rail__text">
              <span class="rail__name">{{ creatorName(creator) }}</span>
              <span class="rail__meta">@{{ creator.user?.username }}</span>
            </span>
          </RouterLink>
        </li>
      </ul>

      <p v-else class="rail__empty">No creators listed yet.</p>

      <RouterLink class="rail__more" to="/creators">Browse creators</RouterLink>
    </section>
  </div>
</template>

<style scoped>
.rail {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.rail__section {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.rail__section + .rail__section {
  padding-top: var(--space-5);
  border-top: 1px solid var(--border-hairline);
}

.rail__title {
  margin-bottom: var(--space-1);
  font-size: var(--text-sm);
  font-weight: 600;
  letter-spacing: 0;
  color: var(--text-primary);
}

.rail__list {
  display: flex;
  flex-direction: column;
}

.rail__row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  min-height: 44px;
  padding-block: var(--space-2);
  color: inherit;
  text-decoration: none;
  border-radius: var(--radius-sm);
}

.rail__row:hover .rail__name {
  color: var(--tide);
}

.rail__date {
  flex: none;
  min-width: 52px;
  color: var(--text-secondary);
  font-size: var(--text-xs);
  letter-spacing: 0.01em;
}

.rail__text {
  display: flex;
  min-width: 0;
  flex-direction: column;
}

.rail__name {
  overflow: hidden;
  color: var(--text-primary);
  font-size: var(--text-sm);
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
  transition: color var(--dur-fast) var(--ease);
}

.rail__meta {
  overflow: hidden;
  color: var(--text-secondary);
  font-size: var(--text-xs);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.rail__empty {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.rail__loading {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding-block: var(--space-2);
}

.rail__loading .skeleton-bar {
  width: 100%;
}

.rail__loading .skeleton-bar:nth-child(2) {
  width: 72%;
}

.rail__more {
  display: inline-flex;
  align-items: center;
  align-self: flex-start;
  min-height: 44px;
  color: var(--tide);
  font-size: var(--text-sm);
  font-weight: 500;
  text-decoration: none;
}

.rail__more:hover {
  color: var(--text-primary);
}
</style>
