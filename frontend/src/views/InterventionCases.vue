<template>
  <div>
    <div class="d-flex align-center flex-wrap ga-2 mb-4">
      <h1 class="text-h5">{{ $t('iv_title') }}</h1>
      <v-spacer />
      <v-btn v-if="caseId" variant="text" :to="`/${tenantId}/intervention-cases`">{{ $t('iv_back') }}</v-btn>
    </div>
    <InterventionCapabilityNotice :tenant-id="tenantId" :token="auth.accessToken" />
    <v-alert v-if="error" type="error" class="mb-4" role="alert">{{ error }}</v-alert>
    <v-btn v-if="caseId && !loading && !detail && error" class="mb-4" variant="outlined" @click="load">{{ $t('iv_reload_case') }}</v-btn>
    <v-progress-linear v-if="loading" indeterminate :aria-label="$t('iv_loading')" />

    <template v-if="!caseId">
      <v-card>
        <v-card-title>{{ $t('iv_queue') }}</v-card-title>
        <v-card-text>
          <v-select v-model="filter" :items="filters" item-title="title" item-value="value" :label="$t('iv_filter')" max-width="300" />
          <p class="text-body-2 mb-3">{{ $t('iv_total') }}: {{ total }} · {{ $t('iv_order') }}</p>
          <v-list v-if="items.length">
            <v-list-item v-for="item in items" :key="item.id" :to="`/${tenantId}/intervention-cases/${item.id}`">
              <v-list-item-title>{{ caseReference(item.conversation_id, item.id) }}</v-list-item-title>
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
      <v-alert v-if="detail.source_status === 'CHANGED'" type="warning" class="mb-4">
        {{ detail.reviewed_source_current ? $t('iv_changed_reviewed') : $t('iv_changed_review_required') }}
        <v-btn v-if="auth.canView('messages')" variant="text" :to="`/${tenantId}/messages?conv=${encodeURIComponent(detail.case.conversation_id)}`">{{ $t('iv_open_updated_conversation') }}</v-btn>
      </v-alert>
      <v-card class="mb-4">
        <v-card-title>{{ caseReference(detail.case.conversation_id, detail.case.id) }}</v-card-title>
        <v-card-text>
          <p>{{ $t('iv_state') }}: {{ stateLabel(detail.case.work_state) }} · {{ $t('iv_assessment') }}: {{ assessmentLabel(detail.case.human_assessment) }} · {{ revisionLabel(detail.case.version) }}</p>
          <p v-if="detail.case.data_owner_id">{{ $t('iv_data_owner') }}: {{ person(detail.case.data_owner_id) }}</p>
          <p v-if="detail.case.assignee_id">{{ $t('iv_assignee') }}: {{ person(detail.case.assignee_id) }}</p>
          <p v-if="detail.case.due_at">{{ $t('iv_due') }}: {{ formatDate(detail.case.due_at) }} <v-chip v-if="overdue(detail.case)" color="error" size="small">{{ $t('iv_overdue') }}</v-chip></p>
          <p v-if="detail.case.outcome_text">{{ $t('iv_outcome') }}: {{ detail.case.outcome_text }}</p>
          <p v-if="detail.case.human_explanation">{{ $t('iv_human_explanation') }}: {{ detail.case.human_explanation }}</p>
          <p v-if="detail.case.human_next_action">{{ $t('iv_human_next') }}: {{ detail.case.human_next_action }}</p>
        </v-card-text>
      </v-card>
      <v-card v-if="proposal" class="mb-4">
        <v-card-title>{{ $t('iv_current_proposal') }}</v-card-title>
        <v-card-text>
          <v-alert v-if="proposal.prompt_version === 'r101-direct-seed'" type="info" class="mb-3">{{ $t('iv_fixture') }}</v-alert>
          <p>{{ $t('iv_kind') }}: {{ proposalKindLabel(proposal.kind) }}</p>
          <p>{{ $t('iv_explanation') }}: {{ proposal.explanation }}</p>
          <p>{{ $t('iv_next') }}: {{ proposal.next_action }}</p>
          <p>{{ $t('iv_evidence') }}:</p>
          <ul><li v-for="(ref, i) in evidence" :key="i">“{{ ref.quote }}”</li></ul>
        </v-card-text>
      </v-card>
      <v-alert v-else-if="requiresProposal" type="warning" class="mb-4">{{ $t('iv_proposal_unavailable') }}</v-alert>
      <v-alert v-if="detail.case.work_state === 'WAITING_ON_DATA'" type="warning" class="mb-4">
        {{ $t('iv_wait_data') }} {{ $t('iv_rerun_manual') }}
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
            <strong>{{ actionLabel(event.action) }} · {{ revisionLabel(event.case_version) }}</strong> · {{ formatDate(event.created_at) }} · {{ person(event.actor_id) }}
            <p>{{ $t('iv_reason') }}: {{ eventReason(event) }}</p>
            <p class="text-caption">{{ $t('iv_source') }}: {{ sourceLabel(event.source_status) }}</p>
            <details>
              <summary>{{ $t('iv_change') }}</summary>
              <p v-if="!historyForEvent(event).valid">{{ $t('iv_history_unavailable') }}</p>
              <p v-else-if="!historyForEvent(event).changes.length">{{ $t('iv_history_no_changes') }}</p>
              <dl v-else v-for="change in historyForEvent(event).changes" :key="change.field" class="ml-2">
                <dt class="font-weight-medium">{{ change.field }}</dt>
                <dd>{{ $t('iv_before') }}: {{ change.before }}</dd>
                <dd>{{ $t('iv_after') }}: {{ change.after }}</dd>
              </dl>
            </details>
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
import InterventionCapabilityNotice from '../components/InterventionCapabilityNotice.vue'
import { applyInterventionAction, getInterventionCase, listInterventionAssignees, listInterventionCases, InterventionHttpError, type InterventionCase, type InterventionDetail, type InterventionEvent } from '../api/intervention'
import {
  interventionActionKey,
  interventionAssessmentKey,
  interventionCaseReference,
  currentInterventionProposal,
  interventionDispositionKey,
  interventionHistoryChanges,
  interventionEventReasonKey,
  interventionProposalKindKey,
  interventionSourceKey,
  interventionStateKey,
  type InterventionHistoryLabels,
} from '../utils/interventionPresentation'

const route = useRoute(); const auth = useAuthStore(); const { t, locale } = useI18n()
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
const filters = computed(() => [{title:t('iv_active'),value:''}, ...['OPEN','ASSIGNED','WAITING_ON_DATA','RESOLVED','REJECTED'].map(value=>({title:stateLabel(value),value}))])
const proposal = computed(() => currentInterventionProposal(detail.value))
const evidence = computed<{message_id:string;quote:string}[]>(() => { try { const refs = JSON.parse(proposal.value?.evidence || '[]'); return Array.isArray(refs) ? refs.filter(x => x && typeof x.message_id === 'string' && typeof x.quote === 'string') : [] } catch { return [] } })
const availableActions = computed(() => {
  const c = detail.value?.case; if (!c) return []
  let names: string[] = []
  if (c.work_state === 'WAITING_ON_DATA') names = ['ASSIGN_DATA','ESCALATE_DATA']
  else if (c.disposition === 'PENDING_REVIEW' && c.work_state === 'OPEN') names = ['CONFIRM','CORRECT','REJECT']
  else if (c.work_state === 'OPEN' || c.work_state === 'ASSIGNED') names = c.work_state === 'ASSIGNED' ? ['ASSIGN','RESOLVE'] : ['ASSIGN']
  else if (c.work_state === 'RESOLVED') names = ['REOPEN']
  if (detail.value?.source_status === 'CHANGED' && !detail.value.reviewed_source_current && auth.canView('messages') && ['OPEN','ASSIGNED','RESOLVED'].includes(c.work_state) && ['CONFIRM','CORRECT'].includes(c.human_assessment)) names.unshift('ACKNOWLEDGE_SOURCE_CHANGE')
  return names.map(value => ({title:t('iv_act_'+value.toLowerCase()),value}))
})
const needsDue = computed(() => ['ASSIGN','ASSIGN_DATA','REOPEN'].includes(action.value))
const proposalLessAction = computed(() => ['ASSIGN_DATA','ESCALATE_DATA'].includes(action.value))
const requiresProposal = computed(() => availableActions.value.some(item => !['ASSIGN_DATA','ESCALATE_DATA'].includes(item.value)))
const sourceAllowsAction = computed(() => ['ASSIGN_DATA','ESCALATE_DATA'].includes(action.value) || (action.value === 'ACKNOWLEDGE_SOURCE_CHANGE' && !!detail.value?.current_source_digest) || detail.value?.source_status === 'LOCAL_MATCH_CURRENTNESS_UNVERIFIED' || (['ASSIGN','RESOLVE','REOPEN'].includes(action.value) && !!detail.value?.reviewed_source_current))
const canSubmit = computed(() => !!detail.value && !!action.value && sourceAllowsAction.value && !!reason.value.trim() && reason.value.length <= 500 && !busy.value &&
  (proposalLessAction.value || !!proposal.value) &&
  (!needsDue.value || !!dueLocal.value) && (!['ASSIGN','ASSIGN_DATA'].includes(action.value) || !!assigneeId.value) &&
  (action.value !== 'CORRECT' || (!!explanation.value.trim() && !!nextAction.value.trim())) &&
  (!['ASSIGN_DATA','ESCALATE_DATA'].includes(action.value) || !!nextAction.value.trim()) &&
  (action.value !== 'RESOLVE' || !!outcome.value.trim()))
function formatDate(s: string) { const date = new Date(s); return Number.isFinite(date.getTime()) ? date.toLocaleString(locale.value) : t('iv_not_set') }
function overdue(c: InterventionCase) { return !!c.due_at && !['RESOLVED','REJECTED'].includes(c.work_state) && new Date(c.due_at).getTime() < Date.now() }
function caseReference(conversationId: string, id: string) {
  const reference = interventionCaseReference(conversationId, id)
  if (!reference.value) return t('iv_reference_unavailable')
  return t('iv_case_reference', { reference: reference.value })
}
function revisionLabel(version: number) { return t('iv_revision', { version }) }
function person(id: string) { return detail.value?.participants?.find(x => x.user_id === id)?.name || assignees.value.find(x => x.user_id === id)?.name || t('iv_person_unavailable') }
function stateLabel(value:string) { return t(interventionStateKey(value)) }
function assessmentLabel(value:string) { return t(interventionAssessmentKey(value)) }
function sourceLabel(value:string) { return t(interventionSourceKey(value)) }
function actionLabel(value:string) { return t(interventionActionKey(value)) }
function eventReason(event: InterventionEvent) { const key=interventionEventReasonKey(event.action,event.reason); return key ? t(key) : event.reason }
function proposalKindLabel(value:string) { return t(interventionProposalKindKey(value)) }
const historyLabels = computed<InterventionHistoryLabels>(() => ({
  fields: {
    disposition: t('iv_history_disposition'),
    work_state: t('iv_history_work_state'),
    human_assessment: t('iv_history_assessment'),
    human_explanation: t('iv_human_explanation'),
    human_next_action: t('iv_human_next'),
    assignee_id: t('iv_assignee'),
    data_owner_id: t('iv_data_owner'),
    due_at: t('iv_due'),
    outcome_text: t('iv_outcome'),
  },
  notSet: t('iv_not_set'),
  state: value => stateLabel(value),
  disposition: value => t(interventionDispositionKey(value)),
  assessment: value => assessmentLabel(value),
  person: value => person(value),
  date: value => formatDate(value),
}))
function historyForEvent(event: InterventionEvent) {
  return interventionHistoryChanges(event.before, event.after, historyLabels.value)
}
function resetForm() { action.value=''; reason.value=''; explanation.value=''; nextAction.value=''; assigneeId.value=''; dueLocal.value=''; outcome.value='' }
function message(e: unknown, context: 'load' | 'write' = 'write') {
  const fallback = context === 'load' ? 'iv_load_failed' : 'iv_failed'
  if (!(e instanceof InterventionHttpError)) return t(fallback)
  if (e.code === 'completion_unknown_reload') return t('iv_completion_unknown')
  if (e.status === 409) return t('iv_conflict')
  if (e.status === 403) return t('iv_denied')
  if (e.status === 401) return t('iv_session')
  if (e.status === 400 || e.status === 422) return t('iv_review_details')
  return t(fallback)
}
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
  } catch(e) { if (stillCurrent(mine,key,signal)) { error.value=message(e, 'load'); if (e instanceof InterventionHttpError && [401,403].includes(e.status)) { detail.value=null; items.value=[]; total.value=0; nextCursor.value=''; resetForm() } } }
  finally { if (stillCurrent(mine,key,signal)) loading.value=false }
}
async function more() {
  if (!nextCursor.value || loading.value) return
  const mine=generation, signal=controller?.signal, tenant=tenantId.value, token=auth.accessToken, key=currentLifetime()
  if (!signal) return
  loading.value=true
  try { const page=await listInterventionCases(tenant,token,signal,filter.value,nextCursor.value); if (stillCurrent(mine,key,signal)) { items.value.push(...page.items); nextCursor.value=page.next_cursor; total.value=page.total } }
  catch(e) { if (stillCurrent(mine,key,signal)) error.value=message(e, 'load') }
  finally { if (stillCurrent(mine,key,signal)) loading.value=false }
}
async function submit() {
  if (!canSubmit.value || !detail.value || !canEdit.value) return
  const mine=generation, tenant=tenantId.value, id=caseId.value, token=auth.accessToken, c=detail.value.case, p=proposalLessAction.value ? null : proposal.value, key=currentLifetime()
  const signal=controller?.signal; if (!signal) return
  let due: string | undefined
  if (needsDue.value) { const date=new Date(dueLocal.value); if (!Number.isFinite(date.getTime()) || date.getTime()<=Date.now()) { error.value=t('iv_invalid_due'); return }; due=date.toISOString() }
  busy.value=true; error.value=''
  let actionRecorded = false
  try {
    await applyInterventionAction(tenant,id,token,signal,{action:action.value,expected_version:c.version,expected_snapshot_id:c.snapshot_id || '',expected_snapshot_digest:c.snapshot_digest,expected_proposal_id:p?.id || '',expected_revision:p?.revision || 0,expected_current_digest:detail.value.current_source_digest || '',reason:reason.value,explanation:explanation.value,next_action:nextAction.value,assignee_id:assigneeId.value,due_at:due,outcome:outcome.value})
    actionRecorded = true
    if (!stillCurrent(mine,key,signal)) return
    resetForm()
    const data=await getInterventionCase(tenant,id,token,signal)
    if (stillCurrent(mine,key,signal)) detail.value=data
  } catch(e) {
    if (stillCurrent(mine,key,signal)) {
      error.value=actionRecorded ? t('iv_refresh_failed') : message(e); resetForm()
      if (actionRecorded) detail.value=null
      if (!actionRecorded && e instanceof InterventionHttpError && (e.status===409 || e.code==='completion_unknown_reload')) { try { const data=await getInterventionCase(tenant,id,token,signal); if (stillCurrent(mine,key,signal)) detail.value=data } catch { if (stillCurrent(mine,key,signal)) detail.value=null } }
      if (e instanceof InterventionHttpError && [401,403].includes(e.status)) { detail.value=null; assignees.value=[] }
    }
  } finally { if (stillCurrent(mine,key,signal)) busy.value=false }
}
watch(() => [route.params.tenantId,route.params.caseId,auth.accessToken,auth.tenantPerms.role,JSON.stringify(auth.tenantPerms.permissions)], load, {immediate:true})
watch(filter, () => { if (!caseId.value) void load() })
onBeforeUnmount(() => { generation++; controller?.abort(); detail.value=null; items.value=[] })
</script>
