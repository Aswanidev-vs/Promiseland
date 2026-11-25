<script setup lang="ts">
import { computed, ref } from 'vue'
import AppIcon from './AppIcon.vue'

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
    revealable?: boolean
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
    revealable: false,
  },
)

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const generatedId = `pl-input-${++sequence}`
const inputId = computed(() => props.id || generatedId)
const hintId = computed(() => (props.error ? `${inputId.value}-error` : props.hint ? `${inputId.value}-hint` : ''))
const describedBy = computed(() => hintId.value || undefined)

const revealed = ref(false)
const inputType = computed(() => (props.revealable && revealed.value ? 'text' : props.type))

function onInput(event: Event): void {
  emit('update:modelValue', (event.target as HTMLInputElement).value)
}
</script>

<template>
  <div class="field" :class="{ 'field--invalid': !!error }">
    <div v-if="label || $slots['label-action']" class="field__label-row">
      <label v-if="label" class="field__label" :for="inputId">{{ label }}</label>
      <slot name="label-action" />
    </div>

    <div class="field__control">
      <input
        :id="inputId"
        class="field__input"
        :class="{ 'field__input--action': revealable }"
        :type="inputType"
        :name="name || undefined"
        :value="modelValue"
        :placeholder="placeholder || undefined"
        :autocomplete="autocomplete || undefined"
        :disabled="disabled"
        :aria-invalid="!!error || undefined"
        :aria-describedby="describedBy"
        @input="onInput"
      />
      <button
        v-if="revealable"
        type="button"
        class="field__reveal"
        :aria-label="revealed ? 'Hide password' : 'Show password'"
        :aria-pressed="revealed"
        @click="revealed = !revealed"
      >
        <AppIcon :name="revealed ? 'eye-off' : 'eye'" :size="18" />
      </button>
    </div>

    <p v-if="error" :id="hintId" class="field__message field__message--error">
      <AppIcon name="alert" :size="16" />
      <span>{{ error }}</span>
    </p>
    <p v-else-if="hint" :id="hintId" class="field__message">{{ hint }}</p>
  </div>
</template>

<style scoped>
.field {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.field__label-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-4);
}

.field__label {
  color: var(--text-primary);
  font-size: var(--text-sm);
  font-weight: 500;
}

.field__control {
  position: relative;
}

.field__input {
  width: 100%;
  min-height: 44px;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  background: var(--ink-850);
  color: var(--text-primary);
  transition:
    border-color var(--dur-fast) var(--ease),
    box-shadow var(--dur-fast) var(--ease);
}

.field__input--action {
  padding-right: calc(var(--space-3) * 2 + 18px);
}

.field__input::placeholder {
  color: var(--text-secondary);
  opacity: 0.75;
}

.field__input:hover:not(:disabled) {
  border-color: var(--ink-600);
}

.field__input:focus {
  border-color: var(--ember);
  box-shadow: 0 0 0 3px var(--ember-soft);
}

.field__input:focus-visible {
  outline: none;
}

.field__input:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.field--invalid .field__input {
  border-color: var(--alert);
}

.field__reveal {
  position: absolute;
  top: 0;
  right: 0;
  display: grid;
  width: 44px;
  height: 100%;
  place-items: center;
  border: 0;
  border-radius: 0 var(--radius-md) var(--radius-md) 0;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  transition: color var(--dur-fast) var(--ease);
}

.field__reveal:hover {
  color: var(--text-primary);
}

.field__message {
  display: flex;
  align-items: flex-start;
  gap: var(--space-2);
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.field__message--error {
  color: var(--alert);
}

.field__message :deep(.app-icon) {
  margin-top: 2px;
}
</style>
