<script setup lang="ts">
/** One mutual match. No location, contact details or compatibility score. */
import { computed } from 'vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseCard from '@/components/ui/BaseCard.vue'
import type { AdultMatch, UserSummary } from '@/types'

const props = withDefaults(
  defineProps<{
    match: AdultMatch
    currentUserId?: number | null
    busy?: boolean
  }>(),
  { currentUserId: null, busy: false },
)

const emit = defineEmits<{ unmatch: [id: number]; report: [user: UserSummary] }>()

const otherUser = computed<UserSummary>(
  () => props.match.users.find((user) => user.id !== props.currentUserId) ?? props.match.users[0]!,
)
</script>

<template>
  <BaseCard class="adult-match">
    <div class="adult-match__body">
      <h3 class="adult-match__name">{{ otherUser.display_name || otherUser.username }}</h3>
      <p class="adult-match__handle">@{{ otherUser.username }}</p>
      <p class="adult-match__note">You both chose to connect. Messaging is available only while the match is active.</p>
    </div>
    <div class="adult-match__actions">
      <BaseButton size="sm" variant="secondary" :disabled="busy" @click="emit('unmatch', match.id)">
        Unmatch
      </BaseButton>
      <BaseButton
        size="sm"
        variant="ghost"
        class="adult-match__report"
        :disabled="busy"
        @click="emit('report', otherUser)"
      >
        Report
      </BaseButton>
    </div>
  </BaseCard>
</template>

<style scoped>
.adult-match {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-4);
}

.adult-match__body {
  flex: 1;
  min-width: 0;
}

.adult-match__name {
  overflow-wrap: anywhere;
  font-family: var(--font-display);
  font-size: var(--text-lg);
}

.adult-match__handle {
  margin-top: var(--space-1);
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.adult-match__note {
  margin-top: var(--space-3);
  max-width: 56ch;
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.adult-match__actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.adult-match :deep(.adult-match__report.btn--ghost) {
  color: var(--alert);
}
</style>
