<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { LoaderCircle, Paperclip, RefreshCw, Search, Star, Trash2, X } from 'lucide-vue-next'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import EmptyInbox from './EmptyInbox.vue'
import { useMailStore } from '@/stores/mail'
import { addressLabel, formatListTime, initials } from '@/utils/format'

const mail = useMailStore()
const searchInput = ref<HTMLInputElement | null>(null)
const query = ref(mail.search)
const confirmClear = ref(false)
const sentinel = ref<HTMLElement | null>(null)
const scroller = ref<HTMLElement | null>(null)
let debounce: ReturnType<typeof setTimeout> | null = null
let observer: IntersectionObserver | null = null

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

async function doClear() {
  confirmClear.value = false
  await mail.clearAll()
}

defineExpose({ focusSearch: () => searchInput.value?.focus() })
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
      <div class="flex items-center gap-1">
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
        <span class="ml-2 text-xs text-faint tabular-nums">
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
      </div>
    </div>

    <!-- List -->
    <div ref="scroller" class="min-h-0 flex-1 overflow-y-auto" role="listbox" aria-label="Emails">
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
          <button
            :data-id="m.id"
            role="option"
            :aria-selected="mail.selectedId === m.id"
            class="group relative flex w-full gap-3 rounded-md px-3 py-2.5 text-left transition-colors"
            :class="[
              mail.selectedId === m.id ? 'bg-active' : 'hover:bg-hover',
              mail.lastArrivalId === m.id ? 'animate-[lm-arrive_1.2s_ease-out]' : '',
            ]"
            @click="mail.select(m.id)"
          >
            <span
              v-if="!m.isRead"
              class="absolute top-1/2 left-0.5 size-1.5 -translate-y-1/2 rounded-full bg-accent"
              aria-label="Unread"
            />
            <span
              class="mt-0.5 flex size-8 shrink-0 items-center justify-center rounded-full text-[11px] font-semibold text-white"
              :style="{ background: avatarColor(m.from.address || m.from.name) }"
              aria-hidden="true"
            >
              {{ initials(m.from) }}
            </span>
            <span class="min-w-0 flex-1">
              <span class="flex items-baseline gap-2">
                <span class="truncate" :class="m.isRead ? 'text-muted' : 'font-semibold text-fg'">
                  {{ addressLabel(m.from) || '(no sender)' }}
                </span>
                <span class="ml-auto shrink-0 text-[11px] text-faint tabular-nums">{{ formatListTime(m.receivedAt) }}</span>
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
          </button>
        </li>
      </ul>

      <div ref="sentinel" class="h-px" />
      <div v-if="mail.loadingMore" class="flex justify-center py-3 text-faint">
        <LoaderCircle class="size-4 animate-spin" />
      </div>
    </div>

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
