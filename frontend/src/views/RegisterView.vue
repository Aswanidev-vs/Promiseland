<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseInput from '@/components/ui/BaseInput.vue'
import WayfinderMark from '@/components/ui/WayfinderMark.vue'
import { register } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'
import { safeRedirect } from '@/components/features/format'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const form = reactive({ email: '', username: '', displayName: '', password: '', confirm: '' })
const errors = reactive({ email: '', username: '', displayName: '', password: '', confirm: '' })
const formError = ref('')
const submitting = ref(false)

const loginTo = computed(() => ({
  path: '/login',
  query: typeof route.query.redirect === 'string' ? { redirect: route.query.redirect } : {},
}))

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
  return !Object.values(errors).some(Boolean)
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
    formError.value = "We couldn't create your account. Check your details and try again."
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="auth">
    <WayfinderMark size="56px" />
    <h1 class="auth__title">Create your account</h1>
    <p class="auth__intro">Find your communities and start sharing.</p>

    <form class="auth__form" novalidate @submit.prevent="submit">
      <BaseInput
        v-model="form.email"
        label="Email address"
        type="email"
        name="email"
        autocomplete="email"
        placeholder="you@example.com"
        :error="errors.email"
      />
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
        :error="errors.displayName"
      />
      <BaseInput
        v-model="form.password"
        label="Password"
        type="password"
        name="password"
        autocomplete="new-password"
        :error="errors.password"
      />
      <BaseInput
        v-model="form.confirm"
        label="Confirm password"
        type="password"
        name="confirm"
        autocomplete="new-password"
        :error="errors.confirm"
      />

      <p v-if="formError" class="auth__error" role="alert">{{ formError }}</p>

      <BaseButton type="submit" variant="primary" block :loading="submitting">Create account</BaseButton>
    </form>

    <p class="auth__footer">Already have an account? <RouterLink :to="loginTo">Sign in</RouterLink></p>
  </div>
</template>

<style scoped>
.auth {
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

.auth__title {
  margin-top: var(--space-2);
  font-size: var(--text-2xl);
}

.auth__intro {
  color: var(--text-secondary);
}

.auth__form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  margin-top: var(--space-4);
}

.auth__error {
  color: var(--alert);
  font-size: var(--text-sm);
}

.auth__footer {
  color: var(--text-secondary);
  font-size: var(--text-sm);
  text-align: center;
}

@media (max-width: 480px) {
  .auth {
    margin: var(--space-6) auto;
    padding: var(--space-5);
  }
}
</style>
