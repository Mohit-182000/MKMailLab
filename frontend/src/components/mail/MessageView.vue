<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import {
  Download,
  File,
  FileImage,
  FileText,
  LoaderCircle,
  Mail,
  MailOpen,
  Star,
  Trash2,
  TriangleAlert,
} from 'lucide-vue-next'
import CopyButton from '@/components/CopyButton.vue'
import PreviewFrame from './PreviewFrame.vue'
import { contentUrl } from '@/services/backend'
import { useMailStore } from '@/stores/mail'
import { ParseStatus, type Address, type Attachment } from '@/types'
import { addressFull, formatBytes, formatFullDate } from '@/utils/format'

type Tab = 'preview' | 'text' | 'headers' | 'raw' | 'attachments'

const mail = useMailStore()
const tab = ref<Tab>('preview')
const raw = ref('')
const rawState = ref<'idle' | 'loading' | 'ready' | 'error'>('idle')

const m = computed(() => mail.selected)
const attachments = computed(() => m.value?.attachments ?? [])
const fileAttachments = computed(() => attachments.value.filter((a) => !a.inline || !a.contentId))

const tabs = computed<{ id: Tab; label: string; count?: number }[]>(() => [
  { id: 'preview', label: 'Preview' },
  { id: 'text', label: 'Plain text' },
  { id: 'headers', label: 'Headers', count: m.value?.headers?.length },
  { id: 'raw', label: 'Raw' },
  { id: 'attachments', label: 'Attachments', count: attachments.value.length },
])

const recipients = computed(() => {
  const msg = m.value
  if (!msg) return []
  const rows: { label: string; list: Address[] }[] = [{ label: 'To', list: msg.to ?? [] }]
  if (msg.cc?.length) rows.push({ label: 'Cc', list: msg.cc })
  if (msg.bcc?.length) rows.push({ label: 'Bcc', list: msg.bcc })
  if (msg.replyTo?.length) rows.push({ label: 'Reply-To', list: msg.replyTo })
  return rows
})

const headersText = computed(() => (m.value?.headers ?? []).map((h) => `${h.name}: ${h.value}`).join('\n'))

// Reset per-message view state; keep the chosen tab when it still applies.
watch(
  () => m.value?.id,
  () => {
    raw.value = ''
    rawState.value = 'idle'
    if (tab.value === 'attachments' && attachments.value.length === 0) tab.value = 'preview'
    if (tab.value === 'raw') void loadRaw()
  },
)

watch(tab, (t) => {
  if (t === 'raw' && rawState.value === 'idle') void loadRaw()
})

async function loadRaw() {
  const id = m.value?.id
  if (!id) return
  rawState.value = 'loading'
  try {
    const res = await fetch(contentUrl.raw(id))
    if (!res.ok) throw new Error(String(res.status))
    const text = await res.text()
    if (m.value?.id !== id) return
    raw.value = text
    rawState.value = 'ready'
  } catch {
    rawState.value = 'error'
  }
}

function isImage(a: Attachment) {
  return a.contentType.startsWith('image/') && a.contentType !== 'image/svg+xml'
}
</script>

<template>
  <section class="flex h-full min-h-0 flex-col bg-bg" aria-label="Email detail">
    <!-- Nothing selected -->
    <div v-if="mail.selectedId === null" class="flex flex-1 flex-col items-center justify-center text-center text-muted">
      <Mail class="mb-3 size-8 text-faint" />
      <p class="font-medium text-fg">Select an email to inspect it</p>
      <p class="mt-1 text-xs">
        Use <span class="kbd">↑</span> <span class="kbd">↓</span> to move between emails
      </p>
    </div>

    <div v-else-if="mail.detailState === 'loading' && !m" class="flex flex-1 items-center justify-center text-faint">
      <LoaderCircle class="size-5 animate-spin" />
    </div>

    <div v-else-if="!m" class="flex flex-1 items-center justify-center text-muted">This email could not be loaded.</div>

    <template v-else>
      <!-- Header -->
      <header class="border-b border-border bg-panel px-6 pt-5 pb-0">
        <div class="flex items-start gap-4">
          <h1 class="min-w-0 flex-1 text-[17px] leading-snug font-semibold break-words select-text">
            {{ m.subject || '(no subject)' }}
          </h1>
          <div class="flex shrink-0 items-center gap-0.5">
            <button class="btn-icon" :title="m.isStarred ? 'Unstar (S)' : 'Star (S)'" @click="mail.toggleStar(m.id)">
              <Star class="size-4" :class="m.isStarred ? 'fill-warning text-warning' : ''" />
            </button>
            <button class="btn-icon" title="Mark as unread (U)" @click="mail.setRead(m.id, false)">
              <MailOpen class="size-4" />
            </button>
            <button class="btn-icon" title="Save as .eml" @click="mail.saveRaw(m.id)">
              <Download class="size-4" />
            </button>
            <button class="btn-icon hover:!text-danger" title="Delete (Del)" @click="mail.remove(m.id)">
              <Trash2 class="size-4" />
            </button>
          </div>
        </div>

        <dl class="mt-3 grid grid-cols-[72px_1fr] gap-x-3 gap-y-1 text-[12.5px]">
          <dt class="text-muted">From</dt>
          <dd class="min-w-0 truncate select-text">
            <span class="font-medium">{{ m.from.name || m.from.address || '—' }}</span>
            <span v-if="m.from.name" class="ml-1 text-muted">&lt;{{ m.from.address }}&gt;</span>
          </dd>
          <template v-for="row in recipients" :key="row.label">
            <dt class="text-muted">{{ row.label }}</dt>
            <dd class="min-w-0 break-words select-text">
              {{ row.list.map(addressFull).join(', ') || '—' }}
              <span v-if="row.label === 'Bcc'" class="ml-1 text-[11px] text-faint">(from SMTP envelope)</span>
            </dd>
          </template>
          <dt class="text-muted">Received</dt>
          <dd class="text-fg/90">
            {{ formatFullDate(m.receivedAt) }}
            <span class="ml-2 text-muted">· {{ formatBytes(m.size) }}</span>
          </dd>
        </dl>

        <div
          v-if="m.parseStatus !== ParseStatus.ParseOK"
          class="mt-3 flex items-start gap-2 rounded-md border border-warning/30 bg-warning-soft px-3 py-2 text-xs"
        >
          <TriangleAlert class="mt-px size-3.5 shrink-0 text-warning" />
          <div class="select-text">
            <p class="font-medium">
              {{ m.parseStatus === ParseStatus.ParseFailed ? 'This email could not be parsed. Showing what was recoverable.' : 'This email has formatting problems.' }}
            </p>
            <ul class="mt-0.5 list-disc pl-4 text-muted">
              <li v-for="(e, i) in (m.parseErrors ?? []).slice(0, 5)" :key="i">{{ e }}</li>
            </ul>
          </div>
        </div>

        <!-- Tabs -->
        <nav class="mt-4 -mb-px flex gap-1" role="tablist">
          <button
            v-for="t in tabs"
            :key="t.id"
            role="tab"
            :aria-selected="tab === t.id"
            class="relative flex h-9 items-center gap-1.5 px-2.5 text-[12.5px] font-medium transition-colors"
            :class="tab === t.id ? 'text-fg' : 'text-muted hover:text-fg'"
            @click="tab = t.id"
          >
            {{ t.label }}
            <span v-if="t.count" class="rounded bg-panel-2 px-1 text-[10px] text-muted tabular-nums">{{ t.count }}</span>
            <span v-if="tab === t.id" class="absolute inset-x-1.5 bottom-0 h-0.5 rounded-full bg-accent" />
          </button>
        </nav>
      </header>

      <!-- Tab content -->
      <div class="min-h-0 flex-1 overflow-hidden">
        <!-- Preview: sandboxed iframe with no permissions (no scripts, no same-origin, no forms, no popups). -->
        <PreviewFrame v-if="tab === 'preview'" :src="contentUrl.html(m.id)" :message-id="m.id" />

        <div v-else-if="tab === 'text'" class="h-full overflow-auto p-6">
          <pre v-if="m.text" class="font-mono text-[12.5px] leading-relaxed break-words whitespace-pre-wrap select-text">{{ m.text }}</pre>
          <p v-else class="text-muted">This email has no plain-text part.</p>
        </div>

        <div v-else-if="tab === 'headers'" class="h-full overflow-auto p-6">
          <div class="mb-3 flex justify-end">
            <CopyButton :text="headersText" label="Copy headers" with-text />
          </div>
          <table class="w-full text-[12.5px]">
            <tbody>
              <tr v-for="(h, i) in m.headers ?? []" :key="i" class="border-b border-border/60 align-top">
                <td class="w-48 py-1.5 pr-4 font-mono text-xs whitespace-nowrap text-muted select-text">{{ h.name }}</td>
                <td class="py-1.5 font-mono text-xs break-all select-text">{{ h.value }}</td>
              </tr>
            </tbody>
          </table>
        </div>

        <div v-else-if="tab === 'raw'" class="flex h-full flex-col">
          <div class="flex h-9 shrink-0 items-center justify-end gap-1 border-b border-border bg-panel px-3">
            <CopyButton :text="raw" label="Copy source" with-text />
            <button class="btn h-7 px-2 text-xs" @click="mail.saveRaw(m.id)"><Download class="size-3.5" /> Save .eml</button>
          </div>
          <div v-if="rawState === 'loading'" class="flex flex-1 items-center justify-center text-faint">
            <LoaderCircle class="size-5 animate-spin" />
          </div>
          <p v-else-if="rawState === 'error'" class="p-6 text-danger">Could not load the raw source.</p>
          <pre v-else class="min-h-0 flex-1 overflow-auto bg-code p-4 font-mono text-xs leading-relaxed whitespace-pre select-text">{{ raw }}</pre>
        </div>

        <div v-else class="h-full overflow-auto p-6">
          <p v-if="attachments.length === 0" class="text-muted">This email has no attachments.</p>
          <ul v-else class="grid grid-cols-[repeat(auto-fill,minmax(220px,1fr))] gap-3">
            <li v-for="a in attachments" :key="a.id" class="card flex flex-col overflow-hidden">
              <div class="flex h-28 items-center justify-center border-b border-border bg-panel-2">
                <img
                  v-if="isImage(a)"
                  :src="contentUrl.attachment(m.id, a.id)"
                  :alt="a.fileName"
                  class="max-h-full max-w-full object-contain"
                  loading="lazy"
                />
                <FileImage v-else-if="a.contentType.startsWith('image/')" class="size-8 text-faint" />
                <FileText v-else-if="a.contentType.startsWith('text/') || a.contentType === 'application/pdf'" class="size-8 text-faint" />
                <File v-else class="size-8 text-faint" />
              </div>
              <div class="flex items-center gap-2 p-2.5">
                <div class="min-w-0 flex-1">
                  <p class="truncate text-xs font-medium select-text" :title="a.fileName">{{ a.fileName }}</p>
                  <p class="truncate text-[11px] text-muted">
                    {{ a.contentType }} · {{ formatBytes(a.size) }}<span v-if="a.inline"> · inline</span>
                  </p>
                </div>
                <button class="btn-icon size-7" title="Save attachment" @click="mail.saveAttachment(m.id, a.id)">
                  <Download class="size-3.5" />
                </button>
              </div>
            </li>
          </ul>
          <p v-if="fileAttachments.length !== attachments.length" class="mt-4 text-[11px] text-faint">
            Inline images are embedded in the HTML via Content-ID and shown in Preview.
          </p>
        </div>
      </div>
    </template>
  </section>
</template>
