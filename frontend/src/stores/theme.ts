import { ref, watch } from 'vue'
import { defineStore } from 'pinia'

const STORAGE_KEY = 'theme-preference'

export const useThemeStore = defineStore('theme', () => {
  const saved = localStorage.getItem(STORAGE_KEY)
  const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
  const isDark = ref(saved !== null ? saved === 'dark' : prefersDark)

  function toggle() {
    isDark.value = !isDark.value
  }

  watch(isDark, (dark) => {
    document.documentElement.classList.toggle('dark', dark)
    localStorage.setItem(STORAGE_KEY, dark ? 'dark' : 'light')
  }, { immediate: true })

  return { isDark, toggle }
})
