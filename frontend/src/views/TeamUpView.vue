<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Avatar from '@/components/ui/Avatar.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseCard from '@/components/ui/BaseCard.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import SkeletonList from '@/components/ui/SkeletonList.vue'
import TagChip from '@/components/ui/TagChip.vue'
import ApplicationRow from '@/components/features/ApplicationRow.vue'
import {
  applyToTeamUp,
  getTeamUp,
  listApplications,
  reviewApplication,
  updateTeamUp,
} from '@/api/teamups'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import type { TeamUp, TeamUpApplication } from '@/types'
import { errorMessage, formatWhen, isForbidden, isNotFound } from '@/components/features/format'
import { creatorCategoryLabel, eventDateParts, teamUpStatusLabel } from '@/components/features/formatR2'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const ui = useUiStore()
const id = computed(() => Number(route.params.id))

const teamUp = ref<TeamUp | null>(null)
const applications = ref<TeamUpApplication[]>([])
const loading = ref(true)
const loadingApplications = ref(false)
const applicationsError = ref('')
const busy = ref(false)
const reviewingId = ref<number | null>(null)
const error = ref('')
const notFound = ref(false)
const forbidden = ref(false)
const message = ref('')
const applyError = ref('')

const isOwner = computed(
  () => !!teamUp.value && !!auth.user && teamUp.value.owner?.id === auth.user.id,
)
const myApplication = computed(() => teamUp.value?.my_application || '')
const canApply = computed(
  () => !isOwner.value && teamUp.value?.status === 'open' && !myApplication.value,
)
const eventParts = computed(() =>
  teamUp.value?.event ? eventDateParts(teamUp.value.event.starts_at, teamUp.value.event.ends_at) : null,
)

async function loadApplications(): Promise<void> {
  loadingApplications.value = true
  applicationsError.value = ''
  try {
    applications.value = await listApplications(id.value)
  } catch (cause) {
    applications.value = []
    applicationsError.value = errorMessage(cause, 'We could not load applications. Try again.')
  } finally {
    loadingApplications.value = false
  }
}

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  notFound.value = false
  forbidden.value = false
  message.value = ''
  applyError.value = ''
  try {
    teamUp.value = await getTeamUp(id.value)
    if (isOwner.value) await loadApplications()
  } catch (cause) {
    if (isForbidden(cause)) forbidden.value = true
    else if (isNotFound(cause)) notFound.value = true
    else error.value = errorMessage(cause, 'We could not load this team up. Try again.')
  } finally {
    loading.value = false
  }
}

async function apply(): Promise<void> {
  if (!message.value.trim() || busy.value) return
  busy.value = true
  applyError.value = ''
  try {
    await applyToTeamUp(id.value, message.value.trim())
    ui.pushToast({ title: 'Request sent', tone: 'success' })
    message.value = ''
    teamUp.value = await getTeamUp(id.value)
  } catch (cause) {
    applyError.value = errorMessage(cause, 'We could not send your request. Try again.')
  } finally {
    busy.value = false
  }
}

async function review(application: TeamUpApplication, status: 'accepted' | 'rejected'): Promise<void> {
  if (busy.value) return
  reviewingId.value = application.id
  busy.value = true
  try {
    const updated = await reviewApplication(id.value, application.id, status)
    applications.value = applications.value.map((item) => (item.id === updated.id ? updated : item))
    ui.pushToast({ title: status === 'accepted' ? 'Applicant accepted' : 'Applicant declined', tone: 'success' })
  } catch (cause) {
    ui.pushToast({ title: 'Review not saved', message: errorMessage(cause), tone: 'error' })
  } finally {
    reviewingId.value = null
    busy.value = false
  }
}

async function changeStatus(status: 'open' | 'filled' | 'closed'): Promise<void> {
  if (!teamUp.value || busy.value || teamUp.value.status === status) return
  busy.value = true
  try {
    teamUp.value = await updateTeamUp(id.value, { status } as unknown as Parameters<typeof updateTeamUp>[1])
    ui.pushToast({ title: 'Request updated', tone: 'success' })
  } catch (cause) {
    ui.pushToast({ title: 'Request not updated', message: errorMessage(cause), tone: 'error' })
  } finally {
    busy.value = false
  }
}

onMounted(load)
watch(id, load)
</script>

<template>
  <div class="team-up-view">
    <BaseButton variant="ghost" @click="router.back()">Back</BaseButton>

    <SkeletonList v-if="loading" :rows="4" />

    <EmptyState
      v-else-if="notFound"
      title="This team up is not available"
      description="It may have been removed, or the link may be wrong."
      action-label="Browse team ups"
      @action="router.push('/team-ups')"
    />

    <EmptyState
      v-else-if="forbidden"
      title="You do not have access to this request"
      description="Check the request's visibility or sign in again."
      action-label="Browse team ups"
      @action="router.push('/team-ups')"
    />

    <EmptyState
      v-else-if="error || !teamUp"
      title="We could not load this team up"
      :description="error || 'The request is unavailable right now.'"
      action-label="Try again"
      @action="load"
    />

    <template v-else>
      <BaseCard class="team-up-view__header">
        <div class="team-up-view__meta">
          <TagChip :label="teamUpStatusLabel(teamUp.status)" />
          <time v-if="teamUp.created_at" :datetime="teamUp.created_at" class="mono">
            {{ formatWhen(teamUp.created_at) }}
          </time>
        </div>
        <h1 class="team-up-view__title">{{ teamUp.title }}</h1>
        <p v-if="teamUp.description" class="team-up-view__description u-measure">{{ teamUp.description }}</p>

        <div v-if="teamUp.categories?.length" class="team-up-view__categories">
          <TagChip v-for="category in teamUp.categories" :key="category" :label="creatorCategoryLabel(category)" />
        </div>

        <div class="team-up-view__details">
          <div class="team-up-view__detail">
            <span class="team-up-view__label">Started by</span>
            <span class="team-up-view__owner">
              <Avatar
                :name="teamUp.owner?.display_name || teamUp.owner?.username || 'Member'"
                :src="teamUp.owner?.avatar_url"
                size="sm"
              />
              <RouterLink :to="`/u/${teamUp.owner?.username || ''}`">
                {{ teamUp.owner?.display_name || teamUp.owner?.username || 'Member' }}
              </RouterLink>
            </span>
          </div>
          <div class="team-up-view__detail">
            <span class="team-up-view__label">Applicants</span>
            <span class="mono">{{ teamUp.applicant_count || 0 }}</span>
          </div>
          <div v-if="teamUp.event && eventParts" class="team-up-view__detail">
            <span class="team-up-view__label">Linked event</span>
            <span>
              <RouterLink :to="`/events/${teamUp.event.id}`">{{ teamUp.event.title }}</RouterLink>
              <span class="team-up-view__when mono">{{ eventParts.full }} · {{ eventParts.time }}</span>
            </span>
          </div>
        </div>
      </BaseCard>

      <BaseCard v-if="isOwner" class="team-up-view__owner-panel">
        <h2 class="team-up-view__section-title">Manage request</h2>
        <div class="team-up-view__status-actions">
          <BaseButton v-if="teamUp.status !== 'open'" size="sm" :loading="busy" @click="changeStatus('open')">
            Reopen
          </BaseButton>
          <BaseButton v-if="teamUp.status === 'open'" size="sm" :loading="busy" @click="changeStatus('filled')">
            Mark as filled
          </BaseButton>
          <BaseButton
            v-if="teamUp.status !== 'closed'"
            size="sm"
            variant="ghost"
            :loading="busy"
            @click="changeStatus('closed')"
          >
            Close request
          </BaseButton>
        </div>

        <section class="team-up-view__applications" aria-labelledby="applications-title">
          <h3 id="applications-title" class="team-up-view__section-title">Applications</h3>
          <SkeletonList v-if="loadingApplications" :rows="2" />
          <template v-else>
            <p v-if="applicationsError" class="team-up-view__error" role="alert">
              {{ applicationsError }}
              <button type="button" @click="loadApplications">Retry</button>
            </p>
            <p v-else-if="!applications.length" class="team-up-view__note">No applications yet.</p>
            <ul v-else class="team-up-view__application-list">
              <li v-for="application in applications" :key="application.id">
                <ApplicationRow
                  :application="application"
                  :busy="busy && reviewingId === application.id"
                  @review="review(application, $event)"
                />
              </li>
            </ul>
          </template>
        </section>
      </BaseCard>

      <BaseCard v-else-if="canApply" class="team-up-view__apply">
        <h2 class="team-up-view__section-title">Request to join</h2>
        <form class="team-up-view__apply-form" @submit.prevent="apply">
          <label class="team-up-view__label" for="application-message">Your message</label>
          <textarea
            id="application-message"
            v-model="message"
            class="team-up-view__control"
            rows="4"
            maxlength="2000"
            placeholder="Tell the organizer how you would like to help."
          />
          <div class="team-up-view__apply-actions">
            <BaseButton type="submit" variant="primary" :loading="busy" :disabled="!message.trim()">
              Send request
            </BaseButton>
          </div>
          <p v-if="applyError" class="team-up-view__error" role="alert">{{ applyError }}</p>
        </form>
      </BaseCard>

      <BaseCard v-else-if="myApplication" class="team-up-view__status-note">
        <h2 class="team-up-view__section-title">Your application</h2>
        <p class="team-up-view__note">
          {{ myApplication === 'accepted' ? 'Accepted' : myApplication === 'rejected' ? 'Not accepted' : 'Pending review' }}
        </p>
      </BaseCard>

      <p v-else class="team-up-view__note">This request is no longer open.</p>
    </template>
  </div>
</template>

<style scoped>
.team-up-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.team-up-view__header,
.team-up-view__owner-panel,
.team-up-view__apply,
.team-up-view__status-note {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.team-up-view__meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-3);
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.team-up-view__title {
  font-size: var(--text-3xl);
}

.team-up-view__description {
  color: var(--text-secondary);
  white-space: pre-wrap;
}

.team-up-view__categories {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.team-up-view__details {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 200px), 1fr));
  gap: var(--space-4);
  padding-top: var(--space-4);
  border-top: 1px solid var(--border-hairline);
}

.team-up-view__detail {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.team-up-view__label {
  color: var(--text-secondary);
  font-size: var(--text-sm);
  font-weight: 500;
}

.team-up-view__owner {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
}

.team-up-view__owner a,
.team-up-view__detail a {
  color: var(--tide);
}

.team-up-view__when {
  display: block;
  color: var(--text-secondary);
  font-size: var(--text-xs);
}

.team-up-view__section-title {
  font-size: var(--text-xl);
}

.team-up-view__status-actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.team-up-view__applications {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding-top: var(--space-4);
  border-top: 1px solid var(--border-hairline);
}

.team-up-view__application-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.team-up-view__apply-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.team-up-view__control {
  width: 100%;
  min-height: 44px;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  background: var(--ink-850);
  color: var(--text-primary);
  font-family: var(--font-body);
  line-height: var(--leading-body);
  resize: vertical;
}

.team-up-view__apply-actions {
  display: flex;
  justify-content: flex-end;
}

.team-up-view__note {
  color: var(--text-secondary);
}

.team-up-view__error {
  color: var(--alert);
  font-size: var(--text-sm);
}

.team-up-view__error button {
  padding: 0;
  border: 0;
  background: none;
  color: var(--alert);
  text-decoration: underline;
  cursor: pointer;
}
</style>
