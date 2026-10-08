<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import {
  CheckSquare,
  Copy,
  Download,
  LoaderCircle,
  Mail,
  MailOpen,
  Paperclip,
  RefreshCw,
  Search,
  Star,
  StarOff,
  Trash2,
  X,
} from 'lucide-vue-next'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import ContextMenu, { type MenuItem } from '@/components/ContextMenu.vue'
import EmptyInbox from './EmptyInbox.vue'
import { useCopy } from '@/composables/useCopy'
import { useMailStore } from '@/stores/mail'
import type { MessageSummary } from '@/types'
import { addressFull, addressLabel, formatListTime, initials } from '@/utils/format'

const mail = useMailStore()
const { copy } = useCopy()
const searchInput = ref<HTMLInputElement | null>(null)
const query = ref(mail.search)
const confirmClear = ref(false)
const confirmBulk = ref(false)
const sentinel = ref<HTMLElement | null>(null)
const scroller = ref<HTMLElement | null>(null)
let debounce: ReturnType<typeof setTimeout> | null = null
let observer: IntersectionObserver | null = null

const selecting = computed(() => mail.checked.length > 0)
const allChecked = computed(() => mail.items.length > 0 && mail.checked.length === mail.items.length)
const someChecked = computed(() => selecting.value && !allChecked.value)
const checkedItems = computed(() => mail.items.filter((m) => mail.checkedSet.has(m.id)))
const anyUnreadChecked = computed(() => checkedItems.value.some((m) => !m.isRead))
const allStarredChecked = computed(() => checkedItems.value.length > 0 && checkedItems.value.every((m) => m.isStarred))

watch(query, (q) => {
  if (debounce) clearTimeout(debounce)
  debounce = setTimeout(() => mail.setSearch(q), 200)
})

function clearSearch() {
  query.value = ''
  mail.setSearch('')
}

onMounted(() => {
  observer = new IntersectionObserver(
    (entries) => {
      if (entries.some((e) => e.isIntersecting)) void mail.loadMore()
    },
    { root: scroller.value, rootMargin: '200px' },
  )
  if (sentinel.value) observer.observe(sentinel.value)
})
onBeforeUnmount(() => observer?.disconnect())

// Keep the selected row visible during keyboard navigation.
watch(
  () => mail.selectedId,
  (id) => {
    if (id === null) return
    requestAnimationFrame(() => {
      scroller.value?.querySelector(`[data-id="${id}"]`)?.scrollIntoView({ block: 'nearest' })
    })
  },
)

const palette = ['#6366f1', '#0ea5e9', '#14b8a6', '#22c55e', '#f59e0b', '#ef4444', '#ec4899', '#8b5cf6']
function avatarColor(seed: string): string {
  let h = 0
  for (let i = 0; i < seed.length; i++) h = (h * 31 + seed.charCodeAt(i)) >>> 0
  return palette[h % palette.length] ?? '#6366f1'
}

function onRowClick(m: MessageSummary, e: MouseEvent) {
  if (e.ctrlKey || e.metaKey) return mail.toggleCheck(m.id)
  if (e.shiftKey) return mail.toggleCheck(m.id, true)
  void mail.select(m.id)
}

function toggleAll() {
  if (allChecked.value) mail.clearChecks()
  else mail.checkAll()
}

function bulkDelete() {
  if (mail.checked.length > 1) confirmBulk.value = true
  else void mail.deleteChecked()
}

async function doBulkDelete() {
  confirmBulk.value = false
  await mail.deleteChecked()
}

async function doClear() {
  confirmClear.value = false
  await mail.clearAll()
}

// --- context menu ---
const menu = ref<{ open: boolean; x: number; y: number; items: MenuItem[] }>({ open: false, x: 0, y: 0, items: [] })

function openMenu(m: MessageSummary, e: MouseEvent) {
  const items: MenuItem[] = [
    { label: 'Open', icon: Mail, action: () => mail.select(m.id) },
    m.isRead
      ? { label: 'Mark as unread', icon: MailOpen, shortcut: 'U', action: () => mail.setRead(m.id, false) }
      : { label: 'Mark as read', icon: MailOpen, action: () => mail.setRead(m.id, true) },
    { label: m.isStarred ? 'Unstar' : 'Star', icon: m.isStarred ? StarOff : Star, shortcut: 'S', action: () => mail.toggleStar(m.id) },
    { label: mail.checkedSet.has(m.id) ? 'Deselect' : 'Select', icon: CheckSquare, action: () => mail.toggleCheck(m.id) },
    { divider: true },
    { label: 'Save as .eml', icon: Download, action: () => mail.saveRaw(m.id) },
    { label: 'Copy sender', icon: Copy, action: () => copy(addressFull(m.from)) },
    { label: 'Copy recipients', icon: Copy, action: () => copy((m.to ?? []).map(addressFull).join(', ')) },
    { label: 'Copy subject', icon: Copy, action: () => copy(m.subject) },
    ...(m.messageId ? [{ label: 'Copy Message-ID', icon: Copy, action: () => copy(m.messageId) }] : []),
    { divider: true },
    { label: 'Delete', icon: Trash2, danger: true, shortcut: 'Del', action: () => mail.remove(m.id) },
  ]
  menu.value = { open: true, x: e.clientX, y: e.clientY, items }
}

defineExpose({ focusSearch: () => searchInput.value?.focus(), bulkDelete })
</script>

<template>
  <section class="flex h-full min-h-0 flex-col border-r border-border bg-panel" aria-label="Email list">
    <!-- Toolbar -->
    <div class="flex flex-col gap-2 border-b border-border p-3">
      <div class="relative">
        <Search class="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-faint" />
        <input
          ref="searchInput"
          v-model="query"
          type="search"
          class="input pr-14 pl-8"
          placeholder="Search subject, sender, recipient"
          aria-label="Search emails"
          @keydown.esc="clearSearch"
        />
        <button v-if="query" class="absolute top-1/2 right-2 -translate-y-1/2 text-faint hover:text-fg" aria-label="Clear search" @click="clearSearch">
          <X class="size-3.5" />
        </button>
        <span v-else class="kbd absolute top-1/2 right-2 -translate-y-1/2">Ctrl K</span>
      </div>

      <div class="flex h-7 items-center gap-1">
        <label class="flex size-7 shrink-0 cursor-pointer items-center justify-center rounded-md hover:bg-hover" :title="allChecked ? 'Deselect all' : 'Select all (Ctrl+A)'">
          <input
            type="checkbox"
            class="size-3.5 accent-[var(--lm-accent)]"
            :checked="allChecked"
            :indeterminate="someChecked"
            :disabled="mail.items.length === 0"
            aria-label="Select all emails"
            @change="toggleAll"
          />
        </label>

        <!-- Bulk actions when emails are checked -->
        <template v-if="selecting">
          <span class="mr-1 text-xs font-medium tabular-nums">{{ mail.checked.length }} selected</span>
          <div class="ml-auto flex items-center">
            <button
              class="btn-icon size-7"
              :title="anyUnreadChecked ? 'Mark selected as read' : 'Mark selected as unread'"
              @click="mail.setReadChecked(anyUnreadChecked)"
            >
              <MailOpen class="size-3.5" />
            </button>
            <button class="btn-icon size-7" :title="allStarredChecked ? 'Unstar selected' : 'Star selected'" @click="mail.setStarredChecked(!allStarredChecked)">
              <Star class="size-3.5" :class="allStarredChecked ? 'fill-warning text-warning' : ''" />
            </button>
            <button class="btn h-7 border-danger/30 px-2 text-xs text-danger hover:bg-danger-soft" title="Delete selected (Del)" @click="bulkDelete">
              <Trash2 class="size-3.5" /> Delete
            </button>
            <button class="btn-icon size-7" title="Clear selection (Esc)" @click="mail.clearChecks()">
              <X class="size-3.5" />
            </button>
          </div>
        </template>

        <template v-else>
          <button
            class="h-7 rounded-md px-2.5 text-xs font-medium transition-colors"
            :class="!mail.unreadOnly ? 'bg-active text-fg' : 'text-muted hover:bg-hover'"
            @click="mail.setUnreadOnly(false)"
          >
            All
          </button>
          <button
            class="h-7 rounded-md px-2.5 text-xs font-medium transition-colors"
            :class="mail.unreadOnly ? 'bg-active text-fg' : 'text-muted hover:bg-hover'"
            @click="mail.setUnreadOnly(true)"
          >
            Unread
          </button>
          <span class="ml-2 truncate text-xs text-faint tabular-nums">
            {{ mail.total.toLocaleString() }} {{ mail.total === 1 ? 'email' : 'emails' }}
          </span>
          <div class="ml-auto flex items-center">
            <button class="btn-icon size-7" title="Refresh (R)" aria-label="Refresh" @click="mail.load()">
              <RefreshCw class="size-3.5" />
            </button>
            <button
              class="btn-icon size-7 hover:!text-danger"
              title="Delete all emails"
              aria-label="Delete all emails"
              :disabled="mail.total === 0 && !mail.isFiltered"
              @click="confirmClear = true"
            >
              <Trash2 class="size-3.5" />
            </button>
          </div>
        </template>
      </div>
    </div>

    <!-- List -->
    <div ref="scroller" class="min-h-0 flex-1 overflow-y-auto" role="listbox" aria-label="Emails" aria-multiselectable="true">
      <div v-if="mail.listState === 'loading'" class="space-y-px p-1">
        <div v-for="i in 8" :key="i" class="flex gap-3 rounded-md p-3">
          <div class="size-8 animate-pulse rounded-full bg-panel-2" />
          <div class="flex-1 space-y-2">
            <div class="h-3 w-1/2 animate-pulse rounded bg-panel-2" />
            <div class="h-3 w-4/5 animate-pulse rounded bg-panel-2" />
          </div>
        </div>
      </div>

      <template v-else-if="mail.items.length === 0">
        <div v-if="mail.isFiltered" class="flex flex-col items-center px-6 py-16 text-center">
          <Search class="mb-3 size-6 text-faint" />
          <p class="font-medium">No matching emails</p>
          <p class="mt-1 text-xs text-muted">Try a different search or filter.</p>
        </div>
        <EmptyInbox v-else />
      </template>

      <ul v-else class="p-1">
        <li v-for="m in mail.items" :key="m.id">
          <div
            :data-id="m.id"
            role="option"
            :aria-selected="mail.selectedId === m.id"
            class="group relative flex w-full cursor-default gap-3 rounded-md px-3 py-2.5 text-left transition-colors"
            :class="[
              mail.checkedSet.has(m.id) ? 'bg-accent-soft' : mail.selectedId === m.id ? 'bg-active' : 'hover:bg-hover',
              mail.lastArrivalId === m.id ? 'animate-[lm-arrive_1.2s_ease-out]' : '',
            ]"
            @click="onRowClick(m, $event)"
            @contextmenu.prevent="openMenu(m, $event)"
          >
            <span
              v-if="!m.isRead"
              class="absolute top-1/2 left-0.5 size-1.5 -translate-y-1/2 rounded-full bg-accent"
              aria-label="Unread"
            />

            <!-- Avatar turns into a checkbox on hover / while selecting -->
            <button
              type="button"
              class="relative mt-0.5 size-8 shrink-0 rounded-full"
              :aria-label="mail.checkedSet.has(m.id) ? 'Deselect email' : 'Select email'"
              :aria-pressed="mail.checkedSet.has(m.id)"
              @click.stop="mail.toggleCheck(m.id, $event.shiftKey)"
            >
              <span
                class="absolute inset-0 flex items-center justify-center rounded-full text-[11px] font-semibold text-white transition-opacity"
                :class="selecting || mail.checkedSet.has(m.id) ? 'opacity-0' : 'group-hover:opacity-0'"
                :style="{ background: avatarColor(m.from.address || m.from.name) }"
                aria-hidden="true"
              >
                {{ initials(m.from) }}
              </span>
              <span
                class="absolute inset-0 flex items-center justify-center rounded-full border transition-opacity"
                :class="[
                  selecting || mail.checkedSet.has(m.id) ? 'opacity-100' : 'opacity-0 group-hover:opacity-100',
                  mail.checkedSet.has(m.id) ? 'border-accent bg-accent text-accent-fg' : 'border-border-strong bg-panel',
                ]"
              >
                <svg v-if="mail.checkedSet.has(m.id)" viewBox="0 0 16 16" class="size-3.5" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true">
                  <path d="m3.5 8.5 3 3 6-7" stroke-linecap="round" stroke-linejoin="round" />
                </svg>
              </span>
            </button>

            <span class="min-w-0 flex-1">
              <span class="flex h-5 items-center gap-2">
                <span class="truncate" :class="m.isRead ? 'text-muted' : 'font-semibold text-fg'">
                  {{ addressLabel(m.from) || '(no sender)' }}
                </span>
                <span class="ml-auto shrink-0 text-[11px] text-faint tabular-nums group-hover:hidden">{{ formatListTime(m.receivedAt) }}</span>
                <!-- Hover actions -->
                <span class="ml-auto hidden shrink-0 items-center gap-0.5 group-hover:flex">
                  <button
                    type="button"
                    class="btn-icon size-6"
                    :title="m.isRead ? 'Mark as unread' : 'Mark as read'"
                    @click.stop="mail.setRead(m.id, !m.isRead)"
                  >
                    <MailOpen v-if="!m.isRead" class="size-3.5" />
                    <Mail v-else class="size-3.5" />
                  </button>
                  <button type="button" class="btn-icon size-6" :title="m.isStarred ? 'Unstar' : 'Star'" @click.stop="mail.toggleStar(m.id)">
                    <Star class="size-3.5" :class="m.isStarred ? 'fill-warning text-warning' : ''" />
                  </button>
                  <button type="button" class="btn-icon size-6 hover:!bg-danger-soft hover:!text-danger" title="Delete this email" @click.stop="mail.remove(m.id)">
                    <Trash2 class="size-3.5" />
                  </button>
                </span>
              </span>
              <span class="mt-0.5 flex items-center gap-1.5">
                <span class="truncate" :class="m.isRead ? 'text-fg/80' : 'font-medium text-fg'">
                  {{ m.subject || '(no subject)' }}
                </span>
                <Paperclip v-if="m.attachmentCount > 0" class="size-3 shrink-0 text-faint" aria-label="Has attachments" />
                <Star v-if="m.isStarred" class="size-3 shrink-0 fill-warning text-warning" aria-label="Starred" />
              </span>
              <span class="mt-0.5 line-clamp-1 text-xs text-faint">{{ m.snippet || ' ' }}</span>
            </span>
          </div>
        </li>
      </ul>

      <div ref="sentinel" class="h-px" />
      <div v-if="mail.loadingMore" class="flex justify-center py-3 text-faint">
        <LoaderCircle class="size-4 animate-spin" />
      </div>
    </div>

    <ContextMenu :open="menu.open" :x="menu.x" :y="menu.y" :items="menu.items" @close="menu.open = false" />

    <ConfirmDialog
      :open="confirmBulk"
      :title="`Delete ${mail.checked.length} emails?`"
      message="The selected emails will be permanently deleted."
      confirm-label="Delete"
      danger
      @confirm="doBulkDelete"
      @cancel="confirmBulk = false"
    />
    <ConfirmDialog
      :open="confirmClear"
      title="Delete all emails?"
      :message="`This permanently deletes all ${mail.total.toLocaleString()} captured emails. This cannot be undone.`"
      confirm-label="Delete all"
      danger
      @confirm="doClear"
      @cancel="confirmClear = false"
    />
  </section>
</template>

<style>
@keyframes lm-arrive {
  from {
    background-color: var(--lm-accent-soft);
  }
}
</style>
