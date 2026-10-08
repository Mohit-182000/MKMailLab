import { Clipboard, Events } from '@wailsio/runtime'
import { MailService, SMTPService, SystemService } from '#bindings/localmail/internal/bridge'
import type { Backend, BackendEvents } from './backend'

/** Backend implementation that calls the Go services through Wails. */
export function createWailsBackend(): Backend {
  return {
    system: {
      getAppInfo: () => SystemService.GetAppInfo(),
      getRecentLogs: async () => (await SystemService.GetRecentLogs()) ?? [],
    },
    smtp: {
      getStatus: () => SMTPService.GetStatus(),
      start: () => SMTPService.Start(),
      stop: () => SMTPService.Stop(),
      getConfig: () => SMTPService.GetConfig(),
      getDefaultConfig: () => SMTPService.GetDefaultConfig(),
      saveConfig: (cfg) => SMTPService.SaveConfig(cfg),
      sendTestEmail: (msg) => SMTPService.SendTestEmail(msg),
    },
    mail: {
      list: (q) => MailService.List(q),
      open: (id) => MailService.Open(id),
      setRead: (ids, read) => MailService.SetRead(ids, read),
      setStarred: (ids, starred) => MailService.SetStarred(ids, starred),
      delete: (ids) => MailService.Delete(ids),
      deleteAll: () => MailService.DeleteAll(),
      saveRaw: (id) => MailService.SaveRaw(id),
      saveAttachment: (id, attachmentId) => MailService.SaveAttachment(id, attachmentId),
    },
    on<K extends keyof BackendEvents>(event: K, cb: (data: BackendEvents[K]) => void) {
      return Events.On(event, (e) => cb(e.data as BackendEvents[K]))
    },
    copyText: async (text) => {
      await Clipboard.SetText(text)
    },
  }
}
