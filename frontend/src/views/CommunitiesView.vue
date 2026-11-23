<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseCard from '@/components/ui/BaseCard.vue'
import BaseInput from '@/components/ui/BaseInput.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import SkeletonList from '@/components/ui/SkeletonList.vue'
import CommunityCard from '@/components/features/CommunityCard.vue'
import { createCommunity, listCommunities } from '@/api/communities'
import { useUiStore } from '@/stores/ui'
import type { Community } from '@/types'
import { errorMessage, isForbidden } from '@/components/features/format'

const ui = useUiStore()
const router = useRouter()
const query = ref('')
const communities = ref<Community[]>([])
const cursor = ref('')
const loading = ref(true)
const loadingMore = ref(false)
const error = ref('')
const forbidden = ref(false)
const showCreate = ref(false)
const creating = ref(false)
const createError = ref('')
const slugTouched = ref(false)
const form = ref({ name: '', slug: '', description: '', visibility: 'public', rules: '' })
const fieldErrors = ref({ name: '', slug: '', description: '' })
let timer: number | undefined

function slugify(value: string): string {
  return value
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9\s-]/g, '')
    .replace(/[\s-]+/g, '-')
    .replace(/^-|-$/g, '')
}

watch(
  () => form.value.name,
  (name) => {
    if (!slugTouched.value) form.value.slug = slugify(name)
  },
)

async function load(reset = false): Promise<void> {
  if (reset) {
    communities.value = []
    cursor.value = ''
    error.value = ''
    forbidden.value = false
  }
  const more = !reset && cursor.value !== ''
  if (more) loadingMore.value = true
  else loading.value = true
  try {
    const page = await listCommunities({
      ...(query.value.trim() ? { q: query.value.trim() } : {}),
      limit: 20,
      ...(more ? { cursor: cursor.value } : {}),
    })
    communities.value = more ? [...communities.value, ...page.items] : page.items
    cursor.value = page.next_cursor || ''
  } catch (cause) {
    if (isForbidden(cause)) forbidden.value = true
    else error.value = errorMessage(cause, 'We could not load communities. Try again.')
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

function scheduleSearch(): void {
  if (timer !== undefined) window.clearTimeout(timer)
  timer = window.setTimeout(() => void load(true), 300)
}

async function submitCreate(): Promise<void> {
  fieldErrors.value = {
    name: form.value.name.trim() ? '' : 'Give your community a name.',
    slug: /^[a-z0-9-]{3,40}$/.test(form.value.slug) ? '' : 'Use 3 to 40 lowercase letters, numbers, or dashes.',
    description: form.value.description.trim() ? '' : 'Add a short description.',
  }
  if (Object.values(fieldErrors.value).some(Boolean)) return

  creating.value = true
  createError.value = ''
  try {
    const rules = form.value.rules
      .split('\n')
      .map((line) => line.trim())
      .filter(Boolean)
    const created = await createCommunity({
      name: form.value.name.trim(),
      slug: form.value.slug,
      description: form.value.description.trim(),
      visibility: form.value.visibility,
      ...(rules.length ? { rules } : {}),
    })
    ui.pushToast({ title: 'Community created', tone: 'success' })
    showCreate.value = false
    form.value = { name: '', slug: '', description: '', visibility: 'public', rules: '' }
    slugTouched.value = false
    query.value = ''
    await load(true)
    if (created?.slug) await router.push(`/communities/${created.slug}`)
  } catch (cause) {
    createError.value = errorMessage(cause, 'We could not create this community. Try again.')
  } finally {
    creating.value = false
  }
}

onMounted(() => void load(true))
onBeforeUnmount(() => {
  if (timer !== undefined) window.clearTimeout(timer)
})
</script>

<template>
  <div class="communities">
    <header class="communities__header">
      <div>
        <h1>Communities</h1>
        <p class="communities__intro">Find a corner of fandom that feels like yours.</p>
      </div>
      <BaseButton :variant="showCreate ? 'secondary' : 'primary'" @click="showCreate = !showCreate">
        {{ showCreate ? 'Close form' : 'Create community' }}
      </BaseButton>
    </header>

    <BaseCard v-if="showCreate" class="communities__create">
      <h2 class="communities__create-title">New community</h2>
      <form class="communities__form" novalidate @submit.prevent="submitCreate">
        <BaseInput v-model="form.name" label="Name" :error="fieldErrors.name" @update:model-value="slugTouched = false" />
        <BaseInput v-model="form.slug" label="Slug" hint="Used in the community link." :error="fieldErrors.slug" @update:model-value="slugTouched = true" />
        <BaseInput v-model="form.description" label="Description" :error="fieldErrors.description" />
        <label class="communities__label" for="community-visibility">Visibility</label>
        <select id="community-visibility" v-model="form.visibility" class="communities__control">
          <option value="public">Public</option>
          <option value="private">Private</option>
        </select>
        <label class="communities__label" for="community-rules">Rules</label>
        <textarea
          id="community-rules"
          v-model="form.rules"
          class="communities__control communities__textarea"
          rows="3"
          placeholder="One rule per line."
        />
        <p v-if="createError" class="communities__error" role="alert">{{ createError }}</p>
        <div class="communities__form-actions">
          <BaseButton variant="ghost" type="button" @click="showCreate = false">Cancel</BaseButton>
          <BaseButton variant="primary" type="submit" :loading="creating">Create community</BaseButton>
        </div>
      </form>
    </BaseCard>

    <BaseInput
      v-model="query"
      type="search"
      label="Search communities"
      placeholder="Search by name or topic"
      @update:model-value="scheduleSearch"
    />

    <SkeletonList v-if="loading" :rows="4" />

    <EmptyState
      v-else-if="forbidden"
      title="Communities are not available"
      description="You may need to sign in again to browse communities."
      action-label="Go to home"
      @action="router.push('/home')"
    />

    <EmptyState
      v-else-if="error"
      title="We could not load communities"
      :description="error"
      action-label="Try again"
      @action="load(true)"
    />

    <EmptyState
      v-else-if="!communities.length"
      :title="query.trim() ? 'No communities match that search' : 'No communities yet'"
      :description="query.trim() ? 'Try a different name or topic.' : 'Start one for the series or club you care about.'"
      :action-label="query.trim() ? 'Clear search' : 'Create a community'"
      @action="query.trim() ? ((query = ''), load(true)) : (showCreate = true)"
    />

    <template v-else>
      <ul class="communities__list">
        <li v-for="community in communities" :key="community.id">
          <CommunityCard :community="community" />
        </li>
      </ul>
      <div v-if="cursor" class="communities__more">
        <BaseButton :loading="loadingMore" @click="load()">Load more</BaseButton>
      </div>
    </template>
  </div>
</template>

<style scoped>
.communities {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.communities__header {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}

.communities__header h1 {
  font-size: var(--text-2xl);
}

.communities__intro {
  margin-top: var(--space-1);
  color: var(--text-secondary);
}

.communities__create-title {
  font-size: var(--text-xl);
}

.communities__form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  margin-top: var(--space-5);
}

.communities__label {
  color: var(--text-primary);
  font-size: var(--text-sm);
  font-weight: 500;
}

.communities__control {
  width: 100%;
  min-height: 44px;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  background: var(--ink-850);
  color: var(--text-primary);
}

.communities__textarea {
  font-family: var(--font-body);
  line-height: var(--leading-body);
  resize: vertical;
}

.communities__error {
  color: var(--alert);
  font-size: var(--text-sm);
}

.communities__form-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}

.communities__list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.communities__more {
  display: flex;
  justify-content: center;
}
</style>
