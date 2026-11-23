<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import Avatar from '@/components/ui/Avatar.vue'
import BaseCard from '@/components/ui/BaseCard.vue'
import SpoilerShield from '@/components/ui/SpoilerShield.vue'
import TagChip from '@/components/ui/TagChip.vue'
import { bookmarkPost, unbookmarkPost, votePost } from '@/api/posts'
import { useUiStore } from '@/stores/ui'
import type { Post } from '@/types'
import { formatWhen, plural } from './format'
import PollBlock from './PollBlock.vue'
import ReportDialog from './ReportDialog.vue'

const props = withDefaults(defineProps<{ post: Post; compact?: boolean }>(), { compact: false })

const ui = useUiStore()
const score = ref(props.post.vote_score ?? 0)
const myVote = ref<number>(props.post.my_vote ?? 0)
const saved = ref(props.post.bookmarked ?? false)
const busy = ref(false)
const reportOpen = ref(false)

watch(
  () => props.post.id,
  () => {
    score.value = props.post.vote_score ?? 0
    myVote.value = props.post.my_vote ?? 0
    saved.value = props.post.bookmarked ?? false
  },
)

const when = computed(() => formatWhen(props.post.created_at))
const spoilerHint = computed(() =>
  props.post.spoiler_note ? `Marked as a spoiler: ${props.post.spoiler_note}` : undefined,
)

async function vote(value: -1 | 1): Promise<void> {
  if (busy.value) return
  const next = myVote.value === value ? 0 : value
  const delta = next - myVote.value
  score.value += delta
  myVote.value = next
  busy.value = true
  try {
    await votePost(props.post.id, next)
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
    if (next) await bookmarkPost(props.post.id)
    else await unbookmarkPost(props.post.id)
  } catch {
    saved.value = !next
    ui.pushToast({ title: 'Save not updated', message: 'Check your connection and try again.', tone: 'error' })
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <BaseCard class="post-card" :class="{ 'post-card--compact': compact }">
    <header class="post-card__head">
      <Avatar :name="post.author?.display_name || post.author?.username || 'Member'" :src="post.author?.avatar_url" size="sm" />
      <div class="post-card__byline">
        <RouterLink class="post-card__author" :to="`/u/${post.author?.username || ''}`">
          {{ post.author?.display_name || post.author?.username || 'Member' }}
        </RouterLink>
        <span class="post-card__meta">
          <RouterLink v-if="post.community" :to="`/communities/${post.community.slug}`">
            {{ post.community.name }}
          </RouterLink>
          <time v-if="when" :datetime="post.created_at">{{ when }}</time>
        </span>
      </div>
      <TagChip v-if="post.type && post.type !== 'text'" :label="post.type" />
    </header>

    <h2 v-if="post.title" class="post-card__title">
      <RouterLink :to="`/posts/${post.id}`">{{ post.title }}</RouterLink>
    </h2>

    <SpoilerShield v-if="post.spoiler" :hint="spoilerHint">
      <p v-if="post.body" class="post-card__body u-measure">{{ post.body }}</p>
      <div v-if="post.media?.length" class="post-card__media">
        <img
          v-for="item in post.media.slice(0, 2)"
          :key="item.id"
          :src="item.url"
          :alt="post.title || 'Post image'"
          loading="lazy"
        />
      </div>
      <PollBlock v-if="post.poll" :post="post" />
    </SpoilerShield>

    <template v-else>
      <p v-if="post.body" class="post-card__body u-measure">{{ post.body }}</p>
      <div v-if="post.media?.length" class="post-card__media">
        <img
          v-for="item in post.media.slice(0, 2)"
          :key="item.id"
          :src="item.url"
          :alt="post.title || 'Post image'"
          loading="lazy"
        />
      </div>
      <PollBlock v-if="post.poll" :post="post" />
    </template>

    <a v-if="post.link_url" :href="post.link_url" class="post-card__link" rel="noopener noreferrer" target="_blank">
      {{ post.link_url }}
    </a>

    <div v-if="post.anime?.length || post.tags?.length" class="post-card__tags">
      <TagChip v-for="item in post.anime || []" :key="item.id" :label="item.title" tone="tide" />
      <TagChip v-for="tag in post.tags || []" :key="tag" :label="tag" />
    </div>

    <footer class="post-card__actions">
      <div class="post-card__votes">
        <button
          type="button"
          class="post-card__action"
          :class="{ 'post-card__action--on': myVote === 1 }"
          :aria-pressed="myVote === 1"
          aria-label="Upvote post"
          :disabled="busy"
          @click="vote(1)"
        >
          Up
        </button>
        <span class="post-card__score mono" :aria-label="`Score ${score}`">{{ score }}</span>
        <button
          type="button"
          class="post-card__action"
          :class="{ 'post-card__action--on': myVote === -1 }"
          :aria-pressed="myVote === -1"
          aria-label="Downvote post"
          :disabled="busy"
          @click="vote(-1)"
        >
          Down
        </button>
      </div>

      <RouterLink class="post-card__action" :to="`/posts/${post.id}`">
        {{ plural(post.comment_count ?? 0, 'comment') }}
      </RouterLink>
      <button
        type="button"
        class="post-card__action"
        :aria-pressed="saved"
        :disabled="busy"
        @click="toggleBookmark"
      >
        {{ saved ? 'Saved' : 'Save' }}
      </button>
      <button type="button" class="post-card__action post-card__action--report" @click="reportOpen = true">
        Report
      </button>
    </footer>

    <ReportDialog
      v-if="reportOpen"
      target-type="post"
      :target-id="post.id"
      @close="reportOpen = false"
      @submitted="reportOpen = false"
    />
  </BaseCard>
</template>

<style scoped>
.post-card {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.post-card--compact {
  padding: var(--space-4);
}

.post-card__head {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.post-card__byline {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-width: 0;
}

.post-card__author {
  overflow: hidden;
  color: var(--text-primary);
  font-weight: 600;
  text-decoration: none;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.post-card__meta {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.post-card__title {
  font-size: var(--text-xl);
}

.post-card__title a {
  color: var(--text-primary);
  text-decoration: none;
}

.post-card__title a:hover {
  color: var(--tide);
}

.post-card__body {
  overflow-wrap: anywhere;
  white-space: pre-wrap;
  color: var(--text-primary);
}

.post-card__media {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 200px), 1fr));
  gap: var(--space-2);
}

.post-card__media img {
  width: 100%;
  max-height: 360px;
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-md);
  object-fit: cover;
}

.post-card__link {
  overflow-wrap: anywhere;
  font-size: var(--text-sm);
}

.post-card__tags {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.post-card__actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-2) var(--space-4);
  padding-top: var(--space-3);
  border-top: 1px solid var(--border-hairline);
}

.post-card__votes {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
}

.post-card__score {
  min-width: 2.5ch;
  color: var(--text-primary);
  font-size: var(--text-sm);
  text-align: center;
}

.post-card__action {
  min-height: 32px;
  padding: 0 var(--space-2);
  border: 0;
  border-radius: var(--radius-sm);
  background: none;
  color: var(--text-secondary);
  font-size: var(--text-sm);
  text-decoration: none;
  cursor: pointer;
}

.post-card__action:hover:not(:disabled) {
  background: var(--ink-700);
  color: var(--text-primary);
}

.post-card__action--on {
  color: var(--ember);
}

.post-card__action--report {
  margin-left: auto;
}
</style>
