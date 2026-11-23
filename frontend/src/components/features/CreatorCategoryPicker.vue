<script setup lang="ts">
import { computed } from 'vue'
import TagChip from '@/components/ui/TagChip.vue'
import { CREATOR_CATEGORIES } from './formatR2'

/** Single- or multi-select chips for creator/team-up categories. */
const props = withDefaults(
  defineProps<{
    modelValue: string | string[]
    multiple?: boolean
    label?: string
    hint?: string
    includeNone?: boolean
    noneValue?: string
    noneLabel?: string
  }>(),
  {
    multiple: false,
    label: 'Creator category',
    hint: '',
    includeNone: true,
    noneValue: '',
    noneLabel: 'None',
  },
)

const emit = defineEmits<{ 'update:modelValue': [value: string | string[]] }>()

const selected = computed(() =>
  Array.isArray(props.modelValue) ? props.modelValue : [props.modelValue],
)

function isOn(value: string): boolean {
  return selected.value.includes(value)
}

function toggle(value: string): void {
  if (props.multiple) {
    const current = Array.isArray(props.modelValue) ? props.modelValue : []
    emit('update:modelValue', isOn(value) ? current.filter((item) => item !== value) : [...current, value])
    return
  }
  emit('update:modelValue', props.modelValue === value ? props.noneValue : value)
}
</script>

<template>
  <fieldset class="category-picker">
    <legend class="category-picker__label">{{ label }}</legend>
    <div class="category-picker__options">
      <button
        v-if="includeNone"
        type="button"
        class="category-picker__option"
        :aria-pressed="isOn(noneValue)"
        @click="toggle(noneValue)"
      >
        <TagChip :label="noneLabel" :selected="isOn(noneValue)" />
      </button>
      <button
        v-for="item in CREATOR_CATEGORIES"
        :key="item.value"
        type="button"
        class="category-picker__option"
        :aria-pressed="isOn(item.value)"
        @click="toggle(item.value)"
      >
        <TagChip :label="item.label" :selected="isOn(item.value)" />
      </button>
    </div>
    <p v-if="hint" class="category-picker__hint">{{ hint }}</p>
  </fieldset>
</template>

<style scoped>
.category-picker {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  margin: 0;
  padding: 0;
  border: 0;
}

.category-picker__label {
  padding: 0;
  color: var(--text-primary);
  font-size: var(--text-sm);
  font-weight: 500;
}

.category-picker__options {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.category-picker__option {
  min-height: 36px;
  padding: 0;
  border: 0;
  background: none;
  cursor: pointer;
}

.category-picker__option:focus-within {
  outline: 2px solid var(--ember);
  outline-offset: 2px;
  border-radius: var(--radius-sm);
}

.category-picker__hint {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}
</style>
