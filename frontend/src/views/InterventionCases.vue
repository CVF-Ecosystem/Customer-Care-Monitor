<template>
  <div>
    <div class="d-flex align-center flex-wrap ga-2 mb-4">
      <h1 class="text-h5">{{ $t('iv_title') }}</h1>
      <v-spacer />
      <v-btn v-if="caseId" variant="text" :to="`/${tenantId}/intervention-cases`">{{ $t('iv_back') }}</v-btn>
    </div>
    <v-alert v-if="error" type="error" class="mb-4" role="alert">{{ error }}</v-alert>
    <v-progress-linear v-if="loading" indeterminate aria-label="Loading intervention cases" />

    <template v-if="!caseId">
      <v-card>
        <v-card-title>{{ $t('iv_queue') }}</v-card-title>
        <v-card-text>
          <v-select v-model="filter" :items="filters" item-title="title" item-value="value" :label="$t('iv_filter')" max-width="300" />
          <p class="text-body-2 mb-3">{{ $t('iv_total') }}: {{ total }} · {{ $t('iv_order') }}</p>
          <v-list v-if="items.length">
            <v-list-item v-for="item in items" :key="item.id" :to="`/${tenantId}/intervention-cases/${item.id}`">
              <v-list-item-title>{{ item.conversation_id }}</v-list-item-title>
              <v-list-item-subtitle>{{ stateLabel(item.work_state) }} · {{ formatDate(item.created_at) }} <span v-if="item.due_at">· {{ $t('iv_due') }} {{ formatDate(item.due_at) }}</span></v-list-item-subtitle>
              <v-chip v-if="overdue(item)" color="error" size="small">{{ $t('iv_overdue') }}</v-chip>
            </v-list-item>
          </v-list>
          <p v-else-if="!loading">{{ $t('iv_empty') }}</p>
          <v-btn v-if="nextCursor" :disabled="loading" @click="more">{{ $t('iv_more') }}</v-btn>
        </v-card-text>
      </v-card>
    </template>

    <template v-else-if="detail">
      <v-alert :type="detail.source_status === 'LOCAL_MATCH_CURRENTNESS_UNVERIFIED' ? 'info' : 'warning'" class="mb-4">
        {{ $t('iv_source') }}: {{ sourceLabel(detail.source_status) }}. {{ $t('iv_local_only') }}
      </v-alert>
      <v-card class="mb-4">
        <v-card-title>{{ $t('iv_case') }} {{ detail.case.conversation_id }}</v-card-title>
        <v-card-text>
          <p>{{ $t('iv_state') }}: {{ stateLabel(detail.case.work_state) }} · {{ $t('iv_assessment') }}: {{ assessmentLabel(detail.case.human_assessment) }} · v{{ detail.case.version }}</p>
          <p v-if="detail.case.data_owner_id">{{ $t('iv_data_owner') }}: {{ person(detail.case.data_owner_id) }}</p>
          <p v-if="detail.case.assignee_id">{{ $t('iv_assignee') }}: {{ person(detail.case.assignee_id) }}</p>
          <p v-if="detail.case.due_at">{{ $t('iv_due') }}: {{ formatDate(detail.case.due_at) }} <v-chip v-if="overdue(detail.case)" color="error" size="small">{{ $t('iv_overdue') }}</v-chip></p>
          <p v-if="detail.case.outcome_text">{{ $t('iv_outcome') }}: {{ detail.case.outcome_text }}</p>
          <p v-if="detail.case.human_explanation">{{ $t('iv_human_explanation') }}: {{ detail.case.human_explanation }}</p>
          <p v-if="detail.case.human_next_action">{{ $t('iv_human_next') }}: {{ detail.case.human_next_action }}</p>
        </v-card-text>
      </v-card>
      <v-card v-if="proposal" class="mb-4">
        <v-card-title>{{ $t('iv_original') }}</v-card-title>
        <v-card-text>
          <v-alert v-if="proposal.prompt_version === 'r101-direct-seed'" type="info" class="mb-3">{{ $t('iv_fixture') }}</v-alert>
          <p>{{ $t('iv_kind') }}: {{ proposal.kind }}</p>
          <p>{{ $t('iv_explanation') }}: {{ proposal.explanation }}</p>
          <p>{{ $t('iv_next') }}: {{ proposal.next_action }}</p>
          <p>{{ $t('iv_evidence') }}:</p>
          <ul><li v-for="(ref, i) in evidence" :key="i">{{ ref.message_id }}: “{{ ref.quote }}”</li></ul>
          <p class="text-caption mt-2">{{ proposal.provider }} {{ proposal.model }} · {{ proposal.snapshot_digest }}</p>
        </v-card-text>
      </v-card>
      <v-alert v-if="detail.case.work_state === 'WAITING_ON_DATA'" type="warning" class="mb-4">
        {{ $t('iv_wait_data') }} {{ detail.case.reason_code }}. {{ $t('iv_rerun_manual') }}
      </v-alert>
      <v-card v-if="canEdit" class="mb-4">
        <v-card-title>{{ $t('iv_action') }}</v-card-title>
        <v-card-text>
          <v-select v-model="action" :items="availableActions" item-title="title" item-value="value" :label="$t('iv_action')" :disabled="busy" />
          <v-textarea v-model="reason" :label="$t('iv_reason')" :disabled="busy" :counter="500" rows="2" />
          <template v-if="action === 'CORRECT'">
            <v-textarea v-model="explanation" :label="$t('iv_human_explanation')" :counter="2000" :disabled="busy" />
            <v-textarea v-model="nextAction" :label="$t('iv_human_next')" :counter="2000" :disabled="busy" />
          </template>
          <template v-if="action === 'ASSIGN' || action === 'ASSIGN_DATA'">
            <v-select v-model="assigneeId" :items="assignees" item-title="name" item-value="user_id" :label="action === 'ASSIGN_DATA' ? $t('iv_data_owner') : $t('iv_assignee')" :disabled="busy" />
          </template>
          <v-text-field v-if="needsDue" v-model="dueLocal" type="datetime-local" :label="$t('iv_due')" :disabled="busy" />
          <v-textarea v-if="action === 'ASSIGN_DATA' || action === 'ESCALATE_DATA'" v-model="nextAction" :label="$t('iv_human_next')" :counter="500" :disabled="busy" />
          <v-textarea v-if="action === 'RESOLVE'" v-model="outcome" :label="$t('iv_outcome')" :counter="2000" :disabled="busy" />
          <v-btn color="primary" :loading="busy" :disabled="!canSubmit" @click="submit">{{ $t('iv_save') }}</v-btn>
        </v-card-text>
      </v-card>
      <v-card>
        <v-card-title>{{ $t('iv_history') }}</v-card-title>
        <v-card-text>
          <p v-if="!detail.human_events.length">{{ $t('iv_empty_history') }}</p>
          <div v-for="event in detail.human_events" :key="event.id" class="mb-3 pb-3 border-b">
            <strong>{{ actionLabel(event.action) }} · v{{ event.case_version }}</strong> · {{ formatDate(event.created_at) }} · {{ person(event.actor_id) }}
            <p>{{ $t('iv_reason') }}: {{ event.reason }}</p>
            <p class="text-caption">{{ $t('iv_source') }}: {{ sourceLabel(event.source_status) }}</p>
            <details><summary>{{ $t('iv_change') }}</summary><pre class="text-caption overflow-auto">{{ event.before }} → {{ event.after }}</pre></details>
          </div>
        </v-card-text>
      </v-card>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '../stores/auth'
import { applyInterventionAction, getInterventionCase, listInterventionAssignees, listInterventionCases, InterventionHttpError, type InterventionCase, type InterventionDetail } from '../api/intervention'

const route = useRoute(); const auth = useAuthStore(); const { t } = useI18n()
const tenantId = computed(() => String(route.params.tenantId || ''))
const caseId = computed(() => String(route.params.caseId || ''))
const canEdit = computed(() => auth.canEdit('intervention_cases') && auth.canView('intervention_cases'))
const items = ref<InterventionCase[]>([]); const detail = ref<InterventionDetail | null>(null)
const assignees = ref<{user_id: string; name: string}[]>([])
const filter = ref(''); const total = ref(0); const nextCursor = ref('')
const loading = ref(false); const busy = ref(false); const error = ref('')
const action = ref(''); const reason = ref(''); const explanation = ref(''); const nextAction = ref('')
const assigneeId = ref(''); const dueLocal = ref(''); const outcome = ref('')
let generation = 0; let controller: AbortController | null = null
const currentLifetime = () => JSON.stringify([tenantId.value,caseId.value,auth.accessToken,auth.tenantPerms.role,auth.tenantPerms.permissions])
const stillCurrent = (mine:number,key:string,signal:AbortSignal) => mine===generation && key===currentLifetime() && !signal.aborted
const filters = computed(() => [{title:t('iv_active'),value:''}, ...['OPEN','ASSIGNED','WAITING_ON_DATA','RESOLVED','REJECTED'].map(value=>({title:value,value}))])
const proposal = computed(() => detail.value?.proposals.at(-1))
const evidence = computed<{message_id:string;quote:string}[]>(() => { try { const refs = JSON.parse(proposal.value?.evidence || '[]'); return Array.isArray(refs) ? refs.filter(x => x && typeof x.message_id === 'string' && typeof x.quote === 'string') : [] } catch { return [] } })
const availableActions = computed(() => {
  const c = detail.value?.case; if (!c) return []
  let names: string[] = []
  if (c.work_state === 'WAITING_ON_DATA') names = ['ASSIGN_DATA','ESCALATE_DATA']
  else if (c.disposition === 'PENDING_REVIEW' && c.work_state === 'OPEN') names = ['CONFIRM','CORRECT','REJECT']
  else if (c.work_state === 'OPEN' || c.work_state === 'ASSIGNED') names = c.work_state === 'ASSIGNED' ? ['ASSIGN','RESOLVE'] : ['ASSIGN']
  else if (c.work_state === 'RESOLVED') names = ['REOPEN']
  return names.map(value => ({title:t('iv_act_'+value.toLowerCase()),value}))
})
const needsDue = computed(() => ['ASSIGN','ASSIGN_DATA','REOPEN'].includes(action.value))
const sourceAllowsAction = computed(() => ['ASSIGN_DATA','ESCALATE_DATA'].includes(action.value) || detail.value?.source_status === 'LOCAL_MATCH_CURRENTNESS_UNVERIFIED')
const canSubmit = computed(() => !!detail.value && !!action.value && sourceAllowsAction.value && !!reason.value.trim() && reason.value.length <= 500 && !busy.value &&
  (!needsDue.value || !!dueLocal.value) && (!['ASSIGN','ASSIGN_DATA'].includes(action.value) || !!assigneeId.value) &&
  (action.value !== 'CORRECT' || (!!explanation.value.trim() && !!nextAction.value.trim())) &&
  (!['ASSIGN_DATA','ESCALATE_DATA'].includes(action.value) || !!nextAction.value.trim()) &&
  (action.value !== 'RESOLVE' || !!outcome.value.trim()))
function formatDate(s: string) { return new Date(s).toLocaleString() }
function overdue(c: InterventionCase) { return !!c.due_at && !['RESOLVED','REJECTED'].includes(c.work_state) && new Date(c.due_at).getTime() < Date.now() }
function person(id: string) { return assignees.value.find(x => x.user_id === id)?.name || id }
function knownLabel(prefix:string,value:string,known:string[]) { return known.includes(value) ? t(`${prefix}${value.toLowerCase()}`) : `${t('iv_unknown')} (${value || '—'})` }
function stateLabel(value:string) { return knownLabel('iv_state_',value,['OPEN','ASSIGNED','WAITING_ON_DATA','RESOLVED','REJECTED']) }
function assessmentLabel(value:string) { return value ? knownLabel('iv_assessment_',value,['CONFIRM','CORRECT','REJECT']) : t('iv_pending') }
function sourceLabel(value:string) { return knownLabel('iv_source_',value,['LOCAL_MATCH_CURRENTNESS_UNVERIFIED','CHANGED','INCOMPLETE','UNAVAILABLE']) }
function actionLabel(value:string) { return knownLabel('iv_act_',value,['CONFIRM','CORRECT','REJECT','ASSIGN','RESOLVE','REOPEN','ASSIGN_DATA','ESCALATE_DATA']) }
function resetForm() { action.value=''; reason.value=''; explanation.value=''; nextAction.value=''; assigneeId.value=''; dueLocal.value=''; outcome.value='' }
function message(e: unknown) { if (e instanceof InterventionHttpError) return e.code === 'completion_unknown_reload' ? t('iv_completion_unknown') : e.status === 409 ? t('iv_conflict') : e.status === 403 ? t('iv_denied') : e.status === 401 ? t('iv_session') : `${t('iv_failed')}: ${e.code}`; return t('iv_failed') }
function freshController() { controller?.abort(); controller = new AbortController(); return controller.signal }
async function load() {
  const mine=++generation, tenant=tenantId.value, id=caseId.value, token=auth.accessToken
  const signal=freshController(), key=currentLifetime(); items.value=[]; detail.value=null; assignees.value=[]; nextCursor.value=''; total.value=0; error.value=''; busy.value=false; loading.value=false; resetForm()
  if (!tenant || !token || !auth.canView('intervention_cases')) return
  loading.value=true
  try {
    if (id) {
      const data=await getInterventionCase(tenant,id,token,signal)
      if (!stillCurrent(mine,key,signal)) return
      detail.value=data
      if (canEdit.value) { const users=await listInterventionAssignees(tenant,token,signal); if (stillCurrent(mine,key,signal)) assignees.value=users.items }
    } else {
      const page=await listInterventionCases(tenant,token,signal,filter.value)
      if (stillCurrent(mine,key,signal)) { items.value=page.items; total.value=page.total; nextCursor.value=page.next_cursor }
    }
  } catch(e) { if (stillCurrent(mine,key,signal)) { error.value=message(e); if (e instanceof InterventionHttpError && [401,403].includes(e.status)) { detail.value=null; items.value=[]; total.value=0; nextCursor.value=''; resetForm() } } }
  finally { if (stillCurrent(mine,key,signal)) loading.value=false }
}
async function more() {
  if (!nextCursor.value || loading.value) return
  const mine=generation, signal=controller?.signal, tenant=tenantId.value, token=auth.accessToken, key=currentLifetime()
  if (!signal) return
  loading.value=true
  try { const page=await listInterventionCases(tenant,token,signal,filter.value,nextCursor.value); if (stillCurrent(mine,key,signal)) { items.value.push(...page.items); nextCursor.value=page.next_cursor; total.value=page.total } }
  catch(e) { if (stillCurrent(mine,key,signal)) error.value=message(e) }
  finally { if (stillCurrent(mine,key,signal)) loading.value=false }
}
async function submit() {
  if (!canSubmit.value || !detail.value || !canEdit.value) return
  const mine=generation, tenant=tenantId.value, id=caseId.value, token=auth.accessToken, c=detail.value.case, p=proposal.value, key=currentLifetime()
  const signal=controller?.signal; if (!signal) return
  let due: string | undefined
  if (needsDue.value) { const date=new Date(dueLocal.value); if (!Number.isFinite(date.getTime()) || date.getTime()<=Date.now()) { error.value=t('iv_invalid_due'); return }; due=date.toISOString() }
  busy.value=true; error.value=''
  try {
    await applyInterventionAction(tenant,id,token,signal,{action:action.value,expected_version:c.version,expected_snapshot_id:c.snapshot_id || '',expected_snapshot_digest:c.snapshot_digest,expected_proposal_id:p?.id || '',expected_revision:p?.revision || 0,reason:reason.value,explanation:explanation.value,next_action:nextAction.value,assignee_id:assigneeId.value,due_at:due,outcome:outcome.value})
    if (!stillCurrent(mine,key,signal)) return
    resetForm()
    const data=await getInterventionCase(tenant,id,token,signal)
    if (stillCurrent(mine,key,signal)) detail.value=data
  } catch(e) {
    if (stillCurrent(mine,key,signal)) {
      error.value=message(e); resetForm()
      if (e instanceof InterventionHttpError && (e.status===409 || e.code==='completion_unknown_reload')) { try { const data=await getInterventionCase(tenant,id,token,signal); if (stillCurrent(mine,key,signal)) detail.value=data } catch { if (stillCurrent(mine,key,signal)) detail.value=null } }
      if (e instanceof InterventionHttpError && [401,403].includes(e.status)) { detail.value=null; assignees.value=[] }
    }
  } finally { if (stillCurrent(mine,key,signal)) busy.value=false }
}
watch(() => [route.params.tenantId,route.params.caseId,auth.accessToken,auth.tenantPerms.role,JSON.stringify(auth.tenantPerms.permissions)], load, {immediate:true})
watch(filter, () => { if (!caseId.value) void load() })
onBeforeUnmount(() => { generation++; controller?.abort(); detail.value=null; items.value=[] })
</script>
