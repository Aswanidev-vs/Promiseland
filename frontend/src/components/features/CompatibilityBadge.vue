<script setup lang="ts">
import { computed } from 'vue'
import TagChip from '@/components/ui/TagChip.vue'
import type { SharedInterests } from '@/types'

const props = defineProps<{
  explanation: string
  shared: SharedInterests
  sharedCount?: number
}>()

const groups = [
  { key: 'anime', label: 'Anime', tone: 'tide' },
  { key: 'genres', label: 'Genres', tone: 'default' },
  { key: 'interests', label: 'Interests', tone: 'default' },
  { key: 'communities', label: 'Communities', tone: 'default' },
] as const

const activeGroups = computed(() =>
  groups.filter((group) => (props.shared?.[group.key]?.length ?? 0) > 0),
)

const countLabel = computed(() => {
  const count = props.sharedCount ?? 0
  return `${count} ${count === 1 ? 'thing' : 'things'}`
})
</script>

<template>
  <section class="compat" aria-label="Why you are seeing this person">
    <div class="compat__head">
      <span class="compat__marker" aria-hidden="true" />
      <span class="compat__label">In common</span>
      <span v-if="sharedCount != null" class="compat__count mono">{{ countLabel }}</span>
    </div>

    <p class="compat__explanation">{{ explanation }}</p>

    <div v-if="activeGroups.length" class="compat__groups">
      <div v-for="group in activeGroups" :key="group.key" class="compat__group">
        <span class="compat__group-label">{{ group.label }}</span>
        <div class="compat__chips">
          <TagChip
            v-for="value in shared[group.key]"
            :key="value"
            :label="value"
            :tone="group.tone"
          />
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.compat {
  padding: var(--space-4);
  border: 1px solid var(--border-hairline);
  border-left: 2px solid var(--ink-600);
  border-radius: var(--radius-md);
  background: var(--ink-850);
}

.compat__head {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.compat__marker {
  width: 6px;
  height: 6px;
  border-radius: var(--radius-pill);
  background: var(--ink-600);
}

.compat__label {
  color: var(--text-secondary);
  font-size: var(--text-sm);
  font-weight: 500;
}

.compat__count {
  color: var(--text-secondary);
  font-size: var(--text-xs);
}

.compat__explanation {
  margin-top: var(--space-3);
  max-width: 58ch;
  color: var(--text-primary);
  font-size: var(--text-lg);
  font-weight: 500;
  letter-spacing: -0.01em;
  line-height: 1.5;
}

.compat__groups {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  margin-top: var(--space-4);
  padding-top: var(--space-4);
  border-top: 1px solid var(--border-hairline);
}

.compat__group {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: var(--space-2) var(--space-3);
}

.compat__group-label {
  flex: none;
  min-width: 84px;
  color: var(--text-muted);
  font-size: var(--text-xs);
}

.compat__chips {
  display: flex;
  flex: 1;
  flex-wrap: wrap;
  gap: var(--space-2);
  min-width: 0;
}

/* Long anime or community names must wrap instead of pushing the card wide. */
.compat__chips :deep(.chip) {
  max-width: 100%;
  white-space: normal;
  overflow-wrap: anywhere;
}

@media (max-width: 480px) {
  .compat {
    padding: var(--space-3);
  }

  .compat__explanation {
    font-size: var(--text-base);
  }

  .compat__group-label {
    min-width: 0;
    width: 100%;
  }
}
</style>
