<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseInput from '@/components/ui/BaseInput.vue'
import AppIcon from '@/components/ui/AppIcon.vue'
import AuthLayout from '@/components/layout/AuthLayout.vue'
import { register } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'
import { useGuestStore } from '@/stores/guest'
import { safeRedirect } from '@/components/features/format'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const guest = useGuestStore()

const form = reactive({ email: '', username: '', displayName: '', password: '', confirm: '' })
const errors = reactive({ email: '', username: '', displayName: '', password: '', confirm: '' })
const formError = ref('')
const consentError = ref('')
const submitting = ref(false)
const consent = reactive({ terms: false, privacy: false })

const loginTo = computed(() => ({
  path: '/login',
  query: typeof route.query.redirect === 'string' ? { redirect: route.query.redirect } : {},
}))

const passwordHint = computed(() =>
  form.password.length === 0 || form.password.length >= 8
    ? 'At least 8 characters.'
    : `${8 - form.password.length} more characters needed.`,
)

function clearError(field: keyof typeof errors): void {
  if (errors[field]) errors[field] = ''
  if (formError.value) formError.value = ''
}

watch(() => form.email, () => clearError('email'))
watch(() => form.username, () => clearError('username'))
watch(() => form.displayName, () => clearError('displayName'))
watch(() => form.password, () => clearError('password'))
watch(() => form.confirm, () => clearError('confirm'))
watch(
  () => [consent.terms, consent.privacy],
  () => {
    consentError.value = ''
  },
)

function validate(): boolean {
  errors.email = form.email.trim() ? (/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email.trim()) ? '' : 'Enter a valid email address.') : 'Enter your email address.'
  errors.username = !form.username.trim()
    ? 'Choose a username.'
    : /^[a-z0-9_]{3,30}$/.test(form.username.trim())
      ? ''
      : 'Use 3 to 30 lowercase letters, numbers, or underscores.'
  errors.displayName = form.displayName.trim() ? '' : 'Enter the name people will see.'
  errors.password = form.password.length >= 8 ? '' : 'Use at least 8 characters.'
  errors.confirm = form.confirm === form.password ? '' : 'Passwords do not match.'
  consentError.value =
    consent.terms && consent.privacy
      ? ''
      : 'Agree to the terms of use and the privacy policy to create an account.'
  return !Object.values(errors).some(Boolean) && !consentError.value
}

async function submit(): Promise<void> {
  if (!validate()) return
  submitting.value = true
  formError.value = ''
  try {
    await register({
      email: form.email.trim(),
      username: form.username.trim(),
      displayName: form.displayName.trim(),
      password: form.password,
    })
    await auth.fetchMe()
    await router.push(safeRedirect(route.query.redirect))
  } catch {
    formError.value = "We couldn't create your account. Check the fields above and try again."
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <AuthLayout
    title="Create your account"
    intro="One account for communities, your feed and events."
  >
    <form class="auth-form" novalidate @submit.prevent="submit">
      <BaseInput
        v-model="form.email"
        label="Email address"
        type="email"
        name="email"
        autocomplete="email"
        placeholder="you@example.com"
        :error="errors.email"
      />

      <div class="auth-form__row">
        <BaseInput
          v-model="form.username"
          label="Username"
          name="username"
          autocomplete="username"
          placeholder="your_handle"
          hint="Lowercase letters, numbers, and underscores."
          :error="errors.username"
        />
        <BaseInput
          v-model="form.displayName"
          label="Display name"
          name="displayName"
          autocomplete="nickname"
          placeholder="Mira"
          :error="errors.displayName"
        />
      </div>

      <BaseInput
        v-model="form.password"
        label="Password"
        type="password"
        name="password"
        autocomplete="new-password"
        revealable
        :hint="passwordHint"
        :error="errors.password"
      />
      <BaseInput
        v-model="form.confirm"
        label="Confirm password"
        type="password"
        name="confirm"
        autocomplete="new-password"
        revealable
        :error="errors.confirm"
      />

      <fieldset class="consent">
        <legend class="consent__legend">Before you join</legend>
        <label class="consent__row">
          <input v-model="consent.terms" type="checkbox" class="consent__box" />
          <span>
            I agree to the <a href="/terms" target="_blank" rel="noopener">terms of use</a>.
          </span>
        </label>
        <label class="consent__row">
          <input v-model="consent.privacy" type="checkbox" class="consent__box" />
          <span>
            I agree to the <a href="/privacy" target="_blank" rel="noopener">privacy policy</a>.
          </span>
        </label>
        <p v-if="consentError" class="consent__error" role="alert">
          <AppIcon name="alert" :size="16" />
          <span>{{ consentError }}</span>
        </p>
      </fieldset>

      <p v-if="formError" class="auth-form__error" role="alert">
        <span class="auth-form__error-icon"><AppIcon name="alert" :size="18" /></span>
        <span>{{ formError }}</span>
      </p>

      <BaseButton type="submit" variant="primary" block :loading="submitting">Create account</BaseButton>
    </form>

    <template #footer>
      <p class="auth-switch">Already have an account? <RouterLink :to="loginTo">Sign in</RouterLink></p>
      <p class="auth-switch">
        <RouterLink to="/events" @click="guest.enter()">Continue as a guest</RouterLink>
      </p>
    </template>
  </AuthLayout>
</template>

<style scoped>
.auth-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.auth-form__row {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--space-4);
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

.auth-switch {
  color: var(--text-secondary);
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

.consent {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  margin: 0;
  padding: var(--space-4);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-md);
  background: var(--ink-850);
}

.consent__legend {
  padding: 0 var(--space-1);
  color: var(--text-secondary);
  font-size: var(--text-xs);
}

.consent__row {
  display: flex;
  align-items: flex-start;
  gap: var(--space-3);
  min-height: 44px;
  padding-block: var(--space-2);
  color: var(--text-secondary);
  font-size: var(--text-sm);
  line-height: 1.5;
  cursor: pointer;
}

.consent__box {
  flex: none;
  width: 18px;
  height: 18px;
  margin: 2px 0 0;
  accent-color: var(--ember);
}

.consent__row a {
  color: var(--tide);
  text-decoration: none;
}

.consent__row a:hover {
  color: var(--text-primary);
}

.consent__error {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--alert);
  font-size: var(--text-sm);
}

@media (max-width: 767px) {
  .auth-form {
    gap: var(--space-4);
  }

  .consent {
    padding: var(--space-3);
  }

  .consent__row {
    padding-block: var(--space-1);
  }
}

@media (max-width: 560px) {
  .auth-form__row {
    grid-template-columns: minmax(0, 1fr);
    gap: var(--space-5);
  }
}
</style>
