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
import InterventionCases from '../views/InterventionCases.vue'
import { useAuthStore } from '../stores/auth'
import { InterventionHttpError } from '../api/intervention'

const calls = vi.hoisted(() => ({ detail:vi.fn(), assignees:vi.fn(), list:vi.fn(), action:vi.fn() }))
vi.mock('../api/intervention', () => ({
  getInterventionCase: (...args: unknown[]) => calls.detail(...args),
  listInterventionAssignees: (...args: unknown[]) => calls.assignees(...args),
  listInterventionCases: (...args: unknown[]) => calls.list(...args),
  applyInterventionAction: (...args: unknown[]) => calls.action(...args),
  InterventionHttpError: class extends Error { status:number; code:string; constructor(status:number,code:string){super(code);this.status=status;this.code=code} },
}))

function deferred<T>() { let resolve!: (v:T)=>void; const promise=new Promise<T>(r=>{resolve=r}); return {promise,resolve} }
function detail(id:string) { return {case:{id,tenant_id:'t',conversation_id:id,version:1,snapshot_id:'s',snapshot_digest:'d',disposition:'PENDING_REVIEW',work_state:'OPEN',human_assessment:'',human_explanation:'',human_next_action:'',outcome_text:'',data_owner_id:null,assignee_id:null,due_at:null},proposals:[{id:'p',revision:1,run_id:'r',snapshot_id:'s',snapshot_digest:'d',kind:'complaint',explanation:'Original '+id,next_action:'Check',evidence:'[]',provider:'seed',model:'none',prompt_version:'r101-direct-seed',created_at:'2026-10-15T00:00:00Z'}],human_events:[],source_status:'LOCAL_MATCH_CURRENTNESS_UNVERIFIED'} }

async function mounted(initial='A') {
  const pinia=createPinia(); setActivePinia(pinia)
  const auth=useAuthStore(); auth.tenantPerms={role:'owner',permissions:{}}
  const router=createRouter({history:createMemoryHistory(),routes:[{path:'/:tenantId/intervention-cases/:caseId',component:InterventionCases}]})
  const i18n=createI18n({legacy:false,locale:'en',messages:{en:enMessages}})
  await router.push(`/t/intervention-cases/${initial}`); await router.isReady()
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
