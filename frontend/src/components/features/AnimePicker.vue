<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue'
import TagChip from '@/components/ui/TagChip.vue'
import { searchAnime } from '@/api/anime'
import type { Anime } from '@/types'
import { errorMessage } from './format'

const props = defineProps<{
  modelValue: Anime[]
  label?: string
}>()

const emit = defineEmits<{ 'update:modelValue': [value: Anime[]] }>()

const query = ref('')
const results = ref<Anime[]>([])
const loading = ref(false)
const error = ref('')
const searched = ref(false)
let timer: number | undefined

function runSearch(): void {
  if (timer !== undefined) window.clearTimeout(timer)
  const term = query.value.trim()
  if (term.length < 2) {
    results.value = []
    searched.value = false
    error.value = ''
    return
  }
  timer = window.setTimeout(async () => {
    loading.value = true
    error.value = ''
    try {
      results.value = await searchAnime(term)
      searched.value = true
    } catch (cause) {
      error.value = errorMessage(cause, 'We could not search anime right now. Try again.')
      results.value = []
    } finally {
      loading.value = false
    }
  }, 250)
}

function add(item: Anime): void {
  if (props.modelValue.some((selected) => selected.id === item.id)) return
  emit('update:modelValue', [...props.modelValue, item])
  query.value = ''
  results.value = []
  searched.value = false
}

function remove(id: number): void {
  emit('update:modelValue', props.modelValue.filter((selected) => selected.id !== id))
}

onBeforeUnmount(() => {
  if (timer !== undefined) window.clearTimeout(timer)
})
</script>

<template>
  <div class="anime-picker">
    <label class="anime-picker__label" for="anime-search">{{ label || 'Favourite anime' }}</label>
    <input
      id="anime-search"
      v-model="query"
      class="anime-picker__input"
      type="search"
      placeholder="Search by title"
      autocomplete="off"
      @input="runSearch"
    />

    <div v-if="modelValue.length" class="anime-picker__selected">
      <span v-for="item in modelValue" :key="item.id" class="anime-picker__chip">
        <TagChip :label="item.title" tone="ember" />
        <button type="button" :aria-label="`Remove ${item.title}`" @click="remove(item.id)">Remove</button>
      </span>
    </div>

    <p v-if="loading" class="anime-picker__status" role="status">Searching titles…</p>
    <p v-else-if="error" class="anime-picker__error" role="alert">
      {{ error }} <button type="button" @click="runSearch">Retry</button>
    </p>
    <p v-else-if="!query.trim()" class="anime-picker__status">Type at least two characters to find a title.</p>
    <p v-else-if="query.trim().length < 2" class="anime-picker__status">Keep typing to search titles.</p>
    <p v-else-if="searched && !results.length" class="anime-picker__status">No titles found. Try another search.</p>

    <ul v-if="results.length" class="anime-picker__results">
      <li v-for="item in results" :key="item.id">
        <button
          type="button"
          class="anime-picker__result"
          :disabled="modelValue.some((selected) => selected.id === item.id)"
          @click="add(item)"
        >
          <span>{{ item.title }}</span>
          <span class="anime-picker__format mono">{{ item.format || 'title' }}</span>
        </button>
      </li>
    </ul>
  </div>
</template>

<style scoped>
.anime-picker {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.anime-picker__label {
  color: var(--text-primary);
  font-size: var(--text-sm);
  font-weight: 500;
}

.anime-picker__input {
  width: 100%;
  min-height: 44px;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  background: var(--ink-850);
  color: var(--text-primary);
}

.anime-picker__input:focus {
  border-color: var(--ember);
  outline: none;
}

.anime-picker__selected {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.anime-picker__chip {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
}

.anime-picker__chip button {
  min-height: 28px;
  padding: 0 var(--space-1);
  border: 0;
  background: none;
  color: var(--text-secondary);
  font-size: var(--text-xs);
  cursor: pointer;
}

.anime-picker__chip button:hover {
  color: var(--alert);
}

.anime-picker__status,
.anime-picker__error {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.anime-picker__error {
  color: var(--alert);
}

.anime-picker__error button {
  padding: 0;
  border: 0;
  background: none;
  color: var(--alert);
  text-decoration: underline;
  cursor: pointer;
}

.anime-picker__results {
  display: flex;
  max-height: 220px;
  flex-direction: column;
  overflow-y: auto;
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-md);
}

.anime-picker__result {
  display: flex;
  width: 100%;
  min-height: 44px;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-3);
  border: 0;
  border-bottom: 1px solid var(--border-hairline);
  background: var(--ink-800);
  color: var(--text-primary);
  text-align: left;
  cursor: pointer;
}

.anime-picker__result:last-child {
  border-bottom: 0;
}

.anime-picker__result:hover:not(:disabled) {
  background: var(--ink-700);
}

.anime-picker__result:disabled {
  color: var(--text-muted);
  cursor: default;
}

.anime-picker__format {
  color: var(--text-secondary);
  font-size: var(--text-xs);
  text-transform: capitalize;
}
</style>
