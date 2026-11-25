<script setup lang="ts">
withDefaults(
  defineProps<{
    interactive?: boolean
    sunken?: boolean
    padded?: boolean
  }>(),
  { interactive: false, padded: true, sunken: false },
)
</script>

<template>
  <article
    class="card"
    :class="{ 'card--interactive': interactive, 'card--flat': !padded, 'card--sunken': sunken }"
  >
    <slot />
  </article>
</template>

<style scoped>
.card {
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-lg);
  background: var(--ink-800);
  box-shadow: var(--shadow-1), var(--lift-top);
}

.card--sunken {
  background: var(--ink-850);
  box-shadow: none;
}

.card:not(.card--flat) {
  padding: var(--space-5);
}

.card--interactive {
  transition:
    border-color var(--dur-fast) var(--ease),
    box-shadow var(--dur-fast) var(--ease);
}

.card--interactive:hover {
  border-color: var(--border-strong);
  box-shadow: var(--shadow-2), var(--lift-top);
}
</style>
