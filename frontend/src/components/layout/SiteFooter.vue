<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useGuestStore } from '@/stores/guest'

const router = useRouter()
const auth = useAuthStore()
const guest = useGuestStore()

const year = new Date().getFullYear()

const accountLinks = computed(() =>
  auth.isAuthenticated
    ? [
        { label: 'Your feed', to: '/home' },
        { label: 'Your settings', to: '/settings' },
      ]
    : [
        { label: 'Sign in', to: '/login' },
        { label: 'Create account', to: '/register' },
      ],
)

function browseAsGuest(): void {
  guest.enter()
  void router.push('/events')
}
</script>

<template>
  <footer class="footer">
    <div class="footer__grid">
      <div class="footer__brand">
        <RouterLink to="/" class="footer__brand-link" aria-label="PromisedLand home">
          <img class="footer__logo" src="/logo.png" alt="" />
        </RouterLink>
        <p class="footer__tagline">Your fandom. Your people. Your place.</p>
        <p class="footer__note">
          An anime-first social network: spoiler-aware communities, a creator feed and events you
          can show up to.
        </p>
      </div>

      <nav class="footer__column" aria-label="Explore">
        <h2 class="footer__heading">Explore</h2>
        <RouterLink to="/">Home</RouterLink>
        <RouterLink to="/events">Events</RouterLink>
        <RouterLink to="/creators">Creators</RouterLink>
      </nav>

      <nav class="footer__column" aria-label="Account">
        <h2 class="footer__heading">Account</h2>
        <RouterLink v-for="link in accountLinks" :key="link.to" :to="link.to">
          {{ link.label }}
        </RouterLink>
        <button v-if="!auth.isAuthenticated && !guest.isGuest" type="button" class="footer__guest" @click="browseAsGuest">
          Continue as a guest
        </button>
      </nav>

      <nav class="footer__column" aria-label="Legal">
        <h2 class="footer__heading">Legal</h2>
        <RouterLink to="/terms">Terms of use</RouterLink>
        <RouterLink to="/privacy">Privacy policy</RouterLink>
      </nav>
    </div>

    <div class="footer__base">
      <p>© {{ year }} PromisedLand</p>
      <p>PromisedLand is a working brand name.</p>
    </div>
  </footer>
</template>

<style scoped>
.footer {
  border-top: 1px solid var(--border-hairline);
  padding-block: var(--space-12) var(--space-8);
}

.footer__grid {
  display: grid;
  grid-template-columns: minmax(0, 1.4fr) repeat(3, minmax(0, 1fr));
  gap: var(--space-10);
}

.footer__brand {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  max-width: 34ch;
}

.footer__brand-link {
  display: inline-flex;
  width: fit-content;
  border-radius: var(--radius-sm);
  text-decoration: none;
}

.footer__logo {
  width: 180px;
  height: auto;
  object-fit: contain;
}

.footer__tagline {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.footer__note {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.footer__column {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--space-1);
}

.footer__heading {
  padding-bottom: var(--space-2);
  color: var(--text-primary);
  font-family: var(--font-body);
  font-size: var(--text-sm);
  font-weight: 600;
}

.footer__column a,
.footer__guest {
  display: inline-flex;
  align-items: center;
  min-height: 44px;
  padding: 0;
  border: 0;
  background: none;
  color: var(--text-secondary);
  font-size: var(--text-sm);
  text-align: left;
  text-decoration: none;
  cursor: pointer;
  transition: color var(--dur-fast) var(--ease);
}

.footer__column a:hover,
.footer__guest:hover {
  color: var(--text-primary);
}

.footer__base {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-4);
  justify-content: space-between;
  margin-top: var(--space-10);
  padding-top: var(--space-6);
  border-top: 1px solid var(--border-hairline);
  color: var(--text-secondary);
  font-size: var(--text-xs);
}

@media (max-width: 1023px) {
  .footer__grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--space-8);
  }
}

@media (max-width: 599px) {
  .footer__grid {
    grid-template-columns: minmax(0, 1fr);
    gap: var(--space-6);
  }
}
</style>
