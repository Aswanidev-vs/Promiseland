<script setup lang="ts">
import { useUiStore } from '@/stores/ui'
import AppIcon from './AppIcon.vue'

const ui = useUiStore()
</script>

<template>
  <div class="toast-host" aria-live="polite" aria-label="Notifications">
    <TransitionGroup name="toast">
      <div v-for="toast in ui.toasts" :key="toast.id" class="toast" :class="`toast--${toast.tone}`">
        <div class="toast__copy">
          <p class="toast__title">{{ toast.title }}</p>
          <p v-if="toast.message" class="toast__message">{{ toast.message }}</p>
        </div>
        <button
          type="button"
          class="toast__close"
          :aria-label="`Dismiss notification: ${toast.title}`"
          @click="ui.dismissToast(toast.id)"
        >
          <AppIcon name="close" :size="16" />
        </button>
      </div>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.toast-host {
  position: fixed;
  right: var(--space-4);
  bottom: calc(var(--bottomnav-h) + var(--space-4));
  z-index: 50;
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  width: min(360px, calc(100vw - var(--space-8)));
  pointer-events: none;
}

.toast {
  display: flex;
  align-items: flex-start;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  border: 1px solid var(--border-strong);
  border-left: 3px solid var(--tide);
  border-radius: var(--radius-md);
  background: var(--ink-800);
  box-shadow: var(--shadow-2);
  pointer-events: auto;
}

.toast--success {
  border-left-color: var(--ok);
}

.toast--error {
  border-left-color: var(--alert);
}

.toast__copy {
  flex: 1;
}

.toast__title {
  font-weight: 500;
}

.toast__message {
  margin-top: var(--space-1);
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

.toast__close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
}

.toast__close:hover {
  color: var(--text-primary);
  background: var(--ink-700);
}

.toast-enter-active,
.toast-leave-active {
  transition:
    opacity var(--dur) var(--ease),
    transform var(--dur) var(--ease);
}

.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateY(8px);
}
</style>
