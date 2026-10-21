<template>
  <v-card class="mb-4" data-testid="execution-status">
    <v-card-title>{{ t('iv_execution_heading') }}</v-card-title>
    <v-card-text>
      <p v-if="loading">{{ t('iv_loading') }}</p>
      <v-alert v-if="error" type="warning" role="status" class="mb-3">{{ error }}</v-alert>
      <p v-if="truncated">{{ t('iv_execution_more_history') }}</p>
      <p v-if="!loading && !requests.length">{{ t('iv_execution_no_request') }}</p>
      <div v-for="request in requests" :key="request.id" class="mb-2">
        <v-btn size="small" variant="text" :disabled="loading" @click="selectRequest(request.id)">
          {{ t('iv_execution_saved_request') }} · {{ formatDate(request.created_at) }}
        </v-btn>
      </div>
      <template v-if="selected">
        <p>{{ t('iv_execution_saved_intent') }}: {{ t('iv_execution_intent') }}</p>
        <p>{{ t('iv_execution_observed') }}: {{ statusLabel }}</p>
        <p v-if="status?.published_case_version">{{ t('iv_execution_published_version', { version: status.published_case_version }) }}</p>
        <v-alert v-if="hold" type="warning" role="status" class="mb-3">{{ t('iv_execution_hold') }}</v-alert>
        <p v-if="readiness && canExecute && !hold">{{ readinessLabel }}</p>
        <v-btn v-if="canExecute && readiness?.state === 'READY' && !hold" color="primary" :disabled="loading || sending || parentBusy || !readinessAlive" :loading="sending" @click="executeOnce">
          {{ t('iv_execution_start') }}
        </v-btn>
        <v-btn class="ml-2" variant="text" :disabled="loading || sending" @click="refreshSelected">{{ t('iv_execution_refresh') }}</v-btn>
      </template>
    </v-card-text>
  </v-card>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  executeInterventionRerun, getInterventionExecutionReadiness, getInterventionExecutionStatus,
  listInterventionRerunRequests, InterventionHttpError,
  type InterventionCase, type InterventionDetail, type InterventionExecutionReadiness,
  type InterventionExecutionStatus as Status, type InterventionRerunRequest,
} from '../api/intervention'
import { beginExecutionHold, executionHoldKey, readExecutionHold, updateExecutionHold, type ExecutionHold } from './interventionExecutionHold'

const props = defineProps<{
  tenantId: string; token: string; actorId: string; role: string; permissionsKey: string
  caseRecord: InterventionCase; canRead: boolean; canExecute: boolean; parentBusy: boolean
  signal?: AbortSignal; refreshCase: () => Promise<InterventionDetail | null>
}>()
const {t,locale}=useI18n()
const requests=ref<InterventionRerunRequest[]>([])
const selectedId=ref('')
const status=ref<Status|null>(null)
const readiness=ref<InterventionExecutionReadiness|null>(null)
const hold=ref<ExecutionHold|null>(null)
const loading=ref(false), sending=ref(false), truncated=ref(false), error=ref('')
const readinessAlive=ref(false)
const selected=computed(()=>requests.value.find(r=>r.id===selectedId.value)||null)
const lifetime=computed(()=>JSON.stringify([props.tenantId,props.caseRecord.id,props.caseRecord.version,props.token,props.actorId,props.role,props.permissionsKey,props.canRead,props.canExecute]))
const markerKey=computed(()=>selected.value && props.actorId ? executionHoldKey(props.tenantId,props.caseRecord.id,selected.value.id,props.actorId):'')
let generation=0, controller=new AbortController(), detachExternal:(()=>void)|null=null, mounted=true
let expiryTimer:ReturnType<typeof setTimeout>|null=null
function clearReadiness() {readiness.value=null;readinessAlive.value=false;if(expiryTimer){clearTimeout(expiryTimer);expiryTimer=null}}
function deadlineFromServer(value:InterventionExecutionReadiness, requestStarted:number) {
  return requestStarted+Math.max(0,Date.parse(value.expires_at||'')-Date.parse(value.server_now)-1_000)
}
function installReadiness(value:InterventionExecutionReadiness, requestStarted:number) {
  clearReadiness();readiness.value=value
  if(value.state!=='READY')return
  const remaining=deadlineFromServer(value,requestStarted)-performance.now()
  if(remaining<=0)return
  readinessAlive.value=true
  expiryTimer=setTimeout(()=>{readinessAlive.value=false;expiryTimer=null},remaining)
}
function current(mine:number,key:string) {return mounted && mine===generation && key===lifetime.value && !controller.signal.aborted && !props.signal?.aborted}
function resetController() {
  controller.abort(); detachExternal?.(); controller=new AbortController()
  if (props.signal) {
    const outer=props.signal, inner=controller
    const abort=()=>inner.abort()
    if (outer.aborted) abort()
    else {outer.addEventListener('abort',abort,{once:true});detachExternal=()=>outer.removeEventListener('abort',abort)}
  }
}
function safeError(e:unknown) {
  if (e instanceof InterventionHttpError && e.status===401) return t('iv_session')
  if (e instanceof InterventionHttpError && e.status===403) return t('iv_denied')
  if (e instanceof InterventionHttpError && e.status===404) return t('iv_followup_missing')
  return t('iv_execution_unavailable')
}
function loadHold() {
  try {hold.value=markerKey.value ? readExecutionHold(localStorage,markerKey.value):null}
  catch {hold.value={state:'UNCERTAIN',at:''}}
}
function formatDate(s:string) {const d=new Date(s);return Number.isFinite(d.getTime())?d.toLocaleString(locale.value):t('iv_not_set')}
const statusLabel=computed(()=>{
  if (!status.value) return t('iv_execution_unknown')
  if (status.value.phase==='MANUAL_HOLD') return t('iv_execution_manual_hold')
  const labels:Record<string,string>={NOT_STARTED:'iv_execution_not_started',CLAIMED:'iv_execution_claimed',CALL_MAY_HAVE_STARTED:'iv_execution_processing',RESPONSE_RECORDED:'iv_execution_processing',PUBLISHED:'iv_execution_published'}
  return t(labels[status.value.phase]||'iv_execution_unknown')
})
const readinessLabel=computed(()=>{
  const labels:Record<string,string>={READY:'iv_execution_ready',AWAITING_OPERATOR_AUTHORITY:'iv_execution_need_operator',PROCESS_DISABLED:'iv_execution_disabled',STALE_OR_INELIGIBLE:'iv_execution_stale',ALREADY_STARTED:'iv_execution_already_started'}
  return t(labels[readiness.value?.state||'']||'iv_execution_unknown')
})
function validStatus(value:Status, requestId:string):boolean {
  return !!value && value.request_id===requestId && typeof value.request_intent==='string' && typeof value.execution_present==='boolean' && typeof value.observed_at==='string' && Number.isFinite(Date.parse(value.observed_at)) &&
    ['NOT_STARTED','CLAIMED','CALL_MAY_HAVE_STARTED','RESPONSE_RECORDED','PUBLISHED','MANUAL_HOLD'].includes(value.phase) &&
    (value.execution_present || value.phase==='NOT_STARTED') && (value.phase!=='PUBLISHED' || (Number.isSafeInteger(value.published_case_version) && value.proposal_available===true))
}
function validReadiness(value:InterventionExecutionReadiness, requestId:string):boolean {
  return !!value && value.request_id===requestId && typeof value.server_now==='string' && Number.isFinite(Date.parse(value.server_now)) &&
    ['READY','AWAITING_OPERATOR_AUTHORITY','PROCESS_DISABLED','STALE_OR_INELIGIBLE','ALREADY_STARTED'].includes(value.state) &&
    (value.state!=='READY' || (Number.isSafeInteger(value.expected_case_version) && Number(value.expected_case_version)>0 && typeof value.grant_handle==='string' && !!value.grant_handle && typeof value.expires_at==='string' && Date.parse(value.expires_at)>Date.parse(value.server_now)))
}
async function readSelected(mine:number,key:string,includeReadiness=true,readSignal:AbortSignal=controller.signal) {
  const requestId=selectedId.value
  if (!requestId || !current(mine,key) || readSignal.aborted) return
  clearReadiness()
  try {
    const observed=await getInterventionExecutionStatus(props.tenantId,props.caseRecord.id,requestId,props.token,readSignal)
    if (!current(mine,key)||readSignal.aborted||requestId!==selectedId.value) return
    if (!validStatus(observed,requestId)) throw new Error('invalid status')
    status.value=observed
    if (observed.execution_present && markerKey.value) {try {updateExecutionHold(localStorage,markerKey.value,'OBSERVED')} catch {/* keep hold */};loadHold()}
  } catch(e) {if (current(mine,key)&&!readSignal.aborted) {status.value=null;error.value=safeError(e)};return}
  if (!includeReadiness || !props.canExecute || !props.actorId || hold.value || !current(mine,key) || readSignal.aborted) return
  try {
    const requestStarted=performance.now()
    const observed=await getInterventionExecutionReadiness(props.tenantId,props.caseRecord.id,requestId,props.token,readSignal)
    if (!current(mine,key)||readSignal.aborted||requestId!==selectedId.value) return
    if (!validReadiness(observed,requestId)) throw new Error('invalid readiness')
    installReadiness(observed,requestStarted)
  } catch(e) {if (current(mine,key)&&!readSignal.aborted) error.value=safeError(e)}
}
async function load() {
  const mine=++generation,key=lifetime.value
  resetController();requests.value=[];selectedId.value='';status.value=null;clearReadiness();hold.value=null;error.value='';truncated.value=false;loading.value=false;sending.value=false
  if (!props.canRead || !props.tenantId || !props.token) return
  loading.value=true
  try {
    let cursor='',pages=0
    const seen=new Set<string>()
    while (pages<5 && current(mine,key)) {
      const page=await listInterventionRerunRequests(props.tenantId,props.caseRecord.id,props.token,controller.signal,cursor)
      if (!current(mine,key)) return
      if (!Array.isArray(page.items)||page.items.length>20||typeof page.has_more!=='boolean'||typeof page.next_cursor!=='string'||page.items.some(r=>!r||r.case_id!==props.caseRecord.id||!r.id||!Number.isFinite(Date.parse(r.created_at)))) throw new Error('invalid requests')
      requests.value.push(...page.items)
      pages++
      if (!page.has_more) break
      if (!page.next_cursor||seen.has(page.next_cursor)) throw new Error('invalid cursor')
      seen.add(page.next_cursor);cursor=page.next_cursor
      if (pages===5) truncated.value=true
    }
    if (!current(mine,key)) return
    const ordered=[...requests.value].sort((a,b)=>Date.parse(a.created_at)-Date.parse(b.created_at)||a.id.localeCompare(b.id))
    selectedId.value=ordered.at(-1)?.id||''
    loadHold()
    await readSelected(mine,key)
  } catch(e) {if(current(mine,key)) error.value=safeError(e)}
  finally {if(current(mine,key)) loading.value=false}
}
function selectRequest(id:string) {
  if(sending.value)return
  if (!requests.value.some(r=>r.id===id)) return
  const mine=generation,key=lifetime.value
  selectedId.value=id;status.value=null;clearReadiness();error.value='';loadHold()
  void readSelected(mine,key)
}
async function refreshSelected() {
  if (!selected.value || loading.value || sending.value) return
  const mine=generation,key=lifetime.value
  error.value='';loading.value=true
  await readSelected(mine,key)
  if (current(mine,key)) loading.value=false
}
function delayObservation(ms:number,signal:AbortSignal):Promise<boolean> {
  return new Promise(resolve=>{
    if(signal.aborted){resolve(false);return}
    const timer=setTimeout(()=>{signal.removeEventListener('abort',abort);resolve(true)},ms)
    const abort=()=>{clearTimeout(timer);resolve(false)}
    signal.addEventListener('abort',abort,{once:true})
  })
}
async function observeAfterSend(mine:number,key:string,requestId:string) {
  const deadline=performance.now()+60_000
  const poll=new AbortController()
  const parentSignal=controller.signal
  const abort=()=>poll.abort()
  parentSignal.addEventListener('abort',abort,{once:true})
  if(parentSignal.aborted)abort()
  const deadlineTimer=setTimeout(abort,60_000)
  try {
    for(let i=0;i<12 && current(mine,key) && !poll.signal.aborted && selectedId.value===requestId;i++){
      if(i>0 && !(await delayObservation(5_000,poll.signal))) return
      if(!current(mine,key)||poll.signal.aborted||selectedId.value!==requestId||performance.now()>=deadline)return
      await readSelected(mine,key,false,poll.signal)
      if(poll.signal.aborted||performance.now()>=deadline)return
      if(status.value?.phase==='PUBLISHED'||status.value?.phase==='MANUAL_HOLD'){
        try {await props.refreshCase()} catch {/* status remains available */}
        return
      }
    }
  } finally {
    clearTimeout(deadlineTimer)
    parentSignal.removeEventListener('abort',abort)
    poll.abort()
  }
}
async function executeOnce() {
  if (sending.value||loading.value||props.parentBusy||!props.canExecute||!props.actorId||!selected.value||hold.value||readiness.value?.state!=='READY'||!readinessAlive.value) return
  const mine=generation,key=lifetime.value,requestId=selected.value.id,marker=markerKey.value
  sending.value=true;error.value=''
  try {
    const freshDetail=await props.refreshCase()
    if(!current(mine,key)||!freshDetail||freshDetail.case.id!==props.caseRecord.id||selectedId.value!==requestId) return
    const requestStarted=performance.now()
    const fresh=await getInterventionExecutionReadiness(props.tenantId,props.caseRecord.id,requestId,props.token,controller.signal)
    if(!current(mine,key)||!validReadiness(fresh,requestId)||fresh.state!=='READY'||fresh.expected_case_version!==freshDetail.case.version||!fresh.grant_handle||!fresh.expires_at||Date.parse(fresh.expires_at)<=Date.parse(fresh.server_now)){
      clearReadiness();error.value=t('iv_execution_changed');return
    }
    installReadiness(fresh,requestStarted)
    if(!window.confirm(t('iv_execution_confirm'))) return
    if(!current(mine,key)||!props.canExecute||marker!==markerKey.value||!readinessAlive.value||performance.now()>=deadlineFromServer(fresh,requestStarted)) {error.value=t('iv_execution_changed');return}
    if(!beginExecutionHold(localStorage,marker)){loadHold();error.value=t('iv_execution_storage_unavailable');return}
    loadHold()
    try {
      await executeInterventionRerun(props.tenantId,props.caseRecord.id,requestId,props.token,controller.signal,{expected_case_version:fresh.expected_case_version!,grant_id:fresh.grant_handle})
      if(!current(mine,key)) return
      await observeAfterSend(mine,key,requestId)
    }catch(e){
      updateExecutionHold(localStorage,marker,'UNCERTAIN');loadHold()
      if(current(mine,key))error.value=e instanceof InterventionHttpError && e.status===409?t('iv_execution_changed'):t('iv_execution_uncertain')
    }
  }catch(e){if(current(mine,key))error.value=safeError(e)}
  finally{if(current(mine,key))sending.value=false}
}
watch(lifetime,()=>{void load()},{immediate:true})
watch(()=>props.signal,()=>{void load()})
onBeforeUnmount(()=>{mounted=false;generation++;controller.abort();detachExternal?.();clearReadiness()})
</script>
