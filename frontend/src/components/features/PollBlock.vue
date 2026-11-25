<script setup lang="ts">
import { computed } from 'vue'
import type { Post } from '@/types'
import TagChip from '@/components/ui/TagChip.vue'

interface PollOptionView {
  id: number
  text: string
  votes: number
}

interface PollView {
  question: string
  options: PollOptionView[]
  total_votes: number
  my_vote: number | null
}

const props = defineProps<{ post: Post }>()
const emit = defineEmits<{ vote: [optionId: number] }>()

const poll = computed<PollView | undefined>(() => props.post.poll as PollView | undefined)

function percentage(votes: number): number {
  const total = poll.value?.total_votes || 0
  return total > 0 ? Math.round((votes / total) * 100) : 0
}
</script>

<template>
  <div v-if="poll" class="poll">
    <p class="poll__question">{{ poll.question }}</p>
    <ul class="poll__options">
      <li v-for="option in poll.options" :key="option.id">
        <button
          type="button"
          class="poll__option"
          :class="{ 'poll__option--mine': poll.my_vote === option.id }"
          :aria-pressed="poll.my_vote === option.id"
          :disabled="poll.my_vote !== null"
          @click="emit('vote', option.id)"
        >
          <span class="poll__text">{{ option.text }}</span>
          <span v-if="poll.my_vote !== null" class="poll__result mono">
            {{ percentage(option.votes) }}% ({{ option.votes }})
          </span>
          <span
            v-if="poll.my_vote !== null"
            class="poll__bar"
            :class="{ 'poll__bar--winner': poll.my_vote === option.id }"
            :style="{ width: `${percentage(option.votes)}%` }"
          />
        </button>
      </li>
    </ul>
    <div class="poll__meta">
      <TagChip :label="`${poll.total_votes} votes`" />
      <span v-if="poll.my_vote !== null">You voted on this poll</span>
      <span v-else>Poll voting opens when the API supports it</span>
    </div>
  </div>
</template>

<style scoped>
.poll {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding: var(--space-4);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-md);
  background: var(--ink-850);
}

.poll__question {
  font-weight: 600;
}

.poll__options {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.poll__option {
  display: flex;
  flex-wrap: wrap;
  width: 100%;
  min-height: 44px;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-sm);
  background: var(--ink-800);
  color: var(--text-primary);
  text-align: left;
  cursor: pointer;
}

.poll__option:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.poll__option--mine {
  border-color: var(--ember);
  border-left: 2px solid var(--ember);
  background: var(--ember-soft);
}

.poll__bar {
  flex-basis: 100%;
  height: 6px;
  border-radius: var(--radius-sm);
  background: var(--ink-600);
}

.poll__bar--winner {
  background: var(--ember);
}

.poll__result {
  color: var(--text-secondary);
  font-size: var(--text-sm);
  white-space: nowrap;
}

.poll__meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-2);
  color: var(--text-secondary);
  font-size: var(--text-sm);
}
</style>
