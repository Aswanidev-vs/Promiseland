<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseCard from '@/components/ui/BaseCard.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import SkeletonList from '@/components/ui/SkeletonList.vue'
import TagChip from '@/components/ui/TagChip.vue'
import CreatorCategoryPicker from '@/components/features/CreatorCategoryPicker.vue'
import TeamUpCard from '@/components/features/TeamUpCard.vue'
import { listTeamUps } from '@/api/teamups'
import type { TeamUp } from '@/types'
import { errorMessage, isForbidden } from '@/components/features/format'
import { TEAM_UP_STATUSES } from '@/components/features/formatR2'

const router = useRouter()

const status = ref('')
const category = ref('')
const teamUps = ref<TeamUp[]>([])
const cursor = ref('')
const loading = ref(true)
const loadingMore = ref(false)
const error = ref('')
const forbidden = ref(false)

const hasFilters = computed(() => !!(status.value || category.value))
// The backend applies the same filters; this is a safety net for stale caches.
const filteredTeamUps = computed(() =>
  teamUps.value.filter(
    (item) =>
      (!status.value || item.status === status.value) &&
      (!category.value || (item.categories || []).includes(category.value)),
  ),
)

function clearFilters(): void {
  status.value = ''
  category.value = ''
  void load(true)
}

async function load(reset = false): Promise<void> {
  if (reset) {
    teamUps.value = []
    cursor.value = ''
    error.value = ''
    forbidden.value = false
  }
  const more = !reset && cursor.value !== ''
  if (more) loadingMore.value = true
  else loading.value = true
  try {
    const page = await listTeamUps({
      ...(status.value ? { status: status.value } : {}),
      ...(category.value ? { category: category.value } : {}),
      limit: 20,
      ...(more ? { cursor: cursor.value } : {}),
    })
    teamUps.value = more ? [...teamUps.value, ...page.items] : page.items
    cursor.value = page.next_cursor || ''
  } catch (cause) {
    if (isForbidden(cause)) forbidden.value = true
    else error.value = errorMessage(cause, 'We could not load team ups. Try again.')
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

function setStatus(value: string): void {
  status.value = value
  void load(true)
}

function setCategory(value: string | string[]): void {
  category.value = String(value)
  void load(true)
}

onMounted(() => void load(true))
</script>

<template>
  <div class="team-ups">
    <header class="team-ups__header">
      <div>
        <h1>Team ups</h1>
        <p class="team-ups__intro">Find a group for the next convention, shoot, or watch party.</p>
      </div>
      <BaseButton variant="primary" to="/team-ups/new">Start a team up</BaseButton>
    </header>

    <BaseCard class="team-ups__filters">
      <fieldset class="team-ups__statuses">
        <legend class="team-ups__label">Status</legend>
        <div class="team-ups__chips">
          <button type="button" class="team-ups__chip" :aria-pressed="!status" @click="setStatus('')">
            <TagChip label="All" :selected="!status" />
          </button>
          <button
            v-for="item in TEAM_UP_STATUSES"
            :key="item.value"
            type="button"
            class="team-ups__chip"
            :aria-pressed="status === item.value"
            @click="setStatus(item.value)"
          >
            <TagChip :label="item.label" :selected="status === item.value" />
          </button>
        </div>
      </fieldset>

      <CreatorCategoryPicker
        :model-value="category"
        label="Category"
        none-label="All categories"
        @update:model-value="setCategory"
      />

      <BaseButton v-if="hasFilters" size="sm" variant="ghost" @click="clearFilters">Clear filters</BaseButton>
    </BaseCard>

    <SkeletonList v-if="loading" :rows="4" />

    <EmptyState
      v-else-if="forbidden"
      title="Team ups are not available"
      description="You may need to sign in again to browse requests."
      action-label="Go to home"
      @action="router.push('/home')"
    />

    <EmptyState
      v-else-if="error"
      title="We could not load team ups"
      :description="error"
      action-label="Try again"
      @action="load(true)"
    />

    <EmptyState
      v-else-if="!filteredTeamUps.length"
      :title="hasFilters ? 'No team ups match these filters' : 'No team ups yet'"
      :description="hasFilters ? 'Try another status or category.' : 'Start a request and invite others to join you.'"
      :action-label="hasFilters ? 'Clear filters' : 'Start a team up'"
      @action="hasFilters ? clearFilters() : router.push('/team-ups/new')"
    />

    <template v-else>
      <ul class="team-ups__list">
        <li v-for="teamUp in filteredTeamUps" :key="teamUp.id"><TeamUpCard :team-up="teamUp" /></li>
      </ul>
      <div v-if="cursor" class="team-ups__more">
        <BaseButton :loading="loadingMore" @click="load()">Load more</BaseButton>
      </div>
      <p v-else class="team-ups__end">That is every request for now.</p>
    </template>
  </div>
</template>

<style scoped>
.team-ups {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.team-ups__header {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}

.team-ups__header h1 {
  font-size: var(--text-2xl);
}

.team-ups__intro {
  margin-top: var(--space-1);
  color: var(--text-secondary);
}

.team-ups__filters {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  align-items: flex-start;
}

.team-ups__statuses {
  margin: 0;
  padding: 0;
  border: 0;
}

.team-ups__label {
  display: block;
  margin-bottom: var(--space-2);
  color: var(--text-primary);
  font-size: var(--text-sm);
  font-weight: 500;
}

.team-ups__chips {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.team-ups__chip {
  min-height: 44px;
  padding: 0;
  border: 0;
  background: none;
  cursor: pointer;
}

.team-ups__chip:focus-within {
  outline: 2px solid var(--ember);
  outline-offset: 2px;
  border-radius: var(--radius-sm);
}

.team-ups__list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.team-ups__more {
  display: flex;
  justify-content: center;
}

.team-ups__end {
  color: var(--text-secondary);
  font-size: var(--text-sm);
  text-align: center;
}
</style>
