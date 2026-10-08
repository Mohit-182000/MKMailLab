<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { TriangleAlert } from 'lucide-vue-next'
import { useSmtpStore } from '@/stores/smtp'
import { useToastStore } from '@/stores/toast'
import { AuthMode, type SmtpConfig } from '@/types'
import { errorMessage } from '@/utils/errors'

const smtp = useSmtpStore()
const toast = useToastStore()

const form = ref<SmtpConfig | null>(null)
const error = ref<string | null>(null)
const showPassword = ref(false)

const dirty = computed(() => {
  if (!form.value || !smtp.config) return false
  return JSON.stringify(form.value) !== JSON.stringify(smtp.config)
})

// Adopt the saved config whenever it changes, unless the user has unsaved edits.
watch(
  () => smtp.config,
  (c) => {
    if (c && !dirty.value) form.value = { ...c }
  },
  { immediate: true },
)

const exposed = computed(() => {
  const h = form.value?.host.trim()
  return !!h && h !== '127.0.0.1' && h !== 'localhost' && h !== '::1'
})

const authOptions = [
  { value: AuthMode.AuthAny, title: 'Accept any', desc: 'Clients may send any credentials (recommended).' },
  { value: AuthMode.AuthNone, title: 'Disabled', desc: 'AUTH is not offered. Clients must not log in.' },
  { value: AuthMode.AuthRequired, title: 'Required', desc: 'Only the username and password below are accepted.' },
]

async function save() {
  if (!form.value) return
  error.value = null
  try {
    const res = await smtp.save({ ...form.value, port: Number(form.value.port), maxMessageMb: Number(form.value.maxMessageMb), maxConnections: Number(form.value.maxConnections) })
    form.value = { ...res.config }
    if (res.status.error) {
      toast.error('Settings saved, but the server could not start', res.status.error)
    } else {
      toast.success('Settings saved', res.status.running ? `SMTP server restarted on ${res.status.host}:${res.status.port}` : undefined)
    }
    if (res.warning) toast.info('Network access enabled', res.warning)
  } catch (err) {
    error.value = errorMessage(err)
  }
}

function reset() {
  if (smtp.config) form.value = { ...smtp.config }
  error.value = null
}

async function restoreDefaults() {
  form.value = await smtp.defaults()
}
</script>

<template>
  <section v-if="form" class="card">
    <header class="border-b border-border px-5 py-3.5">
      <h2 class="text-sm font-semibold">Server settings</h2>
      <p class="mt-0.5 text-xs text-muted">Saving restarts the SMTP server if it is running.</p>
    </header>

    <form class="space-y-5 p-5" @submit.prevent="save">
      <div class="grid grid-cols-[1fr_120px] gap-3">
        <div>
          <label class="label" for="smtp-host">Listen address</label>
          <input id="smtp-host" v-model="form.host" class="input font-mono" list="host-options" spellcheck="false" />
          <datalist id="host-options">
            <option value="127.0.0.1">This computer only</option>
            <option value="0.0.0.0">All network interfaces</option>
          </datalist>
        </div>
        <div>
          <label class="label" for="smtp-port">Port</label>
          <input id="smtp-port" v-model.number="form.port" type="number" min="1" max="65535" class="input font-mono" />
        </div>
      </div>

      <div v-if="exposed" class="flex gap-2 rounded-md border border-warning/30 bg-warning-soft px-3 py-2 text-xs">
        <TriangleAlert class="mt-px size-3.5 shrink-0 text-warning" />
        <span>Other computers on your network will be able to send email to MKMailLab. Use <code class="font-mono">127.0.0.1</code> unless you need this.</span>
      </div>

      <fieldset>
        <legend class="label">Authentication</legend>
        <div class="grid grid-cols-3 gap-2">
          <label
            v-for="o in authOptions"
            :key="o.value"
            class="cursor-pointer rounded-md border p-3 transition-colors"
            :class="form.authMode === o.value ? 'border-accent bg-accent-soft' : 'border-border hover:bg-hover'"
          >
            <input v-model="form.authMode" type="radio" name="auth" :value="o.value" class="sr-only" />
            <span class="block text-xs font-semibold">{{ o.title }}</span>
            <span class="mt-0.5 block text-[11px] leading-snug text-muted">{{ o.desc }}</span>
          </label>
        </div>
      </fieldset>

      <div v-if="form.authMode === AuthMode.AuthRequired" class="grid grid-cols-2 gap-3">
        <div>
          <label class="label" for="smtp-user">Username</label>
          <input id="smtp-user" v-model="form.username" class="input font-mono" autocomplete="off" spellcheck="false" />
        </div>
        <div>
          <label class="label" for="smtp-pass">Password</label>
          <div class="relative">
            <input
              id="smtp-pass"
              v-model="form.password"
              :type="showPassword ? 'text' : 'password'"
              class="input pr-14 font-mono"
              autocomplete="new-password"
            />
            <button type="button" class="absolute top-1/2 right-2 -translate-y-1/2 text-[11px] text-muted hover:text-fg" @click="showPassword = !showPassword">
              {{ showPassword ? 'Hide' : 'Show' }}
            </button>
          </div>
        </div>
      </div>

      <div class="grid grid-cols-2 gap-3">
        <div>
          <label class="label" for="smtp-size">Maximum message size (MB)</label>
          <input id="smtp-size" v-model.number="form.maxMessageMb" type="number" min="1" max="500" class="input" />
        </div>
        <div>
          <label class="label" for="smtp-conns">Maximum connections</label>
          <input id="smtp-conns" v-model.number="form.maxConnections" type="number" min="1" max="10000" class="input" />
        </div>
      </div>

      <label class="flex cursor-pointer items-center justify-between gap-4 rounded-md border border-border px-3 py-2.5">
        <span>
          <span class="block text-xs font-semibold">Start SMTP server automatically</span>
          <span class="block text-[11px] text-muted">Start listening as soon as MKMailLab opens.</span>
        </span>
        <input v-model="form.autoStart" type="checkbox" class="size-4 accent-[var(--lm-accent)]" />
      </label>

      <p v-if="error" class="rounded-md border border-danger/30 bg-danger-soft px-3 py-2 text-xs text-danger" role="alert">{{ error }}</p>

      <div class="flex items-center gap-2 pt-1">
        <button type="submit" class="btn btn-primary" :disabled="!dirty || smtp.busy">
          {{ smtp.busy ? 'Saving…' : 'Save changes' }}
        </button>
        <button type="button" class="btn btn-ghost" :disabled="!dirty" @click="reset">Discard</button>
        <button type="button" class="btn btn-ghost ml-auto text-muted" @click="restoreDefaults">Restore defaults</button>
      </div>
    </form>
  </section>
</template>
