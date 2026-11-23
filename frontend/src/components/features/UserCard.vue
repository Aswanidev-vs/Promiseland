<script setup lang="ts">
import { RouterLink } from 'vue-router'
import Avatar from '@/components/ui/Avatar.vue'
import BaseCard from '@/components/ui/BaseCard.vue'
import type { UserSummary } from '@/types'

defineProps<{ user: UserSummary }>()
</script>

<template>
  <BaseCard interactive class="user-card">
    <RouterLink class="user-card__link" :to="`/u/${user.username}`">
      <Avatar :name="user.display_name || user.username" :src="user.avatar_url" />
      <span class="user-card__info">
        <span class="user-card__name">{{ user.display_name || user.username }}</span>
        <span class="user-card__handle">@{{ user.username }}</span>
      </span>
    </RouterLink>
  </BaseCard>
</template>

<style scoped>
.user-card__link {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  color: inherit;
  text-decoration: none;
}

.user-card__link:hover .user-card__name {
  color: var(--tide);
}

.user-card__info {
  display: flex;
  min-width: 0;
  flex-direction: column;
}

.user-card__name {
  overflow: hidden;
  color: var(--text-primary);
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-card__handle {
  color: var(--text-secondary);
  font-family: var(--font-mono);
  font-size: var(--text-sm);
}
</style>
