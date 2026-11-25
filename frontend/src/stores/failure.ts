import { defineStore } from 'pinia'
import { ref } from 'vue'

/**
 * One place for failures the whole shell has to show: an unreachable API (503)
 * or a render crash caught by the app-level handler (500).
 */
export const useFailureStore = defineStore('failure', () => {
  const serviceUnavailable = ref(false)
  const renderFailed = ref(false)

  function markServiceUnavailable(value = true): void {
    serviceUnavailable.value = value
  }

  function markRenderFailed(): void {
    renderFailed.value = true
  }

  return { serviceUnavailable, renderFailed, markServiceUnavailable, markRenderFailed }
})
