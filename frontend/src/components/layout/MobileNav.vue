<script setup lang="ts">
import { RouterLink } from 'vue-router'
import AppIcon from '@/components/ui/AppIcon.vue'
import type { IconName } from '@/components/ui/icons'

// R1/R2: attach real `to` values as these routes ship.
interface Tab {
  label: string
  icon: IconName
  to?: string
}

const tabs: Tab[] = [
  { label: 'Home', icon: 'home', to: '/' },
  { label: 'Communities', icon: 'communities' },
  { label: 'Events', icon: 'events' },
  { label: 'Notifications', icon: 'bell' },
  { label: 'Profile', icon: 'profile' },
]
</script>

<template>
  <nav class="mobile-nav" aria-label="Primary mobile">
    <ul class="mobile-nav__list">
      <li v-for="tab in tabs" :key="tab.label" class="mobile-nav__item">
        <RouterLink
          v-if="tab.to"
          :to="tab.to"
          class="mobile-nav__tab"
          active-class="mobile-nav__tab--active"
        >
          <AppIcon :name="tab.icon" :size="22" />
          <span>{{ tab.label }}</span>
        </RouterLink>
        <span
          v-else
          class="mobile-nav__tab mobile-nav__tab--soon"
          aria-disabled="true"
          :title="`${tab.label} arrives in a later release`"
          :aria-label="`${tab.label}, arriving in a later release`"
        >
          <AppIcon :name="tab.icon" :size="22" />
          <span>{{ tab.label }}</span>
        </span>
      </li>
    </ul>
  </nav>
</template>

<style scoped>
.mobile-nav {
  position: fixed;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 30;
  height: var(--bottomnav-h);
  border-top: 1px solid var(--border-hairline);
  background: var(--ink-850);
}

.mobile-nav__list {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  height: 100%;
}

.mobile-nav__item {
  min-width: 0;
}

.mobile-nav__tab {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 3px;
  width: 100%;
  height: 100%;
  color: var(--text-secondary);
  font-size: 10px;
  line-height: 1.1;
  text-align: center;
  text-decoration: none;
}

a.mobile-nav__tab:hover {
  color: var(--text-primary);
}

.mobile-nav__tab--active {
  color: var(--ember);
}

.mobile-nav__tab--soon {
  cursor: default;
  opacity: 0.65;
}
</style>
