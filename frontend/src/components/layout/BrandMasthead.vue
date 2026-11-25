<script setup lang="ts">
import { computed } from 'vue'
import ThemeToggle from '@/components/ui/ThemeToggle.vue'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
// The mark goes to the app for a signed-in visitor and to the landing for a guest.
const homeTo = computed(() => (auth.isAuthenticated ? '/home' : '/'))
</script>

<template>
  <header class="masthead">
    <RouterLink :to="homeTo" class="masthead__brand" aria-label="PromisedLand home">
      <img class="masthead__logo" src="/logo.png" alt="" width="26" height="26" />
      <span class="masthead__wordmark">Promised<span class="masthead__accent">Land</span></span>
    </RouterLink>

    <div class="masthead__actions">
      <slot name="actions" />
      <ThemeToggle />
    </div>
  </header>
</template>

<style scoped>
.masthead {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-4);
  margin-inline: calc(var(--space-6) * -1);
  padding: var(--space-3) var(--space-6);
  border-bottom: 1px solid var(--border-hairline);
}

.masthead__brand {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  min-height: 44px;
  color: inherit;
  text-decoration: none;
}

.masthead__brand:hover {
  color: inherit;
}

.masthead__logo {
  border-radius: var(--radius-sm);
  object-fit: cover;
}

.masthead__wordmark {
  font-family: var(--font-display);
  font-size: var(--text-lg);
  font-weight: 600;
  letter-spacing: -0.02em;
  color: var(--text-primary);
}

.masthead__accent {
  color: var(--ember);
}

.masthead__actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

@media (max-width: 767px) {
  .masthead {
    margin-inline: calc(var(--space-4) * -1);
    padding-inline: var(--space-4);
  }
}
</style>
