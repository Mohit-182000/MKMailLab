<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Inbox, Send } from 'lucide-vue-next'
import CopyButton from '@/components/CopyButton.vue'
import { useSmtpStore } from '@/stores/smtp'
import { useToastStore } from '@/stores/toast'
import { errorMessage } from '@/utils/errors'

const smtp = useSmtpStore()
const toast = useToastStore()
const router = useRouter()
const sending = ref(false)

const address = computed(() => (smtp.connection ? `${smtp.connection.host}:${smtp.connection.port}` : ''))

async function sendTest() {
  sending.value = true
  try {
    await smtp.sendTest({
      from: 'LocalMail <test@localmail.local>',
      to: 'you@example.com',
      subject: 'Hello from LocalMail 👋',
      body: '<h2 style="font-family:sans-serif">It works!</h2><p style="font-family:sans-serif">This test email was delivered through your local SMTP server.</p>',
      isHtml: true,
    })
  } catch (err) {
    toast.error('Test email failed', errorMessage(err))
  } finally {
    sending.value = false
  }
}
</script>

<template>
  <div class="flex flex-col items-center px-6 py-14 text-center">
    <div class="mb-4 flex size-12 items-center justify-center rounded-xl bg-accent-soft text-accent">
      <Inbox class="size-6" />
    </div>
    <p class="text-[14px] font-semibold">No emails yet</p>
    <p class="mt-1 max-w-64 text-xs text-muted">
      {{ smtp.running ? 'Point your application at the SMTP server below and send an email.' : 'Start the SMTP server, then send an email from your application.' }}
    </p>

    <div v-if="address" class="mt-5 flex items-center gap-1 rounded-md border border-border bg-panel-2 py-1 pr-1 pl-3">
      <span class="font-mono text-xs select-text">{{ address }}</span>
      <CopyButton :text="address" label="Copy address" />
    </div>

    <div class="mt-5 flex gap-2">
      <button v-if="!smtp.running" class="btn btn-primary" :disabled="smtp.busy" @click="smtp.start()">Start SMTP server</button>
      <button v-else class="btn" :disabled="sending" @click="sendTest">
        <Send class="size-3.5" /> {{ sending ? 'Sending…' : 'Send test email' }}
      </button>
      <button class="btn btn-ghost" @click="router.push('/config')">Setup guide</button>
    </div>
  </div>
</template>
