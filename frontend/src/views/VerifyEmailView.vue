<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import EmptyState from '@/components/ui/EmptyState.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import { verifyEmail } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { errorMessage } from '@/components/features/format'

type State = 'loading' | 'success' | 'error' | 'missing'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const ui = useUiStore()
const state = ref<State>('loading')
const error = ref('')

const token = computed(() => (typeof route.query.token === 'string' ? route.query.token : ''))

async function verify(): Promise<void> {
  if (!token.value) {
    state.value = 'missing'
    return
  }
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
  <div class="verify">
    <p v-if="state === 'loading'" class="verify__status" role="status">Verifying your email…</p>

    <EmptyState
      v-else-if="state === 'success'"
      title="Email verified"
      description="Your account is ready. Your communities and feed are waiting."
      action-label="Go to home"
      @action="router.push('/home')"
    />

    <EmptyState
      v-else-if="state === 'missing'"
      title="No verification link found"
      description="Open the verification link from your email to confirm your address."
      action-label="Go to home"
      @action="router.push('/home')"
    />

    <div v-else class="verify__failure">
      <EmptyState
        title="We could not verify your email"
        description="The link may have expired or already been used. Check your email for a newer link."
        action-label="Go to home"
        @action="router.push('/home')"
      />
      <p class="verify__detail" role="alert">{{ error }}</p>
      <BaseButton variant="ghost" @click="verify">Retry verification</BaseButton>
    </div>
  </div>
</template>

<style scoped>
.verify {
  margin: var(--space-12) auto;
}

.verify__status,
.verify__detail {
  color: var(--text-secondary);
  text-align: center;
}

.verify__failure {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-4);
}
</style>
