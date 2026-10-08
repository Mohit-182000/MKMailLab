import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { getBackend } from '@/services/backend'
import type { SaveResult, SmtpConfig, SmtpStatus, TestEmail } from '@/types'
import { errorMessage } from '@/utils/errors'
import { connectionInfo } from '@/utils/snippets'
import { useToastStore } from './toast'

/** SMTP server status, configuration and controls. */
export const useSmtpStore = defineStore('smtp', () => {
  const status = ref<SmtpStatus | null>(null)
  const config = ref<SmtpConfig | null>(null)
  const busy = ref(false)
  let unsubscribe: (() => void) | null = null

  const running = computed(() => status.value?.running ?? false)
  const hasError = computed(() => !!status.value?.error)
  const connection = computed(() =>
    config.value ? connectionInfo(config.value, running.value ? status.value?.port : undefined) : null,
  )

  async function init(): Promise<void> {
    const backend = await getBackend()
    unsubscribe?.()
    unsubscribe = backend.on('smtp:status', (s) => {
      status.value = s
    })
    const [s, c] = await Promise.all([backend.smtp.getStatus(), backend.smtp.getConfig()])
    status.value = s
    config.value = c
  }

  async function start(): Promise<void> {
    await run(async (b) => {
      status.value = await b.smtp.start()
    }, 'Could not start SMTP server')
  }

  async function stop(): Promise<void> {
    await run(async (b) => {
      status.value = await b.smtp.stop()
    }, 'Could not stop SMTP server')
  }

  async function toggle(): Promise<void> {
    return running.value ? stop() : start()
  }

  /** Saves settings; throws so forms can show field errors. */
  async function save(cfg: SmtpConfig): Promise<SaveResult> {
    const backend = await getBackend()
    busy.value = true
    try {
      const res = await backend.smtp.saveConfig(cfg)
      config.value = res.config
      status.value = res.status
      return res
    } finally {
      busy.value = false
    }
  }

  async function defaults(): Promise<SmtpConfig> {
    return (await getBackend()).smtp.getDefaultConfig()
  }

  async function sendTest(msg: TestEmail): Promise<void> {
    await (await getBackend()).smtp.sendTestEmail(msg)
  }

  async function run(fn: (b: Awaited<ReturnType<typeof getBackend>>) => Promise<void>, failTitle: string) {
    const toast = useToastStore()
    busy.value = true
    try {
      await fn(await getBackend())
    } catch (err) {
      toast.error(failTitle, errorMessage(err))
    } finally {
      busy.value = false
    }
  }

  return { status, config, busy, running, hasError, connection, init, start, stop, toggle, save, defaults, sendTest }
})
