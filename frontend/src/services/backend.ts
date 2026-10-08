import type {
  AppInfo,
  ListQuery,
  LogEntry,
  MailChange,
  Message,
  MessageSummary,
  Page,
  SaveResult,
  SmtpConfig,
  SmtpStatus,
  TestEmail,
} from '@/types'

/** Payload types of backend → frontend events. */
export interface BackendEvents {
  'smtp:status': SmtpStatus
  'mail:received': MessageSummary
  'mail:changed': MailChange
}

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
  smtp: {
    getStatus(): Promise<SmtpStatus>
    start(): Promise<SmtpStatus>
    stop(): Promise<SmtpStatus>
    getConfig(): Promise<SmtpConfig>
    getDefaultConfig(): Promise<SmtpConfig>
    saveConfig(cfg: SmtpConfig): Promise<SaveResult>
    sendTestEmail(msg: TestEmail): Promise<void>
  }
  mail: {
    list(q: ListQuery): Promise<Page>
    open(id: number): Promise<Message | null>
    setRead(ids: number[], read: boolean): Promise<void>
    setStarred(ids: number[], starred: boolean): Promise<void>
    delete(ids: number[]): Promise<void>
    deleteAll(): Promise<void>
    saveRaw(id: number): Promise<string>
    saveAttachment(messageId: number, attachmentId: number): Promise<string>
  }
  /** Subscribes to a backend event; returns an unsubscribe function. */
  on<K extends keyof BackendEvents>(event: K, cb: (data: BackendEvents[K]) => void): () => void
  /** Copies text to the system clipboard. */
  copyText(text: string): Promise<void>
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

/** URL helpers for content served by the in-process asset handler. */
export const contentUrl = {
  html: (id: number) => `/lm/messages/${id}/html`,
  raw: (id: number) => `/lm/messages/${id}/raw`,
  attachment: (id: number, attachmentId: number) => `/lm/messages/${id}/attachments/${attachmentId}`,
}
