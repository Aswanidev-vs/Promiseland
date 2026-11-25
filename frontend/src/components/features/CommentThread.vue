<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import Avatar from '@/components/ui/Avatar.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import { addComment } from '@/api/posts'
import { useUiStore } from '@/stores/ui'
import type { Comment } from '@/types'
import { errorMessage, formatWhen } from './format'
import ReportDialog from './ReportDialog.vue'

const props = defineProps<{
  comments: Comment[]
  postId: number
}>()

const emit = defineEmits<{ refresh: [] }>()

const ui = useUiStore()
const draft = ref('')
const replyTo = ref<number | null>(null)
const replyDraft = ref('')
const submitting = ref(false)
const error = ref('')
const reportTarget = ref<number | null>(null)

const roots = computed(() => props.comments.filter((comment) => comment.parent_id === null))
function repliesOf(id: number): Comment[] {
  return props.comments.filter((comment) => comment.parent_id === id)
}

async function publish(body: string, parentId?: number): Promise<void> {
  if (!body.trim() || submitting.value) return
  submitting.value = true
  error.value = ''
  try {
    await addComment(props.postId, body.trim(), parentId)
    ui.pushToast({ title: 'Comment added', tone: 'success' })
    if (parentId) {
      replyTo.value = null
      replyDraft.value = ''
    } else {
      draft.value = ''
    }
    emit('refresh')
  } catch (cause) {
    error.value = errorMessage(cause, 'We could not add your comment. Try again.')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <section class="thread" aria-labelledby="thread-title">
    <h2 id="thread-title" class="thread__title">Discussion</h2>

    <form class="thread__composer" @submit.prevent="publish(draft)">
      <label class="thread__label" for="comment-draft">Add a comment</label>
      <textarea
        id="comment-draft"
        v-model="draft"
        class="thread__textarea"
        rows="3"
        maxlength="4000"
        placeholder="Add to the discussion."
      />
      <div class="thread__composer-actions">
        <BaseButton type="submit" variant="primary" size="sm" :loading="submitting" :disabled="!draft.trim()">
          Post comment
        </BaseButton>
      </div>
    </form>

    <p v-if="error" class="thread__error" role="alert">{{ error }}</p>

    <p v-if="!comments.length" class="thread__empty">No comments yet. Start the conversation.</p>

    <ul class="thread__list">
      <li v-for="comment in roots" :key="comment.id" class="thread__item">
        <article class="comment">
          <Avatar
            :name="comment.author?.display_name || comment.author?.username || 'Member'"
            :src="comment.author?.avatar_url"
            size="sm"
          />
          <div class="comment__body">
            <div class="comment__head">
              <RouterLink class="comment__author" :to="`/u/${comment.author?.username || ''}`">
                {{ comment.author?.display_name || comment.author?.username || 'Member' }}
              </RouterLink>
              <time v-if="comment.created_at" class="comment__time" :datetime="comment.created_at">
                {{ formatWhen(comment.created_at) }}
              </time>
              <span v-if="comment.edited_at" class="comment__edited">edited</span>
            </div>
            <p v-if="comment.deleted" class="comment__deleted">This comment was removed.</p>
            <p v-else class="comment__text">{{ comment.body }}</p>
            <div v-if="!comment.deleted" class="comment__actions">
              <button type="button" class="comment__action" @click="replyTo = replyTo === comment.id ? null : comment.id">
                Reply
              </button>
              <button type="button" class="comment__action" @click="reportTarget = comment.id">Report</button>
            </div>

            <form
              v-if="replyTo === comment.id"
              class="thread__composer thread__composer--reply"
              @submit.prevent="publish(replyDraft, comment.id)"
            >
              <label class="thread__label" :for="`reply-${comment.id}`">Reply to this comment</label>
              <textarea
                :id="`reply-${comment.id}`"
                v-model="replyDraft"
                class="thread__textarea"
                rows="2"
                maxlength="4000"
              />
              <div class="thread__composer-actions">
                <BaseButton type="button" size="sm" variant="ghost" @click="replyTo = null">Cancel</BaseButton>
                <BaseButton type="submit" size="sm" :loading="submitting" :disabled="!replyDraft.trim()">
                  Post reply
                </BaseButton>
              </div>
            </form>
          </div>
        </article>

        <ul v-if="repliesOf(comment.id).length" class="thread__replies">
          <li v-for="reply in repliesOf(comment.id)" :key="reply.id" class="comment comment--reply">
            <Avatar
              :name="reply.author?.display_name || reply.author?.username || 'Member'"
              :src="reply.author?.avatar_url"
              size="sm"
            />
            <div class="comment__body">
              <div class="comment__head">
                <RouterLink class="comment__author" :to="`/u/${reply.author?.username || ''}`">
                  {{ reply.author?.display_name || reply.author?.username || 'Member' }}
                </RouterLink>
                <time v-if="reply.created_at" class="comment__time" :datetime="reply.created_at">
                  {{ formatWhen(reply.created_at) }}
                </time>
              </div>
              <p v-if="reply.deleted" class="comment__deleted">This comment was removed.</p>
              <p v-else class="comment__text">{{ reply.body }}</p>
              <div v-if="!reply.deleted" class="comment__actions">
                <button type="button" class="comment__action" @click="reportTarget = reply.id">Report</button>
              </div>
            </div>
          </li>
        </ul>
      </li>
    </ul>

    <ReportDialog
      v-if="reportTarget !== null"
      target-type="comment"
      :target-id="reportTarget"
      @close="reportTarget = null"
      @submitted="reportTarget = null"
    />
  </section>
</template>

<style scoped>
.thread {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.thread__title {
  font-size: var(--text-xl);
  font-weight: 500;
}

.thread__composer {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-4);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-md);
  background: var(--ink-850);
}

.thread__composer--reply {
  margin-top: var(--space-3);
}

.thread__label {
  color: var(--text-secondary);
  font-size: var(--text-sm);
  font-weight: 500;
}

.thread__textarea {
  width: 100%;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  background: var(--ink-800);
  color: var(--text-primary);
  font-family: var(--font-body);
  line-height: var(--leading-body);
  resize: vertical;
}

.thread__composer-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}

.thread__error {
  color: var(--alert);
  font-size: var(--text-sm);
}

.thread__empty {
  color: var(--text-secondary);
}

.thread__list {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.comment {
  display: flex;
  gap: var(--space-3);
}

.comment__body {
  min-width: 0;
  flex: 1;
}

.comment__head {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: var(--space-2);
}

.comment__author {
  color: var(--text-primary);
  font-weight: 600;
  text-decoration: none;
}

.comment__time,
.comment__edited {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.comment__text,
.comment__deleted {
  margin-top: var(--space-1);
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}

.comment__deleted {
  color: var(--text-secondary);
  font-style: italic;
}

.comment__actions {
  display: flex;
  gap: var(--space-3);
  margin-top: var(--space-2);
}

.comment__action {
  min-height: 44px;
  padding: 0;
  border: 0;
  background: none;
  color: var(--text-secondary);
  font-size: var(--text-sm);
  cursor: pointer;
  transition:
    background-color var(--dur-fast) var(--ease),
    color var(--dur-fast) var(--ease);
}

.comment__action:hover {
  background: var(--ink-700);
  color: var(--text-primary);
}

.comment__action:active {
  background: var(--ink-700);
  color: var(--text-primary);
}

.comment__action:focus-visible {
  outline: 2px solid var(--ember);
  outline-offset: 2px;
}

.thread__replies {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  margin: var(--space-3) 0 0 var(--space-6);
  padding-left: var(--space-4);
  border-left: 1px solid var(--border-hairline);
}

@media (max-width: 480px) {
  .thread__replies {
    margin-left: var(--space-2);
    padding-left: var(--space-3);
  }
}
</style>
