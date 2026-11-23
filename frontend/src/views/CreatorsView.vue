<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseCard from '@/components/ui/BaseCard.vue'
import BaseInput from '@/components/ui/BaseInput.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import SkeletonList from '@/components/ui/SkeletonList.vue'
import CreatorCard from '@/components/features/CreatorCard.vue'
import CreatorCategoryPicker from '@/components/features/CreatorCategoryPicker.vue'
import { listCreators } from '@/api/creators'
import type { Creator } from '@/types'
import { errorMessage, isForbidden } from '@/components/features/format'

const router = useRouter()

const query = ref('')
const category = ref('')
const creators = ref<Creator[]>([])
const cursor = ref('')
const loading = ref(true)
const loadingMore = ref(false)
const error = ref('')
const forbidden = ref(false)
let timer: number | undefined

const hasFilters = computed(() => !!(query.value.trim() || category.value))

async function load(reset = false): Promise<void> {
  if (reset) {
    creators.value = []
    cursor.value = ''
    error.value = ''
    forbidden.value = false
  }
  const more = !reset && cursor.value !== ''
  if (more) loadingMore.value = true
  else loading.value = true
  try {
    const page = await listCreators({
      ...(query.value.trim() ? { q: query.value.trim() } : {}),
      ...(category.value ? { category: category.value } : {}),
      limit: 20,
      ...(more ? { cursor: cursor.value } : {}),
    } as Parameters<typeof listCreators>[0])
    creators.value = more ? [...creators.value, ...page.items] : page.items
    cursor.value = page.next_cursor || ''
  } catch (cause) {
    if (isForbidden(cause)) forbidden.value = true
    else error.value = errorMessage(cause, 'We could not load creators. Try again.')
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

function scheduleSearch(): void {
  if (timer !== undefined) window.clearTimeout(timer)
  timer = window.setTimeout(() => void load(true), 300)
}

function setCategory(value: string | string[]): void {
  category.value = String(value)
  void load(true)
}

function clearFilters(): void {
  query.value = ''
  category.value = ''
  void load(true)
}

onMounted(() => void load(true))
onBeforeUnmount(() => {
  if (timer !== undefined) window.clearTimeout(timer)
})
</script>

<template>
  <div class="creators">
    <header class="creators__header">
      <div>
        <h1>Creators</h1>
        <p class="creators__intro">Cosplay, crossplay, art, photography, and the people behind it.</p>
      </div>
      <BaseButton variant="primary" to="/posts/new">Share your work</BaseButton>
    </header>

    <BaseCard class="creators__filters">
      <BaseInput
        v-model="query"
        type="search"
        label="Search creators"
        placeholder="Search by name or handle"
        @update:model-value="scheduleSearch"
      />
      <CreatorCategoryPicker
        :model-value="category"
        label="Creator category"
        none-label="All categories"
        :include-none="true"
        @update:model-value="setCategory"
      />
      <BaseButton v-if="hasFilters" size="sm" variant="ghost" @click="clearFilters">Clear filters</BaseButton>
    </BaseCard>

    <SkeletonList v-if="loading" :rows="4" />

    <EmptyState
      v-else-if="forbidden"
      title="Creators are not available"
      description="You may need to sign in again to browse creator portfolios."
      action-label="Go to home"
      @action="router.push('/home')"
    />

    <EmptyState
      v-else-if="error"
      title="We could not load creators"
      :description="error"
      action-label="Try again"
      @action="load(true)"
    />

    <EmptyState
      v-else-if="!creators.length"
      :title="hasFilters ? 'No creators match these filters' : 'No creators yet'"
      :description="hasFilters ? 'Try another category or clear your filters.' : 'Tag a post as creator work to appear here.'"
      :action-label="hasFilters ? 'Clear filters' : 'Share your work'"
      @action="hasFilters ? clearFilters() : router.push('/posts/new')"
    />

    <template v-else>
      <ul class="creators__grid">
        <li v-for="creator in creators" :key="creator.user?.id"><CreatorCard :creator="creator" /></li>
      </ul>
      <div v-if="cursor" class="creators__more">
        <BaseButton :loading="loadingMore" @click="load()">Load more</BaseButton>
      </div>
      <p v-else class="creators__end">That is everyone for now.</p>
    </template>
  </div>
</template>

<style scoped>
.creators {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.creators__header {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}

.creators__header h1 {
  font-size: var(--text-2xl);
}

.creators__intro {
  margin-top: var(--space-1);
  color: var(--text-secondary);
}

.creators__filters {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  align-items: flex-start;
}

.creators__filters > :first-child {
  width: 100%;
}

.creators__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(100%, 280px), 1fr));
  gap: var(--space-4);
  align-items: stretch;
}

.creators__more {
  display: flex;
  justify-content: center;
}

.creators__end {
  color: var(--text-secondary);
  font-size: var(--text-sm);
  text-align: center;
}
</style>
