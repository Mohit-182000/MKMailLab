<script setup lang="ts">
import { computed, ref } from 'vue'
import { Eye, EyeOff, KeyRound } from 'lucide-vue-next'
import CopyButton from '@/components/CopyButton.vue'
import { useSmtpStore } from '@/stores/smtp'
import { AuthMode } from '@/types'

const smtp = useSmtpStore()
const showPassword = ref(false)

const c = computed(() => smtp.connection)
const mode = computed(() => smtp.config?.authMode)

const rows = computed(() => {
  const info = c.value
  if (!info) return []
  const anyCreds = mode.value === AuthMode.AuthAny
  return [
    { key: 'host', label: 'SMTP host', value: info.host, copy: info.host },
    { key: 'port', label: 'SMTP port', value: String(info.port), copy: String(info.port) },
    {
      key: 'username',
      label: 'Username',
      value: info.authRequired ? info.username : anyCreds ? 'Any value, or leave empty' : 'Leave empty',
      copy: info.authRequired ? info.username : '',
      hint: !info.authRequired,
    },
    {
      key: 'password',
      label: 'Password',
      value: info.authRequired ? info.password : anyCreds ? 'Any value, or leave empty' : 'Leave empty',
      copy: info.authRequired ? info.password : '',
      hint: !info.authRequired,
      secret: info.authRequired,
    },
    { key: 'encryption', label: 'Encryption', value: 'None (plain SMTP, no TLS)', copy: '', hint: true },
  ]
})

const envBlock = computed(() => {
  const info = c.value
  if (!info) return ''
  return `SMTP_HOST=${info.host}\nSMTP_PORT=${info.port}\nSMTP_USERNAME=${info.username}\nSMTP_PASSWORD=${info.password}\nSMTP_ENCRYPTION=none`
})

const authSummary = computed(() => {
  switch (mode.value) {
    case AuthMode.AuthRequired:
      return 'Authentication is required. Use exactly these credentials.'
    case AuthMode.AuthNone:
      return 'Authentication is disabled. Do not send credentials.'
    default:
      return 'Authentication is optional. Any username and password are accepted.'
  }
})
</script>

<template>
  <section class="card">
    <header class="flex items-center justify-between border-b border-border px-5 py-3.5">
      <div>
        <h2 class="text-sm font-semibold">Connection details</h2>
        <p class="mt-0.5 text-xs text-muted">Use these settings in your application's mail configuration.</p>
      </div>
      <CopyButton :text="envBlock" label="Copy all" with-text />
    </header>

    <dl class="divide-y divide-border">
      <div v-for="r in rows" :key="r.key" class="flex h-11 items-center gap-4 px-5">
        <dt class="w-28 shrink-0 text-xs text-muted">{{ r.label }}</dt>
        <dd class="flex min-w-0 flex-1 items-center gap-2">
          <span v-if="r.hint" class="truncate text-xs text-faint italic">{{ r.value }}</span>
          <span v-else class="truncate font-mono text-[13px] select-text">
            {{ r.secret && !showPassword ? '•'.repeat(Math.max(r.value.length, 8)) : r.value }}
          </span>
          <button
            v-if="r.secret"
            class="btn-icon size-7"
            :title="showPassword ? 'Hide password' : 'Show password'"
            @click="showPassword = !showPassword"
          >
            <EyeOff v-if="showPassword" class="size-3.5" />
            <Eye v-else class="size-3.5" />
          </button>
        </dd>
        <CopyButton v-if="r.copy" :text="r.copy" :label="`Copy ${r.label.toLowerCase()}`" />
      </div>
    </dl>

    <p class="flex items-center gap-2 border-t border-border px-5 py-3 text-xs text-muted">
      <KeyRound class="size-3.5 shrink-0" /> {{ authSummary }}
    </p>
  </section>
</template>
