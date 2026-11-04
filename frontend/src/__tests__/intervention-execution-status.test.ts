// @vitest-environment happy-dom
import {afterEach,beforeEach,describe,expect,it,vi} from 'vitest'
import {flushPromises,mount} from '@vue/test-utils'
import {createI18n} from 'vue-i18n'
import {createVuetify} from 'vuetify'
import * as components from 'vuetify/components'
import * as directives from 'vuetify/directives'
import en from '../i18n/en'
import viMessages from '../i18n/vi'
import InterventionExecutionStatus from '../components/InterventionExecutionStatus.vue'
import {executionHoldKey,readExecutionHold} from '../components/interventionExecutionHold'
import type {InterventionCase,InterventionDetail} from '../api/intervention'

const calls=vi.hoisted(()=>({list:vi.fn(),status:vi.fn(),readiness:vi.fn(),execute:vi.fn()}))
vi.mock('../api/intervention',()=>({
  listInterventionRerunRequests:(...args:unknown[])=>calls.list(...args),
  getInterventionExecutionStatus:(...args:unknown[])=>calls.status(...args),
  getInterventionExecutionReadiness:(...args:unknown[])=>calls.readiness(...args),
  executeInterventionRerun:(...args:unknown[])=>calls.execute(...args),
  InterventionHttpError:class extends Error {status:number;code:string;constructor(status:number,code:string){super(code);this.status=status;this.code=code}},
}))
const id=(n:number)=>`00000000-0000-4000-8000-${String(n).padStart(12,'0')}`
const tenant=id(1),caseId=id(2),actor=id(3),request=id(4),grant=id(5)
const caseRecord:InterventionCase={id:caseId,tenant_id:tenant,job_id:id(6),conversation_id:id(7),version:2,run_id:id(8),snapshot_id:id(9),snapshot_digest:'a'.repeat(64),disposition:'',work_state:'OPEN',reason_code:'',next_action:'',human_assessment:'',human_explanation:'',human_next_action:'',data_owner_id:null,assignee_id:null,due_at:null,outcome_text:'',created_at:'2026-10-21T00:00:00Z',updated_at:'2026-10-21T00:00:00Z'}
const detail:InterventionDetail={case:caseRecord,proposals:[],human_events:[],source_status:'LOCAL_MATCH_CURRENTNESS_UNVERIFIED'}
const saved={id:request,case_id:caseId,data_revision_id:id(10),data_revision_digest:'b'.repeat(64),original_snapshot_id:id(11),original_snapshot_digest:'c'.repeat(64),reason:'Saved',status:'AWAITING_EXECUTION_AUTHORITY',execution:'NOT_RUN',case_version:2,created_at:'2026-10-21T00:00:00Z'}
const statusNoRow={request_id:request,request_intent:'AWAITING_EXECUTION_AUTHORITY',execution_present:false,phase:'NOT_STARTED',observed_at:'2026-10-21T00:00:00Z'}
function ready(){const now=new Date();return {state:'READY',request_id:request,server_now:now.toISOString(),expected_case_version:2,expires_at:new Date(now.getTime()+60_000).toISOString(),grant_handle:grant}}
function deferred<T>() {let resolve!:(value:T)=>void,reject!:(reason:unknown)=>void;const promise=new Promise<T>((yes,no)=>{resolve=yes;reject=no});return {promise,resolve,reject}}
function mounted(writer=true,refreshCase=vi.fn().mockResolvedValue(detail)) {
  const i18n=createI18n({legacy:false,locale:'en',messages:{en,vi:viMessages}})
  const vuetify=createVuetify({components,directives})
  return mount(InterventionExecutionStatus,{global:{plugins:[i18n,vuetify]},props:{tenantId:tenant,token:'session',actorId:actor,role:'member',permissionsKey:writer?'rw-jw':'r',caseRecord,canRead:true,canExecute:writer,parentBusy:false,refreshCase}})
}
describe('execution UI source lifetime and one-shot controls',()=>{
  afterEach(()=>vi.useRealTimers())
  beforeEach(()=>{
    vi.restoreAllMocks();localStorage.clear();calls.list.mockReset().mockResolvedValue({items:[saved],has_more:false,next_cursor:''});calls.status.mockReset().mockResolvedValue(statusNoRow);calls.readiness.mockReset().mockImplementation(async()=>ready());calls.execute.mockReset().mockResolvedValue({execution:{id:id(12),state:'PUBLISHED'},replayed:false});Object.defineProperty(window,'confirm',{configurable:true,value:vi.fn(()=>true)})
  })
  it('shows historical request after the case is OPEN and keeps reader controls absent',async()=>{
    const wrapper=mounted(false);await flushPromises()
    expect(wrapper.text()).toContain('Saved request')
    expect(wrapper.text()).toContain('No processing record observed')
    expect(wrapper.text()).not.toContain('Start assessment')
    expect(calls.readiness).not.toHaveBeenCalled()
    wrapper.unmount()
  })
  it('sends no POST if the pre-send marker cannot be saved',async()=>{
    const wrapper=mounted();await flushPromises()
    const original=Storage.prototype.setItem
    const storageSpy=vi.spyOn(Storage.prototype,'setItem').mockImplementation(function(this:Storage,key:string,value:string){if(key.startsWith('ccmai:execution:'))throw new Error('full');return original.call(this,key,value)})
    const button=wrapper.findAll('button').find(item=>item.text().includes('Start assessment'))
    expect(button).toBeTruthy();await button!.trigger('click');await flushPromises()
    expect(calls.execute).not.toHaveBeenCalled()
    storageSpy.mockRestore()
    wrapper.unmount()
  })
  it('keeps an uncertain hold across remount and no-row status, with no resend',async()=>{
    calls.execute.mockRejectedValueOnce(new Error('connection lost'))
    const first=mounted();await flushPromises()
    await first.findAll('button').find(item=>item.text().includes('Start assessment'))!.trigger('click');await flushPromises()
    const key=executionHoldKey(tenant,caseId,request,actor)
    expect(readExecutionHold(localStorage,key)?.state).toBe('UNCERTAIN')
    expect(calls.execute).toHaveBeenCalledTimes(1)
    first.unmount()
    const second=mounted();await flushPromises()
    expect(second.text()).toContain('An operator must reconcile')
    expect(second.text()).not.toContain('Start assessment')
    expect(calls.execute).toHaveBeenCalledTimes(1)
    second.unmount()
  })
  it('uses exactly the two server-supplied POST fields on one deliberate action',async()=>{
    calls.status.mockResolvedValueOnce(statusNoRow).mockResolvedValueOnce({request_id:request,request_intent:'AWAITING_EXECUTION_AUTHORITY',execution_present:true,phase:'PUBLISHED',observed_at:'2026-10-21T00:00:00Z',published_case_version:3,proposal_available:true})
    const wrapper=mounted();await flushPromises()
    const button=wrapper.findAll('button').find(item=>item.text().includes('Start assessment'))!
    expect(button.attributes('disabled')).toBeUndefined()
    await button.trigger('click');await flushPromises()
    expect(window.confirm).toHaveBeenCalledTimes(1)
    expect(calls.execute).toHaveBeenCalledTimes(1)
    expect(calls.execute.mock.calls[0][5]).toEqual({expected_case_version:2,grant_id:grant})
    expect(readExecutionHold(localStorage,executionHoldKey(tenant,caseId,request,actor))?.state).toBe('OBSERVED')
    wrapper.unmount()
  })
  it('drops a late readiness result after actor and permission switch',async()=>{
    const late=deferred<ReturnType<typeof ready>>()
    calls.readiness.mockReturnValueOnce(late.promise)
    const wrapper=mounted();await flushPromises()
    await wrapper.setProps({actorId:id(15),permissionsKey:'r',canExecute:false})
    late.resolve(ready());await flushPromises()
    expect(wrapper.text()).not.toContain('Start assessment')
    expect(calls.execute).not.toHaveBeenCalled()
    wrapper.unmount()
  })
  it('does not send if the server deadline elapses while confirmation is open',async()=>{
    let tick=10_000
    vi.spyOn(performance,'now').mockImplementation(()=>tick)
    Object.defineProperty(window,'confirm',{configurable:true,value:vi.fn(()=>{tick+=120_000;return true})})
    const wrapper=mounted();await flushPromises()
    await wrapper.findAll('button').find(item=>item.text().includes('Start assessment'))!.trigger('click');await flushPromises()
    expect(calls.execute).not.toHaveBeenCalled()
    wrapper.unmount()
  })
  it('locks rapid duplicate clicks to one POST',async()=>{
    const pending=deferred<unknown>()
    calls.execute.mockReturnValueOnce(pending.promise)
    const wrapper=mounted();await flushPromises()
    const button=wrapper.findAll('button').find(item=>item.text().includes('Start assessment'))!
    await button.trigger('click');await flushPromises()
    await button.trigger('click');await flushPromises()
    expect(calls.execute).toHaveBeenCalledTimes(1)
    pending.reject(new Error('lost'));await flushPromises()
    wrapper.unmount()
  })
  it('clears READY after an unknown status refresh',async()=>{
    const wrapper=mounted();await flushPromises()
    expect(wrapper.text()).toContain('Start assessment')
    calls.status.mockRejectedValueOnce(new Error('unreadable'))
    await wrapper.findAll('button').find(item=>item.text().includes('Check progress'))!.trigger('click');await flushPromises()
    expect(wrapper.text()).not.toContain('Start assessment')
    wrapper.unmount()
  })
  it('aborts a stalled status read at the one-minute observation limit and leaves manual refresh usable',async()=>{
    let pollSignal:AbortSignal|undefined
    calls.status.mockResolvedValueOnce(statusNoRow).mockImplementationOnce((...args:unknown[])=>new Promise((_resolve,reject)=>{
      pollSignal=args[4] as AbortSignal
      pollSignal.addEventListener('abort',()=>reject(new DOMException('Aborted','AbortError')),{once:true})
    })).mockResolvedValue(statusNoRow)
    const wrapper=mounted();await flushPromises()
    vi.useFakeTimers({toFake:['setTimeout','clearTimeout','Date','performance']})
    await wrapper.findAll('button').find(item=>item.text().includes('Start assessment'))!.trigger('click');await flushPromises()
    expect(calls.status).toHaveBeenCalledTimes(2)
    expect(pollSignal?.aborted).toBe(false)
    await vi.advanceTimersByTimeAsync(60_000);await flushPromises()
    expect(pollSignal?.aborted).toBe(true)
    expect(calls.status).toHaveBeenCalledTimes(2)
    expect(calls.execute).toHaveBeenCalledTimes(1)
    await wrapper.findAll('button').find(item=>item.text().includes('Check progress'))!.trigger('click');await flushPromises()
    expect(calls.status).toHaveBeenCalledTimes(3)
    expect(calls.execute).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })
})

// R140 (M3-06): only the newest selected read may change presentation. Obsolete promises below
// deliberately ignore abort and settle late; every selection-only case asserts zero POST.
describe('selection-read ownership across saved requests',()=>{
  const reqA=id(20),reqB=id(21)
  const savedA={...saved,id:reqA,created_at:'2026-10-20T00:00:00Z'},savedB={...saved,id:reqB,created_at:'2026-10-21T00:00:00Z'}
  const noRow=(r:string)=>({...statusNoRow,request_id:r})
  const claimed=(r:string)=>({request_id:r,request_intent:'AWAITING_EXECUTION_AUTHORITY',execution_present:true,phase:'CLAIMED',observed_at:'2026-10-21T00:00:00Z'})
  const readyFor=(r:string,ms=60_000)=>{const now=new Date();return {state:'READY',request_id:r,server_now:now.toISOString(),expected_case_version:2,expires_at:new Date(now.getTime()+ms).toISOString(),grant_handle:grant}}
  const stateFor=(r:string,state:string)=>({state,request_id:r,server_now:new Date().toISOString()})
  const TEXT={ready:'Ready for one deliberate assessment request',stale:'The case or available capacity has changed',operator:'An operator must prepare approval',notStarted:'No processing record observed',claimed:'Request accepted; processing has not been confirmed',unavailable:'Progress cannot be checked right now',denied:'You no longer have permission',unknown:'Progress cannot be checked now',hold:'An operator must reconcile',loading:'Loading case information',start:'Start assessment',refresh:'Check progress'}
  type W=ReturnType<typeof mounted>
  const btn=(w:W,text:string)=>w.findAll('button').find(b=>b.text().includes(text))
  const pick=(w:W,which:'A'|'B')=>w.findAll('button').filter(b=>b.text().includes('Saved request'))[which==='A'?0:1]!
  const click=async(b:ReturnType<W['find']>|undefined)=>{expect(b).toBeTruthy();expect(b!.attributes('disabled')).toBeUndefined();await b!.trigger('click');await flushPromises()}
  const holdOf=(r:string)=>readExecutionHold(localStorage,executionHoldKey(tenant,caseId,r,actor))
  const signalOf=(fn:typeof calls.status,i:number)=>fn.mock.calls[i][4] as AbortSignal
  beforeEach(()=>{
    vi.restoreAllMocks();localStorage.clear()
    calls.list.mockReset().mockResolvedValue({items:[savedA,savedB],has_more:false,next_cursor:''})
    calls.status.mockReset().mockImplementation(async(...a:unknown[])=>noRow(a[2] as string))
    calls.readiness.mockReset().mockImplementation(async(...a:unknown[])=>readyFor(a[2] as string))
    calls.execute.mockReset().mockResolvedValue({execution:{id:id(12),state:'PUBLISHED'},replayed:false})
    Object.defineProperty(window,'confirm',{configurable:true,value:vi.fn(()=>true)})
  })
  afterEach(()=>vi.useRealTimers())

  it('S01: a stale rejection for A cannot clear B after the user switched to B',async()=>{
    const oldA=deferred<unknown>();calls.status.mockResolvedValueOnce(noRow(reqB)).mockReturnValueOnce(oldA.promise)
    const w=mounted();await flushPromises()
    expect(w.text()).toContain(TEXT.ready)
    await click(pick(w,'A'));await click(pick(w,'B'))
    expect(w.text()).toContain(TEXT.notStarted);expect(w.text()).toContain(TEXT.ready)
    const readinessBefore=calls.readiness.mock.calls.length
    oldA.reject(new Error('stale A failure'));await flushPromises()
    expect(w.text()).not.toContain(TEXT.unavailable)
    expect(w.text()).toContain(TEXT.notStarted);expect(w.text()).toContain(TEXT.ready)
    expect(btn(w,TEXT.start)?.attributes('disabled')).toBeUndefined()
    expect(calls.readiness.mock.calls.length).toBe(readinessBefore)
    expect(calls.execute).not.toHaveBeenCalled()
    w.unmount()
  })

  it('S02: A-new wins over A-old although the request ids match (A -> B -> A)',async()=>{
    const oldA=deferred<unknown>()
    calls.status.mockResolvedValueOnce(noRow(reqB)).mockReturnValueOnce(oldA.promise)
    calls.readiness.mockResolvedValueOnce(readyFor(reqB)).mockResolvedValueOnce(readyFor(reqB)).mockResolvedValueOnce(stateFor(reqA,'STALE_OR_INELIGIBLE'))
    const w=mounted();await flushPromises()
    await click(pick(w,'A'));await click(pick(w,'B'));await click(pick(w,'A'))
    expect(w.text()).toContain(TEXT.notStarted);expect(w.text()).toContain(TEXT.stale)
    const readinessBefore=calls.readiness.mock.calls.length
    oldA.resolve(claimed(reqA));await flushPromises()
    expect(w.text()).not.toContain(TEXT.claimed)
    expect(w.text()).toContain(TEXT.notStarted);expect(w.text()).toContain(TEXT.stale)
    expect(w.text()).not.toContain(TEXT.unavailable)
    expect(btn(w,TEXT.start)).toBeUndefined()
    expect(calls.readiness.mock.calls.length).toBe(readinessBefore)
    expect(holdOf(reqA)).toBeNull()
    expect(calls.execute).not.toHaveBeenCalled()
    w.unmount()
  })

  it.each([['resolve'],['reject']] as const)('S03: old A readiness %s after A -> B cannot change B readiness, button or error',async(settle)=>{
    const oldReadiness=deferred<unknown>()
    calls.readiness.mockResolvedValueOnce(readyFor(reqB)).mockReturnValueOnce(oldReadiness.promise).mockResolvedValueOnce(stateFor(reqB,'AWAITING_OPERATOR_AUTHORITY'))
    const w=mounted();await flushPromises()
    await click(pick(w,'A'));await click(pick(w,'B'))
    expect(w.text()).toContain(TEXT.operator)
    if(settle==='resolve')oldReadiness.resolve(readyFor(reqA));else oldReadiness.reject(new Error('old readiness'))
    await flushPromises()
    expect(w.text()).toContain(TEXT.operator)
    expect(w.text()).not.toContain(TEXT.unavailable)
    expect(btn(w,TEXT.start)).toBeUndefined()
    expect(calls.execute).not.toHaveBeenCalled()
    w.unmount()
  })

  it.each([['resolve'],['reject']] as const)('S03: old A readiness %s after A -> B -> A cannot change the newer A readiness',async(settle)=>{
    const oldReadiness=deferred<unknown>(),newReadiness=deferred<unknown>()
    calls.readiness.mockResolvedValueOnce(readyFor(reqB)).mockReturnValueOnce(oldReadiness.promise).mockResolvedValueOnce(stateFor(reqB,'AWAITING_OPERATOR_AUTHORITY')).mockReturnValueOnce(newReadiness.promise)
    const w=mounted();await flushPromises()
    await click(pick(w,'A'));await click(pick(w,'B'));await click(pick(w,'A'))
    expect(signalOf(calls.readiness,1).aborted).toBe(true)
    if(settle==='resolve')oldReadiness.resolve(readyFor(reqA));else oldReadiness.reject(new Error('old readiness'))
    await flushPromises()
    expect(btn(w,TEXT.start)).toBeUndefined()
    expect(w.text()).not.toContain(TEXT.ready)
    expect(w.text()).not.toContain(TEXT.unavailable)
    newReadiness.resolve(stateFor(reqA,'STALE_OR_INELIGIBLE'));await flushPromises()
    expect(w.text()).toContain(TEXT.stale)
    expect(w.text()).not.toContain(TEXT.unavailable)
    expect(btn(w,TEXT.start)).toBeUndefined()
    expect(calls.execute).not.toHaveBeenCalled()
    w.unmount()
  })

  it('S03: an obsolete short-lived READY cannot replace the current expiry timer',async()=>{
    const w=mounted();await flushPromises()
    vi.useFakeTimers({toFake:['setTimeout','clearTimeout','Date','performance']})
    const oldReadiness=deferred<unknown>()
    calls.readiness.mockReturnValueOnce(oldReadiness.promise).mockResolvedValueOnce(readyFor(reqB)).mockImplementationOnce(async()=>readyFor(reqA,60_000))
    await click(pick(w,'A'));await click(pick(w,'B'));await click(pick(w,'A'))
    expect(btn(w,TEXT.start)?.attributes('disabled')).toBeUndefined()
    oldReadiness.resolve(readyFor(reqA,3_000));await flushPromises()
    await vi.advanceTimersByTimeAsync(2_500);await flushPromises()
    expect(btn(w,TEXT.start)?.attributes('disabled')).toBeUndefined()
    await vi.advanceTimersByTimeAsync(60_000);await flushPromises()
    expect(btn(w,TEXT.start)?.attributes('disabled')).toBeDefined()
    expect(calls.execute).not.toHaveBeenCalled()
    w.unmount()
  })

  it('S04: a newer refresh of the same request wins over the overlapping selection read and keeps its spinner',async()=>{
    const selectionRead=deferred<unknown>(),refreshRead=deferred<unknown>()
    calls.status.mockResolvedValueOnce(noRow(reqB)).mockReturnValueOnce(selectionRead.promise).mockReturnValueOnce(refreshRead.promise)
    const w=mounted();await flushPromises()
    await click(pick(w,'A'))
    await click(btn(w,TEXT.refresh))
    expect(w.text()).toContain(TEXT.loading)
    const readinessBefore=calls.readiness.mock.calls.length
    selectionRead.resolve(claimed(reqA));await flushPromises()
    expect(w.text()).toContain(TEXT.loading)
    expect(w.text()).not.toContain(TEXT.claimed)
    expect(calls.readiness.mock.calls.length).toBe(readinessBefore)
    refreshRead.resolve(noRow(reqA));await flushPromises()
    expect(w.text()).not.toContain(TEXT.loading)
    expect(w.text()).toContain(TEXT.notStarted);expect(w.text()).toContain(TEXT.ready)
    expect(calls.readiness.mock.calls.length).toBe(readinessBefore+1)
    expect(calls.execute).not.toHaveBeenCalled()
    w.unmount()
  })

  it('S04: a stale selection failure after the newer refresh finished leaves the refreshed state',async()=>{
    const selectionRead=deferred<unknown>()
    calls.status.mockResolvedValueOnce(noRow(reqB)).mockReturnValueOnce(selectionRead.promise)
    const w=mounted();await flushPromises()
    await click(pick(w,'A'));await click(btn(w,TEXT.refresh))
    expect(w.text()).not.toContain(TEXT.loading);expect(w.text()).toContain(TEXT.ready)
    selectionRead.reject(new Error('late'));await flushPromises()
    expect(w.text()).not.toContain(TEXT.unavailable)
    expect(w.text()).toContain(TEXT.notStarted);expect(w.text()).toContain(TEXT.ready)
    expect(calls.execute).not.toHaveBeenCalled()
    w.unmount()
  })

  it('S05: an obsolete status promise that ignores abort causes no mutation and no follow-up readiness request',async()=>{
    const oldA=deferred<unknown>()
    calls.status.mockResolvedValueOnce(noRow(reqB)).mockReturnValueOnce(oldA.promise)
    const w=mounted();await flushPromises()
    await click(pick(w,'A'));await click(pick(w,'B'));await click(pick(w,'A'))
    expect(signalOf(calls.status,1).aborted).toBe(true)
    expect(signalOf(calls.status,3).aborted).toBe(false)
    const statusBefore=calls.status.mock.calls.length,readinessBefore=calls.readiness.mock.calls.length
    oldA.resolve(noRow(reqA));await flushPromises()
    expect(calls.status.mock.calls.length).toBe(statusBefore)
    expect(calls.readiness.mock.calls.length).toBe(readinessBefore)
    expect(w.text()).toContain(TEXT.ready);expect(w.text()).not.toContain(TEXT.unavailable)
    expect(btn(w,TEXT.start)?.attributes('disabled')).toBeUndefined()
    expect(calls.execute).not.toHaveBeenCalled()
    w.unmount()
  })

  it.each([
    ['actor',{actorId:id(15)}],['permission',{permissionsKey:'r',canExecute:false}],['token',{token:'session-2'}],
    ['tenant',{tenantId:id(16)}],['case',{caseRecord:{...caseRecord,version:3}}],
  ] as const)('S06: a %s change while a selection read is pending drops the late result',async(_name,patch)=>{
    const oldA=deferred<unknown>()
    calls.status.mockResolvedValueOnce(noRow(reqB)).mockReturnValueOnce(oldA.promise)
    const w=mounted();await flushPromises()
    await click(pick(w,'A'))
    const pendingSignal=signalOf(calls.status,1)
    await w.setProps(patch as Record<string,unknown>);await flushPromises()
    expect(pendingSignal.aborted).toBe(true)
    const statusBefore=calls.status.mock.calls.length,readinessBefore=calls.readiness.mock.calls.length
    oldA.reject(new Error('late'));await flushPromises()
    expect(w.text()).not.toContain(TEXT.unavailable)
    expect(w.text()).toContain(TEXT.notStarted)
    expect(calls.status.mock.calls.length).toBe(statusBefore);expect(calls.readiness.mock.calls.length).toBe(readinessBefore)
    expect(calls.execute).not.toHaveBeenCalled()
    w.unmount()
  })

  it('S06: unmount disposes the expiry timer and drops a pending read without follow-up requests',async()=>{
    const w=mounted();await flushPromises()
    vi.useFakeTimers({toFake:['setTimeout','clearTimeout','Date','performance']})
    await click(btn(w,TEXT.refresh))
    expect(vi.getTimerCount()).toBe(1)
    const oldA=deferred<unknown>();calls.status.mockReturnValueOnce(oldA.promise)
    await click(pick(w,'A'))
    expect(vi.getTimerCount()).toBe(0)
    const pendingSignal=signalOf(calls.status,calls.status.mock.calls.length-1)
    w.unmount()
    expect(pendingSignal.aborted).toBe(true)
    const readinessBefore=calls.readiness.mock.calls.length
    oldA.resolve(noRow(reqA));await flushPromises()
    expect(calls.readiness.mock.calls.length).toBe(readinessBefore)
    expect(vi.getTimerCount()).toBe(0)
    expect(calls.execute).not.toHaveBeenCalled()
  })

  it('S06: unmount with a current READY timer leaves no owned timer behind',async()=>{
    const w=mounted();await flushPromises()
    vi.useFakeTimers({toFake:['setTimeout','clearTimeout','Date','performance']})
    await click(btn(w,TEXT.refresh))
    expect(vi.getTimerCount()).toBe(1)
    w.unmount()
    expect(vi.getTimerCount()).toBe(0)
  })

  it('S07: current read failures stay visible and localized; current success renders normally',async()=>{
    const {InterventionHttpError}=await import('../api/intervention')
    calls.status.mockResolvedValueOnce(noRow(reqB)).mockRejectedValueOnce(new InterventionHttpError(403,'denied'))
    const w=mounted();await flushPromises()
    await click(pick(w,'A'))
    expect(w.text()).toContain(TEXT.denied);expect(w.text()).toContain(TEXT.unknown)
    await click(pick(w,'B'))
    expect(w.text()).not.toContain(TEXT.denied);expect(w.text()).toContain(TEXT.notStarted);expect(w.text()).toContain(TEXT.ready)
    calls.readiness.mockRejectedValueOnce(new Error('readiness down'))
    await click(pick(w,'A'))
    expect(w.text()).toContain(TEXT.unavailable);expect(btn(w,TEXT.start)).toBeUndefined()
    const i18n=createI18n({legacy:false,locale:'vi',messages:{en,vi:viMessages}})
    expect(String(i18n.global.t('iv_execution_unavailable'))).not.toBe('iv_execution_unavailable')
    expect(calls.execute).not.toHaveBeenCalled()
    w.unmount()
  })

  it('S08: an obsolete observed-execution read cannot rewrite the current durable uncertain hold',async()=>{
    localStorage.setItem(executionHoldKey(tenant,caseId,reqA,actor),JSON.stringify({state:'UNCERTAIN',at:'2026-10-21T00:00:00Z'}))
    const oldA=deferred<unknown>()
    calls.status.mockResolvedValueOnce(noRow(reqB)).mockReturnValueOnce(oldA.promise)
    const w=mounted();await flushPromises()
    expect(w.text()).not.toContain(TEXT.hold)
    await click(pick(w,'A'));expect(w.text()).toContain(TEXT.hold)
    await click(pick(w,'B'));expect(w.text()).not.toContain(TEXT.hold)
    await click(pick(w,'A'));expect(w.text()).toContain(TEXT.hold)
    const readinessBefore=calls.readiness.mock.calls.length
    oldA.resolve(claimed(reqA));await flushPromises()
    expect(holdOf(reqA)?.state).toBe('UNCERTAIN')
    expect(holdOf(reqB)).toBeNull()
    expect(w.text()).toContain(TEXT.hold);expect(w.text()).not.toContain(TEXT.claimed)
    expect(btn(w,TEXT.start)).toBeUndefined()
    expect(calls.readiness.mock.calls.length).toBe(readinessBefore)
    expect(calls.execute).not.toHaveBeenCalled()
    w.unmount()
  })
})
