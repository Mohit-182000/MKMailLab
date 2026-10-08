/** Viewport presets for the email preview. width 0 = fill the panel. */
export interface DevicePreset {
  id: string
  label: string
  group: 'Fit' | 'Desktop' | 'Email client' | 'Tablet' | 'Mobile'
  width: number
}

export const DEVICE_PRESETS: DevicePreset[] = [
  { id: 'fit', label: 'Fit to panel', group: 'Fit', width: 0 },

  { id: 'desktop-4k', label: '4K desktop', group: 'Desktop', width: 2560 },
  { id: 'desktop-fhd', label: 'Full HD desktop', group: 'Desktop', width: 1920 },
  { id: 'desktop', label: 'Desktop', group: 'Desktop', width: 1440 },
  { id: 'laptop', label: 'Laptop', group: 'Desktop', width: 1280 },
  { id: 'laptop-small', label: 'Small laptop', group: 'Desktop', width: 1024 },

  { id: 'client-pane', label: 'Email client reading pane', group: 'Email client', width: 600 },
  { id: 'client-narrow', label: 'Narrow reading pane', group: 'Email client', width: 480 },

  { id: 'ipad-pro-l', label: 'iPad Pro (landscape)', group: 'Tablet', width: 1366 },
  { id: 'ipad-pro', label: 'iPad Pro', group: 'Tablet', width: 1024 },
  { id: 'ipad-air', label: 'iPad Air', group: 'Tablet', width: 820 },
  { id: 'ipad-mini', label: 'iPad mini', group: 'Tablet', width: 768 },
  { id: 'android-tablet', label: 'Android tablet', group: 'Tablet', width: 800 },

  { id: 'iphone-pro-max', label: 'iPhone Pro Max', group: 'Mobile', width: 430 },
  { id: 'pixel', label: 'Pixel 8', group: 'Mobile', width: 412 },
  { id: 'iphone', label: 'iPhone 15 / 16', group: 'Mobile', width: 393 },
  { id: 'iphone-se', label: 'iPhone SE', group: 'Mobile', width: 375 },
  { id: 'galaxy', label: 'Galaxy S', group: 'Mobile', width: 360 },
  { id: 'small-phone', label: 'Small phone', group: 'Mobile', width: 320 },
]

export const MIN_CUSTOM_WIDTH = 240
export const MAX_CUSTOM_WIDTH = 3840

export function clampWidth(w: number): number {
  if (!Number.isFinite(w)) return 0
  return Math.round(Math.min(MAX_CUSTOM_WIDTH, Math.max(MIN_CUSTOM_WIDTH, w)))
}

export interface FrameLayout {
  /** CSS width of the iframe in px (0 = 100%). */
  width: number
  /** CSS height of the iframe in px. */
  height: number
  /** Scale applied so wide viewports fit the panel (≤ 1). */
  scale: number
}

/**
 * Computes iframe size and scale. Viewports wider than the available area are
 * rendered at their true width (so media queries behave correctly) and scaled
 * down visually to fit.
 */
export function frameLayout(target: number, availWidth: number, availHeight: number): FrameLayout {
  if (target <= 0 || availWidth <= 0) return { width: 0, height: Math.max(availHeight, 0), scale: 1 }
  const scale = Math.min(1, availWidth / target)
  return { width: target, height: Math.max(availHeight / scale, 0), scale }
}
