import type { Backend, BackendEvents } from './backend'
import {
  AuthMode,
  ChangeKind,
  SmtpState,
  type AppInfo,
  type ListQuery,
  type Message,
  type MessageSummary,
  type SmtpConfig,
  type SmtpStatus,
} from '@/types'

type Listener = (data: unknown) => void

/**
 * In-memory Backend for unit tests (and, later, browser-only UI work).
 * Behaviour mirrors the Go services closely enough to test stores.
 */
export class FakeBackend implements Backend {
  messages: Message[] = []
  config: SmtpConfig = {
    host: '127.0.0.1',
    port: 1025,
    authMode: AuthMode.AuthAny,
    username: '',
    password: '',
    maxMessageMb: 25,
    maxConnections: 100,
    autoStart: true,
  }
  status: SmtpStatus = {
    state: SmtpState.StateRunning,
    running: true,
    host: '127.0.0.1',
    port: 1025,
    error: '',
    startedAt: null,
    received: 0,
    activeSessions: 0,
  }
  clipboard = ''
  opened: string[] = []
  calls: string[] = []
  private listeners = new Map<string, Set<Listener>>()
  private nextId = 1

  /** Adds a message as if received over SMTP and emits mail:received. */
  receive(partial: Partial<Message> = {}): Message {
    const id = this.nextId++
    const msg: Message = {
      id,
      messageId: `<${id}@test>`,
      subject: `Message ${id}`,
      from: { name: 'App', address: 'app@example.com' },
      to: [{ name: '', address: 'user@example.com' }],
      snippet: 'hello',
      receivedAt: new Date(Date.UTC(2026, 9, 8, 10, 0, id)).toISOString(),
      size: 100,
      attachmentCount: 0,
      hasHtml: true,
      isRead: false,
      isStarred: false,
      cc: [],
      bcc: [],
      replyTo: [],
      envelopeFrom: 'app@example.com',
      envelopeTo: ['user@example.com'],
      date: null,
      headers: [],
      text: 'hello',
      html: '<p>hello</p>',
      attachments: [],
      parseStatus: 'ok' as Message['parseStatus'],
      parseErrors: [],
      remoteAddr: '127.0.0.1:1',
      helo: 'test',
      ...partial,
    }
    this.messages.unshift(msg)
    this.emit('mail:received', summary(msg))
    return msg
  }

  emit<K extends keyof BackendEvents>(event: K, data: BackendEvents[K]): void {
    this.listeners.get(event)?.forEach((l) => l(data))
  }

  on<K extends keyof BackendEvents>(event: K, cb: (data: BackendEvents[K]) => void): () => void {
    const set = this.listeners.get(event) ?? new Set()
    set.add(cb as Listener)
    this.listeners.set(event, set)
    return () => set.delete(cb as Listener)
  }

  async copyText(text: string) {
    this.clipboard = text
  }

  async openExternal(url: string) {
    this.opened.push(url)
  }

  system = {
    getAppInfo: async (): Promise<AppInfo> => ({
      name: 'MKMailLab',
      version: '0.0.0-test',
      commit: 'test',
      dev: true,
      portable: false,
      dataDir: 'C:\\data',
      databasePath: 'C:\\data\\mkmaillab.db',
      logDir: 'C:\\data\\logs',
      logFile: 'C:\\data\\logs\\mkmaillab.log',
      startedAt: 0,
      goVersion: 'go',
      platform: 'windows/amd64',
    }),
    getRecentLogs: async () => [],
  }

  smtp = {
    getStatus: async () => this.status,
    start: async () => (this.status = { ...this.status, running: true, state: SmtpState.StateRunning }),
    stop: async () => (this.status = { ...this.status, running: false, state: SmtpState.StateStopped }),
    getConfig: async () => this.config,
    getDefaultConfig: async () => ({ ...this.config, port: 1025 }),
    saveConfig: async (cfg: SmtpConfig) => {
      this.config = cfg
      return { config: cfg, status: this.status, warning: '' }
    },
    sendTestEmail: async () => {
      this.calls.push('sendTestEmail')
    },
  }

  mail = {
    list: async (q: ListQuery) => {
      const s = q.search.toLowerCase()
      let all = this.messages.filter(
        (m) => (!q.unreadOnly || !m.isRead) && (!s || m.subject.toLowerCase().includes(s)),
      )
      const total = all.length
      const unread = all.filter((m) => !m.isRead).length
      const start = q.cursor ? Number(q.cursor) : 0
      const limit = q.limit || 50
      all = all.slice(start, start + limit)
      const next = start + limit < total ? String(start + limit) : ''
      return { items: all.map(summary), nextCursor: next, total, unread }
    },
    open: async (id: number) => {
      const m = this.messages.find((x) => x.id === id) ?? null
      if (m && !m.isRead) {
        m.isRead = true
        this.emit('mail:changed', { kind: ChangeKind.ChangeRead, ids: [id] })
      }
      return m ? { ...m } : null
    },
    setRead: async (ids: number[], read: boolean) => {
      this.messages.forEach((m) => ids.includes(m.id) && (m.isRead = read))
      this.emit('mail:changed', { kind: read ? ChangeKind.ChangeRead : ChangeKind.ChangeUnread, ids })
    },
    setStarred: async (ids: number[], starred: boolean) => {
      this.messages.forEach((m) => ids.includes(m.id) && (m.isStarred = starred))
      this.emit('mail:changed', { kind: starred ? ChangeKind.ChangeStarred : ChangeKind.ChangeUnstarred, ids })
    },
    delete: async (ids: number[]) => {
      this.messages = this.messages.filter((m) => !ids.includes(m.id))
      this.emit('mail:changed', { kind: ChangeKind.ChangeDeleted, ids })
    },
    deleteAll: async () => {
      this.messages = []
      this.emit('mail:changed', { kind: ChangeKind.ChangeCleared, ids: [] })
    },
    saveRaw: async () => 'C:\\saved.eml',
    saveAttachment: async () => 'C:\\saved.bin',
  }
}

function summary(m: Message): MessageSummary {
  const { id, messageId, subject, from, to, snippet, receivedAt, size, attachmentCount, hasHtml, isRead, isStarred } = m
  return { id, messageId, subject, from, to, snippet, receivedAt, size, attachmentCount, hasHtml, isRead, isStarred }
}
