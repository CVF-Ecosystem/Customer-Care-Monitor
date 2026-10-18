// @vitest-environment happy-dom
import { afterEach, describe, expect, it, vi } from 'vitest'
import { applyInterventionAction, InterventionHttpError } from '../api/intervention'

describe('R101 human action transport', () => {
  afterEach(() => { vi.unstubAllGlobals(); vi.useRealTimers() })

  it('sends exactly one authenticated write and returns a 401 without refresh or replay', async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({error:'authorization_required'}), {status:401,headers:{'content-type':'application/json'}}))
    vi.stubGlobal('fetch', fetcher)
    const signal = new AbortController().signal
    const action = {action:'CONFIRM',expected_version:1,expected_snapshot_id:'s',expected_snapshot_digest:'d',expected_proposal_id:'p',expected_revision:1,reason:'Reviewed'}
    await expect(applyInterventionAction('tenant','case','old-token',signal,action)).rejects.toMatchObject({status:401,code:'authorization_required'} satisfies Partial<InterventionHttpError>)
    expect(fetcher).toHaveBeenCalledTimes(1)
    expect(fetcher.mock.calls[0][0]).toBe('/api/v1/tenants/tenant/intervention-cases/case/actions')
    expect(fetcher.mock.calls[0][1]).toMatchObject({method:'POST',redirect:'error',headers:{Authorization:'Bearer old-token'}})
    expect(fetcher.mock.calls[0][1].signal).toBeInstanceOf(AbortSignal)
  })

  it('rejects redirect and timeout as uncertain write outcomes without sending again', async () => {
    vi.useFakeTimers()
    const fetcher = vi.fn().mockImplementation((_url:string, init:RequestInit) => new Promise((_resolve,reject) => {
      init.signal?.addEventListener('abort', () => reject(new DOMException('Aborted','AbortError')))
    }))
    vi.stubGlobal('fetch', fetcher)
    const action = {action:'CONFIRM',expected_version:1,expected_snapshot_id:'s',expected_snapshot_digest:'d',expected_proposal_id:'p',expected_revision:1,reason:'Reviewed'}
    const pending=applyInterventionAction('tenant','case','token',new AbortController().signal,action)
    const rejected = expect(pending).rejects.toMatchObject({code:'completion_unknown_reload'})
    await vi.advanceTimersByTimeAsync(20_001)
    await rejected
    expect(fetcher).toHaveBeenCalledTimes(1)
    expect(fetcher.mock.calls[0][1].redirect).toBe('error')
    vi.useRealTimers()
  })
})
