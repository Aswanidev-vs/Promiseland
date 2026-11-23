<script setup lang="ts">
import BaseButton from '@/components/ui/BaseButton.vue'
import WayfinderMark from '@/components/ui/WayfinderMark.vue'
import AppIcon from '@/components/ui/AppIcon.vue'
import type { IconName } from '@/components/ui/icons'

// R1: point this at '/home' when the authenticated feed route is registered.
const entryTo = '/'

interface Feature {
  title: string
  body: string
  icon: IconName
}

const features: Feature[] = [
  {
    title: 'Find your people',
    body: 'Join communities for the series you love, the characters you talk about, and the clubs near you.',
    icon: 'communities',
  },
  {
    title: 'Share what you make',
    body: 'Post cosplay, fan art, and photos with spoiler controls that respect where you are in the story.',
    icon: 'creative',
  },
  {
    title: 'Meet beyond the thread',
    body: 'Turn online conversations into meetups, screenings, and convention days with people who share your interests.',
    icon: 'events',
  },
]
</script>

<template>
  <div class="landing">
    <section class="hero" aria-labelledby="hero-title">
      <div class="hero__copy">
        <p class="hero__eyebrow reveal" style="--stagger: 0">
          <span class="hero__beacon" aria-hidden="true" />
          A harbour for anime fans
        </p>

        <h1 id="hero-title" class="hero__title reveal" style="--stagger: 1">
          <span>Your fandom.</span>
          <span>Your people.</span>
          <span>Your place.</span>
        </h1>

        <p class="hero__lede reveal" style="--stagger: 2">
          PromisedLand brings your anime interests, communities, creative work, and real-world meetups together in one welcoming place.
        </p>

        <div class="hero__action reveal" style="--stagger: 3">
          <BaseButton variant="primary" size="lg" :to="entryTo">Enter PromisedLand</BaseButton>
        </div>

        <p class="hero__note reveal" style="--stagger: 4">
          The home feed opens with the social release.
        </p>
      </div>

      <div class="hero__chart reveal" style="--stagger: 2">
        <div class="hero__chart-frame">
          <WayfinderMark
            detailed
            size="min(100%, 400px)"
            title="The wayfinder: a compass constellation marking your place in fandom"
          />
          <span class="hero__chart-caption mono">The wayfinder — find your bearing</span>
        </div>
      </div>
    </section>

    <section class="features" aria-labelledby="features-title">
      <header class="features__head reveal" style="--stagger: 5">
        <h2 id="features-title">Built for fandom, in all its forms</h2>
        <p>Start with the series you love. Stay for the people who are still talking after the credits.</p>
      </header>

      <ul class="features__grid">
        <li
          v-for="(feature, index) in features"
          :key="feature.title"
          class="feature reveal"
          :style="{ '--stagger': index + 6 }"
        >
          <span class="feature__glyph" aria-hidden="true">
            <AppIcon :name="feature.icon" :size="22" />
          </span>
          <h3>{{ feature.title }}</h3>
          <p>{{ feature.body }}</p>
        </li>
      </ul>
    </section>

    <footer class="landing__footer">
      <WayfinderMark size="28px" />
      <span class="landing__footer-name">PromisedLand</span>
      <span class="landing__footer-tagline">Your fandom. Your people. Your place.</span>
    </footer>
  </div>
</template>

<style scoped>
.landing {
  padding-top: var(--space-10);
}

/* ── Hero ──────────────────────────────────────────────────────────────────── */
.hero {
  display: grid;
  grid-template-columns: minmax(0, 1.05fr) minmax(0, 0.95fr);
  align-items: center;
  gap: var(--space-12);
  padding-bottom: var(--space-20);
}

.hero__copy {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--space-6);
}

.hero__eyebrow {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--tide);
  font-family: var(--font-mono);
  font-size: var(--text-xs);
  letter-spacing: 0.02em;
}

.hero__beacon {
  width: 7px;
  height: 7px;
  border: 1px solid var(--tide);
  border-radius: var(--radius-pill);
}

.hero__title {
  display: flex;
  flex-direction: column;
  font-size: clamp(var(--text-3xl), 5.2vw, var(--text-4xl));
  font-weight: 600;
}

.hero__lede {
  max-width: 48ch;
  color: var(--text-secondary);
  font-size: var(--text-lg);
}

.hero__action {
  padding-top: var(--space-2);
}

.hero__note {
  color: var(--text-muted);
  font-size: var(--text-sm);
}

.hero__chart {
  position: relative;
}

.hero__chart-frame {
  position: relative;
  display: grid;
  overflow: hidden;
  place-items: center;
  aspect-ratio: 1;
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-lg);
  background: var(--ink-850);
}

.hero__chart-frame::before {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(var(--border-hairline) 1px, transparent 1px),
    linear-gradient(90deg, var(--border-hairline) 1px, transparent 1px);
  background-size: 40px 40px;
  content: '';
  opacity: 0.35;
  pointer-events: none;
}

.hero__chart-frame :deep(.wayfinder) {
  position: relative;
  z-index: 1;
}

.hero__chart-caption {
  position: absolute;
  right: var(--space-4);
  bottom: var(--space-4);
  left: var(--space-4);
  z-index: 1;
  color: var(--text-secondary);
  font-size: var(--text-xs);
  text-align: center;
}

/* ── Features ──────────────────────────────────────────────────────────────── */
.features {
  padding-block: var(--space-12) var(--space-16);
  border-top: 1px solid var(--border-hairline);
}

.features__head {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  max-width: 55ch;
  padding-bottom: var(--space-10);
}

.features__head p {
  color: var(--text-secondary);
  font-size: var(--text-lg);
}

.features__grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--space-8);
}

.feature {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--space-3);
  padding-left: var(--space-6);
  border-left: 1px solid var(--border-hairline);
}

.feature:first-child {
  padding-left: 0;
  border-left: 0;
}

.feature__glyph {
  display: grid;
  width: 44px;
  height: 44px;
  place-items: center;
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-md);
  background: var(--ink-800);
  color: var(--tide);
}

.feature h3 {
  font-size: var(--text-xl);
}

.feature p {
  color: var(--text-secondary);
  max-width: 34ch;
}

/* ── Footer ────────────────────────────────────────────────────────────────── */
.landing__footer {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-3);
  padding-block: var(--space-6) var(--space-10);
  border-top: 1px solid var(--border-hairline);
}

.landing__footer-name {
  font-family: var(--font-display);
  font-size: var(--text-base);
  font-weight: 600;
}

.landing__footer-tagline {
  margin-left: auto;
  color: var(--text-muted);
  font-size: var(--text-sm);
}

@media (max-width: 1023px) {
  .hero {
    grid-template-columns: minmax(0, 1fr);
    gap: var(--space-10);
  }

  .hero__chart {
    max-width: 420px;
  }
}

@media (max-width: 767px) {
  .landing {
    padding-top: var(--space-6);
  }

  .hero {
    padding-bottom: var(--space-12);
  }

  .hero__title {
    font-size: clamp(2.25rem, 11vw, var(--text-3xl));
  }

  .features__grid {
    grid-template-columns: minmax(0, 1fr);
    gap: var(--space-6);
  }

  .feature,
  .feature:first-child {
    padding: var(--space-4) 0 0;
    border-top: 1px solid var(--border-hairline);
    border-left: 0;
  }

  .landing__footer-tagline {
    width: 100%;
    margin-left: 0;
  }
}
</style>
