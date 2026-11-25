<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import Avatar from '@/components/ui/Avatar.vue'
import BaseCard from '@/components/ui/BaseCard.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import CompatibilityBadge from './CompatibilityBadge.vue'
import type { DiscoveryCandidate } from '@/types'

const props = withDefaults(
  defineProps<{
    candidate: DiscoveryCandidate
    busy?: boolean
  }>(),
  { busy: false },
)

const emit = defineEmits<{ connect: []; notInterested: [] }>()

const status = computed(() => props.candidate.connection_status ?? 'none')

const statusNote = computed(() => {
  if (status.value === 'pending_out') return 'Request sent'
  if (status.value === 'pending_in') return 'Wants to connect'
  if (status.value === 'accepted') return 'Connected'
  return ''
})

const statusTone = computed(() => (status.value === 'accepted' ? 'ok' : 'muted'))
</script>

<template>
  <BaseCard class="discovery-card">
    <div class="discovery-card__head">
      <div class="discovery-card__identity">
        <Avatar
          :name="candidate.user.display_name || candidate.user.username"
          :src="candidate.user.avatar_url"
        />
        <span class="discovery-card__name">
          {{ candidate.user.display_name || candidate.user.username }}
        </span>
        <span class="discovery-card__handle">@{{ candidate.user.username }}</span>
      </div>
      <span v-if="statusNote" class="discovery-card__status" :class="`discovery-card__status--${statusTone}`">
        {{ statusNote }}
      </span>
    </div>

    <CompatibilityBadge
      :explanation="candidate.explanation"
      :shared="candidate.shared"
      :shared-count="candidate.shared_count"
    />

    <div class="discovery-card__actions">
      <BaseButton
        v-if="status === 'none'"
        variant="primary"
        size="sm"
        :disabled="busy"
        @click="emit('connect')"
      >
        Connect
      </BaseButton>
      <RouterLink
        v-else-if="status === 'pending_in'"
        class="discovery-card__link"
        to="/connections"
      >
        Review request
      </RouterLink>
      <span v-else-if="status === 'accepted'" class="discovery-card__connected">
        You're connected
      </span>

      <BaseButton variant="secondary" size="sm" :to="`/u/${candidate.user.username}`">
        View profile
      </BaseButton>
      <BaseButton
        v-if="status !== 'accepted'"
        variant="ghost"
        size="sm"
        :disabled="busy"
        @click="emit('notInterested')"
      >
        Not interested
      </BaseButton>
    </div>
  </BaseCard>
</template>

<style scoped>
.discovery-card {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.discovery-card__identity {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  min-width: 0;
}

.discovery-card__name {
  overflow: hidden;
  color: var(--text-primary);
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.discovery-card__handle {
  color: var(--text-secondary);
  font-size: var(--text-xs);
}

.discovery-card__head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2) var(--space-4);
}

.discovery-card__status {
  flex: none;
  font-size: var(--text-xs);
  font-weight: 500;
}

.discovery-card__status--muted {
  color: var(--text-secondary);
}

.discovery-card__status--ok {
  color: var(--ok);
}

.discovery-card__actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-2);
}

.discovery-card__link {
  font-size: var(--text-sm);
  font-weight: 500;
}

.discovery-card__connected {
  color: var(--ok);
  font-size: var(--text-sm);
}
</style>
