// @vitest-environment happy-dom
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { createVuetify } from 'vuetify'
import * as components from 'vuetify/components'
import * as directives from 'vuetify/directives'
import enMessages from '../i18n/en'
import viMessages from '../i18n/vi'
import InterventionDataFollowup from '../components/InterventionDataFollowup.vue'
import { InterventionHttpError, type InterventionCase, type InterventionDataRevision, type InterventionDetail, type InterventionRerunRequest } from '../api/intervention'

const calls = vi.hoisted(() => ({ revisions:vi.fn(), requests:vi.fn(), capture:vi.fn(), rerun:vi.fn() }))
vi.mock('../api/intervention', () => ({
  listInterventionDataRevisions: (...args: unknown[]) => calls.revisions(...args),
  listInterventionRerunRequests: (...args: unknown[]) => calls.requests(...args),
  createInterventionDataRevision: (...args: unknown[]) => calls.capture(...args),
  createInterventionRerunRequest: (...args: unknown[]) => calls.rerun(...args),
  InterventionHttpError: class extends Error { status:number; code:string; constructor(status:number,code:string){super(code);this.status=status;this.code=code} },
}))

const CASE_ID='a9a9a9a9-a9a9-4a9a-8a9a-a9a9a9a9a9a9'
const REVISION_ID='b8b8b8b8-b8b8-48b8-88b8-b8b8b8b8b8b8'
const REQUEST_ID='c7c7c7c7-c7c7-47c7-87c7-c7c7c7c7c7c7'
const SNAPSHOT_ID='d6d6d6d6-d6d6-46d6-86d6-d6d6d6d6d6d6'
const SNAPSHOT_DIGEST='a'.repeat(64)
const REVISION_DIGEST='b'.repeat(64)
const emptyPage=()=>({items:[],has_more:false,next_cursor:''})

function caseRecord(overrides:Partial<InterventionCase>={}): InterventionCase {
  return {id:CASE_ID,tenant_id:'tenant-one',job_id:'job-one',conversation_id:'conversation-one',version:7,run_id:'run-one',snapshot_id:SNAPSHOT_ID,snapshot_digest:SNAPSHOT_DIGEST,disposition:'WAIT_DATA',work_state:'WAITING_ON_DATA',reason_code:'',next_action:'',human_assessment:'',human_explanation:'',human_next_action:'',data_owner_id:null,assignee_id:null,due_at:null,outcome_text:'',created_at:'2026-10-19T08:00:00Z',updated_at:'2026-10-19T08:00:00Z',...overrides}
}
function revision(overrides:Partial<InterventionDataRevision>={}): InterventionDataRevision {
  return {id:REVISION_ID,case_id:CASE_ID,originating_case_version:7,case_version:8,original_snapshot_id:SNAPSHOT_ID,original_snapshot_digest:SNAPSHOT_DIGEST,digest:REVISION_DIGEST,coverage:'complete',coverage_reasons:[],message_count:2,observation_status:'LOCAL_MATCH_CURRENTNESS_UNVERIFIED',reason:'New information arrived',created_at:'2026-10-19T09:00:00Z',...overrides}
}
function rerunRequest(overrides:Partial<InterventionRerunRequest>={}): InterventionRerunRequest {
  return {id:REQUEST_ID,case_id:CASE_ID,data_revision_id:REVISION_ID,data_revision_digest:REVISION_DIGEST,original_snapshot_id:SNAPSHOT_ID,original_snapshot_digest:SNAPSHOT_DIGEST,reason:'Please review the new information',status:'AWAITING_EXECUTION_AUTHORITY',execution:'NOT_RUN',case_version:9,created_at:'2026-10-19T09:10:00Z',...overrides}
}
function detail(row:InterventionCase): InterventionDetail {
  return {case:row,proposals:[],human_events:[],source_status:'LOCAL_MATCH_CURRENTNESS_UNVERIFIED'}
}
async function mounted(overrides:Partial<{
  role:string; canRead:boolean; canCapture:boolean; canRequest:boolean; parentBusy:boolean; lifetimeKey:string;
  caseRecord:InterventionCase; locale:'en'|'vi'; refreshCase:()=>Promise<InterventionDetail|null>; signal:AbortSignal
}>={}) {
  const {locale='en',...panelProps}=overrides
  const record=panelProps.caseRecord || caseRecord()
  const refresh=panelProps.refreshCase || vi.fn().mockResolvedValue(detail(record))
  const wrapper=mount(InterventionDataFollowup,{props:{tenantId:'tenant-one',token:'session-token',role:'owner',caseRecord:record,canRead:true,canCapture:true,canRequest:true,parentBusy:false,lifetimeKey:'tenant-one|case-one|session-token|owner|rw',signal:new AbortController().signal,refreshCase:refresh,...panelProps},global:{plugins:[createI18n({legacy:false,locale,messages:{en:enMessages,vi:viMessages}}),createVuetify({components,directives})]}})
  await flushPromises()
  return wrapper
}

describe('WAIT_DATA follow-up panel', () => {
  beforeEach(() => {
    calls.revisions.mockReset().mockResolvedValue(emptyPage())
    calls.requests.mockReset().mockResolvedValue(emptyPage())
    calls.capture.mockReset()
    calls.rerun.mockReset()
    vi.stubGlobal('crypto',{randomUUID:()=> 'intent-key-unique-1'})
  })

  it('requires an explicit reload after an unavailable capture and never resubmits automatically', async () => {
    calls.capture.mockRejectedValueOnce(new InterventionHttpError(503,'source_unavailable')).mockResolvedValueOnce({revision:revision(),case_version:8})
    const refresh=vi.fn().mockResolvedValue(detail(caseRecord({version:8})))
    const wrapper=await mounted({refreshCase:refresh})
    const vm=wrapper.vm as unknown as {captureReason:string;capture:()=>Promise<void>}
    vm.captureReason='New local content was synchronized'
    const first=vm.capture(); await flushPromises(); await first
    expect(calls.capture).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('temporarily unavailable')
    expect(wrapper.text()).toContain('Reload and review')
    await vm.capture(); expect(calls.capture).toHaveBeenCalledTimes(1)

    await wrapper.get('button').trigger('click')
    await flushPromises()
    expect(refresh).toHaveBeenCalledTimes(1)
    vm.captureReason='New local content was synchronized'
    const second=vm.capture(); await flushPromises(); await second
    expect(calls.capture).toHaveBeenCalledTimes(2)
    expect(calls.capture.mock.lastCall?.[4]).toEqual({expected_version:7,expected_snapshot_id:SNAPSHOT_ID,expected_snapshot_digest:SNAPSHOT_DIGEST,reason:'New local content was synchronized'})
    expect(wrapper.text()).toContain('Updated information was recorded locally.')
    expect(wrapper.text()).not.toContain('source_unavailable')
    wrapper.unmount()
  })

  it('locks a write after any server 5xx until the user explicitly reloads', async () => {
    calls.capture.mockRejectedValueOnce(new InterventionHttpError(502,'gateway_unavailable'))
    const wrapper=await mounted()
    const vm=wrapper.vm as unknown as {captureReason:string;capture:()=>Promise<void>;reloadRequired:boolean}
    vm.captureReason='New local content was synchronized'
    await vm.capture(); await flushPromises()
    expect(calls.capture).toHaveBeenCalledTimes(1)
    expect(vm.reloadRequired).toBe(true)
    expect(wrapper.text()).toContain('temporarily unavailable')
    expect(wrapper.text()).toContain('Reload and review')
    await vm.capture()
    expect(calls.capture).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('binds one request to the latest complete local revision and shows that it does not start an assessment', async () => {
    calls.revisions.mockResolvedValue({items:[revision()],has_more:false,next_cursor:''})
    calls.requests.mockResolvedValue(emptyPage())
    calls.rerun.mockResolvedValue({request:rerunRequest(),case_version:9,replayed:false})
    const wrapper=await mounted({caseRecord:caseRecord({version:8}),locale:'vi'})
    const vm=wrapper.vm as unknown as {requestReason:string;requestRerun:()=>Promise<void>}
    vm.requestReason='Đánh giá thông tin mới'
    await vm.requestRerun(); await flushPromises()
    expect(calls.rerun).toHaveBeenCalledTimes(1)
    expect(calls.rerun.mock.lastCall?.[4]).toMatchObject({
      expected_version:8,expected_snapshot_id:SNAPSHOT_ID,expected_snapshot_digest:SNAPSHOT_DIGEST,
      data_revision_id:REVISION_ID,data_revision_digest:REVISION_DIGEST,reason:'Đánh giá thông tin mới',request_key:'intent-key-unique-1',
    })
    const text=wrapper.text()
    expect(text).toContain('Lưu yêu cầu không khởi chạy đánh giá AI mới')
    expect(text).toContain('Đang chờ cấp quyền riêng')
    expect(text).toContain('Chưa bắt đầu đánh giá mới')
    for (const raw of ['AWAITING_EXECUTION_AUTHORITY','NOT_RUN',REVISION_ID,REQUEST_ID,REVISION_DIGEST,'intent-key-unique-1']) expect(text).not.toContain(raw)
    wrapper.unmount()
  })

  it('does not treat the last row of a partial first page as latest and blocks on incomplete newest coverage', async () => {
    const old=revision({id:REVISION_ID,created_at:'2026-10-19T08:00:00Z'})
    const latest=revision({id:'e5e5e5e5-e5e5-45e5-85e5-e5e5e5e5e5e5',coverage:'partial',coverage_reasons:['SOURCE_INCOMPLETE'],observation_status:'INCOMPLETE',created_at:'2026-10-19T09:00:00Z'})
    calls.revisions.mockResolvedValueOnce({items:[old],has_more:true,next_cursor:'page-two'}).mockResolvedValueOnce({items:[latest],has_more:false,next_cursor:''})
    const wrapper=await mounted()
    expect(wrapper.text()).toContain('Load the remaining information history')
    expect(wrapper.findAll('button').some(button=>button.text().includes('Save reassessment request'))).toBe(false)
    const more=wrapper.findAll('button').find(button=>button.text().includes('Show more'))
    expect(more).toBeTruthy()
    await more!.trigger('click'); await flushPromises()
    expect(wrapper.text()).toContain('The latest information record is incomplete')
    expect(calls.revisions).toHaveBeenCalledTimes(2)
    expect(wrapper.text()).not.toContain('SOURCE_INCOMPLETE')
    wrapper.unmount()
  })

  it('fails closed for unknown roles, preserves reader history, and hides request controls without jobs write', async () => {
    const unknown=await mounted({role:'contractor',canRead:true,canCapture:true,canRequest:true})
    expect(calls.revisions).not.toHaveBeenCalled()
    expect(unknown.text()).toContain('unavailable for this account')
    expect(unknown.findAll('button').some(button=>button.text().includes('Record updated information'))).toBe(false)
    unknown.unmount()

    calls.revisions.mockResolvedValue({items:[revision()],has_more:false,next_cursor:''})
    const reader=await mounted({role:'member',canCapture:false,canRequest:false})
    expect(reader.text()).toContain('Saved information records')
    expect(reader.findAll('button').some(button=>button.text().includes('Record updated information'))).toBe(false)
    expect(reader.findAll('button').some(button=>button.text().includes('Save reassessment request'))).toBe(false)
    reader.unmount()

    const noJobs=await mounted({role:'member',canCapture:true,canRequest:false})
    expect(noJobs.findAll('button').some(button=>button.text().includes('Record updated information'))).toBe(true)
    expect(noJobs.findAll('button').some(button=>button.text().includes('Save reassessment request'))).toBe(false)
    expect(noJobs.text()).toContain('cannot send a reassessment request')
    noJobs.unmount()
  })

  it('clears a stale page when the case lifetime changes and holds a single write lock', async () => {
    let resolveOld!: (page: any)=>void
    calls.revisions.mockImplementationOnce(()=>new Promise(resolve=>{resolveOld=resolve})).mockResolvedValue({items:[revision({case_id:'f4f4f4f4-f4f4-44f4-84f4-f4f4f4f4f4f4'})],has_more:false,next_cursor:''})
    calls.capture.mockReturnValue(new Promise(resolve=>{setTimeout(()=>resolve({revision:revision(),case_version:8}),20)}))
    const wrapper=await mounted()
    const next=caseRecord({id:'f4f4f4f4-f4f4-44f4-84f4-f4f4f4f4f4f4',conversation_id:'conversation-two'})
    await wrapper.setProps({caseRecord:next,lifetimeKey:'tenant-one|case-two|session-token|owner|rw'})
    await flushPromises()
    resolveOld({items:[revision({reason:'STALE CASE CONTENT'})],has_more:false,next_cursor:''})
    await flushPromises()
    expect(wrapper.text()).not.toContain('STALE CASE CONTENT')
    const vm=wrapper.vm as unknown as {captureReason:string;capture:()=>Promise<void>}
    vm.captureReason='Current case content'
    const first=vm.capture(), second=vm.capture(); await flushPromises(); await Promise.all([first,second])
    expect(calls.capture).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('keeps an explicit reload busy when a stale read finishes after signal replacement', async () => {
    let resolveOld!: (page: ReturnType<typeof emptyPage>)=>void
    let resolveRefresh!: (value:InterventionDetail|null)=>void
    calls.revisions.mockImplementationOnce(()=>new Promise<ReturnType<typeof emptyPage>>(resolve=>{resolveOld=resolve}))
    const oldSignal=new AbortController().signal
    const wrapper=await mounted({signal:oldSignal,refreshCase:()=>new Promise<InterventionDetail|null>(resolve=>{resolveRefresh=resolve})})
    const next=caseRecord({id:'f4f4f4f4-f4f4-44f4-84f4-f4f4f4f4f4f4',conversation_id:'conversation-two'})
    const nextSignal=new AbortController().signal
    await wrapper.setProps({caseRecord:next,lifetimeKey:'tenant-one|case-two|session-token|owner|rw',signal:nextSignal})
    await flushPromises()
    const vm=wrapper.vm as unknown as {historyLoading:boolean;reloadExplicitly:()=>Promise<void>}
    expect(vm.historyLoading).toBe(false)
    const reload=vm.reloadExplicitly()
    await flushPromises()
    expect(vm.historyLoading).toBe(true)
    resolveOld(emptyPage()); await flushPromises()
    expect(vm.historyLoading).toBe(true)
    resolveRefresh(detail(next)); await reload; await flushPromises()
    expect(vm.historyLoading).toBe(false)
    wrapper.unmount()
  })

  it('ignores an old post-write refresh after navigation without releasing the new case write', async () => {
    let finishOldRefresh!: (value:InterventionDetail|null)=>void
    let finishNewCapture!: (value:{revision:InterventionDataRevision;case_version:number})=>void
    const oldRefresh=new Promise<InterventionDetail|null>(resolve=>{finishOldRefresh=resolve})
    const newCapture=new Promise<{revision:InterventionDataRevision;case_version:number}>(resolve=>{finishNewCapture=resolve})
    calls.capture.mockResolvedValueOnce({revision:revision(),case_version:8}).mockReturnValueOnce(newCapture)
    const oldSignal=new AbortController().signal
    const wrapper=await mounted({signal:oldSignal,refreshCase:()=>oldRefresh})
    const vm=wrapper.vm as unknown as {captureReason:string;capture:()=>Promise<void>;writingKind:string;error:string;savedCapture:InterventionDataRevision|null}
    vm.captureReason='Original case update'
    const oldWrite=vm.capture(); await flushPromises()
    const nextCaseId='f4f4f4f4-f4f4-44f4-84f4-f4f4f4f4f4f4'
    const nextRecord=caseRecord({id:nextCaseId,conversation_id:'conversation-two'})
    await wrapper.setProps({caseRecord:nextRecord,lifetimeKey:'tenant-one|case-two|session-token|owner|rw',signal:new AbortController().signal,refreshCase:()=>Promise.resolve(detail(caseRecord({id:nextCaseId,conversation_id:'conversation-two',version:8})))})
    await flushPromises()
    vm.captureReason='New case update'
    const newWrite=vm.capture(); await flushPromises()
    expect(vm.writingKind).toBe('capture')
    finishOldRefresh(detail(caseRecord({version:8}))); await oldWrite; await flushPromises()
    expect(vm.writingKind).toBe('capture')
    expect(vm.savedCapture).toBeNull()
    expect(vm.error).toBe('')
    expect(wrapper.text()).not.toContain('could not be refreshed')
    const newRevision=revision({case_id:nextCaseId,reason:'New case update'})
    finishNewCapture({revision:newRevision,case_version:8}); await newWrite; await flushPromises()
    expect(vm.writingKind).toBe('')
    expect(vm.savedCapture?.case_id).toBe(nextCaseId)
    expect(wrapper.text()).not.toContain('Original case update')
    wrapper.unmount()
  })
})
