<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseCard from '@/components/ui/BaseCard.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import SkeletonList from '@/components/ui/SkeletonList.vue'
import TagChip from '@/components/ui/TagChip.vue'
import { listReports, resolveReport } from '@/api/moderation'
import { useUiStore } from '@/stores/ui'
import type { Report } from '@/types'
import { errorMessage, formatWhen, isForbidden, reportReasonLabel } from '@/components/features/format'

const router = useRouter()
const ui = useUiStore()
const reports = ref<Report[]>([])
const loading = ref(true)
const busyId = ref<number | null>(null)
const error = ref('')
const forbidden = ref(false)
const actions: { value: string; label: string }[] = [
  { value: 'remove_content', label: 'Remove content' },
  { value: 'warn_user', label: 'Warn user' },
  { value: 'dismiss', label: 'Dismiss report' },
]
const drafts = new Map<number, { action: string; reason: string }>()

function entryFor(id: number): { action: string; reason: string } {
  let entry = drafts.get(id)
  if (!entry) {
    entry = reactive({ action: 'dismiss', reason: '' })
    drafts.set(id, entry)
  }
  return entry
}

const rows = computed(() => reports.value.map((report) => ({ report, entry: entryFor(report.id) })))

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  forbidden.value = false
  try {
    const page = await listReports({ limit: 50 })
    reports.value = page.items
  } catch (cause) {
    if (isForbidden(cause)) forbidden.value = true
    else error.value = errorMessage(cause, 'We could not load the report queue. Try again.')
  } finally {
    loading.value = false
  }
}

async function resolve(report: Report): Promise<void> {
  const entry = entryFor(report.id)
  if (!entry.reason.trim()) {
    error.value = 'Add a reason for this moderation action.'
    return
  }
  busyId.value = report.id
  error.value = ''
  try {
    await resolveReport(report.id, { action: entry.action, reason: entry.reason.trim() })
    reports.value = reports.value.filter((item) => item.id !== report.id)
    ui.pushToast({ title: 'Report resolved', tone: 'success' })
  } catch (cause) {
    error.value = errorMessage(cause, 'We could not resolve this report. Try again.')
  } finally {
    busyId.value = null
  }
}

onMounted(load)
</script>

<template>
  <div class="moderation">
    <header class="moderation__head">
      <div>
        <h1>Moderation</h1>
        <p class="moderation__intro">Review reports from across the platform.</p>
      </div>
    </header>

    <SkeletonList v-if="loading" :rows="4" />

    <EmptyState
      v-else-if="forbidden"
      title="Moderator access required"
      description="This queue is limited to platform moderators. Regular accounts cannot view or resolve reports."
      action-label="Back to home"
      @action="router.push('/home')"
    />

    <EmptyState
      v-else-if="error && !reports.length"
      title="We could not load the report queue"
      :description="error"
      action-label="Try again"
      @action="load"
    />

    <EmptyState
      v-else-if="!reports.length"
      title="No reports to review"
      description="New reports will appear here with their reason and details."
      action-label="Back to home"
      @action="router.push('/home')"
    />

    <template v-else>
      <p v-if="error" class="moderation__error" role="alert">{{ error }}</p>
      <ul class="moderation__queue">
        <li v-for="row in rows" :key="row.report.id">
          <BaseCard class="moderation__card">
            <div class="moderation__card-head">
              <TagChip :label="reportReasonLabel(row.report.reason)" tone="ember" />
              <TagChip :label="row.report.target_type" />
              <span class="moderation__meta mono">#{{ row.report.id }}</span>
              <time v-if="row.report.created_at" class="moderation__meta" :datetime="row.report.created_at">
                {{ formatWhen(row.report.created_at) }}
              </time>
            </div>
            <h2 class="moderation__target">{{ row.report.target_type }} {{ row.report.target_id }}</h2>
            <p v-if="row.report.details" class="moderation__details">{{ row.report.details }}</p>
            <p v-else class="moderation__details">No additional details were provided.</p>
            <p class="moderation__meta">Reported by account {{ row.report.reporter_id }}</p>

            <form class="moderation__form" @submit.prevent="resolve(row.report)">
              <label class="moderation__label" :for="`action-${row.report.id}`">Action</label>
              <select :id="`action-${row.report.id}`" v-model="row.entry.action" class="moderation__control">
                <option v-for="action in actions" :key="action.value" :value="action.value">{{ action.label }}</option>
              </select>
              <label class="moderation__label" :for="`reason-${row.report.id}`">Reason for action</label>
              <input
                :id="`reason-${row.report.id}`"
                v-model="row.entry.reason"
                class="moderation__control"
                placeholder="Record why you took this action."
              />
              <div class="moderation__actions">
                <BaseButton type="submit" variant="secondary" :loading="busyId === row.report.id">Resolve report</BaseButton>
              </div>
            </form>
          </BaseCard>
        </li>
      </ul>
    </template>
  </div>
</template>

<style scoped>
.moderation {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.moderation__head h1 {
  font-size: var(--text-2xl);
}

.moderation__intro {
  margin-top: var(--space-1);
  color: var(--text-secondary);
}

.moderation__error {
  color: var(--alert);
  font-size: var(--text-sm);
}

.moderation__queue {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.moderation__card {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.moderation__card-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-2);
}

.moderation__meta {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.moderation__target {
  font-size: var(--text-xl);
}

.moderation__details {
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}

.moderation__form {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding-top: var(--space-3);
  border-top: 1px solid var(--border-hairline);
}

.moderation__label {
  color: var(--text-primary);
  font-size: var(--text-sm);
  font-weight: 500;
}

.moderation__control {
  width: 100%;
  min-height: 44px;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  background: var(--ink-850);
  color: var(--text-primary);
}

.moderation__actions {
  display: flex;
  justify-content: flex-end;
  margin-top: var(--space-2);
}
</style>
