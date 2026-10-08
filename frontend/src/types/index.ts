// Domain types used across the UI. Backend DTOs are generated from Go into
// /bindings; they are re-exported here under stable names so components never
// import from generated paths directly.
export type { AppInfo, SaveResult } from '#bindings/localmail/internal/bridge/models'
export type { Entry as LogEntry } from '#bindings/localmail/internal/logging/models'
export type {
  Address,
  Attachment,
  Header,
  ListQuery,
  Message,
  MessageSummary,
  Page,
} from '#bindings/localmail/internal/domain/models'
export { ParseStatus } from '#bindings/localmail/internal/domain/models'
export type { Change as MailChange } from '#bindings/localmail/internal/mailbox/models'
export { ChangeKind } from '#bindings/localmail/internal/mailbox/models'
export type { SMTP as SmtpConfig } from '#bindings/localmail/internal/settings/models'
export { AuthMode } from '#bindings/localmail/internal/settings/models'
export type { Status as SmtpStatus } from '#bindings/localmail/internal/smtpd/models'
export { State as SmtpState } from '#bindings/localmail/internal/smtpd/models'
export type { Message as TestEmail } from '#bindings/localmail/internal/testmail/models'

export type LoadState = 'idle' | 'loading' | 'ready' | 'error'
