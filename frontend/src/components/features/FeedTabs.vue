<script setup lang="ts">
export interface FeedTab {
  value: string
  label: string
}

const props = withDefaults(
  defineProps<{
    modelValue: string
    tabs: FeedTab[]
    label?: string
  }>(),
  { label: 'Feed sections' },
)

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

function select(value: string): void {
  if (value !== props.modelValue) emit('update:modelValue', value)
}
</script>

<template>
  <div class="tabs" role="tablist" :aria-label="label">
    <button
      v-for="tab in tabs"
      :key="tab.value"
      type="button"
      role="tab"
      class="tabs__tab"
      :class="{ 'tabs__tab--active': tab.value === modelValue }"
      :aria-selected="tab.value === modelValue"
      @click="select(tab.value)"
    >
      {{ tab.label }}
    </button>
  </div>
</template>

<style scoped>
.tabs {
  display: flex;
  gap: var(--space-1);
  overflow-x: auto;
  border-bottom: 1px solid var(--border-hairline);
}

.tabs__tab {
  position: relative;
  flex: none;
  min-height: 44px;
  padding: 0 var(--space-4);
  border: 0;
  background: none;
  color: var(--text-secondary);
  cursor: pointer;
  font-size: var(--text-base);
  font-weight: 500;
  white-space: nowrap;
}

.tabs__tab:hover {
  color: var(--text-primary);
}

.tabs__tab--active {
  color: var(--text-primary);
}

.tabs__tab--active::after {
  position: absolute;
  right: var(--space-3);
  bottom: -1px;
  left: var(--space-3);
  height: 2px;
  border-radius: var(--radius-pill);
  background: var(--ember);
  content: '';
}
</style>
