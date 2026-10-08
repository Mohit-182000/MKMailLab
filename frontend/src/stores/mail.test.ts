import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { flushPromises } from '@vue/test-utils'
import { setBackend } from '@/services/backend'
import { FakeBackend } from '@/services/fakeBackend'
import { useMailStore } from './mail'

describe('mail store', () => {
  let fake: FakeBackend

  beforeEach(() => {
    setActivePinia(createPinia())
    fake = new FakeBackend()
    setBackend(fake)
    vi.useFakeTimers({ shouldAdvanceTime: true })
  })

  it('loads the first page and paginates', async () => {
    for (let i = 0; i < 120; i++) fake.receive()
    const mail = useMailStore()
    await mail.init()
    expect(mail.items).toHaveLength(50)
    expect(mail.total).toBe(120)
    expect(mail.hasMore).toBe(true)
    await mail.loadMore()
    await mail.loadMore()
    expect(mail.items).toHaveLength(120)
    expect(mail.hasMore).toBe(false)
  })

  it('prepends live arrivals that match the filter', async () => {
    const mail = useMailStore()
    await mail.init()
    expect(mail.items).toHaveLength(0)

    fake.receive({ subject: 'Welcome aboard' })
    expect(mail.items[0]?.subject).toBe('Welcome aboard')
    expect(mail.inboxUnread).toBe(1)
    expect(mail.lastArrivalId).toBe(mail.items[0]?.id)

    mail.setSearch('invoice')
    await flushPromises()
    expect(mail.items).toHaveLength(0)
    fake.receive({ subject: 'Password reset' })
    expect(mail.items).toHaveLength(0) // does not match "invoice"
    fake.receive({ subject: 'Your invoice #42' })
    expect(mail.items.map((m) => m.subject)).toEqual(['Your invoice #42'])
  })

  it('opening marks read and counts refresh', async () => {
    const a = fake.receive()
    fake.receive()
    const mail = useMailStore()
    await mail.init()
    expect(mail.unread).toBe(2)

    await mail.select(a.id)
    expect(mail.selected?.id).toBe(a.id)
    expect(mail.items.find((m) => m.id === a.id)?.isRead).toBe(true)

    await vi.advanceTimersByTimeAsync(200)
    await flushPromises()
    expect(mail.unread).toBe(1)
    expect(mail.inboxUnread).toBe(1)
  })

  it('deleting the selected email selects a neighbour', async () => {
    const first = fake.receive()
    const second = fake.receive()
    const mail = useMailStore()
    await mail.init()
    // List is newest first: [second, first]
    await mail.select(second.id)
    await mail.remove(second.id)
    await flushPromises()
    expect(mail.items.map((m) => m.id)).toEqual([first.id])
    expect(mail.selectedId).toBe(first.id)
  })

  it('clear all empties the list and selection', async () => {
    const m = fake.receive()
    const mail = useMailStore()
    await mail.init()
    await mail.select(m.id)
    await mail.clearAll()
    await flushPromises()
    expect(mail.items).toHaveLength(0)
    expect(mail.selectedId).toBeNull()
  })

  it('star changes from events update list and detail', async () => {
    const m = fake.receive()
    const mail = useMailStore()
    await mail.init()
    await mail.select(m.id)
    await mail.toggleStar(m.id)
    expect(mail.items[0]?.isStarred).toBe(true)
    expect(mail.selected?.isStarred).toBe(true)
  })

  it('keyboard navigation moves selection within bounds', async () => {
    fake.receive()
    fake.receive()
    fake.receive()
    const mail = useMailStore()
    await mail.init()
    const ids = mail.items.map((m) => m.id)
    await mail.select(ids[0]!)
    mail.selectAdjacent(1)
    await flushPromises()
    expect(mail.selectedId).toBe(ids[1])
    mail.selectAdjacent(-1)
    mail.selectAdjacent(-1)
    await flushPromises()
    expect(mail.selectedId).toBe(ids[0])
  })
})
