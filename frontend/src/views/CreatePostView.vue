<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseInput from '@/components/ui/BaseInput.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import SkeletonList from '@/components/ui/SkeletonList.vue'
import AnimePicker from '@/components/features/AnimePicker.vue'
import { listCommunities } from '@/api/communities'
import { createPost, uploadMedia } from '@/api/posts'
import { useUiStore } from '@/stores/ui'
import type { Anime, Community, Media } from '@/types'
import { errorMessage, isForbidden } from '@/components/features/format'

type PostType = 'text' | 'link' | 'image' | 'question' | 'poll'

const route = useRoute()
const router = useRouter()
const ui = useUiStore()

const types: { value: PostType; label: string }[] = [
  { value: 'text', label: 'Text' },
  { value: 'link', label: 'Link' },
  { value: 'image', label: 'Image' },
  { value: 'question', label: 'Question' },
  { value: 'poll', label: 'Poll' },
]

const type = ref<PostType>('text')
const communities = ref<Community[]>([])
const communitySlug = ref(typeof route.query.community === 'string' ? route.query.community : '')
const title = ref('')
const body = ref('')
const linkUrl = ref('')
const tags = ref('')
const anime = ref<Anime[]>([])
const spoiler = ref(false)
const spoilerNote = ref('')
const season = ref('')
const episode = ref('')
const pollQuestion = ref('')
const pollOptions = ref<string[]>(['', ''])
const media = ref<Media[]>([])
const uploading = ref(false)
const loadingCommunities = ref(true)
const publishing = ref(false)
const error = ref('')
const forbidden = ref(false)
const fieldErrors = ref<Record<string, string>>({})

const canPublish = computed(() => !publishing.value && !uploading.value)

function validate(): boolean {
  const next: Record<string, string> = {}
  if (!title.value.trim()) next.title = 'Add a title.'
  if (type.value === 'link' && !linkUrl.value.trim()) next.linkUrl = 'Add the link you want to share.'
  if (type.value === 'image' && !media.value.length) next.image = 'Upload at least one image.'
  if (type.value === 'poll') {
    if (!pollQuestion.value.trim()) next.pollQuestion = 'Add a poll question.'
    if (pollOptions.value.filter((option) => option.trim()).length < 2) {
      next.pollOptions = 'Add at least two options.'
    }
  }
  if (type.value === 'text' && !body.value.trim()) next.body = 'Add some text for your post.'
  fieldErrors.value = next
  return Object.keys(next).length === 0
}

async function onFile(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  uploading.value = true
  error.value = ''
  try {
    media.value = [...media.value, await uploadMedia(file)]
    fieldErrors.value = { ...fieldErrors.value, image: '' }
    ui.pushToast({ title: 'Image uploaded', tone: 'success' })
  } catch (cause) {
    error.value = errorMessage(cause, 'We could not upload that image. Try again.')
  } finally {
    uploading.value = false
    input.value = ''
  }
}

async function publish(): Promise<void> {
  if (!validate() || !canPublish.value) return
  publishing.value = true
  error.value = ''
  forbidden.value = false
  try {
    // Poll options are passed through createPost: R1 has no separate poll endpoint.
    const input = {
      ...(communitySlug.value ? { communitySlug: communitySlug.value } : {}),
      type: type.value,
      title: title.value.trim(),
      body: body.value.trim(),
      ...(type.value === 'link' ? { linkUrl: linkUrl.value.trim() } : {}),
      ...(media.value.length ? { mediaIds: media.value.map((item) => Number(item.id)) } : {}),
      ...(anime.value.length ? { animeIds: anime.value.map((item) => Number(item.id)) } : {}),
      tags: tags.value
        .split(',')
        .map((tag) => tag.trim())
        .filter(Boolean),
      spoiler: spoiler.value,
      ...(spoiler.value && spoilerNote.value.trim() ? { spoilerNote: spoilerNote.value.trim() } : {}),
      ...(season.value.trim() ? { season: season.value.trim() } : {}),
      ...(episode.value.trim() && Number.isFinite(Number(episode.value))
        ? { episode: Number(episode.value) }
        : {}),
      ...(type.value === 'poll'
        ? { poll: { question: pollQuestion.value.trim(), options: pollOptions.value.filter((option) => option.trim()) } }
        : {}),
    } as unknown as Parameters<typeof createPost>[0]

    const post = await createPost(input)
    ui.pushToast({ title: 'Post published', tone: 'success' })
    await router.push(`/posts/${post.id}`)
  } catch (cause) {
    if (isForbidden(cause)) forbidden.value = true
    else error.value = errorMessage(cause, 'We could not publish your post. Try again.')
  } finally {
    publishing.value = false
  }
}

onMounted(async () => {
  try {
    const page = await listCommunities({ limit: 50 })
    communities.value = page.items
  } catch {
    communities.value = []
  } finally {
    loadingCommunities.value = false
  }
})
</script>

<template>
  <div class="create">
    <h1 class="create__title">Create a post</h1>

    <EmptyState
      v-if="forbidden"
      title="You cannot publish here"
      description="Your account does not have permission to post in this space."
      action-label="Go to home"
      @action="router.push('/home')"
    />

    <form v-else class="create__form" novalidate @submit.prevent="publish">
      <fieldset class="create__types">
        <legend class="create__label">Post type</legend>
        <div class="create__type-list">
          <label v-for="item in types" :key="item.value" class="create__type" :class="{ 'create__type--on': type === item.value }">
            <input v-model="type" type="radio" name="post-type" :value="item.value" />
            {{ item.label }}
          </label>
        </div>
      </fieldset>

      <div class="create__field">
        <label class="create__label" for="post-community">Community</label>
        <SkeletonList v-if="loadingCommunities" :rows="1" />
        <select v-else id="post-community" v-model="communitySlug" class="create__control">
          <option value="">Post to my profile</option>
          <option v-for="community in communities" :key="community.id" :value="community.slug">
            {{ community.name }}
          </option>
        </select>
      </div>

      <BaseInput v-model="title" label="Title" :error="fieldErrors.title" placeholder="What is this about?" />

      <div v-if="type === 'link'" class="create__field">
        <BaseInput v-model="linkUrl" label="Link" type="url" :error="fieldErrors.linkUrl" placeholder="https://" />
      </div>

      <div v-if="type !== 'link'" class="create__field">
        <label class="create__label" for="post-body">Body</label>
        <textarea
          id="post-body"
          v-model="body"
          class="create__control create__textarea"
          rows="6"
          maxlength="10000"
          placeholder="Share your thoughts."
        />
        <p v-if="fieldErrors.body" class="create__error" role="alert">{{ fieldErrors.body }}</p>
      </div>

      <div v-if="type === 'image'" class="create__field">
        <label class="create__label" for="post-image">Image</label>
        <input id="post-image" type="file" accept="image/*" :disabled="uploading" @change="onFile" />
        <p v-if="uploading" class="create__hint" role="status">Uploading image…</p>
        <p v-if="media.length" class="create__hint">{{ media.length }} image attached</p>
        <p v-if="fieldErrors.image" class="create__error" role="alert">{{ fieldErrors.image }}</p>
      </div>

      <div v-if="type === 'poll'" class="create__field">
        <BaseInput v-model="pollQuestion" label="Poll question" :error="fieldErrors.pollQuestion" />
        <span class="create__label">Options</span>
        <div class="create__options">
          <div v-for="(option, index) in pollOptions" :key="`${index}-${option}`" class="create__option">
            <input v-model="pollOptions[index]" class="create__control" :aria-label="`Option ${index + 1}`" />
            <button
              v-if="pollOptions.length > 2"
              type="button"
              class="create__remove"
              :aria-label="`Remove option ${index + 1}`"
              @click="pollOptions.splice(index, 1)"
            >
              Remove
            </button>
          </div>
        </div>
        <BaseButton size="sm" variant="ghost" :disabled="pollOptions.length >= 6" @click="pollOptions.push('')">
          Add option
        </BaseButton>
        <p v-if="fieldErrors.pollOptions" class="create__error" role="alert">{{ fieldErrors.pollOptions }}</p>
      </div>

      <BaseInput v-model="tags" label="Tags" hint="Separate tags with commas." />

      <div class="create__field">
        <AnimePicker v-model="anime" />
      </div>

      <div class="create__field create__field--inline">
        <label class="create__check">
          <input v-model="spoiler" type="checkbox" />
          This post contains spoilers
        </label>
        <div v-if="spoiler" class="create__field">
          <BaseInput v-model="spoilerNote" label="Spoiler note" hint="For example: season 2 finale." />
          <div class="create__row">
            <BaseInput v-model="season" label="Season" />
            <BaseInput v-model="episode" label="Episode" />
          </div>
        </div>
      </div>

      <p v-if="error" class="create__error" role="alert">{{ error }}</p>

      <div class="create__actions">
        <BaseButton variant="ghost" @click="router.push('/home')">Cancel</BaseButton>
        <BaseButton type="submit" variant="primary" :loading="publishing" :disabled="!canPublish">
          Publish post
        </BaseButton>
      </div>
    </form>
  </div>
</template>

<style scoped>
.create {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.create__title {
  font-size: var(--text-2xl);
}

.create__form {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.create__types {
  margin: 0;
  padding: 0;
  border: 0;
}

.create__label {
  display: block;
  margin-bottom: var(--space-2);
  color: var(--text-primary);
  font-size: var(--text-sm);
  font-weight: 500;
}

.create__type-list {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.create__type {
  display: inline-flex;
  min-height: 44px;
  align-items: center;
  padding: 0 var(--space-4);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  color: var(--text-secondary);
  cursor: pointer;
}

.create__type:has(input:checked) {
  border-color: var(--ember);
  color: var(--text-primary);
}

.create__type input {
  position: absolute;
  opacity: 0;
}

.create__type:focus-within {
  outline: 2px solid var(--ember);
  outline-offset: 2px;
}

.create__field {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.create__field--inline {
  gap: var(--space-3);
}

.create__row {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--space-3);
}

.create__control {
  width: 100%;
  min-height: 44px;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  background: var(--ink-850);
  color: var(--text-primary);
  font-family: var(--font-body);
}

.create__textarea {
  line-height: var(--leading-body);
  resize: vertical;
}

.create__options {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.create__option {
  display: flex;
  gap: var(--space-2);
}

.create__remove {
  min-height: 44px;
  padding: 0 var(--space-2);
  border: 0;
  background: none;
  color: var(--text-secondary);
  cursor: pointer;
}

.create__remove:hover {
  color: var(--alert);
}

.create__check {
  display: inline-flex;
  min-height: 44px;
  align-items: center;
  gap: var(--space-2);
  color: var(--text-primary);
  cursor: pointer;
}

.create__check input {
  width: 18px;
  height: 18px;
  accent-color: var(--ember);
}

.create__hint {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.create__error {
  color: var(--alert);
  font-size: var(--text-sm);
}

.create__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}

@media (max-width: 480px) {
  .create__row {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
