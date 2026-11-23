<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import Avatar from '@/components/ui/Avatar.vue'
import BaseCard from '@/components/ui/BaseCard.vue'
import TagChip from '@/components/ui/TagChip.vue'
import { useAuthStore } from '@/stores/auth'
import type { TeamUp } from '@/types'
import { plural } from './format'
import { creatorCategoryLabel, teamUpStatusLabel } from './formatR2'

const props = defineProps<{ teamUp: TeamUp }>()

const auth = useAuthStore()
const isOwner = computed(
  () => !!auth.user && !!props.teamUp.owner && auth.user.id === props.teamUp.owner.id,
)
const categories = computed(() => (props.teamUp.categories || []).slice(0, 4))
const applicationNote = computed(() => {
  const status = props.teamUp.my_application
  if (!status) return 'View request'
  if (status === 'pending') return 'Application pending'
  if (status === 'accepted') return 'Application accepted'
  return 'Application not accepted'
})
</script>

<template>
  <BaseCard interactive class="teamup-card" :padded="false">
    <RouterLink class="teamup-card__link" :to="`/team-ups/${teamUp.id}`">
      <div class="teamup-card__top">
        <TagChip :label="teamUpStatusLabel(teamUp.status)" />
        <span class="teamup-card__applicants mono">{{ plural(teamUp.applicant_count || 0, 'applicant') }}</span>
      </div>

      <h2 class="teamup-card__title">{{ teamUp.title }}</h2>
      <p class="teamup-card__description">{{ teamUp.description || 'No description yet.' }}</p>

      <div v-if="categories.length" class="teamup-card__categories">
        <TagChip v-for="category in categories" :key="category" :label="creatorCategoryLabel(category)" />
      </div>

      <p v-if="teamUp.event" class="teamup-card__event">
        Linked event <span>{{ teamUp.event.title }}</span>
      </p>

      <div class="teamup-card__foot">
        <Avatar
          :name="teamUp.owner?.display_name || teamUp.owner?.username || 'Member'"
          :src="teamUp.owner?.avatar_url"
          size="sm"
        />
        <span class="teamup-card__owner">{{ teamUp.owner?.display_name || teamUp.owner?.username || 'Member' }}</span>
        <span v-if="isOwner" class="teamup-card__manage">Review applications</span>
        <span v-else class="teamup-card__view">{{ applicationNote }}</span>
      </div>
    </RouterLink>
  </BaseCard>
</template>

<style scoped>
.teamup-card__link {
  display: flex;
  height: 100%;
  flex-direction: column;
  gap: var(--space-3);
  padding: var(--space-5);
  color: inherit;
  text-decoration: none;
}

.teamup-card__top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
}

.teamup-card__applicants {
  color: var(--text-secondary);
  font-size: var(--text-xs);
}

.teamup-card__title {
  color: var(--text-primary);
  font-size: var(--text-lg);
}

.teamup-card__link:hover .teamup-card__title {
  color: var(--tide);
}

.teamup-card__description {
  display: -webkit-box;
  overflow: hidden;
  color: var(--text-secondary);
  font-size: var(--text-sm);
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.teamup-card__categories {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-1);
}

.teamup-card__event {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.teamup-card__event span {
  color: var(--tide);
}

.teamup-card__foot {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-2);
  margin-top: auto;
  padding-top: var(--space-3);
  border-top: 1px solid var(--border-hairline);
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.teamup-card__owner {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.teamup-card__manage,
.teamup-card__view {
  margin-left: auto;
  color: var(--tide);
  font-size: var(--text-sm);
}
</style>
