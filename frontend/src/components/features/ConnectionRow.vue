<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import AppIcon from '@/components/ui/AppIcon.vue'
import type { IconName } from '@/components/ui/icons'
import Avatar from '@/components/ui/Avatar.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import type { Connection } from '@/types'
import { formatWhen } from './format'

const props = withDefaults(
  defineProps<{
    connection: Connection
    currentUserId: number
    direction?: 'incoming' | 'outgoing' | 'all'
    busy?: boolean
  }>(),
  { direction: 'all', busy: false },
)

const emit = defineEmits<{ accept: []; reject: []; ignore: []; cancel: [] }>()

const other = computed(() =>
  props.connection.requester.id === props.currentUserId
    ? props.connection.addressee
    : props.connection.requester,
)

const perspective = computed(() => {
  if (props.direction !== 'all') return props.direction
  if (!props.currentUserId) return 'unknown'
  return props.connection.requester.id === props.currentUserId ? 'outgoing' : 'incoming'
})

const isPending = computed(() => props.connection.status === 'pending')
const isIncoming = computed(() => isPending.value && perspective.value === 'incoming')
const isOutgoing = computed(() => isPending.value && perspective.value === 'outgoing')

const statusNote = computed(() => {
  const status = props.connection.status
  if (status === 'accepted') return 'Connected'
  if (status === 'rejected') return 'Declined'
  if (status === 'ignored') return 'Ignored'
  if (isIncoming.value) return 'Wants to connect'
  if (isOutgoing.value) return 'Request sent'
  return ''
})

const statusIcon = computed((): IconName => {
  const status = props.connection.status
  if (status === 'accepted') return 'eye'
  if (status === 'rejected' || status === 'ignored') return 'close'
  if (isIncoming.value || isOutgoing.value) return 'bell'
  return 'eye'
})

const statusTone = computed(() =>
  props.connection.status === 'accepted' ? 'ok' : 'muted',
)

const relativeTime = computed(() => formatWhen(props.connection.created_at))

const absoluteTime = computed(() => {
  const date = new Date(props.connection.created_at)
  return Number.isNaN(date.getTime()) ? '' : date.toLocaleString()
})
</script>

<template>
  <article class="connection-row">
    <div class="connection-row__head">
      <Avatar
        :name="other.display_name || other.username"
        :src="other.avatar_url"
        size="sm"
      />
      <div class="connection-row__identity">
        <p class="connection-row__name">{{ other.display_name || other.username }}</p>
        <p class="connection-row__handle mono">@{{ other.username }}</p>
      </div>
      <time
        v-if="relativeTime"
        class="connection-row__time mono"
        :datetime="connection.created_at"
        :title="absoluteTime"
      >
        {{ relativeTime }}
      </time>
    </div>

    <blockquote v-if="connection.message" class="connection-row__message">
      {{ connection.message }}
    </blockquote>

    <div class="connection-row__footer">
      <span
        v-if="statusNote"
        class="connection-row__status"
        :class="`connection-row__status--${statusTone}`"
      >
        <AppIcon :name="statusIcon" :size="16" />
        <span class="connection-row__status-text">{{ statusNote }}</span>
      </span>
      <div v-if="isIncoming || isOutgoing || connection.status === 'accepted'" class="connection-row__actions">
        <template v-if="isIncoming">
          <BaseButton variant="primary" size="sm" :disabled="busy" @click="emit('accept')">
            Accept
          </BaseButton>
          <BaseButton variant="ghost" size="sm" :disabled="busy" @click="emit('reject')">
            Reject
          </BaseButton>
          <BaseButton variant="ghost" size="sm" :disabled="busy" @click="emit('ignore')">
            Ignore
          </BaseButton>
        </template>
        <template v-else-if="isOutgoing">
          <BaseButton variant="secondary" size="sm" :disabled="busy" @click="emit('cancel')">
            Cancel request
          </BaseButton>
        </template>
        <RouterLink
          v-else
          class="connection-row__message-link"
          :to="{ path: '/chat', query: { user: String(other.id) } }"
        >
          Message
        </RouterLink>
      </div>
    </div>
  </article>
</template>

<style scoped>
.connection-row {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  border-top: 1px solid var(--border-hairline);
}

.connection-row:first-child {
  border-top: 0;
}

.connection-row__head {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  min-width: 0;
}

.connection-row__identity {
  flex: 1;
  min-width: 0;
}

.connection-row__name {
  overflow: hidden;
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.connection-row__handle {
  color: var(--text-secondary);
  font-size: var(--text-xs);
}

.connection-row__time {
  flex: none;
  color: var(--text-secondary);
  font-size: var(--text-xs);
  white-space: nowrap;
}

.connection-row__message {
  margin: 0;
  padding: var(--space-2) var(--space-3);
  border-left: 2px solid var(--ink-600);
  color: var(--text-secondary);
  font-size: var(--text-sm);
  line-height: 1.5;
  overflow-wrap: anywhere;
}

.connection-row__footer {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2) var(--space-4);
}

.connection-row__status {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  flex: none;
  font-size: var(--text-xs);
  font-weight: 500;
}

.connection-row__status--muted {
  color: var(--text-secondary);
}

.connection-row__status--ok {
  color: var(--ok);
}

.connection-row__actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-2);
  margin-left: auto;
}

.connection-row__message-link {
  font-size: var(--text-sm);
  font-weight: 500;
}

@media (max-width: 480px) {
  .connection-row__actions {
    width: 100%;
    margin-left: 0;
  }
}
</style>
