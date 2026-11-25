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
import BaseButton from './BaseButton.vue'

withDefaults(
  defineProps<{
    title: string
    description?: string
    actionLabel?: string
  }>(),
  { description: '', actionLabel: '' },
)

const emit = defineEmits<{ action: [] }>()

const titleId = nextId('empty-title')
</script>

<template>
  <section class="empty" :aria-labelledby="titleId">
    <img class="brand-mark" src="/logo.png" alt="" width="72" height="72" />
    <div class="empty__copy">
      <h2 :id="titleId" class="empty__title">{{ title }}</h2>
      <p v-if="description" class="empty__description">{{ description }}</p>
    </div>
    <BaseButton v-if="actionLabel" variant="primary" @click="emit('action')">
      {{ actionLabel }}
    </BaseButton>
    <slot />
  </section>
</template>

<style scoped>
.empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-5);
  padding: var(--space-12) var(--space-6);
  border: 1px dashed var(--border-hairline);
  border-radius: var(--radius-lg);
  background: var(--ink-850);
  text-align: center;
}

.empty__copy {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  max-width: 42ch;
}

.empty__title {
  font-size: var(--text-xl);
}

.empty__description {
  color: var(--text-secondary);
}
</style>
