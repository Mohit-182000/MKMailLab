<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { Send } from 'lucide-vue-next'
import { useSmtpStore } from '@/stores/smtp'
import { useToastStore } from '@/stores/toast'
import type { TestEmail } from '@/types'
import { errorMessage } from '@/utils/errors'

const smtp = useSmtpStore()
const toast = useToastStore()
const router = useRouter()
const sending = ref(false)

const form = ref<TestEmail>({
  from: 'MKMailLab <test@mkmaillab.local>',
  to: 'you@example.com',
  subject: 'Test email from MKMailLab',
  body: '<h2>Hello!</h2>\n<p>If you can read this, your local SMTP server works.</p>',
  isHtml: true,
})

async function send() {
  sending.value = true
  try {
    await smtp.sendTest(form.value)
    toast.success('Test email sent', 'It should appear in your inbox now.')
    void router.push('/inbox')
  } catch (err) {
    toast.error('Test email failed', errorMessage(err))
  } finally {
    sending.value = false
  }
}
</script>

<template>
  <section class="card">
    <header class="border-b border-border px-5 py-3.5">
      <h2 class="text-sm font-semibold">Send a test email</h2>
      <p class="mt-0.5 text-xs text-muted">Delivered through MKMailLab's own SMTP server to verify it works.</p>
    </header>
    <form class="space-y-3 p-5" @submit.prevent="send">
      <div class="grid grid-cols-2 gap-3">
        <div>
          <label class="label" for="t-from">From</label>
          <input id="t-from" v-model="form.from" class="input" />
        </div>
        <div>
          <label class="label" for="t-to">To</label>
          <input id="t-to" v-model="form.to" class="input" placeholder="a@example.com, b@example.com" />
        </div>
      </div>
      <div>
        <label class="label" for="t-subject">Subject</label>
        <input id="t-subject" v-model="form.subject" class="input" />
      </div>
      <div>
        <div class="mb-1 flex items-center justify-between">
          <label class="label mb-0" for="t-body">Message</label>
          <label class="flex cursor-pointer items-center gap-1.5 text-xs text-muted">
            <input v-model="form.isHtml" type="checkbox" class="accent-[var(--lm-accent)]" /> HTML
          </label>
        </div>
        <textarea id="t-body" v-model="form.body" rows="4" class="input h-auto resize-y py-2 font-mono text-xs leading-relaxed" />
      </div>
      <div class="flex items-center gap-3">
        <button type="submit" class="btn btn-primary" :disabled="sending || !smtp.running">
          <Send class="size-3.5" /> {{ sending ? 'Sending…' : 'Send test email' }}
        </button>
        <span v-if="!smtp.running" class="text-xs text-muted">Start the SMTP server first.</span>
      </div>
    </form>
  </section>
</template>
