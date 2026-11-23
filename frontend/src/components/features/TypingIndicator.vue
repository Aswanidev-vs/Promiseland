<script setup lang="ts">
import { computed } from 'vue'
import { plural } from './format'

const props = withDefaults(defineProps<{ names?: string[] }>(), { names: () => [] })

const text = computed(() => {
  const names = props.names.filter(Boolean)
  if (!names.length) return ''
  if (names.length === 1) return `${names[0]} is typing`
  if (names.length === 2) return `${names[0]} and ${names[1]} are typing`
  return `${names.slice(0, 2).join(', ')} and ${plural(names.length - 2, 'other')} are typing`
})
</script>

<template>
  <p v-if="text" class="typing" aria-live="polite">
    <span class="typing__dots" aria-hidden="true">
      <span />
      <span />
      <span />
    </span>
    <span class="typing__text">{{ text }}</span>
  </p>
</template>

<style scoped>
.typing {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  min-height: 24px;
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.typing__dots {
  display: inline-flex;
  gap: 3px;
}

.typing__dots span {
  width: 4px;
  height: 4px;
  border-radius: var(--radius-pill);
  background: var(--tide);
  animation: typing-blink 1200ms var(--ease) infinite;
}

.typing__dots span:nth-child(2) {
  animation-delay: 160ms;
}

.typing__dots span:nth-child(3) {
  animation-delay: 320ms;
}

@keyframes typing-blink {
  0%,
  60%,
  100% {
    opacity: 0.35;
  }
  30% {
    opacity: 1;
  }
}
</style>
