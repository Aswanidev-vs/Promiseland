<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useGuestStore } from '@/stores/guest'
import Avatar from '@/components/ui/Avatar.vue'
import AppIcon from '@/components/ui/AppIcon.vue'
import type { IconName } from '@/components/ui/icons'

interface NavItem {
  label: string
  icon: IconName
  to?: string
}

interface NavSection {
  label?: string
  items: NavItem[]
}

const route = useRoute()
const auth = useAuthStore()
const guest = useGuestStore()

const homeTo = computed(() => (auth.isAuthenticated ? '/home' : '/'))

const profileTo = computed(() =>
  auth.user?.username ? `/u/${encodeURIComponent(auth.user.username)}` : '/login',
)

const displayName = computed(
  () => auth.user?.display_name || auth.user?.username || 'Your profile',
)
const handle = computed(() => auth.user?.username || '')

// Every destination below is registered and guarded by the router, so each
// primary item is a real link. The disabled branch stays available for
// destinations that are not yet live; it renders honestly (aria-disabled,
// muted, "Soon") and never navigates.
const sections = computed<NavSection[]>(() => {
  if (guest.isGuest) {
    return [
      {
        label: 'Public pages',
        items: [
          { label: 'Events', icon: 'events', to: '/events' },
          { label: 'Creators', icon: 'creative', to: '/creators' },
        ],
      },
      {
        label: 'Your account',
        items: [
          { label: 'Sign in', icon: 'logout', to: '/login' },
          { label: 'Create account', icon: 'profile', to: '/register' },
        ],
      },
    ]
  }

  const base: NavSection[] = [
  {
    items: [
      { label: 'Home', icon: 'home', to: '/home' },
      { label: 'Communities', icon: 'communities', to: '/communities' },
      { label: 'Events', icon: 'events', to: '/events' },
      { label: 'Chat', icon: 'chat', to: '/chat' },
      { label: 'Discover', icon: 'discover', to: '/discover' },
      { label: 'Notifications', icon: 'bell', to: '/notifications' },
      { label: 'Profile', icon: 'profile', to: profileTo.value },
    ],
  },
  {
    label: 'Connect',
    items: [
      { label: 'Creators', icon: 'creative', to: '/creators' },
      { label: 'Team ups', icon: 'communities', to: '/team-ups' },
    ],
  },
  ]

  // The console is for granted staff only; the API refuses everyone else too.
  if (auth.user?.is_staff) {
    base.push({
      label: 'Staff',
      items: [{ label: 'Admin', icon: 'shield', to: '/admin' }],
    })
  }

  return base
})

/** Inclusive match so /communities/:slug keeps Communities lit. */
function isActive(to: string | undefined): boolean {
  if (!to) return false
  if (to === '/') return route.path === '/'
  return route.path === to || route.path.startsWith(`${to}/`)
}
</script>

<template>
  <div class="sidebar">
    <RouterLink :to="homeTo" class="sidebar__brand" aria-label="PromisedLand home">
      <img class="sidebar__logo" src="/logo.png" alt="PromisedLand logo" width="28" height="28" />
      <span class="sidebar__wordmark u-wordmark">Promised<span class="sidebar__accent">Land</span></span>
    </RouterLink>

    <nav aria-label="Main">
      <div v-for="(section, index) in sections" :key="index" class="sidebar__section">
        <p v-if="section.label" class="sidebar__section-label">{{ section.label }}</p>
        <ul class="sidebar__nav">
          <li v-for="item in section.items" :key="item.label">
            <RouterLink
              v-if="item.to"
              :to="item.to"
              class="sidebar__item"
              :class="{ 'sidebar__item--active': isActive(item.to) }"
              :aria-current="isActive(item.to) ? 'page' : undefined"
              :aria-label="item.label"
              :title="item.label"
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
      </div>
    </nav>

    <div class="sidebar__footer">
      <RouterLink
        v-if="auth.isAuthenticated"
        to="/settings"
        class="sidebar__user"
        aria-label="Account settings"
        title="Account settings"
      >
        <Avatar
          :name="auth.user?.display_name || auth.user?.username || 'Member'"
          :src="auth.user?.avatar_url"
          size="sm"
        />
        <span class="sidebar__user-text">
          <span class="sidebar__user-name">{{ displayName }}</span>
          <span v-if="handle" class="sidebar__handle">@{{ handle }}</span>
        </span>
      </RouterLink>
      <RouterLink v-else to="/login" class="sidebar__user" aria-label="Sign in" title="Sign in">
        <span class="sidebar__user-fallback" aria-hidden="true">
          <AppIcon name="profile" :size="20" />
        </span>
        <span class="sidebar__user-text">
          <span class="sidebar__user-name">Sign in</span>
        </span>
      </RouterLink>
    </div>
  </div>
</template>

<style scoped>
.sidebar {
  display: flex;
  flex-direction: column;
  height: 100%;
  gap: var(--space-6);
}

.sidebar__brand {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  min-height: 44px;
  padding: var(--space-1) var(--space-3);
  border-radius: var(--radius-md);
  color: var(--text-primary);
  text-decoration: none;
}

.sidebar__brand:hover {
  background: var(--ink-800);
  color: var(--text-primary);
}

.sidebar__logo {
  width: 28px;
  height: 28px;
  border-radius: var(--radius-sm);
  object-fit: cover;
  border: 1px solid var(--border-hairline);
}

.sidebar__accent {
  color: var(--ember);
}

.sidebar__wordmark {
  font-family: var(--font-display);
  font-size: var(--text-lg);
  font-weight: 600;
  letter-spacing: -0.02em;
  color: var(--paper);
}

.sidebar__section {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.sidebar__section-label {
  margin: 0;
  padding-inline: var(--space-3);
  color: var(--text-muted);
  font-size: var(--text-xs);
}

.sidebar__nav {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.sidebar__item {
  position: relative;
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

.sidebar__item:focus-visible {
  outline: 2px solid var(--ember);
  outline-offset: 2px;
}

/* Active: ink-800 step + parchment text + 2px ember marker on the leading
   edge (::before — never a border, which would shift the layout). */
.sidebar__item--active,
a.sidebar__item--active:hover {
  background: var(--ink-800);
  color: var(--text-primary);
}

.sidebar__item--active::before {
  content: '';
  position: absolute;
  top: var(--space-2);
  bottom: var(--space-2);
  left: 0;
  width: 2px;
  border-radius: var(--radius-pill);
  background: var(--ember);
}

.sidebar__item--soon {
  color: var(--text-muted);
  cursor: default;
}

.sidebar__label {
  flex: 1;
  font-size: var(--text-sm);
  font-weight: 500;
}

.sidebar__soon {
  font-family: var(--font-mono);
  font-size: var(--text-xs);
  color: var(--text-muted);
}

.sidebar__footer {
  margin-top: auto;
}

.sidebar__user {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  min-height: 44px;
  padding: var(--space-1) var(--space-3);
  border-radius: var(--radius-md);
  color: var(--text-primary);
  text-decoration: none;
  transition:
    background-color var(--dur-fast) var(--ease),
    color var(--dur-fast) var(--ease);
}

a.sidebar__user:hover {
  background: var(--ink-800);
  color: var(--text-primary);
}

.sidebar__user:focus-visible {
  outline: 2px solid var(--ember);
  outline-offset: 2px;
}

.sidebar__user-fallback {
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: var(--radius-pill);
  background: var(--ink-700);
  color: var(--text-secondary);
}

.sidebar__user-text {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.sidebar__user-name {
  overflow: hidden;
  font-size: var(--text-sm);
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sidebar__handle {
  overflow: hidden;
  font-family: var(--font-mono);
  font-size: var(--text-xs);
  color: var(--text-muted);
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Tablet: 64px icon rail — labels leave the layout but stay on the links
   (aria-label) and in tooltips (title). */
@media (min-width: 768px) and (max-width: 1023px) {
  .sidebar {
    align-items: center;
    gap: var(--space-4);
  }

  .sidebar__brand {
    justify-content: center;
    padding-inline: 0;
  }

  .sidebar__wordmark,
  .sidebar__label,
  .sidebar__soon,
  .sidebar__section-label,
  .sidebar__user-text {
    display: none;
  }

  .sidebar__section {
    gap: var(--space-1);
  }

  .sidebar__item,
  .sidebar__user {
    justify-content: center;
    width: 48px;
    padding-inline: 0;
  }
}
</style>
