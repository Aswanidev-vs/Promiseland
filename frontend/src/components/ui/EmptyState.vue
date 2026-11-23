<script setup lang="ts">
import WayfinderMark from './WayfinderMark.vue'
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

let sequence = 0
const titleId = `empty-title-${++sequence}`
</script>

<template>
  <section class="empty" :aria-labelledby="titleId">
    <WayfinderMark size="96px" />
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
