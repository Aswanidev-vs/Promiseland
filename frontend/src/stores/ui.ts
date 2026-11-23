import { ref } from 'vue'
import { defineStore } from 'pinia'

export type ToastTone = 'info' | 'success' | 'error'

export interface Toast {
  id: number
  title: string
  message?: string
  tone: ToastTone
}

export interface ToastInput {
  title: string
  message?: string
  tone?: ToastTone
}

const TOAST_LIFETIME = 5000

export const useUiStore = defineStore('ui', () => {
  const toasts = ref<Toast[]>([])
  const sidebarOpen = ref(false)
  const mobileNavOpen = ref(false)
  let nextId = 0

  function dismissToast(id: number): void {
    const index = toasts.value.findIndex((toast) => toast.id === id)
    if (index !== -1) toasts.value.splice(index, 1)
  }

  function pushToast(input: ToastInput): number {
    const id = ++nextId
    toasts.value.push({ tone: 'info', ...input, id })
    if (typeof window !== 'undefined') {
      window.setTimeout(() => dismissToast(id), TOAST_LIFETIME)
    }
    return id
  }

  function toggleSidebar(): void {
    sidebarOpen.value = !sidebarOpen.value
  }

  function toggleMobileNav(): void {
    mobileNavOpen.value = !mobileNavOpen.value
  }

  return {
    toasts,
    sidebarOpen,
    mobileNavOpen,
    pushToast,
    dismissToast,
    toggleSidebar,
    toggleMobileNav,
  }
})
