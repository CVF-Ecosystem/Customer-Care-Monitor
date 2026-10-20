// @vitest-environment happy-dom
import { afterEach, describe, expect, it, vi } from 'vitest'
import { applyInterventionAction, executeInterventionRerun, getInterventionExecutionReadiness, getInterventionExecutionStatus, InterventionHttpError } from '../api/intervention'

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

describe('R137 execution transport shape',()=>{
  afterEach(()=>vi.unstubAllGlobals())
  it('uses bounded uncached GETs and sends exactly two POST fields once',async()=>{
    const fetcher=vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({state:'PROCESS_DISABLED',request_id:'request',server_now:'2026-10-21T00:00:00Z'}),{status:200}))
      .mockResolvedValueOnce(new Response(JSON.stringify({request_id:'request',request_intent:'AWAITING_EXECUTION_AUTHORITY',execution_present:false,phase:'NOT_STARTED',observed_at:'2026-10-21T00:00:00Z'}),{status:200}))
      .mockResolvedValueOnce(new Response(JSON.stringify({execution:{id:'execution',state:'CLAIMED'},replayed:false}),{status:202}))
    vi.stubGlobal('fetch',fetcher)
    const signal=new AbortController().signal
    await getInterventionExecutionReadiness('tenant','case','request','token',signal)
    await getInterventionExecutionStatus('tenant','case','request','token',signal)
    await executeInterventionRerun('tenant','case','request','token',signal,{expected_case_version:2,grant_id:'opaque'})
    expect(fetcher).toHaveBeenCalledTimes(3)
    expect(fetcher.mock.calls[0][0]).toContain('/request/execution-readiness')
    expect(fetcher.mock.calls[1][0]).toContain('/request/execution-status')
    expect(fetcher.mock.calls[0][1]).toMatchObject({method:'GET',cache:'no-store',redirect:'error'})
    expect(fetcher.mock.calls[1][1]).toMatchObject({method:'GET',cache:'no-store',redirect:'error'})
    expect(fetcher.mock.calls[2][1]).toMatchObject({method:'POST',cache:'no-store',redirect:'error'})
    expect(JSON.parse(fetcher.mock.calls[2][1].body)).toEqual({expected_case_version:2,grant_id:'opaque'})
  })
  it('does not replay an execution POST on 401 or 5xx',async()=>{
    const fetcher=vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({error:'authorization_required'}),{status:401}))
      .mockResolvedValueOnce(new Response(JSON.stringify({error:'source_unavailable'}),{status:503}))
    vi.stubGlobal('fetch',fetcher)
    const input={expected_case_version:2,grant_id:'opaque'},signal=new AbortController().signal
    await expect(executeInterventionRerun('tenant','case','request','token',signal,input)).rejects.toMatchObject({status:401})
    await expect(executeInterventionRerun('tenant','case','request','token',signal,input)).rejects.toMatchObject({status:503})
    expect(fetcher).toHaveBeenCalledTimes(2)
  })
})
