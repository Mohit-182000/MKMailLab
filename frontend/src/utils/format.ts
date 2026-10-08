import type { Address } from '@/types'

/** 1536 → "1.5 KB" */
export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) return '—'
  if (bytes < 1024) return `${bytes} B`
  const units = ['KB', 'MB', 'GB']
  let value = bytes / 1024
  let unit = 0
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024
    unit++
  }
  return `${value < 10 ? value.toFixed(1) : Math.round(value)} ${units[unit]}`
}

/**
 * Compact list timestamp: time for today, "Yesterday", day+month this year,
 * full date otherwise.
 */
export function formatListTime(iso: string, now = new Date()): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  const sameDay = (a: Date, b: Date) =>
    a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate()
  if (sameDay(d, now)) {
    return d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })
  }
  const yesterday = new Date(now)
  yesterday.setDate(now.getDate() - 1)
  if (sameDay(d, yesterday)) return 'Yesterday'
  if (d.getFullYear() === now.getFullYear()) {
    return d.toLocaleDateString(undefined, { day: 'numeric', month: 'short' })
  }
  return d.toLocaleDateString(undefined, { day: 'numeric', month: 'short', year: 'numeric' })
}

/** Full, unambiguous timestamp for detail views. */
export function formatFullDate(iso: string | null | undefined): string {
  if (!iso) return '—'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return '—'
  return d.toLocaleString(undefined, {
    weekday: 'short',
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

/** Display label for an address: name if present, otherwise the address. */
export function addressLabel(a: Address | null | undefined): string {
  if (!a) return ''
  return a.name?.trim() || a.address || ''
}

/** "Name <addr>" or "addr". */
export function addressFull(a: Address): string {
  return a.name ? `${a.name} <${a.address}>` : a.address
}

/** Comma-separated labels for a recipient list. */
export function addressList(list: Address[] | null | undefined): string {
  return (list ?? []).map(addressLabel).filter(Boolean).join(', ')
}

/** Initials for an avatar: "Jane Doe" → "JD", "app@x.com" → "A". */
export function initials(a: Address | null | undefined): string {
  const label = addressLabel(a)
  if (!label) return '?'
  const words = label.replace(/@.*/, '').split(/[\s._-]+/).filter(Boolean)
  const letters = words.length > 1 ? (words[0]?.[0] ?? '') + (words[1]?.[0] ?? '') : (words[0]?.[0] ?? '?')
  return letters.toUpperCase()
}
