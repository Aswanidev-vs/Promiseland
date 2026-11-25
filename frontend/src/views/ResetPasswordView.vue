<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import AppIcon from '@/components/ui/AppIcon.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseInput from '@/components/ui/BaseInput.vue'
import AuthLayout from '@/components/layout/AuthLayout.vue'
import { confirmPasswordReset, requestPasswordReset } from '@/api/auth'
import { errorMessage } from '@/components/features/format'

type Step = 'request' | 'sent' | 'confirm' | 'done'

const route = useRoute()
const token = computed(() => (typeof route.query.token === 'string' ? route.query.token : ''))
const step = ref<Step>(token.value ? 'confirm' : 'request')

const form = reactive({ email: '', password: '', confirm: '' })
const errors = reactive({ email: '', password: '', confirm: '' })
const formError = ref('')
const submitting = ref(false)

const title = computed(() => {
  if (step.value === 'sent') return 'Check your email'
  if (step.value === 'confirm') return 'Choose a new password'
  if (step.value === 'done') return 'Password updated'
  return 'Reset your password'
})

const intro = computed(() => {
  if (step.value === 'sent') return 'If an account exists for that address, a reset link is on its way.'
  if (step.value === 'confirm') return 'Use at least eight characters.'
  if (step.value === 'done') return 'Your new password is saved. Sign in to continue.'
  return 'We will email you a link with a time-limited token.'
})

watch(
  () => form.email,
  () => {
    errors.email = ''
    formError.value = ''
  },
)

watch(
  () => form.password,
  () => {
    errors.password = ''
    formError.value = ''
  },
)

watch(
  () => form.confirm,
  () => {
    errors.confirm = ''
    formError.value = ''
  },
)

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
  <AuthLayout :title="title" :intro="intro">
    <form v-if="step === 'request'" class="auth-form" novalidate @submit.prevent="request">
      <BaseInput
        v-model="form.email"
        label="Email address"
        type="email"
        name="email"
        autocomplete="email"
        placeholder="you@example.com"
        :error="errors.email"
      />

      <p v-if="formError" class="auth-form__error" role="alert">
        <span class="auth-form__error-icon"><AppIcon name="alert" :size="18" /></span>
        <span>{{ formError }}</span>
      </p>

      <BaseButton type="submit" variant="primary" block :loading="submitting">Send reset link</BaseButton>
    </form>

    <form v-else-if="step === 'confirm'" class="auth-form" novalidate @submit.prevent="confirm">
      <BaseInput
        v-model="form.password"
        label="New password"
        type="password"
        name="password"
        autocomplete="new-password"
        revealable
        hint="At least 8 characters."
        :error="errors.password"
      />
      <BaseInput
        v-model="form.confirm"
        label="Confirm new password"
        type="password"
        name="confirm"
        autocomplete="new-password"
        revealable
        :error="errors.confirm"
      />

      <p v-if="formError" class="auth-form__error" role="alert">
        <span class="auth-form__error-icon"><AppIcon name="alert" :size="18" /></span>
        <span>{{ formError }}</span>
      </p>

      <BaseButton type="submit" variant="primary" block :loading="submitting">Set new password</BaseButton>
    </form>

    <p v-else-if="step === 'sent'" class="auth-note">
      The link expires after a short while, so open it soon after it arrives.
    </p>
    <p v-else class="auth-note">You can close this tab.</p>

    <template #footer>
      <p v-if="step === 'request' || step === 'confirm'" class="auth-switch">
        <RouterLink to="/login">Back to sign in</RouterLink>
      </p>
      <BaseButton v-else-if="step === 'sent'" variant="secondary" to="/login">Back to sign in</BaseButton>
      <BaseButton v-else variant="primary" to="/login">Sign in</BaseButton>
    </template>
  </AuthLayout>
</template>

<style scoped>
.auth-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.auth-form__error {
  display: flex;
  align-items: flex-start;
  gap: var(--space-2);
  padding: var(--space-3);
  border: 1px solid var(--alert);
  border-radius: var(--radius-md);
  background: var(--ink-850);
  color: var(--text-primary);
  font-size: var(--text-sm);
}

.auth-form__error-icon {
  display: flex;
  flex: none;
  color: var(--alert);
}

.auth-note {
  max-width: 40ch;
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.auth-switch {
  font-size: var(--text-sm);
}

.auth-switch a {
  display: inline-flex;
  align-items: center;
  min-height: 44px;
  color: var(--tide);
  font-weight: 500;
  text-decoration: none;
  transition: color var(--dur-fast) var(--ease);
}

.auth-switch a:hover {
  color: var(--text-primary);
}
</style>
