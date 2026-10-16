// @vitest-environment happy-dom
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import { createI18n } from 'vue-i18n'
import { createVuetify } from 'vuetify'
import * as components from 'vuetify/components'
import * as directives from 'vuetify/directives'
import enMessages from '../i18n/en'
import viMessages from '../i18n/vi'
import InterventionCases from '../views/InterventionCases.vue'
import { useAuthStore } from '../stores/auth'
import { InterventionHttpError, type InterventionDetail } from '../api/intervention'

const calls = vi.hoisted(() => ({ detail:vi.fn(), assignees:vi.fn(), list:vi.fn(), action:vi.fn() }))
vi.mock('../api/intervention', () => ({
  getInterventionCase: (...args: unknown[]) => calls.detail(...args),
  listInterventionAssignees: (...args: unknown[]) => calls.assignees(...args),
  listInterventionCases: (...args: unknown[]) => calls.list(...args),
  applyInterventionAction: (...args: unknown[]) => calls.action(...args),
  InterventionHttpError: class extends Error { status:number; code:string; constructor(status:number,code:string){super(code);this.status=status;this.code=code} },
}))

function deferred<T>() { let resolve!: (v:T)=>void; const promise=new Promise<T>(r=>{resolve=r}); return {promise,resolve} }
function detail(id:string): InterventionDetail { return {case:{id,tenant_id:'t',job_id:'job',conversation_id:id,version:1,snapshot_id:'s',snapshot_digest:'d',disposition:'PENDING_REVIEW',work_state:'OPEN',reason_code:'',next_action:'',human_assessment:'',human_explanation:'',human_next_action:'',outcome_text:'',data_owner_id:null,assignee_id:null,due_at:null,created_at:'2026-10-15T00:00:00Z',updated_at:'2026-10-15T00:00:00Z'},proposals:[{id:'p',revision:1,run_id:'r',snapshot_id:'s',snapshot_digest:'d',kind:'complaint',explanation:'Original '+id,next_action:'Check',evidence:'[]',provider:'private-provider',model:'private-model',prompt_version:'r101-direct-seed',created_at:'2026-10-15T00:00:00Z'}],human_events:[],source_status:'LOCAL_MATCH_CURRENTNESS_UNVERIFIED'} }

async function mounted(initial:string|null='A', locale:'en'|'vi'='en') {
  const pinia=createPinia(); setActivePinia(pinia)
  const auth=useAuthStore(); auth.tenantPerms={role:'owner',permissions:{}}
  const router=createRouter({history:createMemoryHistory(),routes:[{path:'/:tenantId/intervention-cases/:caseId?',component:InterventionCases}]})
  const i18n=createI18n({legacy:false,locale,messages:{en:enMessages,vi:viMessages}})
  await router.push(initial ? `/t/intervention-cases/${initial}` : '/t/intervention-cases'); await router.isReady()
  const wrapper=mount(InterventionCases,{attachTo:document.body,global:{plugins:[router,pinia,i18n,createVuetify({components,directives})]}})
  return {wrapper,router,auth}
}

describe('R101 route lifetime', () => {
  beforeEach(() => { calls.detail.mockReset(); calls.assignees.mockReset(); calls.list.mockReset(); calls.action.mockReset(); localStorage.setItem('cqa_access_token','synthetic-session') })

  it('does not show a late case A response after navigation to case B', async () => {
    const a=deferred<ReturnType<typeof detail>>(), b=deferred<ReturnType<typeof detail>>()
    calls.detail.mockImplementation((_tenant:string,id:string) => id==='A'?a.promise:b.promise)
    calls.assignees.mockResolvedValue({items:[]})
    const {wrapper,router}=await mounted()
    await flushPromises()
    expect(calls.detail).toHaveBeenCalledWith('t','A','synthetic-session',expect.any(AbortSignal))
    await router.push('/t/intervention-cases/B'); await flushPromises()
    b.resolve(detail('CASE-B')); await flushPromises()
    expect(wrapper.text()).toContain('CASE-B')
    a.resolve(detail('CASE-A')); await flushPromises()
    expect(wrapper.text()).toContain('CASE-B')
    expect(wrapper.text()).not.toContain('CASE-A')
    wrapper.unmount()
  })

  it('clears case data and actions when the session or permission changes', async () => {
    calls.detail.mockResolvedValue(detail('CASE-A')); calls.assignees.mockResolvedValue({items:[]})
    const {wrapper,auth}=await mounted(); await flushPromises()
    expect(wrapper.text()).toContain('CASE-A')
    auth.accessToken=''; await flushPromises()
    expect(wrapper.text()).not.toContain('CASE-A')
    expect(wrapper.text()).not.toContain('Record action')
    auth.accessToken='synthetic-session'; auth.tenantPerms={role:'member',permissions:{intervention_cases:'r'}}; await flushPromises()
    expect(wrapper.text()).toContain('CASE-A')
    expect(wrapper.text()).not.toContain('Record action')
    auth.tenantPerms={role:'member',permissions:{}}; await flushPromises()
    expect(wrapper.text()).not.toContain('CASE-A')
    wrapper.unmount()
  })

  it('disables duplicate submit, ignores a late action after navigation, and does not replay conflict', async () => {
    calls.detail.mockImplementation((_tenant:string,id:string)=>Promise.resolve(detail('CASE-'+id)))
    calls.assignees.mockResolvedValue({items:[]})
    const pending=deferred<{case:ReturnType<typeof detail>['case']}>()
    calls.action.mockReturnValueOnce(pending.promise)
    const {wrapper,router}=await mounted(); await flushPromises()
    const vm=wrapper.vm as unknown as { action:string; reason:string; submit:()=>Promise<void> }
    vm.action='CONFIRM'; vm.reason='Checked synthetic proposal'; await flushPromises()
    const first=vm.submit(); const second=vm.submit(); await flushPromises()
    expect(calls.action).toHaveBeenCalledTimes(1)
    await router.push('/t/intervention-cases/B'); await flushPromises()
    pending.resolve({case:detail('CASE-A').case}); await Promise.all([first,second]); await flushPromises()
    expect(wrapper.text()).toContain('CASE-B')
    expect(wrapper.text()).not.toContain('CASE-A')
    wrapper.unmount()

    calls.action.mockReset()
    calls.action.mockRejectedValueOnce(new InterventionHttpError(409,'case_conflict'))
    const next=await mounted(); await flushPromises()
    const other=next.wrapper.vm as unknown as { action:string; reason:string; submit:()=>Promise<void> }
    other.action='CONFIRM'; other.reason='Checked'; await other.submit(); await flushPromises()
    expect(calls.action).toHaveBeenCalledTimes(1)
    expect(calls.detail).toHaveBeenCalledTimes(4) // A/B first mount, then A and explicit conflict reload
    next.wrapper.unmount()
  })
})

describe('intervention case presentation', () => {
  beforeEach(() => { calls.detail.mockReset(); calls.assignees.mockReset(); calls.list.mockReset(); calls.action.mockReset(); localStorage.setItem('cqa_access_token','synthetic-session') })

  it('keeps wire filter values while showing Vietnamese labels', async () => {
    calls.list.mockResolvedValue({items:[],total:0,next_cursor:''})
    const {wrapper}=await mounted(null,'vi'); await flushPromises()
    const vm=wrapper.vm as unknown as { filters:{title:string;value:string}[] }
    expect(vm.filters.map(item=>item.value)).toEqual(['','OPEN','ASSIGNED','WAITING_ON_DATA','RESOLVED','REJECTED'])
    expect(vm.filters.map(item=>item.title)).toEqual(['Đang xử lý','Chờ duyệt hoặc giao việc','Đã giao xử lý','Chờ bổ sung dữ liệu','Đã ghi kết quả','Đã từ chối'])
    expect(wrapper.text()).not.toContain('WAITING_ON_DATA')
    wrapper.unmount()
  })

  it('shows readable history and a short case reference without technical identifiers', async () => {
    const uuid='12345678-abcd-4def-8abc-1234567890ab'
    const data=detail('internal-case-id')
    data.case.conversation_id=uuid
    data.case.work_state='UNRECOGNIZED_STATE'
    data.case.disposition='PRIVATE_DISPOSITION'
    data.source_status='PRIVATE_SOURCE'
    data.proposals[0].kind='PRIVATE_KIND'
    data.proposals[0].evidence=JSON.stringify([{message_id:'private-message-id',quote:'Khách đã nhận được hàng.'}])
    data.proposals[0].snapshot_digest='private-digest'
    data.proposals[0].provider='private-provider'
    data.proposals[0].model='private-model'
    data.human_events=[{
      id:'private-event-id',case_version:4,action:'PRIVATE_ACTION',reason:'Đã kiểm tra hồ sơ',
      source_status:'PRIVATE_SOURCE',actor_id:'actor-12345678-abcd-4def-8abc-1234567890ab',
      before:JSON.stringify({version:3,disposition:'HUMAN_CONFIRM',work_state:'RESOLVED',human_assessment:'CONFIRM',human_explanation:'Đã kiểm tra',human_next_action:'Theo dõi thêm',assignee_id:'assignee-12345678-abcd-4def-8abc-1234567890ab',data_owner_id:null,due_at:'2026-10-20T08:00:00Z',outcome_text:'Đã hẹn khách đổi sản phẩm'}),
      after:JSON.stringify({version:4,disposition:'HUMAN_CONFIRM',work_state:'ASSIGNED',human_assessment:'CONFIRM',human_explanation:'Đã kiểm tra',human_next_action:'Theo dõi thêm',assignee_id:'assignee-12345678-abcd-4def-8abc-1234567890ab',data_owner_id:null,due_at:'2026-10-21T08:00:00Z',outcome_text:''}),
      created_at:'2026-10-16T08:00:00Z',
    }]
    calls.detail.mockResolvedValue(data); calls.assignees.mockResolvedValue({items:[]})
    const {wrapper}=await mounted('internal-case-id','vi'); await flushPromises()
    const text=wrapper.text()
    expect(text).toContain('Mã ca 12345678')
    expect(text).toContain('Trạng thái khác')
    expect(text).toContain('Đề xuất khác')
    expect(text).toContain('Thao tác khác')
    expect(text).toContain('Kết quả xử lý')
    expect(text).toContain('Đã hẹn khách đổi sản phẩm')
    expect(text).toContain('Chưa có thông tin')
    expect(text).not.toContain(uuid)
    expect(text).not.toContain('private-event-id')
    expect(text).not.toContain('private-message-id')
    expect(text).not.toContain('private-digest')
    expect(text).not.toContain('private-provider')
    expect(text).not.toContain('private-model')
    expect(text).not.toContain('PRIVATE_SOURCE')
    expect(text).not.toContain('PRIVATE_KIND')
    expect(text).not.toContain('PRIVATE_ACTION')
    expect(text).not.toContain('M2')
    expect(text).not.toContain('{"version"')
    wrapper.unmount()
  })

  it('uses a safe localized message for an unrecognized load failure', async () => {
    calls.detail.mockRejectedValue(new InterventionHttpError(500,'database_connection_failed'))
    const {wrapper}=await mounted('CASE-A','vi'); await flushPromises()
    expect(wrapper.text()).toContain('Không thể tải thông tin lúc này. Hãy thử lại sau.')
    expect(wrapper.text()).not.toContain('database_connection_failed')
    wrapper.unmount()
  })

  it('clears stale actions when a saved action cannot be refreshed and never replays the write', async () => {
    calls.detail.mockResolvedValueOnce(detail('CASE-A')).mockRejectedValueOnce(new InterventionHttpError(500,'private_refresh_failure'))
    calls.assignees.mockResolvedValue({items:[]})
    calls.action.mockResolvedValue({case:detail('CASE-A').case})
    const {wrapper}=await mounted('CASE-A','vi'); await flushPromises()
    const vm=wrapper.vm as unknown as { action:string; reason:string; submit:()=>Promise<void> }
    vm.action='CONFIRM'; vm.reason='Đã kiểm tra đề xuất'; await flushPromises()
    await vm.submit(); await flushPromises()
    expect(calls.action).toHaveBeenCalledTimes(1)
    expect(calls.detail).toHaveBeenCalledTimes(2)
    expect(wrapper.text()).toContain('Thao tác đã được ghi nhận, nhưng chưa thể tải lại thông tin ca.')
    expect(wrapper.text()).not.toContain('private_refresh_failure')
    expect(wrapper.text()).not.toContain('Ghi nhận thao tác')
    expect(wrapper.text()).toContain('Tải lại ca')
    wrapper.unmount()
  })

  it('shows a safe message for an unrecognized write failure without retrying it', async () => {
    calls.detail.mockResolvedValue(detail('CASE-A')); calls.assignees.mockResolvedValue({items:[]})
    calls.action.mockRejectedValueOnce(new InterventionHttpError(500,'private_write_failure'))
    const {wrapper}=await mounted('CASE-A','vi'); await flushPromises()
    const vm=wrapper.vm as unknown as { action:string; reason:string; submit:()=>Promise<void> }
    vm.action='CONFIRM'; vm.reason='Đã kiểm tra đề xuất'; await flushPromises()
    await vm.submit(); await flushPromises()
    expect(calls.action).toHaveBeenCalledTimes(1)
    expect(calls.detail).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('Không thể lưu thao tác lúc này. Hãy thử lại sau.')
    expect(wrapper.text()).not.toContain('private_write_failure')
    wrapper.unmount()
  })
})
