<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import Avatar from '@/components/ui/Avatar.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import SkeletonList from '@/components/ui/SkeletonList.vue'
import FeedTabs, { type FeedTab } from '@/components/features/FeedTabs.vue'
import PostCard from '@/components/features/PostCard.vue'
import { joinCommunity, listCommunities } from '@/api/communities'
import { getFeed } from '@/api/posts'
import { useUiStore } from '@/stores/ui'
import { errorMessage, isForbidden, plural } from '@/components/features/format'
import type { Community, Post } from '@/types'

type Scope = 'for-you' | 'following'

const router = useRouter()
const ui = useUiStore()
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
const suggested = ref<Community[]>([])
const suggestionsLoaded = ref(false)
const joining = ref('')

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
    if (!posts.value.length) void loadSuggestions()
  } catch (cause) {
    if (isForbidden(cause)) forbidden.value = true
    else error.value = errorMessage(cause, 'We could not load your feed. Try again.')
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

/**
 * An empty feed should offer the next useful thing rather than a bigger
 * placeholder, so it proposes real communities the reader can join.
 */
async function loadSuggestions(): Promise<void> {
  if (suggestionsLoaded.value) return
  suggestionsLoaded.value = true
  try {
    const page = await listCommunities({ limit: 4 })
    suggested.value = page.items.filter((item) => !item.is_member).slice(0, 3)
  } catch {
    suggested.value = []
  }
}

async function join(community: Community): Promise<void> {
  joining.value = community.slug
  try {
    await joinCommunity(community.slug)
    suggested.value = suggested.value.filter((item) => item.id !== community.id)
    ui.pushToast({ title: `Joined ${community.name}`, tone: 'success' })
    await load(true)
  } catch (cause) {
    ui.pushToast({
      title: 'Could not join that community',
      message: errorMessage(cause, 'Try again in a moment.'),
      tone: 'error',
    })
  } finally {
    joining.value = ''
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

    <div v-else-if="!posts.length" class="home__start">
      <EmptyState
        :title="scope === 'following' ? 'No posts from people you follow yet' : 'Your feed is quiet'"
        :description="
          scope === 'following'
            ? 'Follow a few people, or join a community and its threads land here.'
            : 'Join a community, or write the first post your corner of fandom sees.'
        "
        :action-label="scope === 'following' ? 'Find people to follow' : 'Create a post'"
        @action="scope === 'following' ? router.push('/discover') : router.push('/posts/new')"
      />

      <section v-if="suggested.length" class="home__suggest" aria-labelledby="home-suggest-title">
        <h2 id="home-suggest-title" class="home__suggest-title">Communities to start with</h2>
        <ul class="home__suggest-list">
          <li v-for="community in suggested" :key="community.id" class="home__suggest-row">
            <Avatar :name="community.name" :src="community.icon_url" size="sm" />
            <span class="home__suggest-text">
              <span class="home__suggest-name">{{ community.name }}</span>
              <span class="home__suggest-meta">{{ plural(community.member_count, 'member') }}</span>
            </span>
            <BaseButton
              size="sm"
              variant="secondary"
              :loading="joining === community.slug"
              @click="join(community)"
            >
              Join
            </BaseButton>
          </li>
        </ul>
        <RouterLink class="home__suggest-more" to="/communities">Browse all communities</RouterLink>
      </section>
    </div>

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
  gap: var(--space-4);
}

.home__header h1 {
  font-family: var(--font-display);
  font-weight: 560;
  font-size: var(--text-3xl);
  letter-spacing: -0.02em;
  line-height: 1.1;
}

.home__feed {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.home__start {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.home__suggest {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding-top: var(--space-5);
  border-top: 1px solid var(--border-hairline);
}

.home__suggest-title {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--text-primary);
}

.home__suggest-list {
  display: flex;
  flex-direction: column;
}

.home__suggest-row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  min-height: 56px;
  padding-block: var(--space-2);
  border-bottom: 1px solid var(--border-hairline);
}

.home__suggest-row:last-child {
  border-bottom: 0;
}

.home__suggest-text {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
}

.home__suggest-name {
  overflow: hidden;
  color: var(--text-primary);
  font-size: var(--text-sm);
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.home__suggest-meta {
  color: var(--text-secondary);
  font-size: var(--text-xs);
}

.home__suggest-more {
  display: inline-flex;
  align-items: center;
  align-self: flex-start;
  min-height: 44px;
  color: var(--tide);
  font-size: var(--text-sm);
  font-weight: 500;
  text-decoration: none;
}

.home__suggest-more:hover {
  color: var(--text-primary);
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
