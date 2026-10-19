export interface InterventionCapability {
  scope: 'synthetic_only'
  generation_configured: boolean
  rerun_configured: boolean
}

// A read-only capability request must not silently turn a failure into "ready".
export async function getInterventionCapability(tenant: string, token: string, signal: AbortSignal): Promise<InterventionCapability> {
  const controller = new AbortController()
  const abort = () => controller.abort()
  signal.addEventListener('abort', abort, { once: true })
  if (signal.aborted) controller.abort()
  const timeout = setTimeout(abort, 15_000)
  try {
    const response = await fetch(`/api/v1/tenants/${encodeURIComponent(tenant)}/intervention-capability`, {
      headers: { Authorization: `Bearer ${token}` }, signal: controller.signal,
      credentials: 'same-origin', redirect: 'error', cache: 'no-store',
    })
    if (!response.ok || response.redirected) throw new Error('capability_unavailable')
    const c: unknown = await response.json()
    if (!c || typeof c !== 'object') throw new Error('capability_invalid')
    const value = c as Record<string, unknown>
    if (value.scope !== 'synthetic_only' || typeof value.generation_configured !== 'boolean' || typeof value.rerun_configured !== 'boolean') throw new Error('capability_invalid')
    return { scope: 'synthetic_only', generation_configured: value.generation_configured, rerun_configured: value.rerun_configured }
  } finally { clearTimeout(timeout); signal.removeEventListener('abort', abort) }
}
