<script setup lang="ts">
import { computed, ref } from 'vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import { cancelRsvp, rsvpEvent } from '@/api/events'
import type { Event, RsvpStatus } from '@/types'
import { errorMessage } from './format'
import { formatCapacity } from './formatR2'

const props = withDefaults(defineProps<{ event: Event; disabled?: boolean }>(), { disabled: false })

const emit = defineEmits<{
  change: [status: RsvpStatus | null]
  updated: [event: Event]
}>()

const busy = ref(false)
const pending = ref<RsvpStatus | null>(null)
const error = ref('')

const current = computed<RsvpStatus | null>(() => {
  const status = props.event.rsvp_status
  return status === 'going' || status === 'maybe' ? status : null
})
const capacityNote = computed(() => formatCapacity(props.event.rsvp_count, props.event.capacity))
const isFull = computed(() => props.event.capacity > 0 && props.event.rsvp_count >= props.event.capacity)
const note = computed(() => {
  if (current.value === 'going') return "You're going. Cancel any time."
  if (current.value === 'maybe') return "You're on the maybe list."
  if (isFull.value) return 'This event is at capacity.'
  return 'Let the organizer know you are interested.'
})

async function respond(status: RsvpStatus): Promise<void> {
  if (busy.value || props.disabled || current.value === status) return
  busy.value = true
  pending.value = status
  error.value = ''
  try {
    const updated = await rsvpEvent(props.event.id, status)
    emit('change', status)
    emit('updated', updated)
  } catch (cause) {
    error.value = errorMessage(cause, 'We could not save your response. Try again.')
  } finally {
    busy.value = false
    pending.value = null
  }
}

async function cancel(): Promise<void> {
  if (busy.value || props.disabled || !current.value) return
  busy.value = true
  pending.value = null
  error.value = ''
  try {
    const updated = await cancelRsvp(props.event.id)
    emit('change', null)
    emit('updated', updated)
  } catch (cause) {
    error.value = errorMessage(cause, 'We could not cancel your response. Try again.')
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <section class="rsvp" aria-labelledby="rsvp-title">
    <div class="rsvp__head">
      <h2 id="rsvp-title" class="rsvp__title">Your response</h2>
      <p class="rsvp__capacity mono">{{ capacityNote }}</p>
    </div>

    <div class="rsvp__actions">
      <BaseButton
        variant="primary"
        data-status="going"
        :aria-pressed="current === 'going'"
        :disabled="disabled"
        :loading="busy && pending === 'going'"
        @click="respond('going')"
      >
        {{ current === 'going' ? 'You are going' : 'Going' }}
      </BaseButton>
      <BaseButton
        variant="secondary"
        data-status="maybe"
        :aria-pressed="current === 'maybe'"
        :disabled="disabled"
        :loading="busy && pending === 'maybe'"
        @click="respond('maybe')"
      >
        {{ current === 'maybe' ? 'You might go' : 'Maybe' }}
      </BaseButton>
      <BaseButton v-if="current" variant="ghost" data-status="cancel" :disabled="disabled || busy" @click="cancel">
        Cancel RSVP
      </BaseButton>
    </div>

    <p class="rsvp__note">{{ note }}</p>
    <p v-if="error" class="rsvp__error" role="alert">{{ error }}</p>
  </section>
</template>

<style scoped>
.rsvp {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding: var(--space-5);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-lg);
  background: var(--ink-800);
}

.rsvp__head {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  justify-content: space-between;
  gap: var(--space-2);
}

.rsvp__title {
  font-size: var(--text-lg);
}

.rsvp__capacity {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.rsvp__actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.rsvp__note {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.rsvp__error {
  color: var(--alert);
  font-size: var(--text-sm);
}
</style>
