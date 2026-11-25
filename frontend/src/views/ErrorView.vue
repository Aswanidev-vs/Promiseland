<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import AppIcon from '@/components/ui/AppIcon.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import BrandMasthead from '@/components/layout/BrandMasthead.vue'
import SiteFooter from '@/components/layout/SiteFooter.vue'

const props = withDefaults(defineProps<{ code?: number | string }>(), { code: 404 })

const router = useRouter()

interface ErrorCopy {
  title: string
  heading: string
  message: string
  hint?: string
  retry?: boolean
  signIn?: boolean
}

const COPY: Record<number, ErrorCopy> = {
  403: {
    title: 'Forbidden',
    heading: 'This page is not yours to open',
    message:
      'You are signed in, but this area needs a different account or permission. If that seems wrong, sign in with the account that should have access.',
    signIn: true,
  },
  404: {
    title: 'Page not found',
    heading: 'This bearing leads nowhere',
    message:
      'The page you were looking for is not here. It may have moved, or the link may be mistyped.',
  },
  429: {
    title: 'Too many requests',
    heading: 'Slow down a moment',
    message:
      'You have made a lot of requests in a short time. Wait a minute, then try again — nothing has been lost.',
    hint: 'Repeated bursts will keep being paused, so give it a moment.',
    retry: true,
  },
  500: {
    title: 'Server error',
    heading: 'Something gave way on our side',
    message:
      'The page could not finish loading. Reloading usually fixes it; if it does not, the fault is ours, not yours.',
    hint: 'If it keeps happening, tell us what you were doing when it broke.',
    retry: true,
  },
  503: {
    title: 'Service unavailable',
    heading: 'The harbour is closed for a moment',
    message:
      'PromisedLand cannot reach its API right now, usually during a restart or a deploy. Nothing about your account has changed.',
    hint: 'Wait a moment and try again.',
    retry: true,
  },
}

const requestCode = computed(() => {
  const parsed = Number(props.code)
  return Number.isInteger(parsed) ? parsed : 0
})

const copy = computed<ErrorCopy>(() => COPY[requestCode.value] ?? COPY[404]!)
const shownCode = computed(() => (COPY[requestCode.value] ? requestCode.value : 404))

function retry(): void {
  window.location.reload()
}
</script>

<template>
  <div class="error-page">
    <BrandMasthead />

    <main class="error-page__body">
      <div class="error-page__plate" aria-hidden="true">
        <svg viewBox="0 0 200 200" class="error-page__chart" focusable="false">
          <circle cx="100" cy="100" r="72" fill="none" stroke="var(--border-hairline)" stroke-width="1" />
          <circle
            cx="100"
            cy="100"
            r="54"
            fill="none"
            stroke="var(--ember)"
            stroke-width="1"
            stroke-opacity="0.45"
            stroke-dasharray="3 10"
          />
          <g fill="none" stroke="var(--text-secondary)" stroke-width="1" stroke-opacity="0.55">
            <path d="M36 64 L84 42" />
            <path d="M118 44 L162 78" stroke-dasharray="2 6" />
            <path d="M38 138 L74 160" stroke-dasharray="2 6" />
            <path d="M128 158 L164 128" />
          </g>
          <g fill="var(--text-secondary)" fill-opacity="0.7">
            <circle cx="36" cy="64" r="2.5" />
            <circle cx="84" cy="42" r="2.5" />
            <circle cx="38" cy="138" r="2.5" />
            <circle cx="164" cy="128" r="2.5" />
          </g>
          <circle cx="162" cy="78" r="3" fill="var(--ember)" />
          <circle cx="118" cy="44" r="3" fill="var(--ember)" fill-opacity="0.5" />
        </svg>
        <span class="error-page__code">{{ shownCode }}</span>
      </div>

      <div class="error-page__copy">
        <p class="error-page__kind">{{ copy.title }}</p>
        <h1 class="error-page__heading">{{ copy.heading }}</h1>
        <p class="error-page__message">{{ copy.message }}</p>
        <p v-if="copy.hint" class="error-page__hint">{{ copy.hint }}</p>

        <div class="error-page__actions">
          <BaseButton v-if="copy.retry" variant="primary" size="md" type="button" @click="retry">
            Try again
          </BaseButton>
          <BaseButton v-else-if="copy.signIn" variant="primary" size="md" to="/login">
            Sign in
          </BaseButton>
          <BaseButton v-else variant="primary" size="md" to="/">Go home</BaseButton>

          <BaseButton v-if="copy.signIn" variant="secondary" size="md" to="/">Go home</BaseButton>
          <BaseButton v-else-if="!copy.retry" variant="secondary" size="md" type="button" @click="router.back()">
            Back to the last page
          </BaseButton>
          <BaseButton v-else variant="secondary" size="md" to="/">Go home</BaseButton>
        </div>

        <p class="error-page__routes">
          <AppIcon name="events" :size="16" />
          <RouterLink to="/events">Browse events</RouterLink>
          <RouterLink to="/creators">Browse creators</RouterLink>
          <RouterLink v-if="copy.retry" to="/status">Service status</RouterLink>
        </p>
      </div>
    </main>

    <SiteFooter />
  </div>
</template>

<style scoped>
.error-page {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

.error-page__body {
  display: grid;
  flex: 1;
  grid-template-columns: minmax(0, 320px) minmax(0, 1fr);
  align-items: center;
  gap: clamp(var(--space-10), 6vw, var(--space-16));
  padding-block: var(--space-16) var(--space-20);
}

.error-page__plate {
  position: relative;
  display: grid;
  place-items: center;
  aspect-ratio: 1;
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-xl);
  background-color: var(--ink-850);
  background-image:
    linear-gradient(var(--border-hairline) 1px, transparent 1px),
    linear-gradient(90deg, var(--border-hairline) 1px, transparent 1px);
  background-size: 32px 32px;
}

.error-page__chart {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
}

.error-page__code {
  position: relative;
  font-family: var(--font-display);
  font-size: clamp(3.5rem, 8vw, 5.5rem);
  font-weight: 600;
  letter-spacing: -0.04em;
  line-height: 1;
  color: var(--text-primary);
}

.error-page__copy {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--space-4);
  max-width: 52ch;
}

.error-page__kind {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.error-page__heading {
  font-family: var(--font-display);
  font-size: var(--text-3xl);
  font-weight: 600;
  letter-spacing: -0.025em;
  line-height: 1.1;
}

.error-page__message {
  color: var(--text-secondary);
  font-size: var(--text-lg);
}

.error-page__hint {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.error-page__actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-3);
  padding-top: var(--space-2);
}

.error-page__routes {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-2);
  padding-top: var(--space-2);
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.error-page__routes a {
  display: inline-flex;
  align-items: center;
  min-height: 44px;
  color: var(--tide);
  text-decoration: none;
}

.error-page__routes a:hover {
  color: var(--text-primary);
}

@media (max-width: 1023px) {
  .error-page__body {
    grid-template-columns: minmax(0, 1fr);
    align-items: start;
    padding-block: var(--space-10) var(--space-16);
  }

  .error-page__plate {
    max-width: 260px;
  }
}
</style>
