<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useNotificationsStore } from '@/stores/notifications'
import { useGuestStore } from '@/stores/guest'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseInput from '@/components/ui/BaseInput.vue'
import Avatar from '@/components/ui/Avatar.vue'
import AppIcon from '@/components/ui/AppIcon.vue'
import ThemeToggle from '@/components/ui/ThemeToggle.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const notifications = useNotificationsStore()
const guest = useGuestStore()

const query = ref('')
const profileTo = computed(() =>
  auth.user?.username ? `/u/${encodeURIComponent(auth.user.username)}` : '/login',
)

/**
 * Screens whose own content already carries the ember CTA (Share a post,
 * Create event, Follow …). Everywhere else the shell Create is the one
 * primary ember action — one ember per screen.
 */
const emberCreateRoutes = new Set([
  'search',
  'discover',
  'chat',
  'chat-conversation',
  'connections',
  'post',
])
const createVariant = computed(() =>
  emberCreateRoutes.has(String(route.name)) ? 'primary' : 'secondary',
)

const unreadLabel = computed(() =>
  notifications.unreadCount > 0
    ? `Notifications, ${notifications.unreadCount} unread`
    : 'Notifications',
)

watch(
  () => auth.isAuthenticated,
  (signedIn) => {
    if (signedIn) void notifications.load()
  },
  { immediate: true },
)

function submitSearch(): void {
  const q = query.value.trim()
  if (!q) return
  void router.push({ name: 'search', query: { q } })
}
</script>

<template>
  <div class="topbar">
    <!-- Brand only lives here on mobile; the sidebar owns it above 768. -->
    <div class="topbar__start">
      <RouterLink to="/" class="topbar__brand" aria-label="PromisedLand home">
        <img class="topbar__logo" src="/logo.png" alt="PromisedLand logo" width="26" height="26" />
        <span class="topbar__wordmark u-wordmark">Promised<span class="topbar__accent">Land</span></span>
      </RouterLink>
    </div>

    <form v-if="!guest.isGuest" class="topbar__search" role="search" @submit.prevent="submitSearch">
      <label class="sr-only" for="topbar-search">Search PromisedLand</label>
      <button type="submit" class="topbar__search-submit" aria-label="Search" title="Search">
        <AppIcon name="search" :size="20" />
      </button>
      <BaseInput
        id="topbar-search"
        v-model="query"
        class="topbar__search-field"
        type="search"
        name="q"
        placeholder="Search communities, posts, events"
        autocomplete="off"
      />
    </form>

    <RouterLink
      v-if="!guest.isGuest"
      to="/search"
      class="topbar__mobile-search"
      aria-label="Search"
      title="Search"
    >
      <AppIcon name="search" :size="20" />
    </RouterLink>

    <div class="topbar__actions">
      <ThemeToggle />
      <template v-if="auth.isAuthenticated">
        <RouterLink
          to="/notifications"
          class="topbar__bell"
          :aria-label="unreadLabel"
          title="Notifications"
        >
          <AppIcon name="bell" :size="20" />
          <span v-if="notifications.unreadCount > 0" class="topbar__dot" aria-hidden="true" />
        </RouterLink>

        <BaseButton class="topbar__create" :variant="createVariant" size="md" to="/posts/new">
          <svg
            class="topbar__create-icon"
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.5"
            stroke-linecap="round"
            stroke-linejoin="round"
            aria-hidden="true"
            focusable="false"
          >
            <path d="M12 5.5v13M5.5 12h13" />
          </svg>
          Create
        </BaseButton>

        <RouterLink :to="profileTo" class="topbar__profile" aria-label="Your profile" title="Your profile">
          <Avatar
            :name="auth.user?.display_name || auth.user?.username || 'Member'"
            :src="auth.user?.avatar_url"
            size="sm"
          />
        </RouterLink>
      </template>
      <template v-else>
        <BaseButton variant="ghost" size="md" to="/login">Sign in</BaseButton>
        <BaseButton class="topbar__register" variant="secondary" size="md" to="/register">
          Create account
        </BaseButton>
      </template>
    </div>
  </div>
</template>

<style scoped>
.topbar {
  display: grid;
  /* The trailing column sizes to its actions, so the guest sign-in controls
     cannot overflow backwards onto the search field. */
  grid-template-columns: minmax(0, 1fr) minmax(0, 480px) auto;
  align-items: center;
  gap: var(--space-4);
  min-height: var(--topbar-h);
}

.topbar__start {
  min-width: 0;
}

.topbar__brand {
  display: none;
  align-items: center;
  gap: var(--space-2);
  min-height: 44px;
  padding-inline: var(--space-1);
  border-radius: var(--radius-md);
  color: var(--text-primary);
  text-decoration: none;
  white-space: nowrap;
}

.topbar__brand:hover {
  background: var(--ink-800);
  color: var(--text-primary);
}

.topbar__logo {
  width: 26px;
  height: 26px;
  border-radius: var(--radius-sm);
  object-fit: cover;
  border: 1px solid var(--border-hairline);
}

.topbar__accent {
  color: var(--ember);
}

/* Fraunces WONK — same treatment as the sidebar wordmark. */
.topbar__wordmark {
  font-family: var(--font-display);
  font-size: var(--text-lg);
  font-weight: 600;
  letter-spacing: -0.02em;
  color: var(--paper);
}

/* Search: the §3 input recipe on the composed field — ink-850, hairline,
   radius md, ember border + ember-soft ring on focus. */
.topbar__search {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  width: 100%;
  max-width: 480px;
  min-width: 0;
  min-height: 44px;
  padding-inline: var(--space-1);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-md);
  background: var(--ink-850);
  transition:
    border-color var(--dur-fast) var(--ease),
    box-shadow var(--dur-fast) var(--ease);
}

.topbar__search:hover {
  border-color: var(--border-strong);
}

.topbar__search:focus-within {
  border-color: var(--ember);
  box-shadow: 0 0 0 3px var(--ember-soft);
}

.topbar__search :deep(.field) {
  flex: 1;
  min-width: 0;
}

.topbar__search :deep(.field__input) {
  min-height: 44px;
  padding: 0;
  border: 0;
  background: transparent;
  font-size: var(--text-sm);
}

.topbar__search :deep(.field__input:hover),
.topbar__search :deep(.field__input:focus) {
  border-color: transparent;
  outline: none;
  box-shadow: none;
}

.topbar__search-submit {
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  padding: 0;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  transition: color var(--dur-fast) var(--ease);
}

.topbar__search-submit:hover {
  color: var(--text-primary);
}

.topbar__search-submit:focus-visible {
  outline: 2px solid var(--ember);
  outline-offset: 2px;
}

.topbar__mobile-search {
  display: none;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border-radius: var(--radius-md);
  color: var(--text-secondary);
  text-decoration: none;
  transition:
    background-color var(--dur-fast) var(--ease),
    color var(--dur-fast) var(--ease);
}

.topbar__mobile-search:hover {
  background: var(--ink-800);
  color: var(--text-primary);
}

.topbar__mobile-search:focus-visible {
  outline: 2px solid var(--ember);
  outline-offset: 2px;
}

.topbar__actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: var(--space-2);
  min-width: 0;
}

.topbar__bell,
.topbar__profile {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border-radius: var(--radius-md);
  color: var(--text-secondary);
  text-decoration: none;
  transition:
    background-color var(--dur-fast) var(--ease),
    color var(--dur-fast) var(--ease);
}

.topbar__bell:hover,
.topbar__profile:hover {
  background: var(--ink-800);
  color: var(--text-primary);
}

.topbar__bell:focus-visible,
.topbar__profile:focus-visible,
.topbar__brand:focus-visible {
  outline: 2px solid var(--ember);
  outline-offset: 2px;
}

.topbar__actions :deep(.btn:focus-visible) {
  outline: 2px solid var(--ember);
  outline-offset: 2px;
}

.topbar__create-icon {
  flex: none;
}

/* Unread marker — a quiet ember dot, never a second accent. */
.topbar__dot {
  position: absolute;
  top: 8px;
  right: 8px;
  width: 8px;
  height: 8px;
  border: 2px solid var(--ink-900);
  border-radius: var(--radius-pill);
  background: var(--ember);
}

.topbar__register {
  white-space: nowrap;
}

/* Tablet: narrower search so the three grid tracks keep it centered. */
@media (min-width: 768px) and (max-width: 1023px) {
  .topbar__search {
    max-width: 320px;
  }
}

@media (max-width: 767px) {
  .topbar {
    grid-template-columns: auto minmax(0, 1fr) auto;
    gap: var(--space-2);
    min-height: 56px;
  }

  .topbar__brand {
    display: flex;
  }

  .topbar__search {
    display: none;
  }

  .topbar__mobile-search {
    display: inline-flex;
    justify-self: end;
  }

  .topbar__bell,
  .topbar__create,
  .topbar__register {
    display: none;
  }
}
</style>
