<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import AppIcon from '@/components/ui/AppIcon.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseCard from '@/components/ui/BaseCard.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import SkeletonList from '@/components/ui/SkeletonList.vue'
import FeedTabs from '@/components/features/FeedTabs.vue'
import DiscoveryCard from '@/components/features/DiscoveryCard.vue'
import ConnectDialog from '@/components/features/ConnectDialog.vue'
import DiscoverabilityToggle from '@/components/features/DiscoverabilityToggle.vue'
import {
  getDiscoverySettings,
  listFriends,
  sendConnection,
  suppressCandidate,
  unsuppressCandidate,
  updateDiscoverySettings,
} from '@/api/discovery'
import { useUiStore } from '@/stores/ui'
import type { DiscoveryCandidate } from '@/types'
import { errorMessage, isForbidden } from '@/components/features/format'

const router = useRouter()
const ui = useUiStore()

const candidates = ref<DiscoveryCandidate[]>([])
const cursor = ref('')
const loading = ref(true)
const loadingMore = ref(false)
const error = ref('')
const forbidden = ref(false)

const tab = ref('for-you')
const tabs = [
  { value: 'for-you', label: 'For you' },
  { value: 'requests', label: 'Requests' },
]

const discoverable = ref(true)
const settingsLoading = ref(true)
const settingsSaving = ref(false)
const settingsError = ref('')

const connectTarget = ref<DiscoveryCandidate | null>(null)
const connectBusy = ref(false)
const connectError = ref('')

const suppressingId = ref<number | null>(null)
const undo = ref<{ candidate: DiscoveryCandidate; index: number } | null>(null)
let undoTimer: number | undefined

async function load(reset = false): Promise<void> {
  if (reset) {
    candidates.value = []
    cursor.value = ''
    error.value = ''
    forbidden.value = false
  }
  const more = !reset && cursor.value !== ''
  if (more) loadingMore.value = true
  else loading.value = true
  try {
    const page = await listFriends({
      limit: 20,
      ...(more ? { cursor: cursor.value } : {}),
    })
    candidates.value = more ? [...candidates.value, ...page.items] : [...page.items]
    cursor.value = page.next_cursor || ''
  } catch (cause) {
    if (isForbidden(cause)) forbidden.value = true
    else error.value = errorMessage(cause, 'We could not load suggestions. Try again.')
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

async function loadSettings(): Promise<void> {
  settingsLoading.value = true
  settingsError.value = ''
  try {
    const settings = await getDiscoverySettings()
    discoverable.value = settings.discoverable
  } catch (cause) {
    settingsError.value = errorMessage(cause, 'We could not load your discovery setting.')
  } finally {
    settingsLoading.value = false
  }
}

async function setDiscoverable(value: boolean): Promise<void> {
  if (settingsSaving.value || value === discoverable.value) return
  const previous = discoverable.value
  discoverable.value = value
  settingsSaving.value = true
  settingsError.value = ''
  try {
    const settings = await updateDiscoverySettings({ discoverable: value })
    discoverable.value = settings.discoverable
    ui.pushToast({
      title: settings.discoverable ? 'Discoverable' : 'Discovery paused',
      message: settings.discoverable
        ? 'People with shared interests can find you again.'
        : 'You will no longer appear in suggestions.',
      tone: 'success',
    })
  } catch (cause) {
    discoverable.value = previous
    settingsError.value = errorMessage(cause, 'We could not update your discovery setting. Try again.')
    ui.pushToast({ title: 'Setting not saved', tone: 'error' })
  } finally {
    settingsSaving.value = false
  }
}

function openConnect(candidate: DiscoveryCandidate): void {
  connectTarget.value = candidate
  connectError.value = ''
}

async function submitConnect(message: string): Promise<void> {
  const target = connectTarget.value
  if (!target || connectBusy.value) return
  connectBusy.value = true
  connectError.value = ''
  try {
    await sendConnection({
      userId: target.user.id,
      ...(message ? { message } : {}),
    })
    const index = candidates.value.findIndex((item) => item.user.id === target.user.id)
    if (index !== -1) {
      candidates.value[index] = { ...candidates.value[index]!, connection_status: 'pending_out' }
    }
    connectTarget.value = null
    ui.pushToast({ title: 'Request sent', message: `Your request is on its way to ${target.user.display_name || target.user.username}.`, tone: 'success' })
  } catch (cause) {
    connectError.value = errorMessage(cause, 'We could not send your request. Try again.')
  } finally {
    connectBusy.value = false
  }
}

function clearUndo(): void {
  if (undoTimer !== undefined) window.clearTimeout(undoTimer)
  undoTimer = undefined
  undo.value = null
}

async function notInterested(candidate: DiscoveryCandidate): Promise<void> {
  if (suppressingId.value !== null) return
  const index = candidates.value.findIndex((item) => item.user.id === candidate.user.id)
  if (index === -1) return
  suppressingId.value = candidate.user.id
  try {
    await suppressCandidate(candidate.user.id)
    candidates.value.splice(index, 1)
    clearUndo()
    undo.value = { candidate, index }
    undoTimer = window.setTimeout(clearUndo, 6000)
  } catch (cause) {
    ui.pushToast({
      title: 'Could not hide suggestion',
      message: errorMessage(cause, 'Try again in a moment.'),
      tone: 'error',
    })
  } finally {
    suppressingId.value = null
  }
}

async function undoSuppress(): Promise<void> {
  const hidden = undo.value
  if (!hidden) return
  try {
    await unsuppressCandidate(hidden.candidate.user.id)
    candidates.value.splice(Math.min(hidden.index, candidates.value.length), 0, hidden.candidate)
    clearUndo()
    ui.pushToast({ title: 'Suggestion restored', tone: 'success' })
  } catch (cause) {
    ui.pushToast({
      title: 'Could not restore suggestion',
      message: errorMessage(cause, 'Try again in a moment.'),
      tone: 'error',
    })
  }
}

onMounted(() => {
  void load(true)
  void loadSettings()
})
onBeforeUnmount(clearUndo)
</script>

<template>
  <div class="discover">
    <header class="discover__header">
      <div class="discover__heading">
        <p class="discover__eyebrow">Discover</p>
        <h1 class="discover__title">Your fandom. Your people. Your place.</h1>
        <p class="discover__intro">People who care about the same things you do.</p>
      </div>
      <DiscoverabilityToggle
        :model-value="discoverable"
        :loading="settingsLoading || settingsSaving"
        @update:model-value="setDiscoverable"
      />
    </header>

    <div v-if="settingsError" class="discover__settings-error" role="alert">
      <span>{{ settingsError }}</span>
      <BaseButton size="sm" variant="ghost" @click="loadSettings">Retry</BaseButton>
    </div>

    <FeedTabs :model-value="tab" :tabs="tabs" label="Discovery sections" @update:model-value="tab = $event" />

    <template v-if="tab === 'requests'">
      <BaseCard class="discover__requests">
        <h2 class="discover__requests-title">Your requests</h2>
        <p class="discover__requests-copy">
          See who wants to connect, check requests you have sent, and review people you are already connected with.
        </p>
        <RouterLink class="discover__link" to="/connections">View your requests</RouterLink>
      </BaseCard>
    </template>

    <template v-else>
      <SkeletonList v-if="loading" :rows="3" avatar />

      <EmptyState
        v-else-if="forbidden"
        title="Discovery is not available"
        description="Discovery needs an active account. Sign in again to continue."
        action-label="Go to home"
        @action="router.push('/home')"
      />

      <EmptyState
        v-else-if="error"
        title="We could not load suggestions"
        :description="error"
        action-label="Try again"
        @action="load(true)"
      />

      <EmptyState
        v-else-if="!candidates.length"
        title="No suggestions right now"
        description="Add a few interests or favourite anime and we will find people who share them."
        action-label="Update your interests"
        @action="router.push('/settings')"
      />

      <template v-else>
        <ul class="discover__list">
          <li v-for="candidate in candidates" :key="candidate.user.id">
            <DiscoveryCard
              :candidate="candidate"
              :busy="suppressingId === candidate.user.id"
              @connect="openConnect(candidate)"
              @not-interested="notInterested(candidate)"
            />
          </li>
        </ul>
        <div v-if="cursor" class="discover__more">
          <BaseButton :loading="loadingMore" @click="load()">Load more</BaseButton>
        </div>
      </template>
    </template>

    <ConnectDialog
      v-if="connectTarget"
      :user="connectTarget.user"
      :busy="connectBusy"
      :error="connectError"
      @close="connectTarget = null"
      @submit="submitConnect"
    />

    <Transition name="undo">
      <div v-if="undo" class="undo-toast" role="status" aria-live="polite">
        <span class="undo-toast__copy">Suggestion hidden.</span>
        <button type="button" class="undo-toast__action" @click="undoSuppress">Undo</button>
        <button
          type="button"
          class="undo-toast__close"
          aria-label="Dismiss"
          @click="clearUndo"
        >
          <AppIcon name="close" :size="16" />
        </button>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.discover {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.discover__header {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-4) var(--space-6);
}

.discover__heading {
  flex: 1;
  min-width: 0;
}

.discover__eyebrow {
  color: var(--text-secondary);
  font-family: var(--font-mono);
  font-size: var(--text-xs);
  letter-spacing: 0.04em;
}

.discover__title {
  margin-top: var(--space-2);
  max-width: 18ch;
  font-family: var(--font-display);
  font-size: var(--text-2xl);
  line-height: 1.2;
}

.discover__intro {
  margin-top: var(--space-2);
  color: var(--text-secondary);
}

.discover__settings-error {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-4);
  border: 1px solid color-mix(in srgb, var(--alert) 40%, transparent);
  border-radius: var(--radius-md);
  color: var(--alert);
  font-size: var(--text-sm);
}

.discover__list {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.discover__more {
  display: flex;
  justify-content: center;
}

.discover__requests {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  align-items: flex-start;
}

.discover__requests-title {
  font-size: var(--text-xl);
}

.discover__requests-copy {
  max-width: 56ch;
  color: var(--text-secondary);
}

.discover__link {
  font-weight: 500;
}

.undo-toast {
  position: fixed;
  right: var(--space-4);
  bottom: calc(var(--bottomnav-h) + var(--space-4));
  z-index: 55;
  display: flex;
  align-items: center;
  gap: var(--space-3);
  width: min(360px, calc(100vw - var(--space-8)));
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  background: var(--ink-800);
  box-shadow: var(--shadow-2);
}

.undo-toast__copy {
  flex: 1;
  font-size: var(--text-sm);
}

.undo-toast__action {
  min-height: 44px;
  padding: 0 var(--space-2);
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--tide);
  cursor: pointer;
  font-size: var(--text-sm);
  font-weight: 500;
}

.undo-toast__action:hover {
  color: var(--text-primary);
}

.undo-toast__close {
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
}

.undo-toast__close:hover {
  background: var(--ink-700);
  color: var(--text-primary);
}

.undo-enter-active,
.undo-leave-active {
  transition:
    opacity var(--dur) var(--ease),
    transform var(--dur) var(--ease);
}

.undo-enter-from,
.undo-leave-to {
  opacity: 0;
  transform: translateY(8px);
}

@media (min-width: 1024px) {
  .discover__title {
    font-size: var(--text-3xl);
  }
}

@media (max-width: 480px) {
  .discover__header {
    flex-direction: column;
  }

  .discover__title {
    max-width: none;
  }
}
</style>
