<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import AppIcon from '@/components/ui/AppIcon.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import SpoilerShield from '@/components/ui/SpoilerShield.vue'
import ThemeToggle from '@/components/ui/ThemeToggle.vue'
import SiteFooter from '@/components/layout/SiteFooter.vue'
import type { IconName } from '@/components/ui/icons'
import { useAuthStore } from '@/stores/auth'
import { useGuestStore } from '@/stores/guest'

const router = useRouter()
const auth = useAuthStore()
const guest = useGuestStore()

/** Both the masthead and the hero use one target and one name for the action. */
const entryTo = computed(() => (auth.isAuthenticated ? '/home' : '/register'))
const entryLabel = computed(() => (auth.isAuthenticated ? 'Go to your feed' : 'Create account'))

interface Experience {
  title: string
  body: string
  icon: IconName
  note?: string
}

// The four PRD experiences (§6.1). Chat is honest about its release.
const experiences: Experience[] = [
  {
    title: 'Discuss',
    body: 'Series-tagged communities with threaded discussions, polls and spoiler curtains that respect where you are in the story.',
    icon: 'communities',
  },
  {
    title: 'Share',
    body: 'Cosplay, fan art and photography in a feed that credits both the title and the person who made it.',
    icon: 'creative',
  },
  {
    title: 'Meet',
    body: 'Screenings, convention days and photo shoots with RSVPs, planned beside the community that runs them.',
    icon: 'events',
  },
  {
    title: 'Chat',
    body: 'Servers, channels and direct messages for watchalongs, and for everything a thread cannot hold.',
    icon: 'chat',
    note: 'Later release',
  },
]

// PRD §7.9 event types — plain pills, one row.
const eventKinds = ['Convention', 'Screening', 'Meetup', 'Photo shoot', 'Workshop']

interface Interest {
  id: string
  label: string
  /** Positions live in the 320-unit chart viewBox. */
  x: number
  y: number
}

// Sample interests for the map. They sit in a loose ring so any two plotted
// stars read as part of one constellation rather than scattered points.
const interests: Interest[] = [
  { id: 'frieren', label: 'Frieren', x: 96, y: 92 },
  { id: 'dungeon-meshi', label: 'Dungeon Meshi', x: 188, y: 60 },
  { id: 'cosplay', label: 'Cosplay craft', x: 238, y: 140 },
  { id: 'fan-art', label: 'Fan art', x: 198, y: 226 },
  { id: 'photography', label: 'Con photography', x: 108, y: 212 },
  { id: 'meetups', label: 'Local meetups', x: 62, y: 154 },
]

const plotted = ref<string[]>([])

function isPlotted(id: string): boolean {
  return plotted.value.includes(id)
}

function toggleInterest(id: string): void {
  plotted.value = isPlotted(id)
    ? plotted.value.filter((entry) => entry !== id)
    : [...plotted.value, id]
}

const plottedInterests = computed(() => interests.filter((interest) => isPlotted(interest.id)))

const constellation = computed(() =>
  plottedInterests.value.map((interest) => `${interest.x},${interest.y}`).join(' '),
)

const readout = computed(() => {
  const names = plottedInterests.value.map((interest) => interest.label)
  if (names.length === 0) return 'Choose an interest to plot it on the map.'
  if (names.length === 1) return `${names[0]} is plotted. Its community sits under the star.`
  const list = `${names.slice(0, -1).join(', ')} and ${names[names.length - 1]}`
  return `${list} are plotted. The lines mark the context they share.`
})

function labelX(interest: Interest): number {
  return interest.x > 176 ? interest.x - 12 : interest.x + 12
}

function labelAnchor(interest: Interest): 'start' | 'end' {
  return interest.x > 176 ? 'end' : 'start'
}

/** Guests get the public pages: events and creators, read-only. */
function browseAsGuest(): void {
  guest.enter()
  void router.push('/events')
}
</script>

<template>
  <div class="landing">
    <!-- ── Masthead ─────────────────────────────────────────────────────────── -->
    <header class="masthead reveal" style="--stagger: 0">
      <RouterLink to="/" class="masthead__brand" aria-label="PromisedLand home">
        <img class="masthead__logo" src="/logo.png" alt="" width="28" height="28" />
        <span class="wordmark">Promised<span class="wordmark__accent">Land</span></span>
      </RouterLink>

      <nav class="masthead__nav" aria-label="Sections on this page">
        <a href="#experiences">Experiences</a>
        <a href="#spoilers">Spoilers</a>
        <a href="#events">Events</a>
      </nav>

      <div class="masthead__actions">
        <ThemeToggle />
        <BaseButton
          v-if="!auth.isAuthenticated"
          class="masthead__signin"
          variant="ghost"
          size="md"
          to="/login"
        >
          Sign in
        </BaseButton>
        <BaseButton class="masthead__entry" variant="secondary" size="md" :to="entryTo">
          {{ entryLabel }}
        </BaseButton>
      </div>
    </header>

    <!-- ── Hero: the promise, plus the interest map ─────────────────────────── -->
    <section class="hero" aria-labelledby="hero-title">
      <div class="hero__copy">
        <h1 id="hero-title" class="hero__title reveal" style="--stagger: 1">
          <span>Your fandom.</span>
          <span>Your people.</span>
          <span>Your place.</span>
        </h1>

        <p class="hero__lede reveal" style="--stagger: 3">
          Communities for the series you follow, a feed for the cosplay and fan art you make, and
          events you can actually show up to.
        </p>

        <div class="hero__action reveal" style="--stagger: 5">
          <BaseButton variant="primary" size="lg" :to="entryTo">{{ entryLabel }}</BaseButton>
          <BaseButton
            v-if="!auth.isAuthenticated"
            variant="secondary"
            size="lg"
            type="button"
            @click="browseAsGuest"
          >
            Continue as a guest
          </BaseButton>
        </div>

        <p class="hero__note reveal" style="--stagger: 6">
          Beta opens with communities, the creator feed and events. Chat follows in a later release.
        </p>
      </div>

      <figure class="map reveal" style="--stagger: 2">
        <div class="map__frame">
          <svg viewBox="0 0 320 320" aria-hidden="true" focusable="false">
            <g class="map__grid">
              <line v-for="tick in [40, 80, 120, 160, 200, 240, 280]" :key="`v-${tick}`" :x1="tick" y1="0" :x2="tick" y2="320" />
              <line v-for="tick in [40, 80, 120, 160, 200, 240, 280]" :key="`h-${tick}`" x1="0" :y1="tick" x2="320" :y2="tick" />
            </g>

            <circle class="map__ring" cx="160" cy="160" r="122" />
            <path class="map__cross" d="M160 26v16M160 278v16M26 160h16M278 160h16" />

            <polyline
              v-if="plottedInterests.length > 1"
              class="map__link"
              :points="constellation"
            />

            <g
              v-for="interest in interests"
              :key="interest.id"
              class="map__star"
              :class="{ 'map__star--plotted': isPlotted(interest.id) }"
            >
              <circle
                v-if="isPlotted(interest.id)"
                class="map__star-ring"
                :cx="interest.x"
                :cy="interest.y"
                r="9"
              />
              <circle
                class="map__star-dot"
                :cx="interest.x"
                :cy="interest.y"
                :r="isPlotted(interest.id) ? 5 : 3"
              />
              <text
                v-if="isPlotted(interest.id)"
                class="map__star-label"
                :x="labelX(interest)"
                :y="interest.y + 4"
                :text-anchor="labelAnchor(interest)"
              >
                {{ interest.label }}
              </text>
            </g>
          </svg>
        </div>

        <div class="map__chips" role="group" aria-label="Plot an interest on the map">
          <button
            v-for="interest in interests"
            :key="interest.id"
            type="button"
            class="map__chip"
            :aria-pressed="isPlotted(interest.id)"
            @click="toggleInterest(interest.id)"
          >
            {{ interest.label }}
          </button>
        </div>

        <figcaption class="map__caption">
          <span class="map__readout" aria-live="polite">{{ readout }}</span>
          <span class="map__sample">Sample map, not live data.</span>
        </figcaption>
      </figure>
    </section>

    <!-- ── Experiences ──────────────────────────────────────────────────────── -->
    <section id="experiences" class="experiences" aria-labelledby="experiences-title">
      <header class="section-head reveal" style="--stagger: 7">
        <h2 id="experiences-title">One interest list, four ways to use it</h2>
        <p>
          You pick the series and crafts you follow once. Communities, the creator feed and events
          arrange themselves around that list.
        </p>
      </header>

      <ul class="experiences__grid">
        <li
          v-for="(item, index) in experiences"
          :key="item.title"
          class="experience reveal"
          :style="{ '--stagger': index + 8 }"
        >
          <span class="experience__glyph" aria-hidden="true">
            <AppIcon :name="item.icon" :size="22" />
          </span>
          <h3>{{ item.title }}</h3>
          <p>{{ item.body }}</p>
          <span v-if="item.note" class="pill">{{ item.note }}</span>
        </li>
      </ul>
    </section>

    <!-- ── Spoilers ─────────────────────────────────────────────────────────── -->
    <section id="spoilers" class="split" aria-labelledby="spoilers-title">
      <div class="split__copy reveal" style="--stagger: 12">
        <h2 id="spoilers-title">Spoilers stay covered until you choose</h2>
        <p>
          Anything marked as a spoiler arrives behind a cover, so the ending of a finale does not
          land in your scroll.
        </p>
        <p class="split__quiet">
          The tag comes from the person posting, so one can be missed. Every post also carries a
          report reason for an unmarked spoiler.
        </p>
      </div>

      <div class="split__demo reveal" style="--stagger: 13">
        <p class="demo__label">A reply marked as a spoiler</p>
        <SpoilerShield
          label="Reveal spoiler"
          hint="Marked for episode 9. Reveal when you are caught up."
        >
          <p class="demo__reply">
            “The lighthouse keeper has been the narrator's father the whole time. The flashback at
            the end of episode 9 re-frames the entire first act.”
          </p>
          <p class="demo__meta">
            <span>Episode 9</span>
            <span>Marked by the poster</span>
          </p>
        </SpoilerShield>
        <p class="demo__sample">Sample post, not live data.</p>
      </div>
    </section>

    <!-- ── Events ───────────────────────────────────────────────────────────── -->
    <section id="events" class="split" aria-labelledby="events-title">
      <div class="split__copy reveal" style="--stagger: 14">
        <h2 id="events-title">From a thread to a meetup</h2>
        <p>
          Events sit beside the communities that plan them. Publish a screening or a convention day,
          collect RSVPs, and keep the plan next to the conversation.
        </p>
        <ul class="kinds">
          <li v-for="kind in eventKinds" :key="kind" class="pill">{{ kind }}</li>
        </ul>
        <RouterLink class="text-link" to="/events">Browse events</RouterLink>
      </div>

      <article class="pass reveal" style="--stagger: 15">
        <div
          class="pass__media"
          role="img"
          aria-label="Poster for a sample event: a constellation above a harbour light at night"
        >
          <svg viewBox="0 0 320 320" preserveAspectRatio="xMidYMid slice" aria-hidden="true" focusable="false">
            <rect width="320" height="320" fill="var(--ink-700)" />

            <g stroke="var(--border-hairline)" stroke-width="1">
              <line v-for="tick in [64, 128, 192, 256]" :key="`pv-${tick}`" :x1="tick" y1="0" :x2="tick" y2="232" />
              <line v-for="tick in [64, 128, 192]" :key="`ph-${tick}`" x1="0" :y1="tick" x2="320" :y2="tick" />
            </g>

            <path
              d="M62 84 L118 56 L186 74 L238 122 L222 176 L152 206 L92 178 L54 130 Z"
              fill="none"
              stroke="var(--tide)"
              stroke-opacity="0.45"
              stroke-width="1"
            />
            <g fill="var(--tide)">
              <circle cx="62" cy="84" r="2.5" />
              <circle cx="118" cy="56" r="2.5" />
              <circle cx="186" cy="74" r="2.5" />
              <circle cx="222" cy="176" r="2.5" />
              <circle cx="152" cy="206" r="2.5" />
              <circle cx="92" cy="178" r="2.5" />
              <circle cx="54" cy="130" r="2.5" />
            </g>
            <circle cx="238" cy="122" r="4" fill="var(--ember)" />

            <g fill="var(--ink-600)">
              <rect x="26" y="192" width="46" height="40" />
              <rect x="252" y="180" width="42" height="52" />
            </g>
            <g fill="var(--text-primary)" fill-opacity="0.6">
              <rect x="260" y="188" width="4" height="6" />
              <rect x="272" y="188" width="4" height="6" />
              <rect x="260" y="202" width="4" height="6" />
            </g>

            <line x1="0" y1="232" x2="320" y2="232" stroke="var(--border-strong)" stroke-width="1" />
            <rect y="232" width="320" height="88" fill="var(--ink-800)" />
            <circle cx="160" cy="214" r="9" fill="var(--ember)" />
            <rect x="155" y="232" width="10" height="52" fill="var(--ember)" fill-opacity="0.28" />
          </svg>

          <div class="pass__date">
            <span class="pass__day">14</span>
            <span class="pass__month">Mar</span>
          </div>
        </div>

        <div class="pass__body">
          <p class="pass__kind">Screening</p>
          <h3 class="pass__title">Harbour Light: episode 12</h3>

          <dl class="pass__details">
            <div>
              <dt>Time</dt>
              <dd>19:00</dd>
            </div>
            <div>
              <dt>Place</dt>
              <dd>Shimokitazawa</dd>
            </div>
            <div>
              <dt>Host</dt>
              <dd>c/harbour-light</dd>
            </div>
            <div>
              <dt>Going</dt>
              <dd class="mono">18 of 24</dd>
            </div>
          </dl>

          <p class="pass__sample">Sample event, not live data.</p>
        </div>
      </article>
    </section>

    <!-- ── Closing ──────────────────────────────────────────────────────────── -->
    <section class="closing" aria-label="Join PromisedLand">
      <div class="closing__inner reveal" style="--stagger: 16">
        <img class="closing__logo" src="/logo.png" alt="PromisedLand" />
        <p class="closing__tagline">Your fandom. Your people. Your place.</p>
        <BaseButton variant="secondary" size="lg" :to="entryTo">{{ entryLabel }}</BaseButton>
      </div>
    </section>

    <SiteFooter />
  </div>
</template>

<style scoped>
.landing {
  display: flex;
  flex-direction: column;
}

/* Wordmark: brand type, no decorative axes. */
.wordmark {
  font-family: var(--font-display);
  font-size: var(--text-lg);
  font-weight: 600;
  letter-spacing: -0.02em;
  line-height: 1.1;
  color: var(--text-primary);
}

.wordmark__accent {
  color: var(--ember);
}

/* ── Masthead ───────────────────────────────────────────────────────────────── */
.masthead {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-4);
  padding-block: var(--space-5);
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

.masthead__nav {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-6);
}

.masthead__nav a {
  display: inline-flex;
  align-items: center;
  min-height: 44px;
  color: var(--text-secondary);
  font-size: var(--text-base);
  font-weight: 500;
  text-decoration: none;
  transition: color var(--dur-fast) var(--ease);
}

.masthead__nav a:hover {
  color: var(--text-primary);
}

.masthead__actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

/* ── Hero ───────────────────────────────────────────────────────────────────── */
.hero {
  display: grid;
  grid-template-columns: minmax(0, 1.02fr) minmax(0, 0.98fr);
  align-items: center;
  gap: var(--space-16);
  padding-block: var(--space-12) var(--space-20);
}

.hero__copy {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--space-6);
}

.hero__title {
  display: flex;
  flex-direction: column;
  font-family: var(--font-display);
  font-size: var(--text-hero);
  font-weight: 600;
  letter-spacing: -0.03em;
  line-height: 1.04;
}

.hero__title span {
  display: block;
}

.hero__lede {
  max-width: 46ch;
  color: var(--text-secondary);
  font-size: var(--text-lg);
}

.hero__action {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-4);
}

.hero__note {
  max-width: 48ch;
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

/* ── Interest map (the signature moment) ────────────────────────────────────── */
.map {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  width: 100%;
  max-width: 520px;
  margin: 0 auto;
}

.map__frame {
  overflow: hidden;
  aspect-ratio: 1;
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-xl);
  background: var(--ink-850);
  box-shadow: var(--shadow-1);
}

.map__frame svg {
  display: block;
  width: 100%;
  height: 100%;
}

.map__grid line {
  stroke: var(--border-hairline);
  stroke-width: 1;
}

.map__ring {
  fill: none;
  stroke: var(--border-hairline);
  stroke-dasharray: 2 8;
}

.map__cross {
  fill: none;
  stroke: var(--border-strong);
  stroke-width: 1;
}

.map__link {
  fill: none;
  stroke: var(--tide);
  stroke-width: 1.5;
  stroke-linejoin: round;
  stroke-opacity: 0.55;
}

.map__star-dot {
  fill: var(--text-secondary);
  opacity: 0.5;
  transition: opacity var(--dur) var(--ease);
}

.map__star-ring {
  fill: none;
  stroke: var(--ember);
  stroke-width: 1;
  stroke-opacity: 0.4;
}

.map__star--plotted .map__star-dot {
  fill: var(--ember);
  opacity: 1;
}

.map__star-label {
  font-family: var(--font-body);
  font-size: 9px;
  font-weight: 500;
  fill: var(--text-primary);
}

.map__caption {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.map__readout,
.map__sample {
  color: var(--text-secondary);
  font-size: var(--text-sm);
  line-height: 1.5;
}

.map__readout {
  min-height: 42px;
}

.map__chips {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.map__chip {
  min-height: 44px;
  padding: 0 var(--space-4);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-pill);
  background: transparent;
  color: var(--text-secondary);
  font-family: var(--font-body);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
  transition:
    color var(--dur-fast) var(--ease),
    background-color var(--dur-fast) var(--ease),
    border-color var(--dur-fast) var(--ease);
}

.map__chip:hover {
  border-color: var(--border-strong);
  color: var(--text-primary);
}

.map__chip[aria-pressed='true'] {
  border-color: var(--ember);
  background: var(--ember-soft);
  color: var(--text-primary);
}

/* ── Shared section rhythm ──────────────────────────────────────────────────── */
.experiences,
.split,
.closing {
  border-top: 1px solid var(--border-hairline);
}

.experiences,
.split {
  padding-block: var(--space-16);
}

#experiences,
#spoilers,
#events {
  scroll-margin-top: var(--space-6);
}

.section-head {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  max-width: 56ch;
  padding-bottom: var(--space-10);
}

.section-head h2,
.split h2 {
  font-size: var(--text-2xl);
  font-weight: 600;
  letter-spacing: -0.02em;
}

.section-head p {
  color: var(--text-secondary);
  font-size: var(--text-lg);
}

/* ── Experiences ────────────────────────────────────────────────────────────── */
.experiences__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.experience {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--space-3);
  padding: var(--space-6);
}

.experience:nth-child(odd) {
  padding-left: 0;
}

.experience:nth-child(even) {
  border-left: 1px solid var(--border-hairline);
}

.experience:nth-child(n + 3) {
  border-top: 1px solid var(--border-hairline);
}

.experience__glyph {
  display: grid;
  width: 40px;
  height: 40px;
  place-items: center;
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-md);
  background: var(--ink-850);
  color: var(--text-secondary);
}

.experience h3 {
  font-size: var(--text-xl);
  font-weight: 600;
  letter-spacing: -0.01em;
}

.experience p {
  max-width: 42ch;
  color: var(--text-secondary);
}

.pill {
  padding: var(--space-1) var(--space-3);
  border-radius: var(--radius-pill);
  background: var(--ink-800);
  color: var(--text-secondary);
  font-size: var(--text-xs);
  line-height: 1.5;
}

/* ── Split sections: spoilers, events ───────────────────────────────────────── */
.split {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  align-items: center;
  gap: var(--space-16);
}

.split__copy {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--space-4);
}

.split__copy p {
  max-width: 52ch;
  color: var(--text-secondary);
}

.split__quiet {
  font-size: var(--text-sm);
}

.split__demo {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-6);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-lg);
  background: var(--ink-850);
}

.demo__label {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.demo__reply {
  color: var(--text-primary);
}

.demo__meta {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-4);
  padding-top: var(--space-3);
  border-top: 1px solid var(--border-hairline);
  color: var(--text-secondary);
  font-size: var(--text-xs);
}

.demo__sample,
.pass__sample {
  color: var(--text-secondary);
  font-size: var(--text-xs);
}

.kinds {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.text-link {
  display: inline-flex;
  align-items: center;
  min-height: 44px;
  color: var(--tide);
  font-weight: 500;
  text-decoration: none;
  transition: color var(--dur-fast) var(--ease);
}

.text-link:hover {
  color: var(--text-primary);
}

/* ── Event pass: the one rich media surface ─────────────────────────────────── */
.pass {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  width: 100%;
  max-width: 380px;
  margin-inline: auto;
  padding: var(--space-4);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-lg);
  background: var(--ink-800);
  box-shadow: var(--shadow-1);
}

.pass__media {
  position: relative;
  overflow: hidden;
  aspect-ratio: 1;
  border-radius: var(--radius-md);
  background: var(--ink-700);
}

.pass__media svg {
  display: block;
  width: 100%;
  height: 100%;
}

.pass__date {
  position: absolute;
  top: var(--space-3);
  left: var(--space-3);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm);
  background: var(--ember);
  color: var(--ink-900);
  line-height: 1;
}

.pass__day {
  font-family: var(--font-display);
  font-size: var(--text-xl);
  font-weight: 600;
}

.pass__month {
  font-size: var(--text-xs);
}

.pass__body {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding: 0 var(--space-2) var(--space-2);
}

.pass__kind {
  color: var(--text-secondary);
  font-size: var(--text-xs);
}

.pass__title {
  font-size: var(--text-xl);
  font-weight: 600;
  letter-spacing: -0.01em;
  line-height: 1.25;
}

.pass__details {
  display: grid;
  gap: 0;
  margin: 0;
  border-top: 1px solid var(--border-hairline);
}

.pass__details div {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: var(--space-4);
  padding-block: var(--space-2);
  border-bottom: 1px solid var(--border-hairline);
}

.pass__details dt {
  color: var(--text-secondary);
  font-size: var(--text-xs);
}

.pass__details dd {
  margin: 0;
  color: var(--text-primary);
  font-size: var(--text-sm);
}

/* ── Closing + footer ───────────────────────────────────────────────────────── */
.closing {
  padding-block: var(--space-16);
}

.closing__inner {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-4);
  max-width: 44ch;
  margin-inline: auto;
  text-align: center;
}

.closing__logo {
  width: min(220px, 70vw);
  height: auto;
  border-radius: var(--radius-sm);
}

.closing__tagline {
  color: var(--text-secondary);
  font-size: var(--text-lg);
}

/* ── Responsive ─────────────────────────────────────────────────────────────── */
@media (max-width: 1023px) {
  .hero {
    grid-template-columns: minmax(0, 1fr);
    gap: var(--space-10);
    padding-block: var(--space-10) var(--space-16);
  }

  .split {
    grid-template-columns: minmax(0, 1fr);
    gap: var(--space-8);
  }

  .split__copy {
    max-width: 62ch;
  }
}

@media (max-width: 767px) {
  .masthead__nav,
  .masthead__signin,
  .masthead__entry {
    display: none;
  }

  .wordmark {
    font-size: var(--text-base);
  }

  .experiences,
  .split,
  .closing {
    padding-block: var(--space-12);
  }

  .section-head {
    padding-bottom: var(--space-6);
  }

  .experiences__grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .experience,
  .experience:nth-child(odd),
  .experience:nth-child(even) {
    padding: var(--space-5) 0;
    border-top: 1px solid var(--border-hairline);
    border-left: 0;
  }

  .experience:first-child {
    padding-top: 0;
    border-top: 0;
  }

  .split__demo {
    padding: var(--space-4);
  }

  .footer__links,
  .footer__tagline {
    width: 100%;
    margin-left: 0;
  }
}
</style>
