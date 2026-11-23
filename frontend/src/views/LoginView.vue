<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseInput from '@/components/ui/BaseInput.vue'
import WayfinderMark from '@/components/ui/WayfinderMark.vue'
import { login } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'
import { safeRedirect } from '@/components/features/format'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const form = reactive({ email: '', password: '' })
const errors = reactive({ email: '', password: '' })
const formError = ref('')
const submitting = ref(false)

const registerTo = computed(() => ({
  path: '/register',
  query: typeof route.query.redirect === 'string' ? { redirect: route.query.redirect } : {},
}))

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
    formError.value = "We couldn't sign you in. Check your email and password, then try again."
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="auth">
    <WayfinderMark size="56px" />
    <h1 class="auth__title">Sign in</h1>
    <p class="auth__intro">Pick up the conversation where you left it.</p>

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
        v-model="form.password"
        label="Password"
        type="password"
        name="password"
        autocomplete="current-password"
        :error="errors.password"
      />

      <p v-if="formError" class="auth__error" role="alert">{{ formError }}</p>

      <BaseButton type="submit" variant="primary" block :loading="submitting">Sign in</BaseButton>
    </form>

    <p class="auth__footer">
      New here? <RouterLink :to="registerTo">Create an account</RouterLink>
    </p>
    <p class="auth__footer">
      <RouterLink to="/reset-password">Forgot your password?</RouterLink>
    </p>
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
