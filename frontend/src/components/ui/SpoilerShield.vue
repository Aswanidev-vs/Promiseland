<script setup lang="ts">
import { ref } from 'vue'
import AppIcon from './AppIcon.vue'

withDefaults(
  defineProps<{
    label?: string
    hint?: string
  }>(),
  { label: '', hint: '' },
)

const revealed = ref(false)
</script>

<template>
  <div class="spoiler" :class="{ 'spoiler--revealed': revealed }">
    <button
      type="button"
      class="spoiler__toggle"
      :aria-expanded="revealed"
      @click="revealed = !revealed"
    >
      <AppIcon name="eye" :size="18" />
      <span>{{ revealed ? 'Hide spoiler' : label || 'Reveal spoiler' }}</span>
    </button>
    <p v-if="!revealed" class="spoiler__hint">
      {{ hint || 'This post may give away the story. Reveal when you are ready.' }}
    </p>
    <div v-else class="spoiler__content">
      <slot />
    </div>
  </div>
</template>

<style scoped>
.spoiler {
  border: 1px dashed var(--alert);
  border-radius: var(--radius-md);
  background: var(--ink-850);
  padding: var(--space-4);
}

.spoiler--revealed {
  border-style: solid;
  border-color: var(--border-hairline);
}

.spoiler__toggle {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  min-height: 44px;
  padding: 0;
  border: 0;
  background: none;
  color: var(--ember);
  font-weight: 500;
  cursor: pointer;
}

.spoiler__toggle:hover {
  color: var(--ember-strong);
}

.spoiler__hint {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.spoiler__content {
  padding-top: var(--space-3);
  color: var(--text-primary);
}
</style>
