<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseInput from '@/components/ui/BaseInput.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import WayfinderMark from '@/components/ui/WayfinderMark.vue'
import { confirmPasswordReset, requestPasswordReset } from '@/api/auth'
import { errorMessage } from '@/components/features/format'

const route = useRoute()
const router = useRouter()
const token = computed(() => (typeof route.query.token === 'string' ? route.query.token : ''))

const step = ref<'request' | 'sent' | 'confirm' | 'done'>(token.value ? 'confirm' : 'request')
const form = reactive({ email: '', password: '', confirm: '' })
const errors = reactive({ email: '', password: '', confirm: '' })
const formError = ref('')
const submitting = ref(false)

function request(): void {
  errors.email = form.email.trim() ? (/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email.trim()) ? '' : 'Enter a valid email address.') : 'Enter your email address.'
  if (errors.email) return
  submitting.value = true
  formError.value = ''
  requestPasswordReset(form.email.trim())
    .then(() => {
      step.value = 'sent'
    })
    .catch(() => {
      formError.value = "We couldn't send a reset link. Check your email address and try again."
    })
    .finally(() => {
      submitting.value = false
    })
}

function confirm(): void {
  errors.password = form.password.length >= 8 ? '' : 'Use at least 8 characters.'
  errors.confirm = form.confirm === form.password ? '' : 'Passwords do not match.'
  if (errors.password || errors.confirm) return
  if (!token.value) {
    formError.value = 'This reset link is missing its token. Request a new link.'
    return
  }
  submitting.value = true
  formError.value = ''
  confirmPasswordReset(token.value, form.password)
    .then(() => {
      step.value = 'done'
    })
    .catch((cause) => {
      formError.value = errorMessage(cause, 'We could not reset your password. Request a new link.')
    })
    .finally(() => {
      submitting.value = false
    })
}
</script>

<template>
  <div class="reset">
    <WayfinderMark size="56px" />

    <template v-if="step === 'request'">
      <h1 class="reset__title">Reset your password</h1>
      <p class="reset__intro">We will email a link with a time-limited token.</p>
      <form class="reset__form" novalidate @submit.prevent="request">
        <BaseInput
          v-model="form.email"
          label="Email address"
          type="email"
          name="email"
          autocomplete="email"
          placeholder="you@example.com"
          :error="errors.email"
        />
        <p v-if="formError" class="reset__error" role="alert">{{ formError }}</p>
        <BaseButton type="submit" variant="primary" block :loading="submitting">Send reset link</BaseButton>
      </form>
    </template>

    <EmptyState
      v-else-if="step === 'sent'"
      title="Check your email"
      description="If an account exists for that address, a reset link is on its way."
      action-label="Back to sign in"
      @action="router.push('/login')"
    />

    <template v-else-if="step === 'confirm'">
      <h1 class="reset__title">Choose a new password</h1>
      <p class="reset__intro">Use at least eight characters.</p>
      <form class="reset__form" novalidate @submit.prevent="confirm">
        <BaseInput
          v-model="form.password"
          label="New password"
          type="password"
          name="password"
          autocomplete="new-password"
          :error="errors.password"
        />
        <BaseInput
          v-model="form.confirm"
          label="Confirm new password"
          type="password"
          name="confirm"
          autocomplete="new-password"
          :error="errors.confirm"
        />
        <p v-if="formError" class="reset__error" role="alert">{{ formError }}</p>
        <BaseButton type="submit" variant="primary" block :loading="submitting">Set new password</BaseButton>
      </form>
    </template>

    <EmptyState
      v-else
      title="Password updated"
      description="Sign in with your new password to continue."
      action-label="Sign in"
      @action="router.push('/login')"
    />

    <p class="reset__footer"><RouterLink to="/login">Back to sign in</RouterLink></p>
  </div>
</template>

<style scoped>
.reset {
  display: flex;
  width: min(100%, 460px);
  flex-direction: column;
  align-items: stretch;
  gap: var(--space-3);
  margin: var(--space-12) auto;
  padding: var(--space-8);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-lg);
  background: var(--ink-800);
}

.reset__title {
  margin-top: var(--space-2);
  font-size: var(--text-2xl);
}

.reset__intro {
  color: var(--text-secondary);
}

.reset__form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  margin-top: var(--space-4);
}

.reset__error {
  color: var(--alert);
  font-size: var(--text-sm);
}

.reset__footer {
  color: var(--text-secondary);
  font-size: var(--text-sm);
  text-align: center;
}

@media (max-width: 480px) {
  .reset {
    margin: var(--space-6) auto;
    padding: var(--space-5);
  }
}
</style>
