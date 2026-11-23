<script setup lang="ts">
import { RouterLink } from 'vue-router'
import WayfinderMark from '@/components/ui/WayfinderMark.vue'
import AppIcon from '@/components/ui/AppIcon.vue'
import type { IconName } from '@/components/ui/icons'

interface NavItem {
  label: string
  icon: IconName
  to?: string
}

// R1/R2/R3: give these items their real `to` values when the routes land.
const items: NavItem[] = [
  { label: 'Home', icon: 'home', to: '/' },
  { label: 'Communities', icon: 'communities' },
  { label: 'Events', icon: 'events' },
  { label: 'Chat', icon: 'chat' },
  { label: 'Discover', icon: 'discover' },
  { label: 'Notifications', icon: 'bell' },
  { label: 'Profile', icon: 'profile' },
]
</script>

<template>
  <div class="sidebar">
    <RouterLink to="/" class="sidebar__brand">
      <WayfinderMark size="38px" />
      <span class="sidebar__wordmark">PromisedLand</span>
    </RouterLink>

    <nav aria-label="Main">
      <ul class="sidebar__nav">
        <li v-for="item in items" :key="item.label">
          <RouterLink
            v-if="item.to"
            :to="item.to"
            class="sidebar__item"
            active-class="sidebar__item--active"
          >
            <AppIcon :name="item.icon" :size="20" />
            <span class="sidebar__label">{{ item.label }}</span>
          </RouterLink>
          <span
            v-else
            class="sidebar__item sidebar__item--soon"
            aria-disabled="true"
            :title="`${item.label} arrives in a later release`"
            :aria-label="`${item.label}, arriving in a later release`"
          >
            <AppIcon :name="item.icon" :size="20" />
            <span class="sidebar__label">{{ item.label }}</span>
            <span class="sidebar__soon" aria-hidden="true">Soon</span>
          </span>
        </li>
      </ul>
    </nav>

    <p class="sidebar__note">
      <span class="sidebar__note-dot" aria-hidden="true" />
      A harbour for anime fans
    </p>
  </div>
</template>

<style scoped>
.sidebar {
  display: flex;
  flex-direction: column;
  height: 100%;
  gap: var(--space-8);
}

.sidebar__brand {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2);
  color: var(--text-primary);
  text-decoration: none;
}

.sidebar__brand:hover {
  color: var(--text-primary);
}

.sidebar__wordmark {
  font-family: var(--font-display);
  font-size: var(--text-lg);
  font-weight: 600;
  letter-spacing: -0.02em;
}

.sidebar__nav {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.sidebar__item {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  min-height: 44px;
  padding: 0 var(--space-3);
  border-radius: var(--radius-md);
  color: var(--text-secondary);
  text-decoration: none;
  transition:
    background-color var(--dur-fast) var(--ease),
    color var(--dur-fast) var(--ease);
}

a.sidebar__item:hover {
  background: var(--ink-800);
  color: var(--text-primary);
}

.sidebar__item--active {
  background: var(--ink-800);
  color: var(--ember);
}

.sidebar__item--soon {
  cursor: default;
  opacity: 0.75;
}

.sidebar__label {
  flex: 1;
  font-size: var(--text-sm);
}

.sidebar__soon {
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--text-muted);
}

.sidebar__note {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-top: auto;
  padding: 0 var(--space-3);
  color: var(--text-muted);
  font-family: var(--font-mono);
  font-size: var(--text-xs);
}

.sidebar__note-dot {
  width: 6px;
  height: 6px;
  border-radius: var(--radius-pill);
  background: var(--ember);
}

@media (min-width: 768px) and (max-width: 1023px) {
  .sidebar {
    align-items: center;
  }

  .sidebar__brand {
    padding: var(--space-1);
  }

  .sidebar__wordmark,
  .sidebar__label,
  .sidebar__soon,
  .sidebar__note {
    display: none;
  }

  .sidebar__item {
    justify-content: center;
    width: 48px;
    padding: 0;
  }
}
</style>
