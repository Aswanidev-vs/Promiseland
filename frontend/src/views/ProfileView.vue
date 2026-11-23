<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Avatar from '@/components/ui/Avatar.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import SkeletonList from '@/components/ui/SkeletonList.vue'
import TagChip from '@/components/ui/TagChip.vue'
import FeedTabs from '@/components/features/FeedTabs.vue'
import PostCard from '@/components/features/PostCard.vue'
import ReportDialog from '@/components/features/ReportDialog.vue'
import { blockUser, followUser, getUser, unblockUser, unfollowUser } from '@/api/users'
import { getFeed } from '@/api/posts'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import type { Post, UserProfile } from '@/types'
import { errorMessage, isForbidden, isNotFound, plural } from '@/components/features/format'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const ui = useUiStore()
const username = computed(() => String(route.params.username || ''))

const profile = ref<UserProfile | null>(null)
const posts = ref<Post[]>([])
const tab = ref('posts')
const loading = ref(true)
const loadingPosts = ref(false)
const busy = ref(false)
const error = ref('')
const notFound = ref(false)
const forbidden = ref(false)
const reportOpen = ref(false)
const actionError = ref('')

const isSelf = computed(() => !!profile.value && !!auth.user && profile.value.id === auth.user.id)
const tabs = computed(() => [
  { value: 'posts', label: 'Posts' },
  { value: 'about', label: 'About' },
])

async function loadPosts(): Promise<void> {
  if (!profile.value) return
  loadingPosts.value = true
  try {
    // R1 has no author-filtered post endpoint; filter the personal feed by author id.
    const page = await getFeed({ scope: 'for-you', limit: 50 })
    posts.value = page.items.filter((post) => post.author?.id === profile.value?.id)
  } catch {
    posts.value = []
  } finally {
    loadingPosts.value = false
  }
}

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  notFound.value = false
  forbidden.value = false
  try {
    profile.value = await getUser(username.value)
    await loadPosts()
  } catch (cause) {
    if (isForbidden(cause)) forbidden.value = true
    else if (isNotFound(cause)) notFound.value = true
    else error.value = errorMessage(cause, 'We could not load this profile. Try again.')
  } finally {
    loading.value = false
  }
}

async function toggleFollow(): Promise<void> {
  if (!profile.value || busy.value) return
  busy.value = true
  actionError.value = ''
  const following = !profile.value.is_following
  try {
    if (following) await followUser(Number(profile.value.id))
    else await unfollowUser(Number(profile.value.id))
    profile.value = { ...profile.value, is_following: following }
    ui.pushToast({ title: following ? 'Following' : 'Unfollowed', tone: 'success' })
  } catch (cause) {
    actionError.value = errorMessage(cause, 'We could not update this follow. Try again.')
  } finally {
    busy.value = false
  }
}

async function toggleBlock(): Promise<void> {
  if (!profile.value || busy.value) return
  busy.value = true
  actionError.value = ''
  const blocking = !profile.value.is_blocked
  try {
    if (blocking) await blockUser(Number(profile.value.id))
    else await unblockUser(Number(profile.value.id))
    profile.value = { ...profile.value, is_blocked: blocking }
    ui.pushToast({ title: blocking ? 'Account blocked' : 'Account unblocked', tone: 'success' })
    if (blocking) posts.value = []
  } catch (cause) {
    actionError.value = errorMessage(cause, 'We could not update this block. Try again.')
  } finally {
    busy.value = false
  }
}

function changeTab(value: string): void {
  tab.value = value
}

onMounted(load)
watch(username, load)
</script>

<template>
  <div class="profile">
    <SkeletonList v-if="loading" :rows="5" avatar />

    <EmptyState
      v-else-if="notFound"
      title="Profile not found"
      description="This username may have changed or the account may be private."
      action-label="Search people"
      @action="router.push('/search?type=users')"
    />

    <EmptyState
      v-else-if="forbidden"
      title="This profile is private"
      description="You do not have permission to see this profile's content."
      action-label="Back to home"
      @action="router.push('/home')"
    />

    <EmptyState
      v-else-if="error || !profile"
      title="We could not load this profile"
      :description="error || 'The profile is unavailable right now.'"
      action-label="Try again"
      @action="load"
    />

    <template v-else>
      <img v-if="profile.banner_url" class="profile__banner" :src="profile.banner_url" alt="" />

      <section class="profile__header" aria-labelledby="profile-name">
        <Avatar :name="profile.display_name || profile.username" :src="profile.avatar_url" size="lg" />
        <div class="profile__identity">
          <h1 id="profile-name" class="profile__name">{{ profile.display_name || profile.username }}</h1>
          <p class="profile__handle mono">@{{ profile.username }}</p>
          <p v-if="profile.bio" class="profile__bio">{{ profile.bio }}</p>
          <p v-if="profile.location" class="profile__location">{{ profile.location }}</p>
        </div>
        <div class="profile__actions">
          <BaseButton v-if="isSelf" variant="secondary" to="/settings">Edit profile</BaseButton>
          <template v-else>
            <BaseButton variant="primary" :loading="busy" @click="toggleFollow">
              {{ profile.is_following ? 'Following' : 'Follow' }}
            </BaseButton>
            <BaseButton variant="secondary" :loading="busy" @click="toggleBlock">
              {{ profile.is_blocked ? 'Unblock' : 'Block' }}
            </BaseButton>
            <BaseButton variant="ghost" @click="reportOpen = true">Report</BaseButton>
          </template>
        </div>
        <p v-if="actionError" class="profile__error" role="alert">{{ actionError }}</p>
      </section>

      <div class="profile__counts">
        <span><strong class="mono">{{ profile.post_count ?? 0 }}</strong> posts</span>
        <span><strong class="mono">{{ profile.follower_count ?? 0 }}</strong> followers</span>
        <span><strong class="mono">{{ profile.following_count ?? 0 }}</strong> following</span>
      </div>

      <FeedTabs :model-value="tab" :tabs="tabs" label="Profile sections" @update:model-value="changeTab" />

      <SkeletonList v-if="tab === 'posts' && loadingPosts" :rows="3" />

      <template v-else-if="tab === 'posts'">
        <ul v-if="posts.length" class="profile__posts">
          <li v-for="post in posts" :key="post.id"><PostCard :post="post" /></li>
        </ul>
        <EmptyState
          v-else
          title="No posts from this profile in your feed"
          description="Post history is not available in this release. Search to find public posts."
          action-label="Search posts"
          @action="router.push({ path: '/search', query: { q: profile.username, type: 'posts' } })"
        />
      </template>

      <section v-else class="profile__about">
        <h2 class="profile__section-title">Interests</h2>
        <div v-if="profile.interests?.length" class="profile__tags">
          <TagChip v-for="interest in profile.interests" :key="interest.id" :label="interest.name" />
        </div>
        <p v-else class="profile__note">No interests listed yet.</p>

        <h2 class="profile__section-title">Favourite anime</h2>
        <div v-if="profile.anime_interests?.length" class="profile__tags">
          <TagChip
            v-for="item in profile.anime_interests"
            :key="item.anime.id"
            :label="item.anime.title"
            tone="tide"
          />
        </div>
        <p v-else class="profile__note">No favourite titles listed yet.</p>

        <p class="profile__note">
          Member since {{ new Date(profile.created_at).toLocaleDateString(undefined, { month: 'long', year: 'numeric' }) }}
        </p>
        <p class="profile__note">{{ plural(profile.follower_count ?? 0, 'follower') }}</p>
      </section>

      <ReportDialog
        v-if="reportOpen"
        target-type="user"
        :target-id="Number(profile.id)"
        @close="reportOpen = false"
        @submitted="reportOpen = false"
      />
    </template>
  </div>
</template>

<style scoped>
.profile {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.profile__banner {
  width: 100%;
  max-height: 180px;
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-lg);
  object-fit: cover;
}

.profile__header {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  gap: var(--space-4);
  padding: var(--space-6);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-lg);
  background: var(--ink-800);
}

.profile__identity {
  flex: 1;
  min-width: 0;
}

.profile__name {
  font-size: var(--text-2xl);
}

.profile__handle {
  margin-top: var(--space-1);
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.profile__bio {
  margin-top: var(--space-3);
  max-width: 62ch;
}

.profile__location {
  margin-top: var(--space-2);
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.profile__actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.profile__error {
  width: 100%;
  color: var(--alert);
  font-size: var(--text-sm);
}

.profile__counts {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-6);
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.profile__counts strong {
  color: var(--text-primary);
}

.profile__posts {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.profile__about {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding: var(--space-6);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-lg);
  background: var(--ink-800);
}

.profile__section-title {
  font-size: var(--text-lg);
}

.profile__tags {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.profile__note {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

@media (max-width: 480px) {
  .profile__header,
  .profile__about {
    padding: var(--space-4);
  }
}
</style>
