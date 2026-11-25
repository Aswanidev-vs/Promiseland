<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import AppIcon from '@/components/ui/AppIcon.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import AuthLayout from '@/components/layout/AuthLayout.vue'
import { verifyEmail } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { errorMessage } from '@/components/features/format'

type State = 'loading' | 'success' | 'error' | 'missing'

const route = useRoute()
const auth = useAuthStore()
const ui = useUiStore()
const state = ref<State>('loading')
const error = ref('')

const token = computed(() => (typeof route.query.token === 'string' ? route.query.token : ''))

const title = computed(() => {
  if (state.value === 'success') return 'Email verified'
  if (state.value === 'missing') return 'No verification link found'
  if (state.value === 'error') return 'We could not verify your email'
  return 'Verifying your email'
})

const intro = computed(() => {
  if (state.value === 'success') return 'Your account is ready. Your communities and feed are waiting.'
  if (state.value === 'missing') return 'Open the verification link from your email to confirm your address.'
  if (state.value === 'error') return 'The link may have expired or already been used. Check your email for a newer link.'
  return 'This takes a moment.'
})

async function verify(): Promise<void> {
  if (!token.value) {
    state.value = 'missing'
    return
  }
  state.value = 'loading'
  error.value = ''
  try {
    await verifyEmail(token.value)
    await auth.fetchMe()
    state.value = 'success'
    ui.pushToast({ title: 'Email verified', tone: 'success' })
  } catch (cause) {
    error.value = errorMessage(cause, 'We could not verify this link. It may have expired.')
    state.value = 'error'
  }
}

onMounted(verify)
</script>

<template>
  <AuthLayout :title="title" :intro="intro">
    <p v-if="state === 'loading'" class="verify__status" role="status">
      <span class="verify__spinner" aria-hidden="true" />
      Checking the link…
    </p>

    <p v-else-if="state === 'error'" class="auth-error" role="alert">
      <span class="auth-error__icon"><AppIcon name="alert" :size="18" /></span>
      <span>{{ error }}</span>
    </p>

    <template #footer>
      <div v-if="state !== 'loading'" class="verify__actions">
        <BaseButton variant="primary" block to="/home">Go to home</BaseButton>
        <BaseButton v-if="state === 'error'" variant="secondary" block @click="verify">
          Retry verification
        </BaseButton>
      </div>
    </template>
  </AuthLayout>
</template>

<style scoped>
.verify__status {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.verify__spinner {
  width: 16px;
  height: 16px;
  border: 2px solid var(--border-strong);
  border-top-color: var(--ember);
  border-radius: var(--radius-pill);
  animation: verify-spin 700ms linear infinite;
}

@keyframes verify-spin {
  to {
    transform: rotate(360deg);
  }
}

.auth-error {
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

.auth-error__icon {
  display: flex;
  flex: none;
  color: var(--alert);
}

.verify__actions {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
</style>
