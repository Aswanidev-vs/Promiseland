<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'

const props = withDefaults(
  defineProps<{
    variant?: 'primary' | 'secondary' | 'ghost' | 'danger'
    size?: 'sm' | 'md' | 'lg'
    type?: 'button' | 'submit' | 'reset'
    disabled?: boolean
    block?: boolean
    loading?: boolean
    to?: string
    title?: string
  }>(),
  {
    variant: 'secondary',
    size: 'md',
    type: 'button',
    disabled: false,
    block: false,
    loading: false,
    to: '',
    title: '',
  },
)

const classes = computed(() => [
  'btn',
  `btn--${props.variant}`,
  `btn--${props.size}`,
  { 'btn--block': props.block, 'btn--loading': props.loading },
])
</script>

<template>
  <RouterLink
    v-if="to && !disabled"
    :to="to"
    class="btn"
    :class="classes"
    :title="title || undefined"
  >
    <slot />
  </RouterLink>
  <button
    v-else
    :type="type"
    class="btn"
    :class="classes"
    :disabled="disabled || loading"
    :aria-busy="loading || undefined"
    :title="title || undefined"
  >
    <span v-if="loading" class="btn__spinner" aria-hidden="true" />
    <slot />
  </button>
</template>

<style scoped>
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  min-height: 44px;
  padding: 0 var(--space-5);
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  font-family: var(--font-body);
  font-size: var(--text-base);
  font-weight: 500;
  line-height: 1.2;
  text-align: center;
  text-decoration: none;
  cursor: pointer;
  transition:
    background-color var(--dur-fast) var(--ease),
    border-color var(--dur-fast) var(--ease),
    color var(--dur-fast) var(--ease),
    opacity var(--dur-fast) var(--ease);
}

.btn--sm {
  padding-inline: var(--space-3);
  font-size: var(--text-sm);
}

.btn--lg {
  padding-inline: var(--space-6);
  font-size: var(--text-lg);
}

.btn--block {
  width: 100%;
}

.btn--primary {
  background: var(--ember);
  color: var(--ink-900);
}

.btn--primary:hover:not(:disabled) {
  background: var(--ember-strong);
}

.btn--secondary {
  border-color: var(--border-strong);
  background: transparent;
  color: var(--text-primary);
}

.btn--secondary:hover:not(:disabled) {
  background: var(--ink-700);
}

.btn--ghost {
  background: transparent;
  color: var(--text-secondary);
}

.btn--ghost:hover:not(:disabled) {
  background: var(--ink-800);
  color: var(--text-primary);
}

.btn--danger {
  background: var(--alert);
  color: var(--ink-900);
}

.btn--danger:hover:not(:disabled) {
  opacity: 0.9;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn__spinner {
  width: 14px;
  height: 14px;
  border: 2px solid currentColor;
  border-top-color: transparent;
  border-radius: 50%;
  animation: btn-spin 700ms linear infinite;
}

@keyframes btn-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
