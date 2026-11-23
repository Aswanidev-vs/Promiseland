<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import BaseButton from '@/components/ui/BaseButton.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import SkeletonList from '@/components/ui/SkeletonList.vue'
import FeedTabs, { type FeedTab } from '@/components/features/FeedTabs.vue'
import PostCard from '@/components/features/PostCard.vue'
import { getFeed } from '@/api/posts'
import type { Post } from '@/types'
import { errorMessage, isForbidden } from '@/components/features/format'

type Scope = 'for-you' | 'following'

const router = useRouter()
const tabs: FeedTab[] = [
  { value: 'for-you', label: 'For you' },
  { value: 'following', label: 'Following' },
]

const scope = ref<Scope>('for-you')
const posts = ref<Post[]>([])
const cursor = ref('')
const loading = ref(true)
const loadingMore = ref(false)
const error = ref('')
const forbidden = ref(false)

async function load(reset = false): Promise<void> {
  if (reset) {
    posts.value = []
    cursor.value = ''
    error.value = ''
    forbidden.value = false
  }
  const more = !reset && cursor.value !== ''
  if (more) loadingMore.value = true
  else loading.value = true
  try {
    const page = await getFeed({
      scope: scope.value,
      limit: 20,
      ...(more ? { cursor: cursor.value } : {}),
    })
    posts.value = more ? [...posts.value, ...page.items] : page.items
    cursor.value = page.next_cursor || ''
  } catch (cause) {
    if (isForbidden(cause)) forbidden.value = true
    else error.value = errorMessage(cause, 'We could not load your feed. Try again.')
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

function changeScope(value: string): void {
  scope.value = value as Scope
  void load(true)
}

onMounted(() => void load(true))
</script>

<template>
  <div class="home">
    <header class="home__header">
      <h1>Home</h1>
      <BaseButton variant="primary" to="/posts/new">Share a post</BaseButton>
    </header>

    <FeedTabs :model-value="scope" :tabs="tabs" label="Feed scope" @update:model-value="changeScope" />

    <SkeletonList v-if="loading" :rows="4" avatar />

    <EmptyState
      v-else-if="forbidden"
      title="This feed is not available"
      description="You may need to sign in again to see personalized posts."
      action-label="Explore communities"
      @action="router.push('/communities')"
    />

    <EmptyState
      v-else-if="error"
      title="We could not load your feed"
      :description="error"
      action-label="Try again"
      @action="load(true)"
    />

    <EmptyState
      v-else-if="!posts.length"
      :title="scope === 'following' ? 'No posts from people you follow yet' : 'No posts yet'"
      :description="
        scope === 'following'
          ? 'Follow a few people or join a community to fill this feed.'
          : 'Be the first to share something with your communities.'
      "
      :action-label="scope === 'following' ? 'Find communities' : 'Create a post'"
      @action="scope === 'following' ? router.push('/communities') : router.push('/posts/new')"
    />

    <template v-else>
      <ul class="home__feed">
        <li v-for="post in posts" :key="post.id">
          <PostCard :post="post" />
        </li>
      </ul>

      <div v-if="cursor" class="home__more">
        <BaseButton :loading="loadingMore" @click="load()">Load more</BaseButton>
      </div>
      <p v-else class="home__end">You are up to date.</p>
    </template>
  </div>
</template>

<style scoped>
.home {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.home__header {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}

.home__header h1 {
  font-size: var(--text-2xl);
}

.home__feed {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.home__more {
  display: flex;
  justify-content: center;
}

.home__end {
  color: var(--text-secondary);
  font-size: var(--text-sm);
  text-align: center;
}
</style>
