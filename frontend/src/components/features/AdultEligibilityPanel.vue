<script setup lang="ts">
/**
 * Calm, non-judgmental gate state for the R5 adult area. Shown whenever the
 * feature is off, the account is not eligible, or access is not permitted.
 * It never renders any dating UI and always offers a way back to /home.
 */
import { RouterLink } from 'vue-router'
import WayfinderMark from '@/components/ui/WayfinderMark.vue'

withDefaults(
  defineProps<{
    /** Server-supplied explanation, when the status endpoint provides one. */
    message?: string
    /** 403-style access state; uses slightly different framing. */
    denied?: boolean
  }>(),
  { message: '', denied: false },
)

const requirements = [
  {
    title: 'Age assurance',
    body: 'Access needs a reliable age check that meets the rules of the region you are in. A birthday typed into a form is not enough.',
  },
  {
    title: 'Legal review',
    body: 'Privacy, consumer and online-safety obligations have to be reviewed before this area can open.',
  },
  {
    title: 'Separate opt-in',
    body: 'Taking part is a separate choice. Nothing here is inferred from your interests, posts, or communities.',
  },
  {
    title: 'Safety controls',
    body: 'Consent, unmatch, block, reporting and prioritised moderation need to be in place and working.',
  },
] as const
</script>

<template>
  <section class="adult-panel" aria-labelledby="adult-panel-title">
    <WayfinderMark size="72px" />
    <div class="adult-panel__copy">
      <p class="adult-panel__eyebrow">Adults only</p>
      <h1 id="adult-panel-title" class="adult-panel__title">
        {{ denied ? 'Access to this area is not available' : 'This area is not available' }}
      </h1>
      <p class="adult-panel__intro">
        {{
          message ||
          'This part of PromisedLand is not open right now. It stays separate from the rest of your account.'
        }}
      </p>
    </div>

    <div class="adult-panel__requirements">
      <h2 class="adult-panel__requirements-title">What needs to be in place first</h2>
      <ul class="adult-panel__list">
        <li v-for="item in requirements" :key="item.title" class="adult-panel__item">
          <h3 class="adult-panel__item-title">{{ item.title }}</h3>
          <p class="adult-panel__item-body">{{ item.body }}</p>
        </li>
      </ul>
    </div>

    <RouterLink class="adult-panel__home" to="/home">Back to home</RouterLink>
  </section>
</template>

<style scoped>
.adult-panel {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--space-6);
  width: 100%;
  padding: var(--space-8);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-lg);
  background: var(--ink-800);
}

.adult-panel__copy {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  max-width: 56ch;
}

.adult-panel__eyebrow {
  color: var(--text-secondary);
  font-family: var(--font-mono);
  font-size: var(--text-xs);
  letter-spacing: 0.04em;
}

.adult-panel__title {
  overflow-wrap: anywhere;
  font-family: var(--font-display);
  font-size: var(--text-2xl);
}

.adult-panel__intro {
  color: var(--text-secondary);
}

.adult-panel__requirements {
  width: 100%;
  padding-top: var(--space-6);
  border-top: 1px solid var(--border-hairline);
}

.adult-panel__requirements-title {
  margin-bottom: var(--space-5);
  font-size: var(--text-lg);
}

.adult-panel__list {
  display: grid;
  gap: var(--space-5);
}

.adult-panel__item-title {
  font-family: var(--font-body);
  font-size: var(--text-base);
  font-weight: 600;
  letter-spacing: 0;
}

.adult-panel__item-body {
  margin-top: var(--space-1);
  max-width: 62ch;
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.adult-panel__home {
  display: inline-flex;
  align-items: center;
  min-height: 44px;
  color: var(--tide);
  font-weight: 500;
  text-decoration: none;
}

.adult-panel__home:hover {
  color: var(--text-primary);
  text-decoration: underline;
}

@media (min-width: 720px) {
  .adult-panel__list {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 480px) {
  .adult-panel {
    padding: var(--space-5);
  }
}
</style>
