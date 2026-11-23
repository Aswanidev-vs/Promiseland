<script setup lang="ts">
import { computed } from 'vue'

let sequence = 0

const props = withDefaults(
  defineProps<{
    modelValue?: string
    label?: string
    id?: string
    type?: string
    placeholder?: string
    hint?: string
    error?: string
    name?: string
    autocomplete?: string
    disabled?: boolean
  }>(),
  {
    modelValue: '',
    label: '',
    id: '',
    type: 'text',
    placeholder: '',
    hint: '',
    error: '',
    name: '',
    autocomplete: '',
    disabled: false,
  },
)

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const generatedId = `pl-input-${++sequence}`
const inputId = computed(() => props.id || generatedId)
const hintId = computed(() => (props.error ? `${inputId.value}-error` : props.hint ? `${inputId.value}-hint` : ''))
const describedBy = computed(() => hintId.value || undefined)

function onInput(event: Event): void {
  emit('update:modelValue', (event.target as HTMLInputElement).value)
}
</script>

<template>
  <div class="field" :class="{ 'field--invalid': !!error }">
    <label v-if="label" class="field__label" :for="inputId">{{ label }}</label>
    <input
      :id="inputId"
      class="field__input"
      :type="type"
      :name="name || undefined"
      :value="modelValue"
      :placeholder="placeholder || undefined"
      :autocomplete="autocomplete || undefined"
      :disabled="disabled"
      :aria-invalid="!!error || undefined"
      :aria-describedby="describedBy"
      @input="onInput"
    />
    <p v-if="error" :id="hintId" class="field__message field__message--error">{{ error }}</p>
    <p v-else-if="hint" :id="hintId" class="field__message">{{ hint }}</p>
  </div>
</template>

<style scoped>
.field {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.field__label {
  color: var(--text-primary);
  font-size: var(--text-sm);
  font-weight: 500;
}

.field__input {
  width: 100%;
  min-height: 44px;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  background: var(--ink-850);
  color: var(--text-primary);
  transition: border-color var(--dur-fast) var(--ease);
}

.field__input::placeholder {
  color: var(--text-muted);
}

.field__input:hover:not(:disabled) {
  border-color: var(--ink-600);
}

.field__input:focus {
  border-color: var(--ember);
  outline: none;
}

.field__input:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.field--invalid .field__input {
  border-color: var(--alert);
}

.field__message {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.field__message--error {
  color: var(--alert);
}
</style>
