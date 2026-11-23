<script setup lang="ts">
withDefaults(
  defineProps<{
    rows?: number
    avatar?: boolean
  }>(),
  { rows: 3, avatar: false },
)
</script>

<template>
  <div class="skeleton-list" role="status" aria-label="Loading content" aria-busy="true">
    <span class="sr-only">Loading</span>
    <ul class="skeleton-list__items">
      <li v-for="row in rows" :key="row" class="skeleton-list__row">
        <span v-if="avatar" class="skeleton-list__avatar skeleton-bar" aria-hidden="true" />
        <span class="skeleton-list__lines" aria-hidden="true">
          <span class="skeleton-bar" :style="{ width: `${70 + (row % 3) * 10}%` }" />
          <span class="skeleton-bar" :style="{ width: `${45 + (row % 4) * 8}%` }" />
        </span>
      </li>
    </ul>
  </div>
</template>

<style scoped>
.skeleton-list__items {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.skeleton-list__row {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  padding: var(--space-4);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-lg);
  background: var(--ink-800);
}

.skeleton-list__avatar {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-pill);
}

.skeleton-list__lines {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: var(--space-2);
}
</style>
