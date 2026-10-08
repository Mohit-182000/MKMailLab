import { ref } from 'vue'
import { getBackend } from '@/services/backend'
import { useToastStore } from '@/stores/toast'

/** Copies text and exposes a short-lived "copied" flag for button feedback. */
export function useCopy() {
  const copiedKey = ref<string | null>(null)
  let timer: ReturnType<typeof setTimeout> | null = null

  async function copy(text: string, key = 'default'): Promise<void> {
    try {
      await (await getBackend()).copyText(text)
      copiedKey.value = key
      if (timer) clearTimeout(timer)
      timer = setTimeout(() => (copiedKey.value = null), 1500)
    } catch {
      useToastStore().error('Could not copy to clipboard')
    }
  }

  return { copy, copiedKey }
}
