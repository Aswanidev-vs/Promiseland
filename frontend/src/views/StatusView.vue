<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import BrandMasthead from '@/components/layout/BrandMasthead.vue'
import SiteFooter from '@/components/layout/SiteFooter.vue'

type ProbeState = 'ok' | 'fail'
type Overall = 'operational' | 'degraded' | 'unreachable'

interface Probe {
  id: string
  label: string
  description: string
  path: string
}

interface ProbeResult {
  state: ProbeState
  status: number | null
  latencyMs: number | null
  detail: string
}

interface HistoryEntry {
  at: number
  overall: Overall
  latencyMs: number | null
}

const probes: Probe[] = [
  {
    id: 'api',
    label: 'API',
    description: 'The service behind every request in the app.',
    path: '/healthz',
  },
  {
    id: 'database',
    label: 'Database readiness',
    description: 'Whether the database can take reads and writes right now.',
    path: '/readyz',
  },
]

const refreshMs = 15000
const timeoutMs = 5000
const historyLimit = 20

const results = ref<Record<string, ProbeResult>>({})
const checking = ref(true)
const lastChecked = ref<number | null>(null)
const autoRefresh = ref(true)
const history = ref<HistoryEntry[]>([])

const overall = computed<Overall>(() => {
  const api = results.value.api
  const database = results.value.database
  if (api?.state === 'ok' && database?.state === 'ok') return 'operational'
  if (api?.state === 'ok' || database?.state === 'ok') return 'degraded'
  return 'unreachable'
})

const headline = computed(() => {
  if (lastChecked.value === null) return 'Checking the service'
  if (overall.value === 'operational') return 'All systems operational'
  if (overall.value === 'degraded') return 'Partially available'
  return 'PromisedLand is unreachable from this browser'
})

const headlineNote = computed(() => {
  if (lastChecked.value === null) return 'The first check runs as soon as this page opens.'
  if (overall.value === 'operational') return 'Sign-in, feeds, events and uploads all have what they need.'
  if (overall.value === 'degraded') return 'One part of the service is answering, the other is not. Requests that need the database will fail.'
  return 'The API did not answer. Nothing you did caused this; check again in a moment.'
})

const summary = computed(() => {
  if (lastChecked.value === null) return 'Waiting for the first result.'
  const time = formatTime(lastChecked.value)
  return autoRefresh.value
    ? `Last checked at ${time}. Repeats every 15 seconds while this tab is open.`
    : `Last checked at ${time}. Automatic checks are paused.`
})

const healthyChecks = computed(() => history.value.filter((entry) => entry.overall === 'operational').length)

const slowestLatency = computed(() =>
  history.value.reduce((max, entry) => Math.max(max, entry.latencyMs ?? 0), 0),
)

let timer: number | undefined

function isStatusPayload(value: unknown): boolean {
  return typeof value === 'object' && value !== null && typeof (value as { status?: unknown }).status === 'string'
}

function apiMessage(value: unknown): string {
  const error = (value as { error?: { code?: unknown; message?: unknown } } | null)?.error
  // The dev proxy and a stopped API both answer with a code meant for the
  // operator; the page says something a visitor can act on instead.
  if (error?.code === 'api_unavailable') return 'The API is not reachable in this environment.'
  return typeof error?.message === 'string' ? error.message : ''
}

async function probe(check: Probe): Promise<ProbeResult> {
  const controller = new AbortController()
  const timeout = window.setTimeout(() => controller.abort(), timeoutMs)
  const started = performance.now()

  try {
    const response = await fetch(check.path, {
      signal: controller.signal,
      headers: { Accept: 'application/json' },
      cache: 'no-store',
    })
    const latencyMs = Math.round(performance.now() - started)

    let payload: unknown = null
    try {
      payload = await response.json()
    } catch {
      return {
        state: 'fail',
        status: response.status,
        latencyMs,
        detail: 'The endpoint answered with something other than JSON.',
      }
    }

    if (!response.ok) {
      return {
        state: 'fail',
        status: response.status,
        latencyMs,
        detail: apiMessage(payload) || `The endpoint answered with HTTP ${response.status}.`,
      }
    }
    if (!isStatusPayload(payload)) {
      return {
        state: 'fail',
        status: response.status,
        latencyMs,
        detail: 'The endpoint answered without a status payload.',
      }
    }
    return { state: 'ok', status: response.status, latencyMs, detail: '' }
  } catch (cause) {
    const aborted = cause instanceof DOMException && cause.name === 'AbortError'
    return {
      state: 'fail',
      status: null,
      latencyMs: null,
      detail: aborted ? 'No answer within 5 seconds.' : 'The service could not be reached.',
    }
  } finally {
    window.clearTimeout(timeout)
  }
}

async function check(): Promise<void> {
  if (checking.value && lastChecked.value !== null) return
  checking.value = true

  const next: Record<string, ProbeResult> = {}
  await Promise.all(
    probes.map(async (item) => {
      next[item.id] = await probe(item)
    }),
  )

  results.value = next
  checking.value = false
  lastChecked.value = Date.now()

  const latencies = Object.values(next)
    .map((result) => result.latencyMs)
    .filter((value): value is number => value !== null)
  history.value = [
    ...history.value,
    {
      at: lastChecked.value,
      overall: overall.value,
      latencyMs: latencies.length ? Math.max(...latencies) : null,
    },
  ].slice(-historyLimit)
}

function schedule(): void {
  window.clearInterval(timer)
  if (!autoRefresh.value) return
  timer = window.setInterval(() => {
    if (!document.hidden) void check()
  }, refreshMs)
}

watch(autoRefresh, schedule)

onMounted(() => {
  void check()
  schedule()
})

onBeforeUnmount(() => {
  window.clearInterval(timer)
})

function formatTime(value: number): string {
  return new Intl.DateTimeFormat(undefined, {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  }).format(new Date(value))
}

function stateLabel(result: ProbeResult | undefined): string {
  if (!result) return 'Checking'
  return result.state === 'ok' ? 'Operational' : 'Not answering'
}

function barHeight(entry: HistoryEntry): number {
  if (entry.latencyMs === null) return 28
  const max = slowestLatency.value
  if (max <= 0) return 8
  return Math.round(8 + (entry.latencyMs / max) * 20)
}

function barTitle(entry: HistoryEntry): string {
  const latency = entry.latencyMs === null ? 'no answer' : `${entry.latencyMs} ms`
  return `${formatTime(entry.at)}: ${entry.overall}, ${latency}`
}
</script>

<template>
  <div class="status-page">
    <BrandMasthead />

    <main class="status-page__body">
      <header class="status-page__head">
        <p class="status-page__kind">Service status</p>
        <h1 class="status-page__headline" aria-live="polite">
          <span class="status-page__dot" :class="`status-page__dot--${lastChecked === null ? 'checking' : overall}`" aria-hidden="true" />
          {{ headline }}
        </h1>
        <p class="status-page__note">{{ headlineNote }}</p>
        <p class="status-page__summary">{{ summary }}</p>

        <div class="status-page__controls">
          <BaseButton variant="secondary" size="md" type="button" :loading="checking" @click="check">
            Check again
          </BaseButton>
          <label class="status-page__auto">
            <input v-model="autoRefresh" type="checkbox" class="status-page__auto-box" />
            <span>Check again every 15 seconds</span>
          </label>
        </div>
      </header>

      <ul class="status-page__checks">
        <li v-for="item in probes" :key="item.id" class="check">
          <div class="check__text">
            <h2 class="check__label">{{ item.label }}</h2>
            <p class="check__description">{{ item.description }}</p>
          </div>

          <div class="check__result">
            <span
              class="check__state"
              :class="{ 'check__state--fail': results[item.id]?.state === 'fail' }"
            >
              {{ stateLabel(results[item.id]) }}
            </span>
            <span v-if="results[item.id]?.latencyMs != null" class="check__meta mono">
              {{ results[item.id]?.latencyMs }} ms
            </span>
            <span v-if="results[item.id]?.status != null" class="check__meta mono">
              HTTP {{ results[item.id]?.status }}
            </span>
          </div>

          <p v-if="results[item.id]?.detail" class="check__detail">
            {{ results[item.id]?.detail }}
          </p>
        </li>
      </ul>

      <section v-if="history.length" class="status-page__history" aria-labelledby="history-title">
        <h2 id="history-title" class="status-page__history-title">Checks since you opened this page</h2>
        <div class="bars" aria-hidden="true">
          <span
            v-for="(entry, index) in history"
            :key="index"
            class="bars__bar"
            :class="`bars__bar--${entry.overall}`"
            :style="{ height: `${barHeight(entry)}px` }"
            :title="barTitle(entry)"
          />
        </div>
        <p class="status-page__note">
          {{ healthyChecks }} of {{ history.length }}
          {{ history.length === 1 ? 'check was' : 'checks were' }} fully healthy. Taller bars mean slower answers.
        </p>
      </section>

      <p class="status-page__note">
        This page reports what this browser can observe right now. It is not a public uptime history, and it
        checks only the API and its database.
      </p>
    </main>

    <SiteFooter />
  </div>
</template>

<style scoped>
.status-page {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

.status-page__body {
  flex: 1;
  width: 100%;
  max-width: 720px;
  padding-block: var(--space-12) var(--space-16);
}

.status-page__head {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding-bottom: var(--space-8);
  border-bottom: 1px solid var(--border-hairline);
}

.status-page__kind {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.status-page__headline {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  font-family: var(--font-display);
  font-size: var(--text-2xl);
  font-weight: 600;
  letter-spacing: -0.02em;
  line-height: 1.15;
}

.status-page__dot {
  flex: none;
  width: 10px;
  height: 10px;
  border-radius: var(--radius-pill);
  background: var(--text-secondary);
}

.status-page__dot--operational {
  background: var(--ok);
}

.status-page__dot--degraded,
.status-page__dot--unreachable {
  background: var(--alert);
}

.status-page__note,
.status-page__summary {
  max-width: 58ch;
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.status-page__summary {
  font-variant-numeric: tabular-nums;
}

.status-page__controls {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-4);
  padding-top: var(--space-2);
}

.status-page__auto {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  min-height: 44px;
  color: var(--text-secondary);
  font-size: var(--text-sm);
  cursor: pointer;
}

.status-page__auto-box {
  width: 18px;
  height: 18px;
  margin: 0;
  accent-color: var(--ember);
}

.status-page__checks {
  display: flex;
  flex-direction: column;
}

.check {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: var(--space-2) var(--space-6);
  padding-block: var(--space-5);
  border-bottom: 1px solid var(--border-hairline);
}

.check__text {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.check__label {
  font-size: var(--text-lg);
  font-weight: 600;
  letter-spacing: -0.01em;
}

.check__description {
  max-width: 46ch;
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.check__result {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  justify-content: flex-end;
  gap: var(--space-3);
}

.check__state {
  color: var(--ok);
  font-size: var(--text-sm);
  font-weight: 600;
}

.check__state--fail {
  color: var(--alert);
}

.check__meta {
  color: var(--text-secondary);
  font-size: var(--text-xs);
}

.check__detail {
  grid-column: 1 / -1;
  color: var(--alert);
  font-size: var(--text-sm);
}

.status-page__history {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding-block: var(--space-8);
  border-bottom: 1px solid var(--border-hairline);
}

.status-page__history-title {
  font-size: var(--text-lg);
  font-weight: 600;
}

.bars {
  display: flex;
  align-items: flex-end;
  gap: var(--space-1);
  height: 32px;
  padding-block: var(--space-1);
}

.bars__bar {
  flex: 1 1 0;
  min-width: 4px;
  border-radius: var(--radius-sm) var(--radius-sm) 0 0;
  background: var(--ok);
}

.bars__bar--degraded,
.bars__bar--unreachable {
  background: var(--alert);
}

.status-page__body > .status-page__note:last-child {
  padding-top: var(--space-6);
}

@media (max-width: 767px) {
  .status-page__body {
    padding-block: var(--space-8) var(--space-12);
  }

  .status-page__headline {
    font-size: var(--text-xl);
  }

  .check {
    grid-template-columns: minmax(0, 1fr);
  }

  .check__result {
    justify-content: flex-start;
  }
}
</style>
