import { describe, expect, it } from 'vitest'
import { addressLabel, addressList, formatBytes, formatListTime, initials } from './format'

describe('formatBytes', () => {
  it('formats sizes', () => {
    expect(formatBytes(0)).toBe('0 B')
    expect(formatBytes(1023)).toBe('1023 B')
    expect(formatBytes(1536)).toBe('1.5 KB')
    expect(formatBytes(25 * 1024 * 1024)).toBe('25 MB')
    expect(formatBytes(-1)).toBe('—')
  })
})

describe('formatListTime', () => {
  const now = new Date(2026, 9, 8, 15, 0, 0)
  it('uses relative buckets', () => {
    expect(formatListTime(new Date(2026, 9, 8, 9, 5).toISOString(), now)).toMatch(/09|9/)
    expect(formatListTime(new Date(2026, 9, 7, 9, 5).toISOString(), now)).toBe('Yesterday')
    expect(formatListTime(new Date(2026, 0, 3).toISOString(), now)).not.toMatch(/2026/)
    expect(formatListTime(new Date(2024, 0, 3).toISOString(), now)).toMatch(/2024/)
    expect(formatListTime('garbage', now)).toBe('')
  })
})

describe('addresses', () => {
  it('labels and initials', () => {
    expect(addressLabel({ name: 'Jane Doe', address: 'jane@x.com' })).toBe('Jane Doe')
    expect(addressLabel({ name: '', address: 'jane@x.com' })).toBe('jane@x.com')
    expect(addressList([{ name: 'A', address: 'a@x' }, { name: '', address: 'b@x' }])).toBe('A, b@x')
    expect(initials({ name: 'Jane Doe', address: '' })).toBe('JD')
    expect(initials({ name: '', address: 'no-reply@x.com' })).toBe('NR')
    expect(initials(null)).toBe('?')
  })
})
