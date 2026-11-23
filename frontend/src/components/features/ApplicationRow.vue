<script setup lang="ts">
import { RouterLink } from 'vue-router'
import Avatar from '@/components/ui/Avatar.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import TagChip from '@/components/ui/TagChip.vue'
import type { TeamUpApplication } from '@/types'
import { formatWhen } from './format'
import { sentenceCase } from './formatR2'

defineProps<{ application: TeamUpApplication; busy?: boolean }>()
const emit = defineEmits<{ review: [status: 'accepted' | 'rejected'] }>()
</script>

<template>
  <li class="application">
    <Avatar
      :name="application.applicant?.display_name || application.applicant?.username || 'Member'"
      :src="application.applicant?.avatar_url"
      size="sm"
    />
    <div class="application__body">
      <div class="application__head">
        <RouterLink class="application__name" :to="`/u/${application.applicant?.username || ''}`">
          {{ application.applicant?.display_name || application.applicant?.username || 'Member' }}
        </RouterLink>
        <time v-if="application.created_at" :datetime="application.created_at">
          {{ formatWhen(application.created_at) }}
        </time>
        <TagChip :label="sentenceCase(application.status)" />
      </div>
      <p class="application__message">{{ application.message || 'No message was added.' }}</p>
      <div v-if="application.status === 'pending'" class="application__actions">
        <BaseButton size="sm" :loading="busy" @click="emit('review', 'accepted')">Accept</BaseButton>
        <BaseButton size="sm" variant="ghost" :disabled="busy" @click="emit('review', 'rejected')">
          Decline
        </BaseButton>
      </div>
    </div>
  </li>
</template>

<style scoped>
.application {
  display: flex;
  gap: var(--space-3);
  padding: var(--space-4);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-md);
  background: var(--ink-850);
}

.application__body {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: var(--space-2);
}

.application__head {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: var(--space-2);
}

.application__name {
  color: var(--text-primary);
  font-weight: 600;
  text-decoration: none;
}

.application__head time {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.application__message {
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}

.application__actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}
</style>
