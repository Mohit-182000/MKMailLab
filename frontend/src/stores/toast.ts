import { defineStore } from 'pinia'
import { ref } from 'vue'

export type ToastKind = 'success' | 'error' | 'info'

export interface Toast {
  id: number
  kind: ToastKind
  title: string
  message?: string
}

/** Unobtrusive notifications in the bottom-right corner. */
export const useToastStore = defineStore('toast', () => {
  const toasts = ref<Toast[]>([])
  let nextId = 1

  function push(kind: ToastKind, title: string, message?: string, timeoutMs = 4000): number {
    const id = nextId++
    toasts.value.push({ id, kind, title, message })
    if (toasts.value.length > 4) toasts.value.shift()
    if (timeoutMs > 0) setTimeout(() => dismiss(id), timeoutMs)
    return id
  }

  function dismiss(id: number): void {
    toasts.value = toasts.value.filter((t) => t.id !== id)
  }

  return {
    toasts,
    push,
    dismiss,
    success: (title: string, message?: string) => push('success', title, message),
    error: (title: string, message?: string) => push('error', title, message, 7000),
    info: (title: string, message?: string) => push('info', title, message),
  }
})
