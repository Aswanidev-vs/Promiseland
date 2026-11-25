<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import BaseButton from '@/components/ui/BaseButton.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import SkeletonList from '@/components/ui/SkeletonList.vue'
import AnimePicker from '@/components/features/AnimePicker.vue'
import InterestPicker from '@/components/features/InterestPicker.vue'
import { updateAnimeInterests, updateInterests } from '@/api/users'
import { useUiStore } from '@/stores/ui'
import type { Anime } from '@/types'
import { isForbidden, errorMessage } from '@/components/features/format'

const router = useRouter()
const ui = useUiStore()

const step = ref(1)
const interestIds = ref<number[]>([])
const anime = ref<Anime[]>([])
const saving = ref(false)
const error = ref('')
const forbidden = ref(false)

async function finish(): Promise<void> {
  saving.value = true
  error.value = ''
  forbidden.value = false
  try {
    await updateInterests(interestIds.value)
    await updateAnimeInterests(
      anime.value.map((item) => ({ animeId: item.id, interestType: 'favorite' })),
    )
    ui.pushToast({ title: 'Preferences saved', message: 'Your feed is ready.', tone: 'success' })
    await router.push('/home')
  } catch (cause) {
    if (isForbidden(cause)) forbidden.value = true
    else error.value = errorMessage(cause, 'We could not save your preferences. Try again.')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="onboarding">
    <p class="onboarding__progress mono">Step {{ step }} of 2</p>
    <h1 class="onboarding__title">Set your bearings</h1>
    <p class="onboarding__intro">
      A few signals help us show you the communities and posts that fit your fandom. You can change these later.
    </p>

    <SkeletonList v-if="saving" :rows="3" />

    <EmptyState
      v-else-if="forbidden"
      title="You cannot complete setup here"
      description="Your account does not have access to these preferences. You can still explore the feed."
      action-label="Go to home"
      @action="router.push('/home')"
    />

    <template v-else>
      <section v-if="step === 1" class="onboarding__step" aria-labelledby="interests-heading">
        <h2 id="interests-heading" class="onboarding__step-title">What are you into?</h2>
        <InterestPicker v-model="interestIds" />
      </section>

      <section v-else class="onboarding__step" aria-labelledby="anime-heading">
        <h2 id="anime-heading" class="onboarding__step-title">Which titles do you follow?</h2>
        <AnimePicker v-model="anime" />
      </section>

      <p v-if="error" class="onboarding__error" role="alert">{{ error }}</p>

      <div class="onboarding__actions">
        <BaseButton v-if="step === 2" variant="ghost" @click="step = 1">Back</BaseButton>
        <BaseButton v-if="step === 1" variant="primary" @click="step = 2">Continue</BaseButton>
        <BaseButton v-else variant="primary" :loading="saving" @click="finish">Finish setup</BaseButton>
      </div>
      <p class="onboarding__skip">Skip for now and set things up in settings later.</p>
    </template>
  </div>
</template>

<style scoped>
.onboarding {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  margin: var(--space-10) auto;
}

.onboarding__progress {
  color: var(--text-muted);
  font-size: var(--text-sm);
}

.onboarding__title {
  font-size: var(--text-2xl);
  font-family: var(--font-display);
  font-weight: 560;
  letter-spacing: -0.015em;
}

.onboarding__intro {
  max-width: 60ch;
  color: var(--text-secondary);
}

.onboarding__step {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  padding: var(--space-6);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-lg);
  background: var(--ink-800);
}

.onboarding__step-title {
  font-size: var(--text-xl);
}

.onboarding__error {
  color: var(--alert);
  font-size: var(--text-sm);
}

.onboarding__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}

.onboarding__skip {
  color: var(--text-secondary);
  font-size: var(--text-sm);
  text-align: right;
}

@media (max-width: 480px) {
  .onboarding {
    margin: var(--space-6) auto;
  }

  .onboarding__step {
    padding: var(--space-4);
  }
}
</style>
