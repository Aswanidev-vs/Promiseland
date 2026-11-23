<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useNotificationsStore } from '@/stores/notifications'
import BaseButton from '@/components/ui/BaseButton.vue'
import Avatar from '@/components/ui/Avatar.vue'
import AppIcon from '@/components/ui/AppIcon.vue'
import WayfinderMark from '@/components/ui/WayfinderMark.vue'

const router = useRouter()
const auth = useAuthStore()
const notifications = useNotificationsStore()

const query = ref('')
const profileTo = computed(() =>
  auth.user?.username ? `/u/${encodeURIComponent(auth.user.username)}` : '/login',
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
    <RouterLink to="/" class="topbar__brand" aria-label="PromisedLand home">
      <WayfinderMark size="30px" />
      <span>PromisedLand</span>
    </RouterLink>

    <form class="topbar__search" role="search" @submit.prevent="submitSearch">
      <label class="sr-only" for="topbar-search">Search PromisedLand</label>
      <button type="submit" class="topbar__search-submit" aria-label="Search">
        <AppIcon name="search" :size="18" />
      </button>
      <input
        id="topbar-search"
        v-model="query"
        class="topbar__search-input"
        type="search"
        name="q"
        placeholder="Search communities, posts, events"
      />
    </form>

    <div class="topbar__actions">
      <template v-if="auth.isAuthenticated">
        <RouterLink to="/notifications" class="topbar__bell" :aria-label="`Notifications, ${notifications.unreadCount} unread`">
          <AppIcon name="bell" :size="20" />
          <span v-if="notifications.unreadCount > 0" class="topbar__badge" aria-hidden="true">
            {{ notifications.unreadCount > 9 ? '9+' : notifications.unreadCount }}
          </span>
        </RouterLink>
        <RouterLink :to="profileTo" class="topbar__profile" aria-label="Your profile">
          <Avatar :name="auth.user?.display_name || auth.user?.username || 'Member'" :src="auth.user?.avatar_url" size="sm" />
        </RouterLink>
        <BaseButton variant="ghost" size="sm" aria-label="Sign out" @click="auth.logout()">
          <AppIcon name="logout" :size="16" />
          <span class="topbar__signout-label">Sign out</span>
        </BaseButton>
      </template>
      <template v-else>
        <BaseButton variant="ghost" size="sm" to="/login">Sign in</BaseButton>
        <BaseButton variant="secondary" size="sm" to="/register">Create account</BaseButton>
      </template>
    </div>
  </div>
</template>

<style scoped>
.topbar {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  min-height: var(--topbar-h);
  padding-block: var(--space-2);
  border-bottom: 1px solid var(--border-hairline);
}

.topbar__brand {
  display: none;
  align-items: center;
  gap: var(--space-2);
  color: var(--text-primary);
  font-family: var(--font-display);
  font-size: var(--text-lg);
  font-weight: 600;
  text-decoration: none;
  white-space: nowrap;
}

.topbar__brand:hover {
  color: var(--text-primary);
}

.topbar__search {
  display: flex;
  flex: 1;
  align-items: center;
  gap: var(--space-2);
  min-height: 44px;
  max-width: 440px;
  padding-inline: var(--space-2);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-md);
  background: var(--ink-850);
  color: var(--text-secondary);
}

.topbar__search:hover {
  border-color: var(--border-strong);
}

.topbar__search:focus-within {
  border-color: var(--ember);
  outline: 2px solid var(--ember);
  outline-offset: 2px;
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
}

.topbar__search-submit:hover {
  color: var(--text-primary);
}

.topbar__search-input {
  flex: 1;
  min-width: 0;
  min-height: 44px;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--text-primary);
  font-size: var(--text-sm);
}

.topbar__search-input:focus {
  outline: none;
}

.topbar__search-input::placeholder {
  color: var(--text-secondary);
  opacity: 0.8;
}

.topbar__actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-left: auto;
}

.topbar__bell,
.topbar__profile {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 44px;
  height: 44px;
  border-radius: var(--radius-md);
  color: var(--text-secondary);
}

.topbar__bell:hover,
.topbar__profile:hover {
  background: var(--ink-800);
  color: var(--text-primary);
}

.topbar__badge {
  position: absolute;
  top: 2px;
  right: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  height: 18px;
  padding-inline: var(--space-1);
  border-radius: var(--radius-pill);
  background: var(--ember);
  color: var(--ink-900);
  font-family: var(--font-mono);
  font-size: 10px;
  font-variant-numeric: tabular-nums;
  line-height: 1;
}

@media (max-width: 767px) {
  .topbar {
    gap: var(--space-2);
  }

  .topbar__brand {
    display: flex;
  }

  .topbar__brand span {
    display: none;
  }

  .topbar__search {
    max-width: none;
  }

  .topbar__search-input::placeholder {
    color: transparent;
  }

  .topbar__signout-label {
    display: none;
  }
}
</style>
