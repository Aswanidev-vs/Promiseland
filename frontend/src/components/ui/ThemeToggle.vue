<script setup lang="ts">
import AppIcon from './AppIcon.vue'
import { useTheme } from '@/composables/useTheme'

const { theme, toggleTheme } = useTheme()
</script>

<template>
  <button
    type="button"
    class="theme-toggle"
    :aria-label="theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'"
    :title="theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'"
    @click="toggleTheme"
  >
    <AppIcon :name="theme === 'dark' ? 'sun' : 'moon'" :size="18" />
  </button>
</template>

<style scoped>
.theme-toggle {
  display: inline-flex;
  flex: none;
  width: 44px;
  height: 44px;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: 0;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  transition:
    background-color var(--dur-fast) var(--ease),
    color var(--dur-fast) var(--ease);
}

.theme-toggle:hover {
  background: var(--ink-800);
  color: var(--text-primary);
}
</style>
