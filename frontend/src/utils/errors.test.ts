import { describe, expect, it } from 'vitest'
import { errorMessage } from './errors'

describe('errorMessage', () => {
  it('extracts messages from common shapes', () => {
    expect(errorMessage(new Error('disk full'))).toBe('disk full')
    expect(errorMessage('plain')).toBe('plain')
    expect(errorMessage({ message: 'from wails' })).toBe('from wails')
  })

  it('falls back for unknown values', () => {
    expect(errorMessage(undefined)).toBe('Something went wrong')
    expect(errorMessage({ message: 42 }, 'custom')).toBe('custom')
    expect(errorMessage(new Error(''), 'custom')).toBe('custom')
  })
})
