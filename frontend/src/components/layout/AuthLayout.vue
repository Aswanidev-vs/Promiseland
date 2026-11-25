<script setup lang="ts">
import BrandMasthead from '@/components/layout/BrandMasthead.vue'

withDefaults(defineProps<{ title: string; intro?: string }>(), { intro: '' })
</script>

<template>
  <div class="auth-page">
    <BrandMasthead />

    <main class="auth-page__body">
      <div class="auth-page__column">
        <div class="auth-page__heading">
          <h1 class="auth-page__title">{{ title }}</h1>
          <p v-if="intro" class="auth-page__intro">{{ intro }}</p>
        </div>

        <slot />
        <slot name="footer" />
      </div>

      <aside class="auth-page__panel" aria-label="About PromisedLand">
        <div class="auth-panel">
          <div class="auth-panel__chart">
            <img
              class="auth-panel__image"
              src="/represent.png"
              alt="PromisedLand"
            />
          </div>

          <p class="auth-panel__promise">
            <span>Your fandom.</span>
            <span>Your people.</span>
            <span>Your place.</span>
          </p>

          <p class="auth-panel__note">
            Spoiler-aware communities, a creator feed and local events, all grown from the
            interests you pick.
          </p>
        </div>
      </aside>
    </main>
  </div>
</template>

<style scoped>
.auth-page {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

/* ── Two-column body: form first, brand panel second ────────────────────────── */
.auth-page__body {
  display: grid;
  flex: 1;
  grid-template-columns: minmax(0, 420px) minmax(0, 400px);
  align-items: center;
  justify-content: center;
  gap: clamp(var(--space-12), 7vw, var(--space-20));
  padding-block: var(--space-12) var(--space-16);
}

.auth-page__column {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
  width: 100%;
  max-width: 420px;
}

.auth-page__heading {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.auth-page__title {
  font-family: var(--font-display);
  font-size: var(--text-3xl);
  font-weight: 600;
  letter-spacing: -0.025em;
  line-height: 1.1;
}

.auth-page__intro {
  max-width: 40ch;
  color: var(--text-secondary);
}

/* ── Brand panel: the one rich surface on an otherwise quiet screen ─────────── */
.auth-page__panel {
  display: flex;
  justify-content: center;
}

.auth-panel {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
  width: min(100%, 400px);
  padding: var(--space-5);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-xl);
  background: var(--ink-850);
  box-shadow: var(--shadow-1);
}

.auth-panel__chart {
  overflow: hidden;
  aspect-ratio: 1;
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-lg);
  background: var(--ink-900);
}

.auth-panel__image {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.auth-panel__promise {
  display: flex;
  flex-direction: column;
  font-family: var(--font-display);
  font-size: var(--text-xl);
  font-weight: 600;
  letter-spacing: -0.02em;
  line-height: 1.2;
  color: var(--text-primary);
}

.auth-panel__note {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

@media (max-width: 1023px) {
  .auth-page__body {
    grid-template-columns: minmax(0, 420px);
    justify-content: center;
    padding-block: var(--space-10) var(--space-16);
  }

  .auth-page__panel {
    display: none;
  }
}

@media (max-width: 767px) {
  .auth-page__masthead {
    margin-inline: calc(var(--space-4) * -1);
    padding-inline: var(--space-4);
  }

  .auth-page__body {
    grid-template-columns: minmax(0, 1fr);
    align-items: start;
    padding-block: var(--space-6) var(--space-12);
  }

  .auth-page__column {
    gap: var(--space-5);
  }

  .auth-page__title {
    font-size: var(--text-2xl);
  }
}
</style>
