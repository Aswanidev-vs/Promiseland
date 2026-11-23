<script setup lang="ts">
/**
 * Release 5 — gated adults-only area. Deliberately unlinked from navigation.
 *
 * The very first call is GET /adult/status. Disabled or ineligible responses
 * render only a requirements panel; no profile form, matches, or discovery UI
 * is mounted in that state. A 404 (flag off) is treated as "not available",
 * not as a network error, and a 403 as permission denied.
 */
import { computed, onMounted, ref, watch } from 'vue'
import BaseCard from '@/components/ui/BaseCard.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import SkeletonList from '@/components/ui/SkeletonList.vue'
import ReportDialog from '@/components/features/ReportDialog.vue'
import AdultEligibilityPanel from '@/components/features/AdultEligibilityPanel.vue'
import AdultMatchCard from '@/components/features/AdultMatchCard.vue'
import AdultProfileForm from '@/components/features/AdultProfileForm.vue'
import { isForbidden, isNotFound } from '@/components/features/format'
import { errorMessage } from '@/api/client'
import {
  getAdultProfile,
  getAdultStatus,
  listAdultMatches,
  reportAdult,
  saveAdultProfile,
  unmatch,
} from '@/api/adult'
import { useAuthStore } from '@/stores/auth'
import type { AdultMatch, AdultProfile, AdultProfileInput, AdultStatus, UserSummary } from '@/types'

const auth = useAuthStore()
const currentUserId = computed(() => auth.user?.id ?? null)

const status = ref<AdultStatus | null>(null)
const statusLoading = ref(true)
const statusError = ref('')
const gateUnavailable = ref(false)
const gateDenied = ref(false)
const available = computed(() => status.value?.enabled === true && status.value?.eligible === true)

const profile = ref<AdultProfile | null>(null)
const matches = ref<AdultMatch[]>([])
const dataLoading = ref(false)
const dataError = ref('')
const dataDenied = ref(false)

const saving = ref(false)
const saveError = ref('')
const saveNotice = ref('')
const busyMatchId = ref<number | null>(null)
const matchNotice = ref('')
const reportTarget = ref<UserSummary | null>(null)

async function loadStatus(): Promise<void> {
  statusLoading.value = true
  statusError.value = ''
  gateUnavailable.value = false
  gateDenied.value = false
  try {
    status.value = await getAdultStatus()
    if (!status.value.enabled || !status.value.eligible) gateUnavailable.value = true
  } catch (cause) {
    status.value = null
    if (isForbidden(cause)) {
      gateDenied.value = true
      gateUnavailable.value = true
    } else if (isNotFound(cause)) {
      // FEATURE_ADULT_DATING is off: the namespace does not exist.
      gateUnavailable.value = true
    } else {
      statusError.value = errorMessage(cause, 'We could not check whether this area is available. Try again.')
    }
  } finally {
    statusLoading.value = false
  }
}

async function loadProfile(): Promise<void> {
  try {
    profile.value = await getAdultProfile()
  } catch (cause) {
    if (isNotFound(cause)) {
      // No opt-in profile yet — the empty form is the correct next step.
      profile.value = null
    } else if (isForbidden(cause)) {
      dataDenied.value = true
    } else {
      dataError.value = errorMessage(cause, 'We could not load your adult profile. Try again.')
    }
  }
}

async function loadMatches(): Promise<void> {
  try {
    matches.value = await listAdultMatches()
  } catch (cause) {
    if (isForbidden(cause)) dataDenied.value = true
    else dataError.value = errorMessage(cause, 'We could not load your matches. Try again.')
  }
}

async function loadData(): Promise<void> {
  dataLoading.value = true
  dataError.value = ''
  dataDenied.value = false
  await Promise.all([loadProfile(), loadMatches()])
  dataLoading.value = false
}

async function saveProfile(input: AdultProfileInput): Promise<void> {
  if (saving.value) return
  saving.value = true
  saveError.value = ''
  saveNotice.value = ''
  try {
    profile.value = await saveAdultProfile(input)
    saveNotice.value =
      'Your adult profile was saved. It is part of this separate opt-in area only, and is not shown on your main profile, posts, or communities. You can change it here at any time.'
  } catch (cause) {
    saveError.value = errorMessage(cause, 'We could not save your profile. Try again.')
  } finally {
    saving.value = false
  }
}

async function removeMatch(id: number): Promise<void> {
  if (busyMatchId.value !== null) return
  busyMatchId.value = id
  matchNotice.value = ''
  try {
    await unmatch(id)
    matches.value = matches.value.filter((match) => match.id !== id)
    matchNotice.value = "Match removed. You will no longer appear in each other's matches."
  } catch (cause) {
    matchNotice.value = errorMessage(cause, 'We could not remove this match. Try again.')
  } finally {
    busyMatchId.value = null
  }
}

function openReport(user: UserSummary): void {
  reportTarget.value = user
  matchNotice.value = ''
}

function submitAdultReport(input: { reason: string; details?: string }): Promise<void> {
  const target = reportTarget.value
  if (!target) return Promise.resolve()
  return reportAdult({ targetType: 'adult_profile', targetId: target.id, ...input })
}

function onReportSubmitted(): void {
  reportTarget.value = null
  matchNotice.value = 'Your report was sent for priority review.'
}

watch(available, (value) => {
  if (value) void loadData()
})

onMounted(() => {
  void loadStatus()
})
</script>

<template>
  <div class="adult">
    <SkeletonList v-if="statusLoading" :rows="3" />

    <AdultEligibilityPanel
      v-else-if="gateUnavailable || gateDenied"
      :denied="gateDenied"
      :message="gateDenied ? 'Your account does not have access to this area.' : status?.message || ''"
    />

    <EmptyState
      v-else-if="statusError"
      title="We could not check availability"
      :description="statusError"
      action-label="Try again"
      @action="loadStatus"
    />

    <template v-else-if="available">
      <SkeletonList v-if="dataLoading" :rows="4" />

      <AdultEligibilityPanel
        v-else-if="dataDenied"
        denied
        message="Your account does not have access to this area."
      />

      <EmptyState
        v-else-if="dataError"
        title="We could not load this area"
        :description="dataError"
        action-label="Try again"
        @action="loadData"
      />

      <template v-else>
        <header class="adult__header">
          <p class="adult__eyebrow">Adults only</p>
          <h1 class="adult__title">Opt-in connections</h1>
          <p class="adult__intro">
            This is a separate part of PromisedLand. It is not part of your main profile, posts, or communities, and
            nothing here is inferred from your interests. You choose what to share here.
          </p>
        </header>

        <BaseCard class="adult__safety">
          <h2 class="adult__section-title">Safety in this area</h2>
          <ul class="adult__safety-list">
            <li>A match happens only when two people both choose it. There is no score or ranking.</li>
            <li>You can unmatch or report someone at any time. Safety reports go to a priority review queue.</li>
            <li>This area has no location, phone number, or private contact details.</li>
          </ul>
        </BaseCard>

        <BaseCard class="adult__profile">
          <h2 class="adult__section-title">Your profile in this area</h2>
          <p v-if="saveNotice" class="adult__notice adult__notice--success" role="status">{{ saveNotice }}</p>
          <AdultProfileForm
            :profile="profile"
            :saving="saving"
            :error="saveError"
            @save="saveProfile"
          />
        </BaseCard>

        <section class="adult__matches" aria-labelledby="adult-matches-title">
          <h2 id="adult-matches-title" class="adult__section-title">Your matches</h2>
          <p v-if="matchNotice" class="adult__notice" role="status">{{ matchNotice }}</p>

          <EmptyState
            v-if="!matches.length"
            title="No matches yet"
            description="Matches appear here when you and another person both choose to connect."
          />

          <ul v-else class="adult__matches-list">
            <li v-for="match in matches" :key="match.id">
              <AdultMatchCard
                :match="match"
                :current-user-id="currentUserId"
                :busy="busyMatchId === match.id"
                @unmatch="removeMatch"
                @report="openReport"
              />
            </li>
          </ul>
        </section>
      </template>
    </template>

    <ReportDialog
      v-if="reportTarget"
      target-type="adult profile"
      :target-id="reportTarget.id"
      :submitter="submitAdultReport"
      @close="reportTarget = null"
      @submitted="onReportSubmitted"
    />
  </div>
</template>

<style scoped>
.adult {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
  min-width: 0;
}

.adult__header {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.adult__eyebrow {
  color: var(--text-secondary);
  font-family: var(--font-mono);
  font-size: var(--text-xs);
  letter-spacing: 0.04em;
}

.adult__title {
  font-family: var(--font-display);
  font-size: var(--text-2xl);
}

.adult__intro {
  max-width: 64ch;
  color: var(--text-secondary);
}

.adult__section-title {
  font-family: var(--font-display);
  font-size: var(--text-xl);
}

.adult__safety {
  border-left: 2px solid var(--border-strong);
  background: var(--ink-850);
}

.adult__safety-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  margin-top: var(--space-3);
  max-width: 72ch;
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.adult__safety-list li {
  padding-left: var(--space-4);
  position: relative;
}

.adult__safety-list li::before {
  position: absolute;
  left: 0;
  color: var(--text-muted);
  content: '—';
}

.adult__profile {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
  min-width: 0;
}

.adult__notice {
  padding: var(--space-3) var(--space-4);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.adult__notice--success {
  border-color: color-mix(in srgb, var(--ok) 45%, transparent);
  color: var(--text-primary);
}

.adult__matches {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  min-width: 0;
  padding-top: var(--space-6);
  border-top: 1px solid var(--border-hairline);
}

.adult__matches-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

@media (min-width: 1024px) {
  .adult__title {
    font-size: var(--text-3xl);
  }
}
</style>
