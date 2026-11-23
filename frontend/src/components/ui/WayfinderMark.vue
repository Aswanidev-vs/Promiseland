<script setup lang="ts">
/**
 * The one signature motif: a lantern-lit harbour wayfinder — compass rose,
 * constellation and an ember bearing. Used in the brand, empty states,
 * the landing hero and the 404. Decorative by default.
 */
withDefaults(
  defineProps<{
    detailed?: boolean
    title?: string
    size?: string
  }>(),
  {
    detailed: false,
    title: '',
    size: '48px',
  },
)

const ticks = Array.from({ length: 24 }, (_, index) => {
  const angle = (index * Math.PI) / 12
  const major = index % 6 === 0
  const inner = major ? 76 : 80
  const outer = 86
  return {
    x1: 100 + Math.cos(angle) * inner,
    y1: 100 + Math.sin(angle) * inner,
    x2: 100 + Math.cos(angle) * outer,
    y2: 100 + Math.sin(angle) * outer,
    major,
  }
})

const constellation = [
  { x: 52, y: 62, ember: false },
  { x: 76, y: 44, ember: false },
  { x: 132, y: 52, ember: true },
  { x: 158, y: 86, ember: false },
  { x: 148, y: 140, ember: false },
  { x: 108, y: 162, ember: false },
  { x: 58, y: 146, ember: false },
  { x: 42, y: 104, ember: false },
]

const constellationPath = `${constellation.map((node) => `${node.x},${node.y}`).join(' L ')} L ${constellation[0]!.x},${constellation[0]!.y}`
</script>

<template>
  <svg
    class="wayfinder"
    :class="{ 'wayfinder--detailed': detailed }"
    :style="{ width: size, height: size }"
    viewBox="0 0 200 200"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
    :role="title ? 'img' : undefined"
    :aria-label="title || undefined"
    :aria-hidden="title ? undefined : 'true'"
    focusable="false"
  >
    <circle cx="100" cy="100" r="88" stroke="var(--ink-600)" stroke-width="1" />
    <circle
      cx="100"
      cy="100"
      r="72"
      stroke="var(--border-strong)"
      stroke-width="1"
      stroke-dasharray="1 7"
      stroke-linecap="round"
    />
    <circle cx="100" cy="100" r="31" stroke="var(--border-hairline)" stroke-width="1" />

    <template v-if="detailed">
      <path :d="`M ${constellationPath}`" stroke="var(--tide)" stroke-opacity="0.42" stroke-width="1" />
      <circle
        v-for="(node, index) in constellation"
        :key="index"
        :cx="node.x"
        :cy="node.y"
        :r="node.ember ? 3.5 : 2.2"
        :fill="node.ember ? 'var(--ember)' : 'var(--tide)'"
      />
      <text x="100" y="17" text-anchor="middle" class="wayfinder__label">N</text>
      <text x="186" y="104" text-anchor="middle" class="wayfinder__label">E</text>
      <text x="100" y="193" text-anchor="middle" class="wayfinder__label">S</text>
      <text x="14" y="104" text-anchor="middle" class="wayfinder__label">W</text>
    </template>

    <line
      v-for="(tick, index) in ticks"
      :key="`tick-${index}`"
      :x1="tick.x1"
      :y1="tick.y1"
      :x2="tick.x2"
      :y2="tick.y2"
      :stroke="tick.major ? 'var(--text-muted)' : 'var(--ink-600)'"
      stroke-width="1"
    />

    <!-- Four-point rose, then the ember bearing that makes it a wayfinder -->
    <path d="M100 36 L105 95 L100 100 L95 95 Z" fill="var(--tide)" />
    <path d="M100 164 L95 105 L100 100 L105 105 Z" fill="var(--ink-600)" />
    <path d="M164 100 L105 95 L100 100 L105 105 Z" fill="var(--ink-700)" />
    <path d="M36 100 L95 95 L100 100 L95 105 Z" fill="var(--ink-700)" />
    <path d="M100 100 L142 58 L109 91 Z" fill="var(--ember)" />
    <path d="M100 100 L58 142 L91 109 Z" fill="var(--ink-600)" />
    <circle cx="100" cy="100" r="4" fill="var(--ink-900)" stroke="var(--ember-strong)" stroke-width="1.5" />
  </svg>
</template>

<style scoped>
.wayfinder {
  flex: none;
}

.wayfinder__label {
  font-family: var(--font-mono);
  font-size: 8px;
  letter-spacing: 0.08em;
  fill: var(--text-muted);
}
</style>
