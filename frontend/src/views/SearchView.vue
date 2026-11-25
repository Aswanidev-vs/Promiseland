<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import BaseInput from '@/components/ui/BaseInput.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import SkeletonList from '@/components/ui/SkeletonList.vue'
import FeedTabs, { type FeedTab } from '@/components/features/FeedTabs.vue'
import CommunityCard from '@/components/features/CommunityCard.vue'
import PostCard from '@/components/features/PostCard.vue'
import UserCard from '@/components/features/UserCard.vue'
import { search } from '@/api/search'
import type { Community, Post, UserSummary } from '@/types'
import { errorMessage, isForbidden } from '@/components/features/format'

type SearchType = 'users' | 'communities' | 'posts'

const route = useRoute()
const router = useRouter()
const tabs: FeedTab[] = [
  { value: 'posts', label: 'Posts' },
  { value: 'communities', label: 'Communities' },
  { value: 'users', label: 'People' },
]

const query = ref(typeof route.query.q === 'string' ? route.query.q : '')
const type = ref<SearchType>(parseType(route.query.type))
const users = ref<UserSummary[]>([])
const communities = ref<Community[]>([])
const posts = ref<Post[]>([])
const loading = ref(false)
const error = ref('')
const forbidden = ref(false)
const searched = ref(false)

function parseType(value: unknown): SearchType {
  return value === 'users' || value === 'communities' || value === 'posts' ? value : 'posts'
}

async function run(): Promise<void> {
  const term = query.value.trim()
  searched.value = true
  error.value = ''
  forbidden.value = false
  if (!term) {
    users.value = []
    communities.value = []
    posts.value = []
    return
  }
  loading.value = true
  try {
    const results = await search({ q: term, type: type.value, limit: 20 })
    users.value = results.users || []
    communities.value = results.communities || []
    posts.value = results.posts || []
  } catch (cause) {
    if (isForbidden(cause)) forbidden.value = true
    else error.value = errorMessage(cause, 'We could not complete your search. Try again.')
  } finally {
    loading.value = false
  }
}

function submit(): void {
  void router.replace({ path: '/search', query: { q: query.value.trim(), type: type.value } })
  void run()
}

function changeType(value: string): void {
  type.value = value as SearchType
  submit()
}

watch(
  () => [route.query.q, route.query.type],
  () => {
    query.value = typeof route.query.q === 'string' ? route.query.q : ''
    type.value = parseType(route.query.type)
    void run()
  },
)

onMounted(run)
</script>

<template>
  <div class="search">
    <h1 class="search__title">Search</h1>

    <form role="search" @submit.prevent="submit">
      <BaseInput v-model="query" type="search" label="Search PromisedLand" placeholder="People, communities, and posts" />
    </form>

    <FeedTabs :model-value="type" :tabs="tabs" label="Result types" @update:model-value="changeType" />

    <SkeletonList v-if="loading" :rows="4" />

    <EmptyState
      v-else-if="forbidden"
      title="Search is not available"
      description="You may need to sign in again to use search."
      action-label="Go to home"
      @action="router.push('/home')"
    />

    <EmptyState
      v-else-if="error"
      title="We could not complete your search"
      :description="error"
      action-label="Try again"
      @action="run"
    />

    <EmptyState
      v-else-if="!query.trim()"
      title="What are you looking for?"
      description="Search for people, communities, or posts by title and topic."
      action-label="Browse communities"
      @action="router.push('/communities')"
    />

    <EmptyState
      v-else-if="searched && !users.length && !communities.length && !posts.length"
      title="No results found"
      description="Try a different term or switch the result type."
      action-label="Clear search"
      @action="((query = ''), submit())"
    />

    <section v-else-if="type === 'users'" class="search__group">
      <h2 class="search__group-title">People <span class="search__count mono">{{ users.length }}</span></h2>
      <ul class="search__list">
        <li v-for="user in users" :key="user.id"><UserCard :user="user" /></li>
      </ul>
    </section>

    <section v-else-if="type === 'communities'" class="search__group">
      <h2 class="search__group-title">Communities <span class="search__count mono">{{ communities.length }}</span></h2>
      <ul class="search__list">
        <li v-for="community in communities" :key="community.id"><CommunityCard :community="community" /></li>
      </ul>
    </section>

    <section v-else class="search__group">
      <h2 class="search__group-title">Posts <span class="search__count mono">{{ posts.length }}</span></h2>
      <ul class="search__list">
        <li v-for="post in posts" :key="post.id"><PostCard :post="post" /></li>
      </ul>
    </section>
  </div>
</template>

<style scoped>
.search {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.search__title {
  font-family: var(--font-display);
  font-weight: 560;
  font-size: var(--text-3xl);
  letter-spacing: -0.02em;
  line-height: 1.1;
}

.search .field__input:focus {
  border-color: var(--ember);
  box-shadow: 0 0 0 3px var(--ember-soft);
}

.search__group-title {
  font-size: var(--text-base);
  font-weight: 500;
}

.search__count {
  color: var(--text-muted);
  font-size: var(--text-sm);
}

.search__list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
</style>
