<script setup lang="ts">
const props = defineProps<{
  modelValue: boolean
  loading?: boolean
}>()

const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

let sequence = 0
const uid = `discoverability-${++sequence}`
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
  justify-content: center;
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
  width: 40px;
  height: 22px;
  padding: 2px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-pill);
  background: var(--ink-600);
  transition: background-color var(--dur-fast) var(--ease);
}

.discoverability__switch--on .discoverability__track {
  border-color: var(--tide);
  background: var(--tide);
}

.discoverability__knob {
  width: 16px;
  height: 16px;
  border-radius: var(--radius-pill);
  background: var(--text-secondary);
  transition: transform var(--dur-fast) var(--ease);
}

.discoverability__switch--on .discoverability__knob {
  background: var(--ink-900);
  transform: translateX(18px);
}
</style>
