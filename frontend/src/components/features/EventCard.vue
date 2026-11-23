<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import Avatar from '@/components/ui/Avatar.vue'
import BaseCard from '@/components/ui/BaseCard.vue'
import TagChip from '@/components/ui/TagChip.vue'
import WayfinderMark from '@/components/ui/WayfinderMark.vue'
import type { Event } from '@/types'
import { plural } from './format'
import { eventDateParts, eventTypeLabel, formatCapacity } from './formatR2'

const props = defineProps<{ event: Event }>()

const parts = computed(() => eventDateParts(props.event.starts_at, props.event.ends_at))
const typeLabel = computed(() => eventTypeLabel(props.event.event_type))
const place = computed(() => {
  if (props.event.is_online) return 'Online event'
  return [props.event.venue_text, props.event.city, props.event.country].filter(Boolean).join(', ') || 'Location to be confirmed'
})
const capacityNote = computed(() => formatCapacity(props.event.rsvp_count, props.event.capacity))
const isFull = computed(() => props.event.capacity > 0 && props.event.rsvp_count >= props.event.capacity)
const fill = computed(() =>
  props.event.capacity > 0
    ? Math.min(100, Math.round((props.event.rsvp_count / props.event.capacity) * 100))
    : 0,
)
const statusNote = computed(() => {
  if (props.event.rsvp_status === 'going') return 'Going'
  if (props.event.rsvp_status === 'maybe') return 'Maybe'
  return ''
})
</script>

<template>
  <BaseCard interactive class="event-card" :padded="false">
    <div class="event-card__media">
      <img v-if="event.cover_url" class="event-card__cover" :src="event.cover_url" :alt="event.title" loading="lazy" />
      <div v-else class="event-card__placeholder" aria-hidden="true">
        <WayfinderMark detailed size="92px" />
      </div>
      <span class="event-card__badge"><TagChip :label="typeLabel" tone="ember" /></span>
    </div>

    <RouterLink class="event-card__body" :to="`/events/${event.id}`">
      <div class="event-card__date" :title="parts.full">
        <span class="event-card__month mono">{{ parts.month }}</span>
        <span class="event-card__day">{{ parts.day }}</span>
        <span class="event-card__weekday mono">{{ parts.weekday }}</span>
      </div>

      <div class="event-card__info">
        <h2 class="event-card__title">{{ event.title }}</h2>
        <p class="event-card__when mono">{{ parts.full }} · {{ parts.time }}</p>
        <p class="event-card__place">{{ place }}</p>
        <div class="event-card__foot">
          <Avatar
            :name="event.organizer?.display_name || event.organizer?.username || 'Member'"
            :src="event.organizer?.avatar_url"
            size="sm"
          />
          <span class="event-card__organizer">{{ event.organizer?.display_name || event.organizer?.username || 'Member' }}</span>
          <TagChip v-if="statusNote" :label="statusNote" tone="tide" />
          <TagChip v-if="isFull && !statusNote" label="At capacity" />
        </div>
      </div>
    </RouterLink>

    <div class="event-card__capacity">
      <div class="event-card__capacity-line">
        <span class="event-card__count mono">{{ capacityNote }}</span>
        <span class="event-card__count mono">{{ plural(event.rsvp_count, 'response') }}</span>
      </div>
      <div
        v-if="event.capacity > 0"
        class="event-card__bar"
        role="progressbar"
        :aria-valuenow="event.rsvp_count"
        :aria-valuemin="0"
        :aria-valuemax="event.capacity"
        :aria-label="capacityNote"
      >
        <span class="event-card__fill" :style="{ width: `${fill}%` }" />
      </div>
    </div>
  </BaseCard>
</template>

<style scoped>
.event-card {
  display: flex;
  overflow: hidden;
  flex-direction: column;
}

.event-card__media {
  position: relative;
  display: flex;
  min-height: 132px;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border-bottom: 1px solid var(--border-hairline);
  background: var(--ink-850);
}

.event-card__cover {
  width: 100%;
  height: 164px;
  object-fit: cover;
}

.event-card__placeholder {
  display: flex;
  width: 100%;
  height: 132px;
  align-items: center;
  justify-content: center;
  border-bottom: 1px solid var(--border-hairline);
  background: var(--ink-900);
  opacity: 0.85;
}

.event-card__badge {
  position: absolute;
  top: var(--space-3);
  left: var(--space-3);
}

.event-card__body {
  display: flex;
  gap: var(--space-4);
  padding: var(--space-5) var(--space-5) var(--space-4);
  color: inherit;
  text-decoration: none;
}

.event-card__date {
  display: flex;
  width: 64px;
  flex: none;
  flex-direction: column;
  align-items: center;
  gap: var(--space-1);
  padding-right: var(--space-4);
  border-right: 1px dashed var(--border-strong);
}

.event-card__month {
  color: var(--text-secondary);
  font-size: var(--text-xs);
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.event-card__day {
  color: var(--text-primary);
  font-family: var(--font-display);
  font-size: var(--text-3xl);
  font-variant-numeric: tabular-nums;
  line-height: 1;
}

.event-card__weekday {
  color: var(--text-secondary);
  font-size: var(--text-xs);
}

.event-card__info {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: var(--space-2);
}

.event-card__title {
  color: var(--text-primary);
  font-size: var(--text-xl);
}

.event-card__body:hover .event-card__title {
  color: var(--tide);
}

.event-card__when {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.event-card__place {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.event-card__foot {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-2);
  margin-top: auto;
  padding-top: var(--space-2);
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.event-card__organizer {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.event-card__capacity {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-3) var(--space-5) var(--space-4);
  border-top: 1px solid var(--border-hairline);
}

.event-card__capacity-line {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  gap: var(--space-2);
}

.event-card__count {
  color: var(--text-secondary);
  font-size: var(--text-xs);
}

.event-card__bar {
  height: 4px;
  overflow: hidden;
  border-radius: var(--radius-pill);
  background: var(--ink-700);
}

.event-card__fill {
  display: block;
  height: 100%;
  border-radius: var(--radius-pill);
  background: var(--ember);
}

@media (max-width: 480px) {
  .event-card__body {
    gap: var(--space-3);
    padding-inline: var(--space-4);
  }

  .event-card__capacity {
    padding-inline: var(--space-4);
  }

  .event-card__date {
    width: 52px;
    padding-right: var(--space-3);
  }
}
</style>
