import type { AppInfo, LogEntry } from '@/types'

/**
 * Backend is the only gateway from the UI to Go. Stores depend on this
 * interface, never on generated bindings, so tests and browser-only
 * development can substitute an in-memory implementation.
 */
export interface Backend {
  system: {
    getAppInfo(): Promise<AppInfo>
    getRecentLogs(): Promise<LogEntry[]>
  }
}

let current: Backend | null = null

/** Returns the active backend. Lazily falls back to the Wails implementation. */
export async function getBackend(): Promise<Backend> {
  if (!current) {
    const { createWailsBackend } = await import('./wailsBackend')
    current = createWailsBackend()
  }
  return current
}

/** Replaces the active backend (tests, mock mode). */
export function setBackend(backend: Backend | null): void {
  current = backend
}
