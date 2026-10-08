import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { setBackend } from '@/services/backend'
import { FakeBackend } from '@/services/fakeBackend'
import { useAppStore } from './app'

describe('app store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('loads app info', async () => {
    setBackend(new FakeBackend())
    const store = useAppStore()
    const pending = store.load()
    expect(store.state).toBe('loading')
    await pending
    expect(store.state).toBe('ready')
    expect(store.info?.name).toBe('LocalMail')
    expect(store.error).toBeNull()
  })

  it('surfaces backend errors', async () => {
    const fake = new FakeBackend()
    fake.system.getAppInfo = async () => {
      throw new Error('boom')
    }
    setBackend(fake)
    const store = useAppStore()
    await store.load()
    expect(store.state).toBe('error')
    expect(store.error).toBe('boom')
    expect(store.info).toBeNull()
  })
})
