<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import Avatar from '@/components/ui/Avatar.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import { addEventComment } from '@/api/events'
import { useUiStore } from '@/stores/ui'
import type { EventComment } from '@/types'
import { errorMessage, formatWhen } from './format'

const props = defineProps<{ comments: EventComment[]; eventId: number }>()
const emit = defineEmits<{ refresh: [] }>()

const ui = useUiStore()
const draft = ref('')
const replyTo = ref<number | null>(null)
const replyDraft = ref('')
const submitting = ref(false)
const error = ref('')

const roots = computed(() => props.comments.filter((comment) => comment.parent_id === null))
function repliesOf(id: number): EventComment[] {
  return props.comments.filter((comment) => comment.parent_id === id)
}

async function publish(body: string, parentId?: number): Promise<void> {
  if (!body.trim() || submitting.value) return
  submitting.value = true
  error.value = ''
  try {
    await addEventComment(props.eventId, body.trim(), parentId)
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
  <section class="event-thread" aria-labelledby="event-thread-title">
    <h2 id="event-thread-title" class="event-thread__title">Discussion</h2>

    <form class="event-thread__composer" @submit.prevent="publish(draft)">
      <label class="event-thread__label" for="event-comment-draft">Add a comment</label>
      <textarea
        id="event-comment-draft"
        v-model="draft"
        class="event-thread__textarea"
        rows="3"
        maxlength="4000"
        placeholder="Ask a question or share a plan."
      />
      <div class="event-thread__actions">
        <BaseButton type="submit" variant="primary" size="sm" :loading="submitting" :disabled="!draft.trim()">
          Post comment
        </BaseButton>
      </div>
    </form>

    <p v-if="error" class="event-thread__error" role="alert">{{ error }}</p>
    <p v-if="!comments.length" class="event-thread__empty">No comments yet. Start the conversation.</p>

    <ul class="event-thread__list">
      <li v-for="comment in roots" :key="comment.id" class="event-thread__item">
        <article class="event-comment">
          <Avatar
            :name="comment.author?.display_name || comment.author?.username || 'Member'"
            :src="comment.author?.avatar_url"
            size="sm"
          />
          <div class="event-comment__body">
            <div class="event-comment__head">
              <RouterLink class="event-comment__author" :to="`/u/${comment.author?.username || ''}`">
                {{ comment.author?.display_name || comment.author?.username || 'Member' }}
              </RouterLink>
              <time v-if="comment.created_at" class="event-comment__time" :datetime="comment.created_at">
                {{ formatWhen(comment.created_at) }}
              </time>
            </div>
            <p v-if="comment.deleted" class="event-comment__deleted">This comment was removed.</p>
            <p v-else class="event-comment__text">{{ comment.body }}</p>
            <button
              v-if="!comment.deleted"
              type="button"
              class="event-comment__action"
              @click="replyTo = replyTo === comment.id ? null : comment.id"
            >
              Reply
            </button>

            <form
              v-if="replyTo === comment.id"
              class="event-thread__composer event-thread__composer--reply"
              @submit.prevent="publish(replyDraft, comment.id)"
            >
              <label class="event-thread__label" :for="`event-reply-${comment.id}`">Reply to this comment</label>
              <textarea :id="`event-reply-${comment.id}`" v-model="replyDraft" class="event-thread__textarea" rows="2" />
              <div class="event-thread__actions">
                <BaseButton type="button" size="sm" variant="ghost" @click="replyTo = null">Cancel</BaseButton>
                <BaseButton type="submit" size="sm" :loading="submitting" :disabled="!replyDraft.trim()">
                  Post reply
                </BaseButton>
              </div>
            </form>
          </div>
        </article>

        <ul v-if="repliesOf(comment.id).length" class="event-thread__replies">
          <li v-for="reply in repliesOf(comment.id)" :key="reply.id" class="event-comment">
            <Avatar
              :name="reply.author?.display_name || reply.author?.username || 'Member'"
              :src="reply.author?.avatar_url"
              size="sm"
            />
            <div class="event-comment__body">
              <div class="event-comment__head">
                <RouterLink class="event-comment__author" :to="`/u/${reply.author?.username || ''}`">
                  {{ reply.author?.display_name || reply.author?.username || 'Member' }}
                </RouterLink>
                <time v-if="reply.created_at" class="event-comment__time" :datetime="reply.created_at">
                  {{ formatWhen(reply.created_at) }}
                </time>
              </div>
              <p v-if="reply.deleted" class="event-comment__deleted">This comment was removed.</p>
              <p v-else class="event-comment__text">{{ reply.body }}</p>
            </div>
          </li>
        </ul>
      </li>
    </ul>
  </section>
</template>

<style scoped>
.event-thread {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.event-thread__title {
  font-size: var(--text-xl);
}

.event-thread__composer {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-4);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-md);
  background: var(--ink-850);
}

.event-thread__composer--reply {
  margin-top: var(--space-3);
}

.event-thread__label {
  color: var(--text-secondary);
  font-size: var(--text-sm);
  font-weight: 500;
}

.event-thread__textarea {
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

.event-thread__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}

.event-thread__error {
  color: var(--alert);
  font-size: var(--text-sm);
}

.event-thread__empty {
  color: var(--text-secondary);
}

.event-thread__list {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.event-comment {
  display: flex;
  gap: var(--space-3);
}

.event-comment__body {
  min-width: 0;
  flex: 1;
}

.event-comment__head {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: var(--space-2);
}

.event-comment__author {
  color: var(--text-primary);
  font-weight: 600;
  text-decoration: none;
}

.event-comment__time {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.event-comment__text,
.event-comment__deleted {
  margin-top: var(--space-1);
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}

.event-comment__deleted {
  color: var(--text-secondary);
  font-style: italic;
}

.event-comment__action {
  min-height: 44px;
  margin-top: var(--space-2);
  padding: 0;
  border: 0;
  background: none;
  color: var(--text-secondary);
  font-size: var(--text-sm);
  cursor: pointer;
}

.event-comment__action:hover {
  color: var(--text-primary);
}

.event-thread__replies {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  margin: var(--space-3) 0 0 var(--space-6);
  padding-left: var(--space-6);
  border-left: 1px solid var(--border-hairline);
}

@media (max-width: 480px) {
  .event-thread__replies {
    margin-left: var(--space-2);
    padding-left: var(--space-3);
  }
}
</style>
