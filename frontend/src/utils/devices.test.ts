import { describe, expect, it } from 'vitest'
import { DEVICE_PRESETS, clampWidth, frameLayout } from './devices'

describe('device presets', () => {
  it('have unique ids and cover all groups', () => {
    const ids = DEVICE_PRESETS.map((p) => p.id)
    expect(new Set(ids).size).toBe(ids.length)
    for (const g of ['Fit', 'Desktop', 'Email client', 'Tablet', 'Mobile']) {
      expect(DEVICE_PRESETS.some((p) => p.group === g)).toBe(true)
    }
  })
})

describe('frameLayout', () => {
  it('fills the panel in fit mode', () => {
    expect(frameLayout(0, 900, 600)).toEqual({ width: 0, height: 600, scale: 1 })
  })

  it('does not scale narrow viewports', () => {
    expect(frameLayout(390, 900, 600)).toEqual({ width: 390, height: 600, scale: 1 })
  })

  it('scales wide viewports down and compensates height', () => {
    const l = frameLayout(1800, 900, 600)
    expect(l.width).toBe(1800)
    expect(l.scale).toBe(0.5)
    expect(l.height).toBe(1200) // visually 600 after scaling
  })
})

describe('clampWidth', () => {
  it('keeps custom widths in range', () => {
    expect(clampWidth(100)).toBe(240)
    expect(clampWidth(99999)).toBe(3840)
    expect(clampWidth(800.4)).toBe(800)
    expect(clampWidth(Number.NaN)).toBe(0)
  })
})
