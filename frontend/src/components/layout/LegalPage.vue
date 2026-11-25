<script setup lang="ts">
import BrandMasthead from './BrandMasthead.vue'
import SiteFooter from './SiteFooter.vue'

withDefaults(defineProps<{ title: string; updated: string; intro?: string }>(), { intro: '' })
</script>

<template>
  <div class="legal">
    <BrandMasthead />

    <main class="legal__body">
      <article class="legal__doc">
        <header class="legal__head">
          <h1 class="legal__title">{{ title }}</h1>
          <p class="legal__meta">Last updated {{ updated }}</p>
          <p v-if="intro" class="legal__intro">{{ intro }}</p>
        </header>

        <slot />
      </article>
    </main>

    <SiteFooter />
  </div>
</template>

<style scoped>
.legal {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

.legal__body {
  flex: 1;
  padding-block: var(--space-12) var(--space-16);
}

.legal__doc {
  max-width: 70ch;
}

.legal__head {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding-bottom: var(--space-8);
  border-bottom: 1px solid var(--border-hairline);
}

.legal__title {
  font-family: var(--font-display);
  font-size: var(--text-3xl);
  font-weight: 600;
  letter-spacing: -0.025em;
  line-height: 1.1;
}

.legal__meta {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.legal__intro {
  color: var(--text-secondary);
  font-size: var(--text-lg);
}

.legal__doc :deep(h2) {
  margin-top: var(--space-8);
  color: var(--text-primary);
  font-size: var(--text-xl);
  font-weight: 600;
  letter-spacing: -0.01em;
}

.legal__doc :deep(h3) {
  margin-top: var(--space-5);
  color: var(--text-primary);
  font-size: var(--text-base);
  font-weight: 600;
}

.legal__doc :deep(p),
.legal__doc :deep(li) {
  margin-top: var(--space-3);
  color: var(--text-secondary);
  font-size: var(--text-base);
  line-height: 1.7;
}

.legal__doc :deep(ul) {
  margin-top: var(--space-3);
  padding-left: var(--space-5);
  list-style: disc;
}

.legal__doc :deep(li + li) {
  margin-top: var(--space-2);
}

.legal__doc :deep(a) {
  color: var(--tide);
  text-decoration: none;
}

.legal__doc :deep(a:hover) {
  color: var(--text-primary);
}

.legal__doc :deep(strong) {
  color: var(--text-primary);
  font-weight: 600;
}
</style>
