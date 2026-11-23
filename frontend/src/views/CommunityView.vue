<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Avatar from '@/components/ui/Avatar.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseCard from '@/components/ui/BaseCard.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import SkeletonList from '@/components/ui/SkeletonList.vue'
import TagChip from '@/components/ui/TagChip.vue'
import FeedTabs from '@/components/features/FeedTabs.vue'
import PostCard from '@/components/features/PostCard.vue'
import UserCard from '@/components/features/UserCard.vue'
import {
  addModerator,
  getCommunity,
  joinCommunity,
  leaveCommunity,
  listCommunityPosts,
  listMembers,
  listRules,
  updateCommunity,
} from '@/api/communities'
import { useUiStore } from '@/stores/ui'
import type { Community, CommunityRule, Post, UserSummary } from '@/types'
import { errorMessage, isForbidden, isNotFound, plural } from '@/components/features/format'

const route = useRoute()
const router = useRouter()
const ui = useUiStore()
const slug = computed(() => String(route.params.slug || ''))

const community = ref<Community | null>(null)
const rules = ref<CommunityRule[]>([])
const posts = ref<Post[]>([])
const members = ref<UserSummary[]>([])
const tab = ref('posts')
const cursor = ref('')
const loading = ref(true)
const loadingList = ref(false)
const busy = ref(false)
const error = ref('')
const notFound = ref(false)
const forbidden = ref(false)
const actionError = ref('')
const editingDescription = ref(false)
const descriptionDraft = ref('')
const moderatorId = ref('')

const isModerator = computed(
  () => community.value?.role === 'owner' || community.value?.role === 'moderator',
)
const tabs = computed(() => [
  { value: 'posts', label: 'Posts' },
  { value: 'members', label: 'Members' },
  { value: 'rules', label: 'Rules' },
])

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  notFound.value = false
  forbidden.value = false
  try {
    const [found, communityRules] = await Promise.all([getCommunity(slug.value), listRules(slug.value)])
    community.value = found
    rules.value = communityRules
    descriptionDraft.value = found.description || ''
    await loadPosts(true)
  } catch (cause) {
    if (isForbidden(cause)) forbidden.value = true
    else if (isNotFound(cause)) notFound.value = true
    else error.value = errorMessage(cause, 'We could not load this community. Try again.')
  } finally {
    loading.value = false
  }
}

async function loadPosts(reset = false): Promise<void> {
  if (reset) {
    posts.value = []
    cursor.value = ''
  }
  loadingList.value = true
  try {
    const page = await listCommunityPosts(slug.value, {
      limit: 20,
      ...(cursor.value ? { cursor: cursor.value } : {}),
    })
    posts.value = reset ? page.items : [...posts.value, ...page.items]
    cursor.value = page.next_cursor || ''
  } catch (cause) {
    actionError.value = errorMessage(cause, 'We could not load posts. Try again.')
  } finally {
    loadingList.value = false
  }
}

async function loadMembers(): Promise<void> {
  loadingList.value = true
  try {
    const page = await listMembers(slug.value, { limit: 20 })
    members.value = page.items
  } catch (cause) {
    actionError.value = errorMessage(cause, 'We could not load members. Try again.')
  } finally {
    loadingList.value = false
  }
}

function changeTab(value: string): void {
  tab.value = value
  if (value === 'members') void loadMembers()
  if (value === 'posts' && !posts.value.length) void loadPosts(true)
}

async function toggleMembership(): Promise<void> {
  if (!community.value || busy.value) return
  busy.value = true
  actionError.value = ''
  const joining = !community.value.is_member
  try {
    if (joining) await joinCommunity(slug.value)
    else await leaveCommunity(slug.value)
    community.value = { ...community.value, is_member: joining }
    ui.pushToast({ title: joining ? 'Joined community' : 'Left community', tone: 'success' })
  } catch (cause) {
    actionError.value = errorMessage(cause, 'We could not update your membership. Try again.')
  } finally {
    busy.value = false
  }
}

async function saveDescription(): Promise<void> {
  if (!descriptionDraft.value.trim()) return
  busy.value = true
  actionError.value = ''
  try {
    const updated = await updateCommunity(slug.value, { description: descriptionDraft.value.trim() })
    community.value = updated
    editingDescription.value = false
    ui.pushToast({ title: 'Description updated', tone: 'success' })
  } catch (cause) {
    actionError.value = errorMessage(cause, 'We could not save the description. Try again.')
  } finally {
    busy.value = false
  }
}

async function inviteModerator(): Promise<void> {
  const id = Number(moderatorId.value)
  if (!Number.isInteger(id) || id <= 0) {
    actionError.value = 'Enter a valid member ID.'
    return
  }
  busy.value = true
  actionError.value = ''
  try {
    await addModerator(slug.value, id)
    moderatorId.value = ''
    ui.pushToast({ title: 'Moderator added', tone: 'success' })
  } catch (cause) {
    actionError.value = errorMessage(cause, 'We could not add that moderator. Try again.')
  } finally {
    busy.value = false
  }
}

onMounted(load)
watch(slug, load)
</script>

<template>
  <div class="community">
    <SkeletonList v-if="loading" :rows="4" />

    <EmptyState
      v-else-if="notFound"
      title="Community not found"
      description="This community may have been removed or the link may be wrong."
      action-label="Discover communities"
      @action="router.push('/communities')"
    />

    <EmptyState
      v-else-if="forbidden"
      title="You do not have access to this community"
      description="This community is private. Request an invitation from one of its moderators."
      action-label="Discover communities"
      @action="router.push('/communities')"
    />

    <EmptyState
      v-else-if="error || !community"
      title="We could not load this community"
      :description="error || 'The community is unavailable right now.'"
      action-label="Try again"
      @action="load"
    />

    <template v-else>
      <img v-if="community.banner_url" class="community__banner" :src="community.banner_url" alt="" />

      <BaseCard class="community__header">
        <div class="community__identity">
          <Avatar :name="community.name" :src="community.icon_url" size="lg" />
          <div>
            <h1 class="community__name">{{ community.name }}</h1>
            <p class="community__handle mono">/c/{{ community.slug }}</p>
          </div>
        </div>
        <p class="community__description">{{ community.description || 'No description yet.' }}</p>
        <div class="community__meta">
          <TagChip :label="community.visibility === 'private' ? 'Private' : 'Public'" />
          <span class="mono">{{ plural(community.member_count ?? 0, 'member') }}</span>
          <span class="mono">{{ plural(community.post_count ?? 0, 'post') }}</span>
          <TagChip v-if="community.role" :label="community.role" tone="ember" />
        </div>
        <div class="community__actions">
          <BaseButton variant="primary" :to="`/posts/new?community=${community.slug}`">Write a post</BaseButton>
          <BaseButton :loading="busy" :variant="community.is_member ? 'secondary' : 'ghost'" @click="toggleMembership">
            {{ community.is_member ? 'Leave community' : 'Join community' }}
          </BaseButton>
        </div>
        <p v-if="actionError" class="community__error" role="alert">{{ actionError }}</p>
      </BaseCard>

      <div class="community__body">
        <div class="community__main">
          <FeedTabs :model-value="tab" :tabs="tabs" label="Community sections" @update:model-value="changeTab" />

          <SkeletonList v-if="loadingList" :rows="3" />

          <template v-else-if="tab === 'posts'">
            <EmptyState
              v-if="!posts.length"
              title="No posts in this community yet"
              description="Start a discussion or share something you have made."
              action-label="Write a post"
              @action="router.push(`/posts/new?community=${community.slug}`)"
            />
            <ul v-else class="community__list">
              <li v-for="post in posts" :key="post.id"><PostCard :post="post" /></li>
            </ul>
            <div v-if="cursor" class="community__more">
              <BaseButton @click="loadPosts(false)">Load more</BaseButton>
            </div>
          </template>

          <template v-else-if="tab === 'members'">
            <p v-if="!members.length" class="community__note">No members to show yet.</p>
            <ul v-else class="community__list">
              <li v-for="member in members" :key="member.id"><UserCard :user="member" /></li>
            </ul>
          </template>

          <template v-else>
            <p v-if="!rules.length" class="community__note">This community has no published rules yet.</p>
            <ol v-else class="community__rules">
              <li v-for="rule in rules" :key="rule.id" class="community__rule">
                <h3>{{ rule.title }}</h3>
                <p>{{ rule.body }}</p>
              </li>
            </ol>
          </template>
        </div>

        <aside class="community__rail" aria-label="Community information">
          <BaseCard>
            <h2 class="community__rail-title">Rules</h2>
            <ol v-if="rules.length" class="community__rail-rules">
              <li v-for="rule in rules.slice(0, 5)" :key="rule.id">{{ rule.title }}</li>
            </ol>
            <p v-else class="community__note">No rules published.</p>
            <BaseButton variant="ghost" size="sm" @click="changeTab('rules')">See all rules</BaseButton>
          </BaseCard>

          <BaseCard v-if="isModerator" class="community__mod">
            <h2 class="community__rail-title">Moderator tools</h2>
            <div class="community__mod-block">
              <label class="community__label" for="community-description">Description</label>
              <textarea
                v-if="editingDescription"
                id="community-description"
                v-model="descriptionDraft"
                class="community__control"
                rows="3"
              />
              <p v-else class="community__note">{{ community.description || 'No description yet.' }}</p>
              <div class="community__mod-actions">
                <BaseButton v-if="editingDescription" size="sm" variant="ghost" @click="editingDescription = false">
                  Cancel
                </BaseButton>
                <BaseButton size="sm" variant="secondary" :loading="busy" @click="editingDescription ? saveDescription() : (editingDescription = true)">
                  {{ editingDescription ? 'Save description' : 'Edit description' }}
                </BaseButton>
              </div>
            </div>
            <div class="community__mod-block">
              <label class="community__label" for="moderator-id">Add a moderator by member ID</label>
              <input
                id="moderator-id"
                v-model="moderatorId"
                class="community__control"
                type="number"
                min="1"
                inputmode="numeric"
              />
              <BaseButton size="sm" variant="secondary" :loading="busy" @click="inviteModerator">Add moderator</BaseButton>
            </div>
          </BaseCard>
        </aside>
      </div>
    </template>
  </div>
</template>

<style scoped>
.community {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.community__banner {
  width: 100%;
  max-height: 200px;
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-lg);
  object-fit: cover;
}

.community__header {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.community__identity {
  display: flex;
  align-items: center;
  gap: var(--space-4);
}

.community__name {
  font-size: var(--text-2xl);
}

.community__handle {
  margin-top: var(--space-1);
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.community__description {
  color: var(--text-secondary);
}

.community__meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-3);
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.community__actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.community__error {
  color: var(--alert);
  font-size: var(--text-sm);
}

.community__body {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 260px;
  gap: var(--space-5);
  align-items: start;
}

.community__main {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: var(--space-4);
}

.community__list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.community__more {
  display: flex;
  justify-content: center;
}

.community__note {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.community__rules {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  counter-reset: rule;
}

.community__rule {
  padding-left: var(--space-6);
  counter-increment: rule;
}

.community__rule::before {
  margin-left: calc(var(--space-6) * -1);
  color: var(--tide);
  font-family: var(--font-mono);
  content: counter(rule) '. ';
}

.community__rule h3 {
  font-size: var(--text-lg);
}

.community__rule p {
  margin-top: var(--space-1);
  color: var(--text-secondary);
}

.community__rail {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.community__rail-title {
  font-size: var(--text-lg);
}

.community__rail-rules {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  margin-block: var(--space-3);
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.community__mod {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.community__mod-block {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.community__label {
  color: var(--text-primary);
  font-size: var(--text-sm);
  font-weight: 500;
}

.community__control {
  width: 100%;
  min-height: 44px;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  background: var(--ink-850);
  color: var(--text-primary);
  font-family: var(--font-body);
  resize: vertical;
}

.community__mod-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}

@media (max-width: 1023px) {
  .community__body {
    grid-template-columns: minmax(0, 1fr);
  }

  .community__rail {
    order: -1;
  }
}
</style>
