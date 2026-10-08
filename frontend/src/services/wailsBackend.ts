import { SystemService } from '#bindings/localmail/internal/bridge'
import type { Backend } from './backend'

/** Backend implementation that calls the Go services through Wails. */
export function createWailsBackend(): Backend {
  return {
    system: {
      getAppInfo: () => SystemService.GetAppInfo(),
      getRecentLogs: async () => (await SystemService.GetRecentLogs()) ?? [],
    },
  }
}
