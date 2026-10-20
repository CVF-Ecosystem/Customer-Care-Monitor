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
