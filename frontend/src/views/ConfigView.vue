<script setup lang="ts">
import { computed } from 'vue'
import { Power } from 'lucide-vue-next'
import ConnectionDetails from '@/components/config/ConnectionDetails.vue'
import FrameworkSnippets from '@/components/config/FrameworkSnippets.vue'
import ServerSettings from '@/components/config/ServerSettings.vue'
import TestEmail from '@/components/config/TestEmail.vue'
import { useAppStore } from '@/stores/app'
import { useSmtpStore } from '@/stores/smtp'

const smtp = useSmtpStore()
const app = useAppStore()

const statusText = computed(() => {
  const s = smtp.status
  if (!s) return 'Checking status…'
  if (s.running) return `Listening on ${s.host}:${s.port} · ${s.received} received this session · ${s.activeSessions} open connection${s.activeSessions === 1 ? '' : 's'}`
  return s.error || 'The server is stopped. Applications cannot send email until it is started.'
})
</script>

<template>
  <div class="h-full overflow-y-auto">
    <div class="mx-auto max-w-6xl space-y-5 p-6">
      <!-- Status banner -->
      <section
        class="card flex items-center gap-4 px-5 py-4"
        :class="smtp.status?.error && !smtp.running ? 'border-danger/40' : ''"
      >
        <div
          class="flex size-10 items-center justify-center rounded-lg"
          :class="smtp.running ? 'bg-success-soft text-success' : smtp.status?.error ? 'bg-danger-soft text-danger' : 'bg-panel-2 text-muted'"
        >
          <Power class="size-5" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="font-semibold">SMTP server {{ smtp.running ? 'running' : smtp.status?.error ? 'failed to start' : 'stopped' }}</p>
          <p class="mt-0.5 truncate text-xs text-muted" :title="statusText">{{ statusText }}</p>
        </div>
        <button
          class="btn"
          :class="smtp.running ? '' : 'btn-primary'"
          :disabled="smtp.busy || !smtp.status"
          @click="smtp.toggle()"
        >
          {{ smtp.running ? 'Disconnect' : 'Connect' }}
        </button>
      </section>

      <div class="grid grid-cols-1 gap-5 xl:grid-cols-2">
        <div class="space-y-5">
          <ConnectionDetails />
          <FrameworkSnippets />
        </div>
        <div class="space-y-5">
          <ServerSettings />
          <TestEmail />
        </div>
      </div>

      <p v-if="app.info" class="pb-2 text-center text-[11px] text-faint select-text">
        MKMailLab {{ app.info.version }} · Data: {{ app.info.dataDir }}
      </p>
    </div>
  </div>
</template>
