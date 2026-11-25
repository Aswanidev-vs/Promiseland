<script lang="ts">
// Module scope on purpose: one counter per module keeps generated ids unique
// across instances. A counter inside <script setup> resets per instance, which
// gives every field the same id and makes labels point at the wrong control.
let sequence = 0

function nextId(prefix: string): string {
  sequence += 1
  return `${prefix}-${sequence}`
}
</script>

<script setup lang="ts">
const props = defineProps<{
  modelValue: boolean
  loading?: boolean
}>()

const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

const uid = nextId('discoverability')
const labelId = `${uid}-label`
const hintId = `${uid}-hint`

function toggle(): void {
  if (!props.loading) emit('update:modelValue', !props.modelValue)
}
</script>

<template>
  <div class="discoverability">
    <div class="discoverability__copy">
      <span :id="labelId" class="discoverability__label">Discoverable</span>
      <span :id="hintId" class="discoverability__hint">
        Let people with shared interests find you.
      </span>
    </div>
    <button
      type="button"
      class="discoverability__switch"
      :class="{ 'discoverability__switch--on': modelValue }"
      role="switch"
      :aria-checked="modelValue"
      :aria-labelledby="labelId"
      :aria-describedby="hintId"
      :disabled="loading"
      :aria-busy="loading || undefined"
      @click="toggle"
    >
      <span class="discoverability__track" aria-hidden="true">
        <span class="discoverability__knob" />
      </span>
      <span class="discoverability__state">{{ modelValue ? 'On' : 'Off' }}</span>
    </button>
  </div>
</template>

<style scoped>
.discoverability {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-4);
  min-width: 0;
  padding: var(--space-3) var(--space-4);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-md);
  background: var(--ink-850);
}

.discoverability__copy {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  min-width: 0;
}

.discoverability__label {
  color: var(--text-primary);
  font-size: var(--text-sm);
  font-weight: 500;
}

.discoverability__hint {
  max-width: 32ch;
  color: var(--text-secondary);
  font-size: var(--text-xs);
  line-height: 1.4;
}

.discoverability__switch {
  display: inline-flex;
  flex: none;
  align-items: center;
  gap: var(--space-3);
  min-width: 44px;
  min-height: 44px;
  border: 0;
  border-radius: var(--radius-md);
  background: transparent;
  cursor: pointer;
}

.discoverability__switch:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.discoverability__track {
  display: flex;
  align-items: center;
  width: var(--space-10);
  height: var(--space-5);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-pill);
  background: var(--ink-600);
  transition: background-color var(--dur-fast) var(--ease);
}

.discoverability__switch--on .discoverability__track {
  border-color: var(--ember);
  background: var(--ember-soft);
}

.discoverability__knob {
  width: var(--space-4);
  height: var(--space-4);
  border-radius: var(--radius-pill);
  background: var(--text-secondary);
  transition: transform var(--dur-fast) var(--ease);
}

.discoverability__switch--on .discoverability__knob {
  background: var(--ember);
  transform: translateX(var(--space-5));
}

.discoverability__state {
  color: var(--text-secondary);
  font-size: var(--text-sm);
  font-weight: 500;
}
</style>
