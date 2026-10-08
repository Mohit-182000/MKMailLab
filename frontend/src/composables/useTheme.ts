import { computed, ref } from 'vue'

export type ThemePreference = 'system' | 'light' | 'dark'

const STORAGE_KEY = 'lm.theme'
const media = typeof window !== 'undefined' ? window.matchMedia('(prefers-color-scheme: dark)') : null

function readPreference(): ThemePreference {
  try {
    const v = localStorage.getItem(STORAGE_KEY)
    if (v === 'light' || v === 'dark' || v === 'system') return v
  } catch {
    // storage unavailable
  }
  return 'system'
}

const preference = ref<ThemePreference>(readPreference())
const systemDark = ref(media?.matches ?? false)
media?.addEventListener('change', (e) => {
  systemDark.value = e.matches
  apply()
})

const resolved = computed<'light' | 'dark'>(() =>
  preference.value === 'system' ? (systemDark.value ? 'dark' : 'light') : preference.value,
)

function apply(): void {
  document.documentElement.dataset.theme = resolved.value
}

/** Light / dark / system theme, persisted per user. */
export function useTheme() {
  function setPreference(p: ThemePreference): void {
    preference.value = p
    try {
      localStorage.setItem(STORAGE_KEY, p)
    } catch {
      // ignore
    }
    apply()
  }

  function cycle(): void {
    const order: ThemePreference[] = ['system', 'light', 'dark']
    setPreference(order[(order.indexOf(preference.value) + 1) % order.length] ?? 'system')
  }

  return { preference, resolved, setPreference, cycle, apply }
}
