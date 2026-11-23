<script setup lang="ts">
/**
 * Chat-specific glyphs extend the R0 icon set with the same 24px grid and
 * 1.5px stroke. AppIcon is left untouched; base glyphs (search, close, ...)
 * are reused from icons.ts so chat never invents a second visual language.
 */
import { iconPaths } from '@/components/ui/icons'

const chatPaths = {
  hash: ['M9 4.5 7 19.5', 'M17 4.5l-2 15', 'M4.5 9h15', 'M3.5 15h15'],
  like: [
    'M7 10.5v9H4.3v-9H7Z',
    'M7 10.5 10.6 4.4c1.7 0 2.5 1.2 2.1 2.8L12 10h6.1c1.4 0 2.3 1.2 2 2.6l-1.6 5.5c-.3 1-1.2 1.7-2.2 1.7H7',
  ],
  heart: ['M12 20s-7.4-4.5-7.4-9.6A4.2 4.2 0 0 1 12 7.5a4.2 4.2 0 0 1 7.4 2.9C19.4 15.5 12 20 12 20Z'],
  star: ['m12 3.8 2.6 5.3 5.8.9-4.2 4.1 1 5.8-5.2-2.8-5.2 2.8 1-5.8L3.6 10l5.8-.9L12 3.8Z'],
  laugh: [
    'M12 3.8a8.2 8.2 0 1 0 0 16.4 8.2 8.2 0 0 0 0-16.4Z',
    'M8.7 14c.8 1.3 2 2 3.3 2s2.5-.7 3.3-2',
    'M9.2 9.5h.01',
    'M14.8 9.5h.01',
  ],
  reply: ['M9.5 8.5 5.5 12.5l4 4', 'M5.5 12.5h8a5.5 5.5 0 0 1 5.5 5.5'],
  report: ['M6 4.5v15', 'M6 5.5h10l-1.8 3.5L16 12.5H6'],
  trash: ['M4.5 7h15', 'M9.5 7V4.5h5V7', 'M6.5 7l.8 12.5h9.4L17.5 7', 'M10.5 10v6', 'M13.5 10v6'],
  send: ['M4.5 12 20 4.5 15.5 20 11 14.5 4.5 12Z', 'M11 14.5 20 4.5'],
  plus: ['M12 5v14', 'M5 12h14'],
  back: ['M15 5 8 12l7 7'],
  group: [
    'M9 11.2a3.2 3.2 0 1 0 0-6.4 3.2 3.2 0 0 0 0 6.4Z',
    'M3.2 19c0-3 2.6-4.8 5.8-4.8s5.8 1.8 5.8 4.8',
    'M15.4 5.2a3.2 3.2 0 0 1 0 6.2',
    'M16.4 14.6c2.4.4 4 1.9 4 4.4',
  ],
  warning: ['M12 4 3 20h18L12 4Z', 'M12 10v4.5', 'M12 17.5h.01'],
  offline: ['M3 3l18 18', 'M5 12.5a11 11 0 0 1 4-2.3', 'M2 9a15 15 0 0 1 4-2.2', 'M9 16a4 4 0 0 1 5.5-.4', 'M12 19.5h.01'],
  members: ['M12 11.5a4 4 0 1 0 0-8 4 4 0 0 0 0 8Z', 'M4.5 20c0-3.6 3.2-6 7.5-6s7.5 2.4 7.5 6'],
  check: ['m5 12.5 4.5 4.5L19 7.5'],
} as const

const paths = { ...iconPaths, ...chatPaths }
export type ChatIconName = keyof typeof paths

const props = withDefaults(defineProps<{ name: ChatIconName; size?: number | string }>(), { size: 20 })
</script>

<template>
  <svg
    class="chat-icon"
    :width="props.size"
    :height="props.size"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="1.5"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
    focusable="false"
  >
    <path v-for="(path, index) in paths[props.name]" :key="index" :d="path" />
  </svg>
</template>

<style scoped>
.chat-icon {
  flex: none;
}
</style>
