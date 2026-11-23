<script setup lang="ts">
import { iconPaths, type IconName } from './icons'

withDefaults(
  defineProps<{
    name: IconName
    size?: number | string
  }>(),
  { size: 20 },
)
</script>

<template>
  <svg
    class="app-icon"
    :width="size"
    :height="size"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="1.5"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
    focusable="false"
  >
    <path v-for="(path, index) in iconPaths[name]" :key="index" :d="path" />
  </svg>
</template>

<style scoped>
.app-icon {
  flex: none;
}
</style>
