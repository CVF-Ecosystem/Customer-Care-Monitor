import { reactive } from 'vue'
import api from '../api'

// CCMAI-RUNTIME-036: the first-run setup status is only ever "confirmed" by a well-formed answer
// from the server. A failure, timeout or malformed payload is "unavailable" and is never treated
// as configured, so the router cannot silently open the app (or clear a credential) on a guess.
export type SetupState = 'unresolved' | 'unavailable' | 'required' | 'configured'

// Upper bound for the status request alone (the shared client keeps its own 120 s default).
export const SETUP_STATUS_TIMEOUT_MS = 8000

export const setupStatus = reactive<{ state: SetupState; retrying: boolean }>({ state: 'unresolved', retrying: false })

let inflight: Promise<SetupState> | null = null

// Only a non-null object whose needs_setup is a real boolean counts as an answer.
export function parseSetupStatus(data: unknown): boolean | null {
  if (data === null || typeof data !== 'object') return null
  const value = (data as { needs_setup?: unknown }).needs_setup
  return typeof value === 'boolean' ? value : null
}

function settle(next: SetupState): SetupState {
  setupStatus.state = next
  setupStatus.retrying = false
  return next
}

function startRequest(): Promise<SetupState> {
  const controller = new AbortController()
  let watchdog: ReturnType<typeof setTimeout> | undefined
  const deadline = new Promise<'timeout'>((resolve) => {
    watchdog = setTimeout(() => {
      controller.abort()
      resolve('timeout')
    }, SETUP_STATUS_TIMEOUT_MS)
  })
  const request = api
    .get('/setup/status', { timeout: SETUP_STATUS_TIMEOUT_MS, signal: controller.signal })
    .then((res: { data?: unknown } | undefined) => parseSetupStatus(res?.data))
    .catch(() => null)
  const run = Promise.race([request, deadline])
    .then((outcome): SetupState => {
      if (outcome === 'timeout' || outcome === null) return settle('unavailable')
      return settle(outcome ? 'required' : 'configured')
    })
    .catch(() => settle('unavailable'))
    .finally(() => {
      clearTimeout(watchdog)
      inflight = null
    })
  inflight = run
  return run
}

// Used by the router guard. Confirmed answers are cached for the page lifetime; an unavailable
// answer is NOT retried here, only by an explicit retrySetupStatus() from the user.
export function requestSetupStatus(): Promise<SetupState> {
  if (inflight) return inflight
  if (setupStatus.state !== 'unresolved') return Promise.resolve(setupStatus.state)
  return startRequest()
}

// Explicit user retry: allowed only while unavailable; concurrent callers share one request.
export function retrySetupStatus(): Promise<SetupState> {
  if (inflight) return inflight
  if (setupStatus.state !== 'unavailable') return Promise.resolve(setupStatus.state)
  setupStatus.retrying = true
  return startRequest()
}

export function markSetupConfigured() {
  settle('configured')
}
