<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import { report } from '@/api/moderation'
import { useUiStore } from '@/stores/ui'
import { errorMessage, REPORT_REASONS } from './format'

const props = withDefaults(
  defineProps<{
    targetType: string
    targetId: number
    /** Chat messages have their own endpoint; other targets use generic /reports. */
    submitter?: (input: { reason: string; details?: string }) => Promise<void>
  }>(),
  { submitter: undefined },
)

const emit = defineEmits<{ close: []; submitted: [] }>()

const ui = useUiStore()
const reason = ref<string>('')
const details = ref('')
const submitting = ref(false)
const error = ref('')

function onKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape') emit('close')
}

onMounted(() => window.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))

async function submit(): Promise<void> {
  if (!reason.value) {
    error.value = 'Choose a reason for this report.'
    return
  }
  if (reason.value === 'other' && !details.value.trim()) {
    error.value = 'Add a short description so moderators know what happened.'
    return
  }
  submitting.value = true
  error.value = ''
  const input = { reason: reason.value, details: details.value.trim() || undefined }
  try {
    if (props.submitter) await props.submitter(input)
    else
      await report({
        targetType: props.targetType,
        targetId: props.targetId,
        ...input,
      })
    ui.pushToast({ title: 'Report sent', message: 'The moderation team will review it.', tone: 'success' })
    emit('submitted')
    emit('close')
  } catch (cause) {
    error.value = errorMessage(cause, 'We could not send your report. Try again.')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="backdrop" @click.self="emit('close')">
    <section class="dialog" role="dialog" aria-modal="true" aria-labelledby="report-title">
      <h2 id="report-title" class="dialog__title">Report this {{ targetType }}</h2>
      <p class="dialog__intro">Tell us what happened. Reports go to moderators, not to the author.</p>

      <form class="dialog__form" @submit.prevent="submit">
        <label class="dialog__label" for="report-reason">Reason</label>
        <select id="report-reason" v-model="reason" class="dialog__control">
          <option value="" disabled>Choose a reason</option>
          <option v-for="item in REPORT_REASONS" :key="item.value" :value="item.value">
            {{ item.label }}
          </option>
        </select>

        <label class="dialog__label" for="report-details">Details <span>(optional)</span></label>
        <textarea
          id="report-details"
          v-model="details"
          class="dialog__control dialog__textarea"
          rows="4"
          maxlength="500"
          placeholder="Add context that will help a moderator."
        />

        <p v-if="error" class="dialog__error" role="alert">{{ error }}</p>

        <div class="dialog__actions">
          <BaseButton type="button" variant="ghost" @click="emit('close')">Cancel</BaseButton>
          <BaseButton type="submit" variant="primary" :loading="submitting">Send report</BaseButton>
        </div>
      </form>
    </section>
  </div>
</template>

<style scoped>
.backdrop {
  position: fixed;
  inset: 0;
  z-index: 60;
  display: grid;
  overflow-y: auto;
  padding: var(--space-4);
  place-items: center;
  background: rgba(0, 0, 0, 0.68);
}

.dialog {
  width: min(100%, 480px);
  padding: var(--space-6);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-lg);
  background: var(--ink-800);
  box-shadow: var(--shadow-2);
}

.dialog__title {
  font-size: var(--text-xl);
}

.dialog__intro {
  margin-top: var(--space-2);
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.dialog__form {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  margin-top: var(--space-6);
}

.dialog__label {
  color: var(--text-primary);
  font-size: var(--text-sm);
  font-weight: 500;
}

.dialog__label span {
  color: var(--text-secondary);
  font-weight: 400;
}

.dialog__control {
  width: 100%;
  min-height: 44px;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  background: var(--ink-850);
  color: var(--text-primary);
}

.dialog__textarea {
  resize: vertical;
  font-family: var(--font-body);
  line-height: var(--leading-body);
}

.dialog__error {
  color: var(--alert);
  font-size: var(--text-sm);
}

.dialog__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-4);
}
</style>
