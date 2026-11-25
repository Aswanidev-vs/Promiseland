<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseCard from '@/components/ui/BaseCard.vue'
import BaseInput from '@/components/ui/BaseInput.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import ImageUploader from '@/components/ui/ImageUploader.vue'
import SkeletonList from '@/components/ui/SkeletonList.vue'
import {
  addModerator,
  addRule,
  getCommunity,
  listRules,
  updateCommunity,
} from '@/api/communities'
import { useUiStore } from '@/stores/ui'
import { errorMessage, isForbidden, isNotFound } from '@/components/features/format'
import type { Community, CommunityRule, Visibility } from '@/types'

const route = useRoute()
const ui = useUiStore()
const slug = computed(() => String(route.params.slug || ''))

const community = ref<Community | null>(null)
const rules = ref<CommunityRule[]>([])
const loading = ref(true)
const error = ref('')
const notFound = ref(false)
const staffOnly = ref(false)

const details = reactive<{ name: string; description: string; visibility: Visibility }>({
  name: '',
  description: '',
  visibility: 'public',
})
const savingDetails = ref(false)
const detailsError = ref('')
const addingRule = ref(false)
const ruleError = ref('')
const rule = reactive({ title: '', body: '' })
const moderatorId = ref('')
const addingModerator = ref(false)

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    const loaded = await getCommunity(slug.value)
    community.value = loaded
    details.name = loaded.name
    details.description = loaded.description || ''
    details.visibility = loaded.visibility === 'private' ? 'private' : 'public'
    staffOnly.value = loaded.role !== 'owner' && loaded.role !== 'moderator'
    if (!staffOnly.value) rules.value = await listRules(slug.value)
  } catch (cause) {
    if (isNotFound(cause)) notFound.value = true
    else if (isForbidden(cause)) staffOnly.value = true
    else error.value = errorMessage(cause, 'We could not load this community. Try again.')
  } finally {
    loading.value = false
  }
}

async function saveDetails(): Promise<void> {
  if (!community.value) return
  detailsError.value = ''
  if (details.name.trim().length < 3) {
    detailsError.value = 'Name must be at least 3 characters.'
    return
  }
  savingDetails.value = true
  try {
    community.value = await updateCommunity(slug.value, {
      name: details.name.trim(),
      description: details.description.trim(),
      visibility: details.visibility,
    })
    ui.pushToast({ title: 'Community details saved', tone: 'success' })
  } catch (cause) {
    detailsError.value = errorMessage(cause, 'We could not save these details. Try again.')
  } finally {
    savingDetails.value = false
  }
}

async function saveImage(field: 'icon_url' | 'banner_url', value: string | null): Promise<void> {
  if (!community.value) return
  const previous = community.value[field]
  community.value = { ...community.value, [field]: value }
  try {
    community.value = await updateCommunity(slug.value, { [field]: value })
    ui.pushToast({
      title: field === 'icon_url' ? 'Community icon updated' : 'Community banner updated',
      tone: 'success',
    })
  } catch (cause) {
    community.value = { ...community.value, [field]: previous }
    ui.pushToast({
      title: 'Could not update the image',
      message: errorMessage(cause, 'Try again in a moment.'),
      tone: 'error',
    })
  }
}

async function publishRule(): Promise<void> {
  ruleError.value = ''
  if (!rule.title.trim()) {
    ruleError.value = 'Give the rule a short title.'
    return
  }
  addingRule.value = true
  try {
    await addRule(slug.value, { title: rule.title.trim(), body: rule.body.trim() })
    rules.value = await listRules(slug.value)
    rule.title = ''
    rule.body = ''
    ui.pushToast({ title: 'Rule added', tone: 'success' })
  } catch (cause) {
    ruleError.value = errorMessage(cause, 'We could not add that rule. Try again.')
  } finally {
    addingRule.value = false
  }
}

async function promoteModerator(): Promise<void> {
  const id = Number(moderatorId.value)
  ruleError.value = ''
  if (!Number.isInteger(id) || id <= 0) {
    ruleError.value = 'Enter a member ID.'
    return
  }
  addingModerator.value = true
  try {
    await addModerator(slug.value, id)
    moderatorId.value = ''
    ui.pushToast({ title: 'Moderator added', tone: 'success' })
  } catch (cause) {
    ruleError.value = errorMessage(cause, 'We could not add that member as a moderator.')
  } finally {
    addingModerator.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="manage">
    <RouterLink v-if="slug" class="manage__back" :to="`/communities/${slug}`">
      Back to community
    </RouterLink>

    <header class="manage__head">
      <h1 class="manage__title">Manage community</h1>
      <p v-if="community" class="manage__sub">
        {{ community.name }} · you are the {{ community.role }}
      </p>
    </header>

    <SkeletonList v-if="loading" :rows="4" />

    <EmptyState
      v-else-if="notFound"
      title="That community does not exist"
      description="It may have been removed, or the link may be mistyped."
      action-label="Browse communities"
      @action="$router.push('/communities')"
    />

    <EmptyState
      v-else-if="staffOnly"
      title="Only community staff can manage this community"
      description="Owners and moderators can change details, appearance, rules and moderators."
      action-label="Back to the community"
      @action="$router.push(`/communities/${slug}`)"
    />

    <EmptyState
      v-else-if="error && !community"
      title="We could not load this community"
      :description="error"
      action-label="Try again"
      @action="load"
    />

    <template v-else-if="community">
      <BaseCard class="manage__section">
        <h2 class="manage__heading">Details</h2>
        <form class="manage__form" novalidate @submit.prevent="saveDetails">
          <BaseInput v-model="details.name" label="Name" />
          <div class="manage__field">
            <label class="manage__label" for="manage-description">Description</label>
            <textarea
              id="manage-description"
              v-model="details.description"
              class="manage__control"
              rows="4"
              maxlength="2000"
            />
          </div>
          <div class="manage__field">
            <label class="manage__label" for="manage-visibility">Visibility</label>
            <select id="manage-visibility" v-model="details.visibility" class="manage__control">
              <option value="public">Public: anyone can find and read it</option>
              <option value="private">Private: members only</option>
            </select>
          </div>
          <p v-if="detailsError" class="manage__error" role="alert">{{ detailsError }}</p>
          <div class="manage__actions">
            <BaseButton type="submit" variant="primary" :loading="savingDetails">
              Save details
            </BaseButton>
          </div>
        </form>
      </BaseCard>

      <BaseCard class="manage__section">
        <h2 class="manage__heading">Appearance</h2>
        <p class="manage__note">The icon appears in lists; the banner sits across the community page.</p>
        <ImageUploader
          :model-value="community.icon_url"
          label="Icon"
          shape="circle"
          :size="72"
          @update:model-value="saveImage('icon_url', $event)"
        />
        <ImageUploader
          :model-value="community.banner_url"
          label="Banner"
          shape="wide"
          hint="Shown across the top of the community page. JPEG, PNG, GIF or WebP up to 5 MB."
          @update:model-value="saveImage('banner_url', $event)"
        />
      </BaseCard>

      <BaseCard class="manage__section">
        <h2 class="manage__heading">Rules</h2>
        <ol v-if="rules.length" class="manage__rules">
          <li v-for="(item, index) in rules" :key="item.id">
            <h3 class="manage__rule-title">{{ index + 1 }}. {{ item.title }}</h3>
            <p v-if="item.body" class="manage__rule-body">{{ item.body }}</p>
          </li>
        </ol>
        <p v-else class="manage__note">No rules published yet.</p>

        <form class="manage__form manage__form--ruled" novalidate @submit.prevent="publishRule">
          <BaseInput v-model="rule.title" label="New rule title" hint="At most 120 characters." />
          <div class="manage__field">
            <label class="manage__label" for="manage-rule-body">Explanation</label>
            <textarea id="manage-rule-body" v-model="rule.body" class="manage__control" rows="3" />
          </div>
          <p v-if="ruleError" class="manage__error" role="alert">{{ ruleError }}</p>
          <div class="manage__actions">
            <BaseButton type="submit" variant="secondary" :loading="addingRule">Add rule</BaseButton>
          </div>
        </form>
      </BaseCard>

      <BaseCard class="manage__section">
        <h2 class="manage__heading">Moderators</h2>
        <p class="manage__note">
          Moderators can remove content, review reports and edit the rules. Add one by their member ID,
          which appears in the members list on the community page.
        </p>
        <form class="manage__form" novalidate @submit.prevent="promoteModerator">
          <BaseInput
            v-model="moderatorId"
            label="Member ID"
            type="number"
            hint="Numbers only, for example 42."
          />
          <div class="manage__actions">
            <BaseButton type="submit" variant="secondary" :loading="addingModerator">
              Add moderator
            </BaseButton>
          </div>
        </form>
      </BaseCard>
    </template>
  </div>
</template>

<style scoped>
.manage {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.manage__back {
  display: inline-flex;
  align-items: center;
  align-self: flex-start;
  min-height: 44px;
  color: var(--text-secondary);
  font-size: var(--text-sm);
  text-decoration: none;
}

.manage__back:hover {
  color: var(--text-primary);
}

.manage__head {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.manage__title {
  font-family: var(--font-display);
  font-size: var(--text-3xl);
  font-weight: 560;
  letter-spacing: -0.02em;
  line-height: 1.1;
}

.manage__sub {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.manage__section {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.manage__heading {
  font-size: var(--text-xl);
  font-weight: 600;
}

.manage__note {
  max-width: 62ch;
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.manage__form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.manage__form--ruled {
  padding-top: var(--space-5);
  border-top: 1px solid var(--border-hairline);
}

.manage__field {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.manage__label {
  color: var(--text-primary);
  font-size: var(--text-sm);
  font-weight: 500;
}

.manage__control {
  width: 100%;
  min-height: 44px;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  background: var(--ink-850);
  color: var(--text-primary);
}

.manage__control:focus {
  border-color: var(--ember);
  box-shadow: 0 0 0 3px var(--ember-soft);
  outline: none;
}

.manage__rules {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.manage__rule-title {
  font-size: var(--text-base);
  font-weight: 600;
}

.manage__rule-body {
  margin-top: var(--space-1);
  max-width: 62ch;
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.manage__error {
  color: var(--alert);
  font-size: var(--text-sm);
}

.manage__actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-3);
}
</style>
