<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import AppIcon from '@/components/ui/AppIcon.vue'
import type { IconName } from '@/components/ui/icons'

interface Tab {
  label: string
  icon: IconName
  to: string
  /** Icon-only tab, used to fit Chat and Discover into the tab bar without overflow. */
  compact?: boolean
}

const auth = useAuthStore()

const profileTo = computed(() =>
  auth.user?.username ? `/u/${encodeURIComponent(auth.user.username)}` : '/login',
)

const tabs = computed<Tab[]>(() => [
  { label: 'Home', icon: 'home', to: '/home' },
  { label: 'Communities', icon: 'communities', to: '/communities' },
  { label: 'Chat', icon: 'chat', to: '/chat', compact: true },
  { label: 'Discover', icon: 'discover', to: '/discover', compact: true },
  { label: 'Events', icon: 'events', to: '/events' },
  { label: 'Notifications', icon: 'bell', to: '/notifications' },
  { label: 'Profile', icon: 'profile', to: profileTo.value },
])
</script>

<template>
  <nav class="mobile-nav" aria-label="Primary mobile">
    <ul class="mobile-nav__list">
      <li v-for="tab in tabs" :key="tab.label" class="mobile-nav__item">
        <RouterLink
          :to="tab.to"
          class="mobile-nav__tab"
          :class="{ 'mobile-nav__tab--compact': tab.compact }"
          active-class="mobile-nav__tab--active"
          :aria-label="tab.label"
          :title="tab.compact ? tab.label : undefined"
        >
          <AppIcon :name="tab.icon" :size="22" />
          <span>{{ tab.label }}</span>
        </RouterLink>
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
  /* Seven equal cells: 375px / 7 ≈ 53px, still above the 44px touch floor. */
  grid-template-columns: repeat(7, minmax(0, 1fr));
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
  overflow: hidden;
}

.mobile-nav__tab span {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mobile-nav__tab--compact span {
  display: none;
}

a.mobile-nav__tab:hover {
  color: var(--text-primary);
}

.mobile-nav__tab--active {
  color: var(--ember);
}
</style>
