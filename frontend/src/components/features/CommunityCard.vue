<script setup lang="ts">
import { RouterLink } from 'vue-router'
import Avatar from '@/components/ui/Avatar.vue'
import BaseCard from '@/components/ui/BaseCard.vue'
import TagChip from '@/components/ui/TagChip.vue'
import type { Community } from '@/types'
import { plural } from './format'

defineProps<{ community: Community }>()
</script>

<template>
  <BaseCard interactive class="community-card">
    <RouterLink class="community-card__link" :to="`/communities/${community.slug}`">
      <Avatar :name="community.name" :src="community.icon_url" />
      <span class="community-card__info">
        <span class="community-card__name">{{ community.name }}</span>
        <span class="community-card__description">{{ community.description || 'No description yet.' }}</span>
        <span class="community-card__meta">
          <TagChip :label="community.visibility === 'private' ? 'Private' : 'Public'" />
          <span class="mono">{{ plural(community.member_count ?? 0, 'member') }}</span>
          <span class="mono">{{ plural(community.post_count ?? 0, 'post') }}</span>
        </span>
      </span>
    </RouterLink>
  </BaseCard>
</template>

<style scoped>
.community-card__link {
  display: flex;
  align-items: flex-start;
  gap: var(--space-4);
  color: inherit;
  text-decoration: none;
}

.community-card__link:hover .community-card__name {
  color: var(--tide);
}

.community-card__info {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: var(--space-2);
}

.community-card__name {
  overflow: hidden;
  color: var(--text-primary);
  font-family: var(--font-display);
  font-size: var(--text-lg);
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.community-card__description {
  display: -webkit-box;
  overflow: hidden;
  color: var(--text-secondary);
  font-size: var(--text-sm);
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.community-card__meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-2);
  color: var(--text-secondary);
  font-size: var(--text-sm);
}
</style>
