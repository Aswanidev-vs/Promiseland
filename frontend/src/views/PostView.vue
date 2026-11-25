<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Avatar from '@/components/ui/Avatar.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import SkeletonList from '@/components/ui/SkeletonList.vue'
import SpoilerShield from '@/components/ui/SpoilerShield.vue'
import TagChip from '@/components/ui/TagChip.vue'
import CommentThread from '@/components/features/CommentThread.vue'
import PollBlock from '@/components/features/PollBlock.vue'
import ReportDialog from '@/components/features/ReportDialog.vue'
import { bookmarkPost, deletePost, getPost, listComments, unbookmarkPost, updatePost, votePost } from '@/api/posts'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import type { Comment, Post } from '@/types'
import { errorMessage, formatWhen, isForbidden, isNotFound, plural } from '@/components/features/format'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const ui = useUiStore()
const id = computed(() => Number(route.params.id))

const post = ref<Post | null>(null)
const comments = ref<Comment[]>([])
const loading = ref(true)
const loadingComments = ref(false)
const busy = ref(false)
const error = ref('')
const notFound = ref(false)
const forbidden = ref(false)
const reportOpen = ref(false)
const editing = ref(false)
const editTitle = ref('')
const editBody = ref('')
const score = ref(0)
const myVote = ref(0)
const saved = ref(false)

const isAuthor = computed(() => !!post.value && !!auth.user && post.value.author?.id === auth.user.id)
const isModerator = computed(
  () => post.value?.community?.role === 'owner' || post.value?.community?.role === 'moderator',
)
const canManage = computed(() => isAuthor.value || isModerator.value)
const spoilerHint = computed(() =>
  post.value?.spoiler_note ? `Marked as a spoiler: ${post.value.spoiler_note}` : undefined,
)

function applyPost(value: Post): void {
  post.value = value
  score.value = value.vote_score ?? 0
  myVote.value = value.my_vote ?? 0
  saved.value = value.bookmarked ?? false
  editTitle.value = value.title || ''
  editBody.value = value.body || ''
}

async function loadComments(): Promise<void> {
  loadingComments.value = true
  try {
    const page = await listComments(id.value, { limit: 50 })
    comments.value = page.items
  } catch {
    comments.value = []
  } finally {
    loadingComments.value = false
  }
}

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  notFound.value = false
  forbidden.value = false
  try {
    applyPost(await getPost(id.value))
    await loadComments()
  } catch (cause) {
    if (isForbidden(cause)) forbidden.value = true
    else if (isNotFound(cause)) notFound.value = true
    else error.value = errorMessage(cause, 'We could not load this post. Try again.')
  } finally {
    loading.value = false
  }
}

async function vote(value: -1 | 1): Promise<void> {
  if (busy.value) return
  const next = myVote.value === value ? 0 : value
  const delta = next - myVote.value
  score.value += delta
  myVote.value = next
  busy.value = true
  try {
    await votePost(id.value, next)
  } catch {
    score.value -= delta
    myVote.value -= delta
    ui.pushToast({ title: 'Vote not saved', message: 'Check your connection and try again.', tone: 'error' })
  } finally {
    busy.value = false
  }
}

async function toggleBookmark(): Promise<void> {
  if (busy.value) return
  const next = !saved.value
  saved.value = next
  busy.value = true
  try {
    if (next) await bookmarkPost(id.value)
    else await unbookmarkPost(id.value)
  } catch {
    saved.value = !next
    ui.pushToast({ title: 'Save not updated', message: 'Check your connection and try again.', tone: 'error' })
  } finally {
    busy.value = false
  }
}

async function saveEdit(): Promise<void> {
  if (!post.value || !editTitle.value.trim()) return
  busy.value = true
  try {
    applyPost(await updatePost(post.value.id, { title: editTitle.value.trim(), body: editBody.value }))
    editing.value = false
    ui.pushToast({ title: 'Post updated', tone: 'success' })
  } catch (cause) {
    ui.pushToast({ title: 'Post not updated', message: errorMessage(cause), tone: 'error' })
  } finally {
    busy.value = false
  }
}

async function removePost(): Promise<void> {
  if (!post.value || busy.value) return
  busy.value = true
  try {
    await deletePost(post.value.id)
    ui.pushToast({ title: 'Post removed', tone: 'success' })
    await router.push('/home')
  } catch (cause) {
    ui.pushToast({ title: 'Post not removed', message: errorMessage(cause), tone: 'error' })
  } finally {
    busy.value = false
  }
}

onMounted(load)
watch(id, load)
</script>

<template>
  <div class="post-view">
    <SkeletonList v-if="loading" :rows="5" avatar />

    <EmptyState
      v-else-if="notFound"
      title="This post is not available"
      description="It may have been removed, or the link may be wrong."
      action-label="Back to home"
      @action="router.push('/home')"
    />

    <EmptyState
      v-else-if="forbidden"
      title="You do not have access to this post"
      description="Check the community or profile privacy settings."
      action-label="Explore communities"
      @action="router.push('/communities')"
    />

    <EmptyState
      v-else-if="error || !post"
      title="We could not load this post"
      :description="error || 'The post is unavailable right now.'"
      action-label="Try again"
      @action="load"
    />

    <template v-else>
      <BaseButton variant="ghost" @click="router.back()">Back</BaseButton>

      <article class="post-detail">
        <header class="post-detail__head">
          <Avatar
            :name="post.author?.display_name || post.author?.username || 'Member'"
            :src="post.author?.avatar_url"
          />
          <div class="post-detail__byline">
            <RouterLink class="post-detail__author" :to="`/u/${post.author?.username || ''}`">
              {{ post.author?.display_name || post.author?.username || 'Member' }}
            </RouterLink>
            <span class="post-detail__meta">
              <RouterLink v-if="post.community" :to="`/communities/${post.community.slug}`">
                {{ post.community.name }}
              </RouterLink>
              <span v-if="post.edited_at" class="post-detail__edited">edited</span>
              <time v-if="post.created_at" :datetime="post.created_at">{{ formatWhen(post.created_at) }}</time>
            </span>
          </div>
          <TagChip v-if="post.type && post.type !== 'text'" :label="post.type" />
        </header>

        <template v-if="editing">
          <label class="post-detail__label" for="edit-title">Title</label>
          <input id="edit-title" v-model="editTitle" class="post-detail__control" />
          <label class="post-detail__label" for="edit-body">Body</label>
          <textarea id="edit-body" v-model="editBody" class="post-detail__control" rows="6" />
          <div class="post-detail__edit-actions">
            <BaseButton size="sm" variant="ghost" @click="editing = false">Cancel</BaseButton>
            <BaseButton size="sm" variant="secondary" :loading="busy" @click="saveEdit">Save changes</BaseButton>
          </div>
        </template>

        <template v-else>
          <h1 class="post-detail__title">{{ post.title }}</h1>

          <SpoilerShield v-if="post.spoiler" :hint="spoilerHint">
            <p v-if="post.body" class="post-detail__body">{{ post.body }}</p>
            <div v-if="post.media?.length" class="post-detail__media">
              <img v-for="item in post.media" :key="item.id" :src="item.url" :alt="post.title || 'Post image'" />
            </div>
            <PollBlock v-if="post.poll" :post="post" />
          </SpoilerShield>

          <template v-else>
            <p v-if="post.body" class="post-detail__body">{{ post.body }}</p>
            <div v-if="post.media?.length" class="post-detail__media">
              <img v-for="item in post.media" :key="item.id" :src="item.url" :alt="post.title || 'Post image'" />
            </div>
            <PollBlock v-if="post.poll" :post="post" />
          </template>

          <a v-if="post.link_url" :href="post.link_url" class="post-detail__link" target="_blank" rel="noopener noreferrer">
            {{ post.link_url }}
          </a>

          <div v-if="post.anime?.length || post.tags?.length" class="post-detail__tags">
            <TagChip v-for="item in post.anime || []" :key="item.id" :label="item.title" tone="tide" />
            <TagChip v-for="tag in post.tags || []" :key="tag" :label="tag" />
          </div>
        </template>

        <footer class="post-detail__actions">
          <div class="post-detail__votes">
            <button
              type="button"
              class="post-detail__action"
              :class="{ 'post-detail__action--on': myVote === 1 }"
              :aria-pressed="myVote === 1"
              aria-label="Upvote post"
              :disabled="busy"
              @click="vote(1)"
            >
              Up
            </button>
            <span class="post-detail__score mono">{{ score }}</span>
            <button
              type="button"
              class="post-detail__action"
              :class="{ 'post-detail__action--on': myVote === -1 }"
              :aria-pressed="myVote === -1"
              aria-label="Downvote post"
              :disabled="busy"
              @click="vote(-1)"
            >
              Down
            </button>
          </div>
          <button type="button" class="post-detail__action" :aria-pressed="saved" :disabled="busy" @click="toggleBookmark">
            {{ saved ? 'Saved' : 'Save' }}
          </button>
          <span class="post-detail__count">{{ plural(post.comment_count ?? 0, 'comment') }}</span>
          <button type="button" class="post-detail__action" @click="reportOpen = true">Report</button>
          <template v-if="canManage">
            <button v-if="!editing" type="button" class="post-detail__action" @click="editing = true">Edit</button>
            <button type="button" class="post-detail__action post-detail__action--danger" :disabled="busy" @click="removePost">
              Remove
            </button>
          </template>
        </footer>
      </article>

      <SkeletonList v-if="loadingComments" :rows="2" />
      <CommentThread v-else :comments="comments" :post-id="post.id" @refresh="loadComments" />

      <ReportDialog
        v-if="reportOpen"
        target-type="post"
        :target-id="post.id"
        @close="reportOpen = false"
        @submitted="reportOpen = false"
      />
    </template>
  </div>
</template>

<style scoped>
.post-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.post-detail {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  padding: var(--space-6);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-lg);
  background: var(--ink-800);
}

.post-detail__head {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.post-detail__byline {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-width: 0;
}

.post-detail__author {
  color: var(--text-primary);
  font-weight: 500;
  text-decoration: none;
}

.post-detail__meta {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  color: var(--text-muted);
  font-size: var(--text-sm);
}

.post-detail__edited {
  color: var(--text-secondary);
}

.post-detail__title {
  margin-top: var(--space-3);
  font-family: var(--font-display);
  font-weight: 560;
  font-size: var(--text-3xl);
  letter-spacing: -0.02em;
  line-height: 1.1;
}

.post-detail__body {
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}

.post-detail__media {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.post-detail__media img {
  width: 100%;
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-md);
  object-fit: cover;
}

.post-detail__link {
  overflow-wrap: anywhere;
  font-size: var(--text-sm);
}

.post-detail__tags {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.post-detail__label {
  color: var(--text-secondary);
  font-size: var(--text-sm);
  font-weight: 500;
}

.post-detail__control {
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

.post-detail__edit-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}

.post-detail__actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-2) var(--space-4);
  padding-top: var(--space-4);
  border-top: 1px solid var(--border-hairline);
}

.post-detail__votes {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
}

.post-detail__score {
  min-width: 2.5ch;
  color: var(--text-primary);
  font-size: var(--text-sm);
  text-align: center;
}

.post-detail__action {
  min-height: 44px;
  padding: 0 var(--space-2);
  border: 0;
  border-radius: var(--radius-sm);
  background: none;
  color: var(--text-secondary);
  font-size: var(--text-sm);
  text-decoration: none;
  cursor: pointer;
  transition:
    background-color var(--dur-fast) var(--ease),
    color var(--dur-fast) var(--ease);
}

.post-detail__action:hover:not(:disabled) {
  background: var(--ink-700);
  color: var(--text-primary);
}

.post-detail__action:active:not(:disabled) {
  background: var(--ink-700);
  color: var(--text-primary);
}

.post-detail__action:focus-visible {
  outline: 2px solid var(--ember);
  outline-offset: 2px;
}

.post-detail__action--on {
  color: var(--ember);
}

.post-detail__action--danger {
  color: var(--alert);
}

.post-detail__count {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

@media (max-width: 480px) {
  .post-detail {
    padding: var(--space-4);
  }
}
</style>
