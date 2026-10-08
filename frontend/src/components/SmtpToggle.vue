<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useSmtpStore } from '@/stores/smtp'

const smtp = useSmtpStore()
const router = useRouter()

const label = computed(() => {
  const s = smtp.status
  if (!s) return 'Connecting…'
  if (s.running) return 'SMTP running'
  if (s.error) return 'SMTP error'
  return 'SMTP stopped'
})

const address = computed(() => {
  const s = smtp.status
  const c = smtp.config
  if (s?.running) return `${s.host}:${s.port}`
  if (c) return `${c.host}:${c.port}`
  return ''
})
</script>

<template>
  <div
    class="flex h-8 items-center gap-2.5 rounded-full border pr-1 pl-3 transition-colors"
    :class="smtp.hasError && !smtp.running ? 'border-danger/40 bg-danger-soft' : 'border-border bg-panel'"
  >
    <button
      type="button"
      class="flex items-center gap-2 text-left"
      :title="smtp.status?.error || 'Open SMTP configuration'"
      @click="router.push('/config')"
    >
      <span class="relative flex size-2">
        <span v-if="smtp.running" class="absolute inline-flex size-full animate-ping rounded-full bg-success opacity-50" />
        <span
          class="relative inline-flex size-2 rounded-full"
          :class="smtp.running ? 'bg-success' : smtp.hasError ? 'bg-danger' : 'bg-faint'"
        />
      </span>
      <span class="text-xs font-medium">{{ label }}</span>
      <span v-if="address" class="font-mono text-[11px] text-muted">{{ address }}</span>
    </button>

    <button
      type="button"
      role="switch"
      :aria-checked="smtp.running"
      :aria-label="smtp.running ? 'Stop SMTP server' : 'Start SMTP server'"
      :title="smtp.running ? 'Disconnect (stop SMTP server)' : 'Connect (start SMTP server)'"
      :disabled="smtp.busy || !smtp.status"
      class="relative inline-flex h-6 w-10 shrink-0 items-center rounded-full transition-colors disabled:opacity-60"
      :class="smtp.running ? 'bg-success' : 'bg-border-strong'"
      @click="smtp.toggle()"
    >
      <span
        class="inline-block size-[18px] rounded-full bg-white shadow transition-transform"
        :class="smtp.running ? 'translate-x-[19px]' : 'translate-x-[3px]'"
      />
    </button>
  </div>
</template>
