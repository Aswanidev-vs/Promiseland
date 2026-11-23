<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{ status?: string; size?: 'sm' | 'md' }>(), {
  status: 'offline',
  size: 'sm',
})

const state = computed(() => {
  const value = (props.status || 'offline').toLowerCase()
  if (value === 'online') return 'online'
  if (value === 'away' || value === 'idle' || value === 'busy' || value === 'dnd') return 'away'
  return 'offline'
})

const label = computed(() => (state.value === 'online' ? 'Online' : state.value === 'away' ? 'Away' : 'Offline'))
</script>

<template>
  <span
    class="presence"
    :class="[`presence--${state}`, `presence--${size}`]"
    role="img"
    :aria-label="label"
    :title="label"
  />
</template>

<style scoped>
.presence {
  display: inline-block;
  flex: none;
  border-radius: var(--radius-pill);
  background: var(--ink-600);
  box-shadow: 0 0 0 2px var(--ink-800);
}

.presence--sm {
  width: 8px;
  height: 8px;
}

.presence--md {
  width: 10px;
  height: 10px;
}

.presence--online {
  background: var(--tide);
}

.presence--away {
  background: var(--text-muted);
}
</style>
