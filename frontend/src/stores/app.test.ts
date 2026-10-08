import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { setBackend, type Backend } from '@/services/backend'
import type { AppInfo } from '@/types'
import { useAppStore } from './app'

const info: AppInfo = {
  name: 'LocalMail',
  version: '0.1.0',
  commit: 'abc',
  dev: true,
  portable: false,
  dataDir: 'C:\\data',
  databasePath: 'C:\\data\\localmail.db',
  logDir: 'C:\\data\\logs',
  logFile: 'C:\\data\\logs\\localmail.log',
  startedAt: 1_700_000_000_000,
  goVersion: 'go1.27.1',
  platform: 'windows/amd64',
}

function fakeBackend(getAppInfo: Backend['system']['getAppInfo']): Backend {
  return { system: { getAppInfo, getRecentLogs: async () => [] } }
}

describe('app store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('loads app info', async () => {
    setBackend(fakeBackend(vi.fn().mockResolvedValue(info)))
    const store = useAppStore()
    const pending = store.load()
    expect(store.state).toBe('loading')
    await pending
    expect(store.state).toBe('ready')
    expect(store.info?.version).toBe('0.1.0')
    expect(store.error).toBeNull()
  })

  it('surfaces backend errors', async () => {
    setBackend(fakeBackend(vi.fn().mockRejectedValue(new Error('boom'))))
    const store = useAppStore()
    await store.load()
    expect(store.state).toBe('error')
    expect(store.error).toBe('boom')
    expect(store.info).toBeNull()
  })
})
