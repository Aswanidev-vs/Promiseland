<script setup lang="ts">
import { onMounted, ref } from 'vue'
import TagChip from '@/components/ui/TagChip.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import SkeletonList from '@/components/ui/SkeletonList.vue'
import { listInterests } from '@/api/users'
import type { Interest } from '@/types'
import { errorMessage } from './format'

const props = defineProps<{ modelValue: number[] }>()
const emit = defineEmits<{ 'update:modelValue': [value: number[]] }>()

const interests = ref<Interest[]>([])
const loading = ref(true)
const error = ref('')

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    interests.value = await listInterests()
  } catch (cause) {
    error.value = errorMessage(cause, 'We could not load interests. Try again.')
  } finally {
    loading.value = false
  }
}

function toggle(id: number): void {
  const selected = props.modelValue.includes(id)
  emit(
    'update:modelValue',
    selected ? props.modelValue.filter((item) => item !== id) : [...props.modelValue, id],
  )
}

onMounted(load)
</script>

<template>
  <div class="interest-picker">
    <p class="interest-picker__label">Interests</p>
    <SkeletonList v-if="loading" :rows="2" />
    <div v-else-if="error" class="interest-picker__error" role="alert">
      <p>{{ error }}</p>
      <BaseButton size="sm" @click="load">Retry</BaseButton>
    </div>
    <p v-else-if="!interests.length" class="interest-picker__empty">
      Interests are not available yet. You can continue and add them later.
    </p>
    <div v-else class="interest-picker__options">
      <button
        v-for="item in interests"
        :key="item.id"
        type="button"
        class="interest-picker__option"
        :class="{ selected: modelValue.includes(item.id) }"
        :aria-pressed="modelValue.includes(item.id)"
        @click="toggle(item.id)"
      >
        <TagChip :label="item.name" :selected="modelValue.includes(item.id)" />
      </button>
    </div>
    <p class="interest-picker__hint">Choose the parts of fandom you want to see more of.</p>
  </div>
</template>

<style scoped>
.interest-picker {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.interest-picker__label {
  color: var(--text-primary);
  font-size: var(--text-sm);
  font-weight: 500;
}

.interest-picker__options {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.interest-picker__option {
  min-height: 44px;
  padding: 0;
  border: 0;
  background: none;
  cursor: pointer;
}

/* Selected chip state, styled here since TagChip is shared. */
.interest-picker__option.selected :deep(.chip) {
  border-color: var(--ember);
  background: var(--ember-soft);
  color: var(--ember);
}

.interest-picker__hint,
.interest-picker__empty {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.interest-picker__error {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-3);
  color: var(--alert);
  font-size: var(--text-sm);
}
</style>
