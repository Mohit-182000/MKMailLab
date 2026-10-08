import { defineStore } from 'pinia'
import { computed, ref, shallowRef } from 'vue'
import { getBackend } from '@/services/backend'
import { ChangeKind, type LoadState, type MailChange, type Message, type MessageSummary } from '@/types'
import { errorMessage } from '@/utils/errors'
import { useToastStore } from './toast'

const PAGE_SIZE = 50

/** Inbox state: paged list, selection, detail, and live updates. */
export const useMailStore = defineStore('mail', () => {
  const items = ref<MessageSummary[]>([])
  const total = ref(0)
  const unread = ref(0)
  const inboxUnread = ref(0) // unfiltered, for the tab badge
  const cursor = ref('')
  const search = ref('')
  const unreadOnly = ref(false)
  const listState = ref<LoadState>('idle')
  const loadingMore = ref(false)

  const selectedId = ref<number | null>(null)
  const selected = shallowRef<Message | null>(null)
  const detailState = ref<LoadState>('idle')
  /** Bumped on every arrival so the UI can play a subtle highlight. */
  const lastArrivalId = ref<number | null>(null)

  const hasMore = computed(() => cursor.value !== '')
  const isFiltered = computed(() => search.value.trim() !== '' || unreadOnly.value)

  let subscribed = false
  let listToken = 0
  let countsTimer: ReturnType<typeof setTimeout> | null = null

  async function init(): Promise<void> {
    if (!subscribed) {
      const backend = await getBackend()
      backend.on('mail:received', onReceived)
      backend.on('mail:changed', onChanged)
      subscribed = true
    }
    await load()
  }

  /** Loads the first page for the current filters. */
  async function load(): Promise<void> {
    const token = ++listToken
    if (items.value.length === 0) listState.value = 'loading'
    try {
      const backend = await getBackend()
      const page = await backend.mail.list({
        search: search.value,
        unreadOnly: unreadOnly.value,
        cursor: '',
        limit: PAGE_SIZE,
      })
      if (token !== listToken) return // a newer request superseded this one
      items.value = page.items ?? []
      total.value = page.total
      unread.value = page.unread
      cursor.value = page.nextCursor
      listState.value = 'ready'
      if (!isFiltered.value) inboxUnread.value = page.unread
      else void refreshCounts()
    } catch (err) {
      if (token !== listToken) return
      listState.value = 'error'
      useToastStore().error('Could not load emails', errorMessage(err))
    }
  }

  async function loadMore(): Promise<void> {
    if (!hasMore.value || loadingMore.value) return
    loadingMore.value = true
    const token = listToken
    try {
      const backend = await getBackend()
      const page = await backend.mail.list({
        search: search.value,
        unreadOnly: unreadOnly.value,
        cursor: cursor.value,
        limit: PAGE_SIZE,
      })
      if (token !== listToken) return
      const known = new Set(items.value.map((m) => m.id))
      items.value.push(...(page.items ?? []).filter((m) => !known.has(m.id)))
      cursor.value = page.nextCursor
      total.value = page.total
      unread.value = page.unread
    } catch (err) {
      useToastStore().error('Could not load more emails', errorMessage(err))
    } finally {
      loadingMore.value = false
    }
  }

  function setSearch(value: string): void {
    if (value === search.value) return
    search.value = value
    void load()
  }

  function setUnreadOnly(value: boolean): void {
    unreadOnly.value = value
    void load()
  }

  async function select(id: number | null): Promise<void> {
    selectedId.value = id
    if (id === null) {
      selected.value = null
      detailState.value = 'idle'
      return
    }
    detailState.value = 'loading'
    try {
      const backend = await getBackend()
      const msg = await backend.mail.open(id)
      if (selectedId.value !== id) return
      selected.value = msg
      detailState.value = msg ? 'ready' : 'error'
      const item = items.value.find((m) => m.id === id)
      if (item) item.isRead = true
    } catch (err) {
      if (selectedId.value !== id) return
      detailState.value = 'error'
      useToastStore().error('Could not open email', errorMessage(err))
    }
  }

  /** Selects the message after/before the current one in the list. */
  function selectAdjacent(step: 1 | -1): void {
    if (items.value.length === 0) return
    const idx = items.value.findIndex((m) => m.id === selectedId.value)
    const next = items.value[Math.min(Math.max(idx + step, 0), items.value.length - 1)]
    if (next && next.id !== selectedId.value) void select(next.id)
  }

  async function setRead(id: number, read: boolean) {
    await act((b) => b.mail.setRead([id], read), 'Could not update email')
  }

  async function toggleStar(id: number) {
    const item = items.value.find((m) => m.id === id)
    const starred = !(item?.isStarred ?? selected.value?.isStarred ?? false)
    await act((b) => b.mail.setStarred([id], starred), 'Could not update email')
  }

  async function remove(id: number) {
    // Move selection to a neighbour before the item disappears.
    if (selectedId.value === id) {
      const idx = items.value.findIndex((m) => m.id === id)
      const neighbour = items.value[idx + 1] ?? items.value[idx - 1]
      void select(neighbour ? neighbour.id : null)
    }
    await act((b) => b.mail.delete([id]), 'Could not delete email')
  }

  async function clearAll() {
    await act((b) => b.mail.deleteAll(), 'Could not delete emails')
  }

  async function saveRaw(id: number) {
    const toast = useToastStore()
    try {
      const path = await (await getBackend()).mail.saveRaw(id)
      if (path) toast.success('Saved .eml file', path)
    } catch (err) {
      toast.error('Could not save email', errorMessage(err))
    }
  }

  async function saveAttachment(messageId: number, attachmentId: number) {
    const toast = useToastStore()
    try {
      const path = await (await getBackend()).mail.saveAttachment(messageId, attachmentId)
      if (path) toast.success('Attachment saved', path)
    } catch (err) {
      toast.error('Could not save attachment', errorMessage(err))
    }
  }

  async function act(fn: (b: Awaited<ReturnType<typeof getBackend>>) => Promise<void>, failTitle: string) {
    try {
      await fn(await getBackend())
    } catch (err) {
      useToastStore().error(failTitle, errorMessage(err))
    }
  }

  // --- live updates ---------------------------------------------------------

  function matchesFilters(m: MessageSummary): boolean {
    if (unreadOnly.value && m.isRead) return false
    const q = search.value.trim().toLowerCase()
    if (!q) return true
    const haystack = [m.subject, m.from.name, m.from.address, ...(m.to ?? []).flatMap((a) => [a.name, a.address])]
      .join(' ')
      .toLowerCase()
    return haystack.includes(q)
  }

  function onReceived(m: MessageSummary): void {
    inboxUnread.value++
    if (!matchesFilters(m) || items.value.some((x) => x.id === m.id)) return
    items.value.unshift(m)
    total.value++
    if (!m.isRead) unread.value++
    lastArrivalId.value = m.id
    if (listState.value !== 'ready') listState.value = 'ready'
  }

  function onChanged(c: MailChange): void {
    const ids = new Set(c.ids ?? [])
    switch (c.kind) {
      case ChangeKind.ChangeRead:
      case ChangeKind.ChangeUnread: {
        const read = c.kind === ChangeKind.ChangeRead
        for (const m of items.value) if (ids.has(m.id)) m.isRead = read
        if (selected.value && ids.has(selected.value.id)) selected.value = { ...selected.value, isRead: read }
        break
      }
      case ChangeKind.ChangeStarred:
      case ChangeKind.ChangeUnstarred: {
        const starred = c.kind === ChangeKind.ChangeStarred
        for (const m of items.value) if (ids.has(m.id)) m.isStarred = starred
        if (selected.value && ids.has(selected.value.id)) selected.value = { ...selected.value, isStarred: starred }
        break
      }
      case ChangeKind.ChangeDeleted:
        items.value = items.value.filter((m) => !ids.has(m.id))
        if (selectedId.value !== null && ids.has(selectedId.value)) void select(null)
        break
      case ChangeKind.ChangeCleared:
        items.value = []
        cursor.value = ''
        void select(null)
        break
    }
    scheduleCountRefresh()
  }

  /** Counts are re-queried (cheap) rather than tracked, so they never drift. */
  function scheduleCountRefresh(): void {
    if (countsTimer) clearTimeout(countsTimer)
    countsTimer = setTimeout(() => void refreshCounts(), 150)
  }

  async function refreshCounts(): Promise<void> {
    try {
      const backend = await getBackend()
      const [filtered, inbox] = await Promise.all([
        backend.mail.list({ search: search.value, unreadOnly: unreadOnly.value, cursor: '', limit: 1 }),
        isFiltered.value ? backend.mail.list({ search: '', unreadOnly: false, cursor: '', limit: 1 }) : null,
      ])
      total.value = filtered.total
      unread.value = filtered.unread
      inboxUnread.value = (inbox ?? filtered).unread
    } catch {
      // Non-critical; the next successful load corrects counts.
    }
  }

  return {
    items, total, unread, inboxUnread, cursor, search, unreadOnly, listState, loadingMore,
    selectedId, selected, detailState, lastArrivalId, hasMore, isFiltered,
    init, load, loadMore, setSearch, setUnreadOnly, select, selectAdjacent,
    setRead, toggleStar, remove, clearAll, saveRaw, saveAttachment,
    // exposed for tests
    onReceived, onChanged,
  }
})
