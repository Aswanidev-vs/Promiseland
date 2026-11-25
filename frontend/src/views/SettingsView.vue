<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseCard from '@/components/ui/BaseCard.vue'
import Avatar from '@/components/ui/Avatar.vue'
import BaseInput from '@/components/ui/BaseInput.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import SkeletonList from '@/components/ui/SkeletonList.vue'
import { getMe } from '@/api/auth'
import { uploadMedia } from '@/api/media'
import { updateMe, updateSettings } from '@/api/users'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { errorMessage, isForbidden } from '@/components/features/format'

const router = useRouter()
const auth = useAuthStore()
const ui = useUiStore()

const form = reactive({
  displayName: '',
  bio: '',
  location: '',
  avatarUrl: '',
  visibility: 'public',
  discoverable: true,
})
const loading = ref(true)
const uploadingAvatar = ref(false)
const uploadError = ref('')
const savingProfile = ref(false)
const savingPrivacy = ref(false)
const error = ref('')
const forbidden = ref(false)
const notice = ref('')

const notifications = [
  { id: 'replies', label: 'Replies and mentions', note: 'Planned' },
  { id: 'follows', label: 'New followers', note: 'Planned' },
  { id: 'community', label: 'Community activity', note: 'Planned' },
]

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  forbidden.value = false
  try {
    const me = await getMe()
    form.displayName = me.display_name || ''
    form.bio = me.bio || ''
    form.location = me.location || ''
    form.avatarUrl = me.avatar_url || ''
    form.visibility = me.visibility === 'private' ? 'private' : 'public'
    form.discoverable = me.discoverable !== false
  } catch (cause) {
    if (isForbidden(cause)) forbidden.value = true
    else error.value = errorMessage(cause, 'We could not load your settings. Try again.')
  } finally {
    loading.value = false
  }
}

const avatarTypes = ['image/jpeg', 'image/png', 'image/gif', 'image/webp']
const maxAvatarBytes = 5 * 1024 * 1024

async function uploadAvatar(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  // Reset the input so choosing the same file again still fires a change.
  input.value = ''
  if (!file) return

  uploadError.value = ''
  if (!avatarTypes.includes(file.type)) {
    uploadError.value = 'Choose a JPEG, PNG, GIF or WebP image.'
    return
  }
  if (file.size > maxAvatarBytes) {
    uploadError.value = 'Images must be 5 MB or smaller.'
    return
  }

  uploadingAvatar.value = true
  try {
    const media = await uploadMedia(file)
    form.avatarUrl = media.url
    notice.value = 'Photo uploaded. Save the profile to keep it.'
  } catch (cause) {
    uploadError.value = errorMessage(cause, 'We could not upload that image. Try again.')
  } finally {
    uploadingAvatar.value = false
  }
}

async function saveProfile(): Promise<void> {
  if (!form.displayName.trim()) {
    error.value = 'Enter the name people will see.'
    return
  }
  savingProfile.value = true
  error.value = ''
  notice.value = ''
  try {
    // The User DTO uses snake_case keys on the wire.
    const patch = {
      display_name: form.displayName.trim(),
      bio: form.bio.trim(),
      location: form.location.trim(),
      avatar_url: form.avatarUrl.trim() || null,
    } as unknown as Parameters<typeof updateMe>[0]
    await updateMe(patch)
    await auth.fetchMe()
    notice.value = 'Profile saved.'
    ui.pushToast({ title: 'Profile saved', tone: 'success' })
  } catch (cause) {
    error.value = errorMessage(cause, 'We could not save your profile. Try again.')
  } finally {
    savingProfile.value = false
  }
}

async function savePrivacy(): Promise<void> {
  savingPrivacy.value = true
  error.value = ''
  notice.value = ''
  try {
    await updateSettings({ visibility: form.visibility, discoverable: form.discoverable })
    await auth.fetchMe()
    notice.value = 'Privacy settings saved.'
    ui.pushToast({ title: 'Privacy saved', tone: 'success' })
  } catch (cause) {
    error.value = errorMessage(cause, 'We could not save your privacy settings. Try again.')
  } finally {
    savingPrivacy.value = false
  }
}

async function signOut(): Promise<void> {
  try {
    await auth.logout()
    await router.push('/login')
  } catch (cause) {
    error.value = errorMessage(cause, 'We could not sign you out. Try again.')
  }
}

onMounted(load)
</script>

<template>
  <div class="settings">
    <h1 class="settings__title">Settings</h1>

    <SkeletonList v-if="loading" :rows="4" />

    <EmptyState
      v-else-if="forbidden"
      title="You do not have access to these settings"
      description="Sign in with your own account to edit your profile."
      action-label="Go to sign in"
      @action="router.push('/login')"
    />

    <EmptyState
      v-else-if="error && !form.displayName"
      title="We could not load your settings"
      :description="error"
      action-label="Try again"
      @action="load"
    />

    <template v-else>
      <p v-if="notice" class="settings__notice" role="status">{{ notice }}</p>
      <p v-if="error" class="settings__error" role="alert">{{ error }}</p>

      <BaseCard class="settings__section">
        <h2 class="settings__heading">Profile</h2>
        <form class="settings__form" novalidate @submit.prevent="saveProfile">
          <BaseInput v-model="form.displayName" label="Display name" />

          <div class="settings__avatar">
            <Avatar
              :name="form.displayName || 'Your profile'"
              :src="form.avatarUrl || null"
              size="lg"
            />
            <div class="settings__avatar-body">
              <input
                class="settings__upload"
                type="file"
                accept="image/png,image/jpeg,image/gif,image/webp"
                aria-label="Upload a profile photo"
                :disabled="uploadingAvatar"
                @change="uploadAvatar"
              />
              <button
                v-if="form.avatarUrl"
                type="button"
                class="settings__remove"
                @click="form.avatarUrl = ''"
              >
                Remove photo
              </button>
              <p class="settings__note">
                JPEG, PNG, GIF or WebP up to 5 MB. {{ uploadingAvatar ? 'Uploading…' : 'Save the profile after uploading.' }}
              </p>
              <p v-if="uploadError" class="settings__error" role="alert">{{ uploadError }}</p>
            </div>
          </div>

          <BaseInput
            v-model="form.avatarUrl"
            label="Or paste an image link"
            type="url"
            placeholder="https://"
          />
          <BaseInput v-model="form.location" label="Location" hint="City or region is optional." />
          <div class="settings__field">
            <label class="settings__label" for="settings-bio">Bio</label>
            <textarea id="settings-bio" v-model="form.bio" class="settings__control" rows="4" maxlength="500" />
          </div>
          <div class="settings__actions">
            <BaseButton type="submit" variant="primary" :loading="savingProfile">Save profile</BaseButton>
          </div>
        </form>
      </BaseCard>

      <BaseCard class="settings__section">
        <h2 class="settings__heading">Privacy and discovery</h2>
        <form class="settings__form" novalidate @submit.prevent="savePrivacy">
          <div class="settings__field">
            <label class="settings__label" for="settings-visibility">Profile visibility</label>
            <select id="settings-visibility" v-model="form.visibility" class="settings__control">
              <option value="public">Public</option>
              <option value="private">Private</option>
            </select>
          </div>
          <label class="settings__check">
            <input v-model="form.discoverable" type="checkbox" />
            Show my profile in discovery
          </label>
          <div class="settings__actions">
            <BaseButton type="submit" variant="secondary" :loading="savingPrivacy">Save privacy settings</BaseButton>
          </div>
        </form>
      </BaseCard>

      <BaseCard class="settings__section">
        <h2 class="settings__heading">Notifications</h2>
        <ul class="settings__list">
          <li v-for="item in notifications" :key="item.id" class="settings__list-item">
            <input :id="`notify-${item.id}`" type="checkbox" disabled />
            <label :for="`notify-${item.id}`">{{ item.label }}</label>
            <span class="settings__planned">Planned</span>
          </li>
        </ul>
        <p class="settings__note">Notification controls will open in a later release.</p>
      </BaseCard>

      <BaseCard class="settings__section">
        <h2 class="settings__heading">Account</h2>
        <div class="settings__actions">
          <BaseButton variant="secondary" @click="signOut">Sign out</BaseButton>
        </div>
        <div class="settings__deactivate">
          <div>
            <h3 class="settings__deactivate-title">Deactivate account</h3>
            <p class="settings__note">Planned. Account deactivation is not available in this release.</p>
          </div>
          <BaseButton variant="danger" disabled>Deactivate account</BaseButton>
        </div>
      </BaseCard>
    </template>
  </div>
</template>

<style scoped>
.settings {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.settings__title {
  font-weight: 560;
  letter-spacing: -0.02em;
}

.settings__section {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.settings__heading {
  font-size: var(--text-xl);
}

.settings__form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.settings__avatar {
  display: flex;
  align-items: flex-start;
  gap: var(--space-4);
}

.settings__avatar-body {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  min-width: 0;
}

.settings__upload {
  max-width: 100%;
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.settings__upload::file-selector-button {
  min-height: 44px;
  margin-right: var(--space-3);
  padding: 0 var(--space-4);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--text-primary);
  font: inherit;
  font-weight: 500;
  cursor: pointer;
  transition:
    background-color var(--dur-fast) var(--ease),
    border-color var(--dur-fast) var(--ease);
}

.settings__upload::file-selector-button:hover {
  border-color: var(--border-hairline);
  background: var(--ink-700);
}

.settings__upload:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.settings__remove {
  display: inline-flex;
  min-height: 44px;
  align-items: center;
  align-self: flex-start;
  padding: 0;
  border: 0;
  background: none;
  color: var(--tide);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
}

.settings__remove:hover {
  color: var(--text-primary);
}

.settings__field {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.settings__label {
  color: var(--text-primary);
  font-size: var(--text-sm);
  font-weight: 500;
}

.settings__control {
  width: 100%;
  min-height: 44px;
  padding: var(--space-3) calc(var(--space-3) + 2px);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-md);
  background: var(--ink-850);
  color: var(--text-primary);
  font-family: var(--font-body);
  resize: vertical;
  transition: border-color var(--dur-fast) var(--ease);
}

.settings__control:hover:not(:disabled) {
  border-color: var(--border-strong);
}

.settings__control:focus {
  border-color: var(--ember);
  outline: none;
  box-shadow: 0 0 0 3px var(--ember-soft);
}

.settings__check {
  display: inline-flex;
  min-height: 44px;
  align-items: center;
  gap: var(--space-2);
  cursor: pointer;
}

.settings__check input,
.settings__list-item input {
  width: 18px;
  height: 18px;
  accent-color: var(--ember);
}

.settings__check input:disabled,
.settings__list-item input:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.settings__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}

.settings__list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.settings__list-item {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.settings__list-item label {
  flex: 1;
  color: var(--text-secondary);
}

.settings__planned {
  padding: 2px var(--space-2);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  font-family: var(--font-mono);
  font-size: var(--text-xs);
}

.settings__note {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.settings__notice {
  color: var(--ok);
  font-size: var(--text-sm);
}

.settings__error {
  color: var(--alert);
  font-size: var(--text-sm);
}

.settings__deactivate {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-4);
  margin-top: var(--space-4);
  padding-top: var(--space-4);
  border-top: 1px solid var(--border-hairline);
}

.settings__deactivate-title {
  font-size: var(--text-lg);
}
</style>
