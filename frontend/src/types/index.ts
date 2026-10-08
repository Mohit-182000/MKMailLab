// Domain types used across the UI. Backend DTOs are generated from Go into
// /bindings; they are re-exported here under stable names so components never
// import from generated paths directly.
export type { AppInfo } from '#bindings/localmail/internal/bridge/models'
export type { Entry as LogEntry } from '#bindings/localmail/internal/logging/models'

export type LoadState = 'idle' | 'loading' | 'ready' | 'error'
