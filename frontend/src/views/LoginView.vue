<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseInput from '@/components/ui/BaseInput.vue'
import AppIcon from '@/components/ui/AppIcon.vue'
import AuthLayout from '@/components/layout/AuthLayout.vue'
import { login } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'
import { useGuestStore } from '@/stores/guest'
import { safeRedirect } from '@/components/features/format'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const guest = useGuestStore()

const form = reactive({ email: '', password: '' })
const errors = reactive({ email: '', password: '' })
const formError = ref('')
const submitting = ref(false)

const registerTo = computed(() => ({
  path: '/register',
  query: typeof route.query.redirect === 'string' ? { redirect: route.query.redirect } : {},
}))

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

function validate(): boolean {
  errors.email = form.email.trim() ? (/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email.trim()) ? '' : 'Enter a valid email address.') : 'Enter your email address.'
  errors.password = form.password ? '' : 'Enter your password.'
  return !errors.email && !errors.password
}

async function submit(): Promise<void> {
  if (!validate()) return
  submitting.value = true
  formError.value = ''
  try {
    await login({ email: form.email.trim(), password: form.password })
    await auth.fetchMe()
    await router.push(safeRedirect(route.query.redirect))
  } catch {
    formError.value = "We couldn't sign you in with that email and password. Check them and try again."
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <AuthLayout title="Sign in" intro="Pick up the conversation where you left it.">
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
      <BaseInput
        v-model="form.password"
        label="Password"
        type="password"
        name="password"
        autocomplete="current-password"
        revealable
        :error="errors.password"
      >
        <template #label-action>
          <RouterLink class="field-link" to="/reset-password">Forgot password?</RouterLink>
        </template>
      </BaseInput>

      <p v-if="formError" class="auth-form__error" role="alert">
        <span class="auth-form__error-icon"><AppIcon name="alert" :size="18" /></span>
        <span>{{ formError }}</span>
      </p>

      <BaseButton type="submit" variant="primary" block :loading="submitting">Sign in</BaseButton>
    </form>

    <template #footer>
      <p class="auth-switch">New here? <RouterLink :to="registerTo">Create an account</RouterLink></p>
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

/* 44px hit area that does not add height to the label row. */
.field-link {
  display: inline-flex;
  align-items: center;
  min-height: 44px;
  margin-block: calc(var(--space-3) * -1);
  color: var(--tide);
  font-size: var(--text-sm);
  font-weight: 500;
  text-decoration: none;
  transition: color var(--dur-fast) var(--ease);
}

.field-link:hover {
  color: var(--text-primary);
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
</style>
