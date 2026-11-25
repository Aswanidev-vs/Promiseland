import { ref } from 'vue'

export type Theme = 'dark' | 'light'

const STORAGE_KEY = 'pl-theme'

function readInitialTheme(): Theme {
  if (typeof window === 'undefined') return 'dark'
  try {
    const saved = window.localStorage.getItem(STORAGE_KEY)
    if (saved === 'light' || saved === 'dark') return saved
  } catch {
    // Ignore storage access errors
  }
  return document.documentElement.getAttribute('data-theme') === 'light' ? 'light' : 'dark'
}

const theme = ref<Theme>(readInitialTheme())

function applyTheme(next: Theme): void {
  theme.value = next
  if (typeof document !== 'undefined') {
    document.documentElement.setAttribute('data-theme', next)
    const meta = document.getElementById('meta-theme-color')
    if (meta) meta.setAttribute('content', next === 'light' ? '#f9f7f3' : '#090a0d')
  }
  try {
    window.localStorage.setItem(STORAGE_KEY, next)
  } catch {
    // Ignore storage access errors
  }
}

export function useTheme() {
  function toggleTheme(): void {
    applyTheme(theme.value === 'dark' ? 'light' : 'dark')
  }

  return {
    theme,
    toggleTheme,
    setTheme: applyTheme,
  }
}
