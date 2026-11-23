<script setup lang="ts">
import BaseButton from '@/components/ui/BaseButton.vue'
import WayfinderMark from '@/components/ui/WayfinderMark.vue'
</script>

<template>
  <section class="not-found" aria-labelledby="not-found-title">
    <div class="not-found__chart">
      <WayfinderMark detailed size="min(100%, 260px)" />
    </div>
    <p class="not-found__code mono">404 — off the chart</p>
    <h1 id="not-found-title">This bearing leads nowhere</h1>
    <p class="not-found__copy">
      The page you were looking for is not here. Your way back to the harbour is just below.
    </p>
    <BaseButton variant="primary" to="/">Return home</BaseButton>
  </section>
</template>

<style scoped>
.not-found {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-4);
  padding-block: var(--space-16) var(--space-20);
  text-align: center;
}

.not-found__chart {
  display: grid;
  width: min(100%, 300px);
  padding: var(--space-4);
  place-items: center;
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-lg);
  background: var(--ink-850);
}

.not-found__code {
  color: var(--text-muted);
  font-size: var(--text-xs);
}

.not-found h1 {
  font-size: var(--text-2xl);
}

.not-found__copy {
  max-width: 42ch;
  color: var(--text-secondary);
}

@media (max-width: 767px) {
  .not-found {
    padding-block: var(--space-10) var(--space-16);
  }
}
</style>
