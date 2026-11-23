<script setup lang="ts">
/**
 * Opt-in profile form for the gated adult area. Local UX validation only —
 * the server remains the source of truth. Emits a camelCase payload; the
 * api/adult.ts module maps it to the snake_case wire shape.
 */
import { ref, watch } from 'vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseInput from '@/components/ui/BaseInput.vue'
import type { AdultIntent, AdultProfile, AdultProfileInput } from '@/types'

const props = withDefaults(
  defineProps<{
    profile?: AdultProfile | null
    saving?: boolean
    /** Save failure passed down from the view. */
    error?: string
  }>(),
  { profile: null, saving: false, error: '' },
)

const emit = defineEmits<{ save: [input: AdultProfileInput] }>()

const displayName = ref('')
const intent = ref<AdultIntent | ''>('')
const bio = ref('')
const boundaries = ref('')

interface FieldErrors {
  displayName?: string
  intent?: string
  bio?: string
  boundaries?: string
}
const errors = ref<FieldErrors>({})

watch(
  () => props.profile,
  (profile) => {
    if (!profile) return
    displayName.value = profile.display_name
    intent.value = profile.intent
    bio.value = profile.bio
    boundaries.value = profile.boundaries
  },
  { immediate: true },
)

function validate(): boolean {
  const next: FieldErrors = {}
  const name = displayName.value.trim()
  if (!name) next.displayName = 'Enter a display name for this area.'
  else if (name.length > 40) next.displayName = 'Keep the display name under 40 characters.'

  if (!intent.value) next.intent = 'Choose what you are here for.'

  const about = bio.value.trim()
  if (!about) next.bio = 'Write a short introduction.'
  else if (about.length > 500) next.bio = 'Keep your introduction under 500 characters.'

  const limits = boundaries.value.trim()
  if (!limits) next.boundaries = 'Add at least one boundary.'
  else if (limits.length > 300) next.boundaries = 'Keep your boundaries under 300 characters.'

  errors.value = next
  return Object.keys(next).length === 0
}

function submit(): void {
  if (props.saving || !validate()) return
  const chosen = intent.value
  if (!chosen) return
  emit('save', {
    displayName: displayName.value.trim(),
    intent: chosen,
    bio: bio.value.trim(),
    boundaries: boundaries.value.trim(),
  })
}
</script>

<template>
  <form class="adult-form" novalidate @submit.prevent="submit">
    <p class="adult-form__intro">
      Your answers here stay in this opt-in area. They are not added to your main profile, posts, or communities.
    </p>

    <BaseInput
      id="adult-display-name"
      v-model="displayName"
      label="Display name"
      hint="The name other people in this area will see."
      :error="errors.displayName"
    />

    <div class="adult-form__field" :class="{ 'adult-form__field--invalid': !!errors.intent }">
      <label class="adult-form__label" for="adult-intent">What are you here for</label>
      <select
        id="adult-intent"
        v-model="intent"
        class="adult-form__control"
        :aria-invalid="!!errors.intent || undefined"
        :aria-describedby="errors.intent ? 'adult-intent-error' : undefined"
      >
        <option value="" disabled>Choose one</option>
        <option value="friendship">Friendship</option>
        <option value="dating">Dating</option>
        <option value="both">Both friendship and dating</option>
      </select>
      <p v-if="errors.intent" id="adult-intent-error" class="adult-form__error" role="alert">
        {{ errors.intent }}
      </p>
    </div>

    <div class="adult-form__field" :class="{ 'adult-form__field--invalid': !!errors.bio }">
      <label class="adult-form__label" for="adult-bio">About you</label>
      <textarea
        id="adult-bio"
        v-model="bio"
        class="adult-form__control adult-form__textarea"
        rows="4"
        maxlength="500"
        placeholder="What you care about, in fandom and otherwise."
        :aria-invalid="!!errors.bio || undefined"
        :aria-describedby="errors.bio ? 'adult-bio-error' : 'adult-bio-hint'"
      />
      <p v-if="errors.bio" id="adult-bio-error" class="adult-form__error" role="alert">{{ errors.bio }}</p>
      <p v-else id="adult-bio-hint" class="adult-form__hint">A few honest sentences are enough.</p>
    </div>

    <div class="adult-form__field" :class="{ 'adult-form__field--invalid': !!errors.boundaries }">
      <label class="adult-form__label" for="adult-boundaries">Your boundaries</label>
      <textarea
        id="adult-boundaries"
        v-model="boundaries"
        class="adult-form__control adult-form__textarea"
        rows="3"
        maxlength="300"
        placeholder="What you want other people to respect."
        :aria-invalid="!!errors.boundaries || undefined"
        :aria-describedby="errors.boundaries ? 'adult-boundaries-error' : 'adult-boundaries-hint'"
      />
      <p v-if="errors.boundaries" id="adult-boundaries-error" class="adult-form__error" role="alert">
        {{ errors.boundaries }}
      </p>
      <p v-else id="adult-boundaries-hint" class="adult-form__hint">For example, pace of conversation or topics to avoid.</p>
    </div>

    <p v-if="error" class="adult-form__submit-error" role="alert">{{ error }}</p>

    <div class="adult-form__actions">
      <BaseButton type="submit" variant="primary" :loading="saving">
        {{ profile ? 'Save changes' : 'Create profile' }}
      </BaseButton>
    </div>
  </form>
</template>

<style scoped>
.adult-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.adult-form__intro {
  max-width: 62ch;
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.adult-form__field {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.adult-form__label {
  color: var(--text-primary);
  font-size: var(--text-sm);
  font-weight: 500;
}

.adult-form__control {
  width: 100%;
  min-height: 44px;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  background: var(--ink-850);
  color: var(--text-primary);
  transition: border-color var(--dur-fast) var(--ease);
}

.adult-form__control:hover {
  border-color: var(--ink-600);
}

.adult-form__control:focus {
  border-color: var(--ember);
  outline: none;
}

.adult-form__field--invalid .adult-form__control {
  border-color: var(--alert);
}

.adult-form__textarea {
  line-height: var(--leading-body);
  resize: vertical;
}

.adult-form__hint {
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.adult-form__error,
.adult-form__submit-error {
  color: var(--alert);
  font-size: var(--text-sm);
}

.adult-form__submit-error {
  padding: var(--space-2) var(--space-3);
  border: 1px solid color-mix(in srgb, var(--alert) 40%, transparent);
  border-radius: var(--radius-md);
}

.adult-form__actions {
  display: flex;
  justify-content: flex-start;
}
</style>
