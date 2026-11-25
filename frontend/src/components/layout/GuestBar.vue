<script setup lang="ts">
import AppIcon from '@/components/ui/AppIcon.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import { useGuestStore } from '@/stores/guest'

const guest = useGuestStore()
</script>

<template>
  <div class="guest-bar">
    <p class="guest-bar__copy">
      <span class="guest-bar__tag">Guest</span>
      You can look around events and creators. An account is what lets you join communities, RSVP
      and post.
    </p>

    <div class="guest-bar__actions">
      <BaseButton variant="primary" size="sm" to="/register">Create account</BaseButton>
      <BaseButton variant="ghost" size="sm" to="/login">Sign in</BaseButton>
      <button
        type="button"
        class="guest-bar__close"
        aria-label="Leave guest mode"
        title="Leave guest mode"
        @click="guest.leave()"
      >
        <AppIcon name="close" :size="16" />
      </button>
    </div>
  </div>
</template>

<style scoped>
.guest-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  margin-inline: calc(var(--space-6) * -1);
  padding: var(--space-2) var(--space-6);
  border-bottom: 1px solid var(--border-hairline);
  background: var(--ink-850);
}

.guest-bar__copy {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-2);
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.guest-bar__tag {
  padding: 2px var(--space-2);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-pill);
  color: var(--text-primary);
  font-size: var(--text-xs);
}

.guest-bar__actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.guest-bar__close {
  display: grid;
  width: 44px;
  height: 44px;
  place-items: center;
  padding: 0;
  border: 0;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  transition:
    background-color var(--dur-fast) var(--ease),
    color var(--dur-fast) var(--ease);
}

.guest-bar__close:hover {
  background: var(--ink-800);
  color: var(--text-primary);
}

@media (max-width: 767px) {
  .guest-bar {
    margin-inline: calc(var(--space-4) * -1);
    padding-inline: var(--space-4);
  }
}
</style>
