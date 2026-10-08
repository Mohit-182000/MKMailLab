import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getBackend } from '@/services/backend'
import type { AppInfo, LoadState } from '@/types'
import { errorMessage } from '@/utils/errors'

/** Application-level state: identity, paths, startup status. */
export const useAppStore = defineStore('app', () => {
  const info = ref<AppInfo | null>(null)
  const state = ref<LoadState>('idle')
  const error = ref<string | null>(null)

  async function load(): Promise<void> {
    state.value = 'loading'
    error.value = null
    try {
      const backend = await getBackend()
      info.value = await backend.system.getAppInfo()
      state.value = 'ready'
    } catch (err) {
      error.value = errorMessage(err, 'Could not reach the MKMailLab backend')
      state.value = 'error'
    }
  }

  return { info, state, error, load }
})
