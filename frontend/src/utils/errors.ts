/**
 * Converts anything thrown by a backend call into a message suitable for the
 * UI. Wails rejects with Error-like objects whose message is the Go error.
 */
export function errorMessage(err: unknown, fallback = 'Something went wrong'): string {
  if (err instanceof Error && err.message) return err.message
  if (typeof err === 'string' && err) return err
  if (err && typeof err === 'object' && 'message' in err) {
    const msg = (err as { message: unknown }).message
    if (typeof msg === 'string' && msg) return msg
  }
  return fallback
}
