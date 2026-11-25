<script setup lang="ts">
withDefaults(
  defineProps<{
    label: string
    tone?: 'default' | 'ember' | 'tide'
    selected?: boolean
  }>(),
  { tone: 'default', selected: false },
)
</script>

<template>
  <span
    class="chip"
    :class="[`chip--${tone}`, { 'chip--selected': selected }]"
  >
    <slot name="icon" />
    {{ label }}
  </span>
</template>

<style scoped>
.chip {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  min-height: 24px;
  padding: var(--space-1) var(--space-3);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-pill);
  background: var(--ink-700);
  color: var(--text-secondary);
  font-family: var(--font-mono);
  font-size: var(--text-xs);
  line-height: 1.3;
  white-space: nowrap;
}

.chip--tide {
  border-color: color-mix(in srgb, var(--tide) 30%, transparent);
  color: var(--tide);
}

.chip--ember {
  border-color: color-mix(in srgb, var(--ember) 32%, transparent);
  color: var(--ember);
}

.chip--selected {
  border-color: var(--ember);
  color: var(--ember-strong);
}
</style>
