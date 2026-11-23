<script setup lang="ts">
import { computed, ref, watch } from 'vue'

const props = withDefaults(
  defineProps<{
    name: string
    src?: string | null
    size?: 'sm' | 'md' | 'lg'
  }>(),
  { src: null, size: 'md' },
)

const failed = ref(false)
watch(() => props.src, () => (failed.value = false))

const initials = computed(() => {
  const parts = props.name.trim().split(/\s+/).filter(Boolean)
  if (parts.length === 0) return '?'
  return (parts[0]![0]! + (parts.length > 1 ? parts[parts.length - 1]![0]! : '')).toUpperCase()
})
</script>

<template>
  <span class="avatar" :class="`avatar--${size}`">
    <img v-if="src && !failed" :src="src" :alt="name" @error="failed = true" />
    <span v-else class="avatar__initials" role="img" :aria-label="name">{{ initials }}</span>
  </span>
</template>

<style scoped>
.avatar {
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-pill);
  background: var(--ink-700);
  color: var(--text-secondary);
}

.avatar--sm {
  width: 28px;
  height: 28px;
  font-size: var(--text-xs);
}

.avatar--md {
  width: 40px;
  height: 40px;
  font-size: var(--text-sm);
}

.avatar--lg {
  width: 64px;
  height: 64px;
  font-size: var(--text-lg);
}

.avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.avatar__initials {
  font-family: var(--font-display);
  font-weight: 600;
}
</style>
