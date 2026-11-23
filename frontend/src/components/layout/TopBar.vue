<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import BaseButton from '@/components/ui/BaseButton.vue'
import Avatar from '@/components/ui/Avatar.vue'
import AppIcon from '@/components/ui/AppIcon.vue'
import WayfinderMark from '@/components/ui/WayfinderMark.vue'

const auth = useAuthStore()
const ui = useUiStore()

// R1: wire these buttons to /login and /register once the auth views exist.
function announceAccounts(): void {
  ui.pushToast({
    title: 'Accounts arrive in Release 1',
    message: 'Sign-in and registration are part of the social foundation.',
  })
}

async function signOut(): Promise<void> {
  try {
    await auth.logout()
    ui.pushToast({ title: 'You are signed out', tone: 'success' })
  } catch {
    ui.pushToast({ title: 'Could not sign out', message: 'Check your connection and try again.', tone: 'error' })
  }
}
</script>

<template>
  <div class="topbar">
    <RouterLink to="/" class="topbar__brand" aria-label="PromisedLand home">
      <WayfinderMark size="30px" />
      <span>PromisedLand</span>
    </RouterLink>

    <button type="button" class="topbar__search" @click="announceAccounts">
      <AppIcon name="search" :size="18" />
      <span class="topbar__search-text">Search communities, posts, events</span>
    </button>

    <div class="topbar__actions">
      <template v-if="auth.isAuthenticated">
        <Avatar :name="auth.user?.displayName || auth.user?.username || 'Member'" size="sm" />
        <BaseButton variant="ghost" size="sm" @click="signOut">
          <AppIcon name="logout" :size="16" />
          Sign out
        </BaseButton>
      </template>
      <template v-else>
        <BaseButton variant="ghost" size="sm" @click="announceAccounts">Sign in</BaseButton>
        <BaseButton variant="secondary" size="sm" @click="announceAccounts">Create account</BaseButton>
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
  gap: var(--space-3);
  min-height: 44px;
  max-width: 440px;
  padding: 0 var(--space-3);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-md);
  background: var(--ink-850);
  color: var(--text-secondary);
  text-align: left;
  cursor: pointer;
  transition: border-color var(--dur-fast) var(--ease);
}

.topbar__search:hover {
  border-color: var(--border-strong);
  color: var(--text-primary);
}

.topbar__search-text {
  font-size: var(--text-sm);
}

.topbar__actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-left: auto;
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
    justify-content: center;
    max-width: none;
  }

  .topbar__search-text {
    display: none;
  }
}
</style>
