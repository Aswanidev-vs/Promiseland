<script setup lang="ts">
import { RouterLink } from 'vue-router'
import Avatar from '@/components/ui/Avatar.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import type { Community } from '@/types'
import { plural } from './format'

defineProps<{ community: Community }>()
</script>

<template>
  <div class="community-card">
    <RouterLink class="community-card__link" :to="`/communities/${community.slug}`">
      <Avatar :name="community.name" :src="community.icon_url" />
      <span class="community-card__info">
        <span class="community-card__name">{{ community.name }}</span>
        <span class="community-card__count mono">{{ plural(community.member_count ?? 0, 'member') }}</span>
        <span class="community-card__description">{{ community.description || 'No description yet.' }}</span>
      </span>
      <span class="community-card__action">
        <BaseButton
          v-if="!community.is_member"
          variant="secondary"
          :to="`/communities/${community.slug}`"
        >
          Join community
        </BaseButton>
        <span v-else class="community-card__joined">Joined</span>
      </span>
    </RouterLink>
  </div>
</template>

<style scoped>
.community-card {
  border-bottom: 1px solid var(--border-hairline);
}

.community-card:last-child {
  border-bottom: 0;
}

.community-card__link {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  color: inherit;
  padding: var(--space-3) 0;
  text-decoration: none;
}

.community-card__link:hover .community-card__name {
  color: var(--tide);
}

.community-card__info {
  display: flex;
  flex: 1;
  min-width: 0;
  flex-direction: column;
  gap: var(--space-1);
}

.community-card__name {
  overflow: hidden;
  color: var(--text-primary);
  font-family: var(--font-display);
  font-size: var(--text-lg);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.community-card__count {
  color: var(--text-muted);
  font-size: var(--text-xs);
}

.community-card__description {
  overflow: hidden;
  color: var(--text-secondary);
  font-size: var(--text-sm);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.community-card__action {
  margin-inline-start: auto;
}

.community-card__joined {
  color: var(--ok);
  font-size: var(--text-sm);
}
</style>
