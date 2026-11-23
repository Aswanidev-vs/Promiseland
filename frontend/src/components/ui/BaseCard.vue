<script setup lang="ts">
withDefaults(
  defineProps<{
    interactive?: boolean
    padded?: boolean
  }>(),
  { interactive: false, padded: true },
)
</script>

<template>
  <article class="card" :class="{ 'card--interactive': interactive, 'card--flat': !padded }">
    <slot />
  </article>
</template>

<style scoped>
.card {
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-lg);
  background: var(--ink-800);
  box-shadow: var(--shadow-1);
}

.card:not(.card--flat) {
  padding: var(--space-5);
}

.card--interactive {
  transition: border-color var(--dur-fast) var(--ease);
}

.card--interactive:hover {
  border-color: var(--border-strong);
}
</style>
