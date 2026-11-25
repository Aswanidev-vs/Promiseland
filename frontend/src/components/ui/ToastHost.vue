<script setup lang="ts">
import AppIcon from './AppIcon.vue'
import type { IconName } from './icons'
import { useUiStore, type ToastTone } from '@/stores/ui'

const ui = useUiStore()

const toneIcon: Record<ToastTone, IconName> = {
  info: 'info',
  success: 'check',
  error: 'alert',
}
</script>

<template>
  <div class="toast-host" aria-live="polite" aria-label="Notifications">
    <TransitionGroup name="toast" tag="ul" class="toast-host__list">
      <li
        v-for="toast in ui.toasts"
        :key="toast.id"
        class="toast"
        :class="`toast--${toast.tone}`"
        role="status"
      >
        <span class="toast__icon" aria-hidden="true">
          <AppIcon :name="toneIcon[toast.tone]" :size="18" />
        </span>

        <div class="toast__copy">
          <p class="toast__title">{{ toast.title }}</p>
          <p v-if="toast.message" class="toast__message">{{ toast.message }}</p>
        </div>

        <button
          type="button"
          class="toast__close"
          :aria-label="`Dismiss notification: ${toast.title}`"
          title="Dismiss"
          @click="ui.dismissToast(toast.id)"
        >
          <AppIcon name="close" :size="16" />
        </button>
      </li>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.toast-host {
  position: fixed;
  right: var(--space-4);
  bottom: calc(var(--bottomnav-h) + var(--space-4));
  z-index: 50;
  width: min(380px, calc(100vw - var(--space-8)));
  pointer-events: none;
}

.toast-host__list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.toast {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: start;
  gap: var(--space-3);
  padding: var(--space-3);
  border: 1px solid var(--border-hairline);
  border-radius: var(--radius-lg);
  background: var(--ink-800);
  box-shadow: var(--shadow-2);
  pointer-events: auto;
}

/* The icon carries the tone; the surface stays quiet. */
.toast__icon {
  display: grid;
  width: 32px;
  height: 32px;
  place-items: center;
  border-radius: var(--radius-sm);
  background: var(--ink-700);
  color: var(--text-secondary);
}

.toast--success .toast__icon {
  color: var(--ok);
}

.toast--error .toast__icon {
  color: var(--alert);
}

.toast__copy {
  min-width: 0;
  padding-top: var(--space-1);
}

.toast__title {
  font-weight: 500;
  line-height: 1.3;
}

.toast__message {
  margin-top: var(--space-1);
  color: var(--text-secondary);
  font-size: var(--text-sm);
  line-height: 1.45;
}

.toast__close {
  display: grid;
  width: 44px;
  height: 44px;
  margin: calc(var(--space-1) * -1) calc(var(--space-2) * -1) 0 0;
  place-items: center;
  border: 0;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  transition:
    background-color var(--dur-fast) var(--ease),
    color var(--dur-fast) var(--ease);
}

.toast__close:hover {
  background: var(--ink-700);
  color: var(--text-primary);
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
  transform: translateY(12px);
}

/* Leaving toasts leave a gap-free stack while the rest slides up. */
.toast-move {
  transition: transform var(--dur) var(--ease);
}

.toast-leave-active {
  position: absolute;
  right: 0;
  left: 0;
}

@media (min-width: 768px) {
  .toast-host {
    bottom: var(--space-6);
  }
}

@media (max-width: 767px) {
  .toast-host {
    right: var(--space-4);
    left: var(--space-4);
    width: auto;
  }
}
</style>
