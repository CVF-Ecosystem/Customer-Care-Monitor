<template>
  <v-card class="mb-4" data-testid="intervention-data-followup">
    <v-card-title>{{ $t('iv_followup_title') }}</v-card-title>
    <v-card-text>
      <p class="mb-3">{{ $t('iv_followup_local_note') }}</p>
      <p class="mb-3">{{ $t('iv_followup_request_note') }}</p>
      <v-alert v-if="!allowedToRead" type="info" class="mb-3">{{ $t('iv_followup_unavailable') }}</v-alert>
      <template v-else>
        <v-alert v-if="error" type="error" role="alert" class="mb-3">{{ error }}</v-alert>
        <v-alert v-if="reloadRequired" type="warning" role="status" class="mb-3">
          {{ $t('iv_followup_reload_required') }}
          <v-btn class="ml-2" variant="text" :disabled="historyLoading || writing || parentBusy" @click="reloadExplicitly">{{ $t('iv_followup_reload') }}</v-btn>
        </v-alert>
        <v-alert v-if="savedCapture" type="success" role="status" class="mb-3">
          {{ $t('iv_followup_capture_saved') }} · {{ coverageLabel(savedCapture.coverage) }} · {{ savedCapture.message_count }} {{ $t('iv_followup_items') }} · {{ formatDate(savedCapture.created_at) }}
          <span v-if="savedCapture.reason"> · {{ savedCapture.reason }}</span>
        </v-alert>
        <v-alert v-if="savedRequest" type="info" role="status" class="mb-3">
          {{ $t('iv_followup_request_saved') }} {{ requestStatusLabel(savedRequest.status) }} · {{ executionLabel(savedRequest.execution) }}
        </v-alert>

        <v-progress-linear v-if="historyLoading" indeterminate class="mb-3" :aria-label="$t('iv_loading')" />

        <section v-if="canCapture" class="mb-5">
          <h3 class="text-subtitle-1 mb-2">{{ $t('iv_followup_capture_heading') }}</h3>
          <v-textarea v-model="captureReason" :label="$t('iv_followup_capture_reason')" :counter="500" rows="2" :disabled="controlsDisabled" />
          <v-btn color="primary" :loading="writingKind === 'capture'" :disabled="controlsDisabled || !captureReason.trim() || captureReason.length > 500" @click="capture">
            {{ $t('iv_followup_capture') }}
          </v-btn>
        </section>

        <section class="mb-5">
          <div class="d-flex align-center flex-wrap ga-2 mb-2">
            <h3 class="text-subtitle-1">{{ $t('iv_followup_revisions') }}</h3>
            <v-btn v-if="revisionMoreAvailable" size="small" variant="text" :disabled="historyLoading || writing || parentBusy" @click="loadMore('revision')">{{ $t('iv_followup_more') }}</v-btn>
          </div>
          <v-alert v-if="revisionTruncated" type="warning" density="compact" class="mb-2">{{ $t('iv_followup_truncated') }}</v-alert>
          <p v-if="!revisions.length && !historyLoading" class="text-body-2">{{ $t('iv_followup_no_revisions') }}</p>
          <article v-for="revision in revisions" :key="revision.id" class="mb-3 pb-2 border-b" data-testid="revision-row">
            <strong>{{ coverageLabel(revision.coverage) }}</strong> · {{ observationLabel(revision.observation_status) }} · {{ revision.message_count }} {{ $t('iv_followup_items') }} · {{ formatDate(revision.created_at) }}
            <p v-if="revision.reason" class="mb-0">{{ $t('iv_followup_reason') }}: {{ revision.reason }}</p>
          </article>
        </section>

        <section class="mb-5">
          <div class="d-flex align-center flex-wrap ga-2 mb-2">
            <h3 class="text-subtitle-1">{{ $t('iv_followup_request_history') }}</h3>
            <v-btn v-if="requestMoreAvailable" size="small" variant="text" :disabled="historyLoading || writing || parentBusy" @click="loadMore('request')">{{ $t('iv_followup_more') }}</v-btn>
          </div>
          <v-alert v-if="requestTruncated" type="warning" density="compact" class="mb-2">{{ $t('iv_followup_truncated') }}</v-alert>
          <p v-if="!requestsWithSaved.length && !historyLoading" class="text-body-2">{{ $t('iv_followup_no_requests') }}</p>
          <article v-for="request in requestsWithSaved" :key="request.id" class="mb-3 pb-2 border-b" data-testid="request-row">
            <strong>{{ requestStatusLabel(request.status) }}</strong> · {{ executionLabel(request.execution) }} · {{ formatDate(request.created_at) }}
            <p v-if="request.reason" class="mb-0">{{ $t('iv_followup_reason') }}: {{ request.reason }}</p>
          </article>
        </section>

        <section v-if="canRequest" class="mb-2">
          <h3 class="text-subtitle-1 mb-2">{{ $t('iv_followup_request_heading') }}</h3>
          <p v-if="requestReadinessLabel" class="text-body-2">{{ requestReadinessLabel }}</p>
          <p v-else-if="hasRequestForLatest" class="text-body-2">{{ $t('iv_followup_pending') }}</p>
          <template v-else>
            <v-textarea v-model="requestReason" :label="$t('iv_followup_request_reason')" :counter="500" rows="2" :disabled="controlsDisabled || !requestReady" />
            <v-btn color="primary" :loading="writingKind === 'request'" :disabled="controlsDisabled || !requestReady || !requestReason.trim() || requestReason.length > 500" @click="requestRerun">
              {{ $t('iv_followup_request') }}
            </v-btn>
          </template>
        </section>
        <p v-else-if="allowedToRead && canCapture" class="text-body-2">{{ $t('iv_followup_no_request_permission') }}</p>
        <p v-else-if="allowedToRead" class="text-body-2">{{ $t('iv_followup_read_only') }}</p>
      </template>
    </v-card-text>
  </v-card>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  createInterventionDataRevision,
  createInterventionRerunRequest,
  InterventionHttpError,
  listInterventionDataRevisions,
  listInterventionRerunRequests,
  type InterventionCase,
  type InterventionDataRevision,
  type InterventionDetail,
  type InterventionRerunRequest,
  type InterventionWaitDataPage,
} from '../api/intervention'

const props = defineProps<{
  tenantId: string
  token: string
  role: string
  caseRecord: InterventionCase
  canRead: boolean
  canCapture: boolean
  canRequest: boolean
  parentBusy: boolean
  lifetimeKey: string
  signal?: AbortSignal
  refreshCase: () => Promise<InterventionDetail | null>
}>()
const emit = defineEmits<{ (event: 'write-busy', value: boolean): void }>()
const { t, locale } = useI18n()

const PAGE_SIZE = 20
const MAX_PAGES = 5
const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i
const SHA256 = /^[0-9a-f]{64}$/i
type Kind = 'revision' | 'request'
type WriteKind = 'capture' | 'request'
const knownRole = computed(() => ['owner','admin','member'].includes(props.role))
const allowedToRead = computed(() => knownRole.value && props.canRead && !!props.token && !!props.tenantId)
const canCapture = computed(() => allowedToRead.value && props.canCapture && props.caseRecord.work_state === 'WAITING_ON_DATA' && !!props.caseRecord.snapshot_id && !!props.caseRecord.snapshot_digest)
const canRequest = computed(() => canCapture.value && props.canRequest)
const revisions = ref<InterventionDataRevision[]>([])
const requests = ref<InterventionRerunRequest[]>([])
const revisionCursor = ref(''); const requestCursor = ref('')
const revisionHasMore = ref(false); const requestHasMore = ref(false)
const revisionPageCount = ref(0); const requestPageCount = ref(0)
const revisionLoaded = ref(false); const requestLoaded = ref(false)
const revisionComplete = ref(false); const requestComplete = ref(false)
const historyLoading = ref(false); const writingKind = ref<'' | WriteKind>('')
const error = ref(''); const reloadRequired = ref(false)
const captureReason = ref(''); const requestReason = ref('')
const savedCapture = ref<InterventionDataRevision | null>(null)
const savedCaptureCaseVersion = ref<number | null>(null)
const savedRequest = ref<InterventionRerunRequest | null>(null)
let generation = 0
let active = true
let controller = new AbortController()
let externalSignal: AbortSignal | undefined
let detachExternal: (() => void) | null = null
let revisionCursors = new Set<string>()
let requestCursors = new Set<string>()
let historyUnavailable = false
const activeReads = new Set<symbol>()

const writing = computed(() => writingKind.value !== '')
const parentBusy = computed(() => props.parentBusy)
const controlsDisabled = computed(() => historyLoading.value || writing.value || parentBusy.value || reloadRequired.value || !allowedToRead.value)
const revisionMoreAvailable = computed(() => revisionHasMore.value && revisionPageCount.value < MAX_PAGES)
const requestMoreAvailable = computed(() => requestHasMore.value && requestPageCount.value < MAX_PAGES)
const revisionTruncated = computed(() => revisionHasMore.value && revisionPageCount.value >= MAX_PAGES)
const requestTruncated = computed(() => requestHasMore.value && requestPageCount.value >= MAX_PAGES)

function validDate(value: string) { return Number.isFinite(Date.parse(value)) }
function validRevision(value: unknown): value is InterventionDataRevision {
  if (!value || typeof value !== 'object') return false
  const row = value as InterventionDataRevision
  return typeof row.id === 'string' && UUID.test(row.id) && row.case_id === props.caseRecord.id &&
    Number.isSafeInteger(row.originating_case_version) && row.originating_case_version >= 0 &&
    Number.isSafeInteger(row.case_version) && row.case_version === row.originating_case_version + 1 &&
    row.original_snapshot_id === props.caseRecord.snapshot_id && row.original_snapshot_digest === props.caseRecord.snapshot_digest &&
    typeof row.digest === 'string' && SHA256.test(row.digest) && typeof row.coverage === 'string' &&
    Array.isArray(row.coverage_reasons) && row.coverage_reasons.every(item => typeof item === 'string') &&
    Number.isSafeInteger(row.message_count) && row.message_count >= 0 && typeof row.observation_status === 'string' &&
    typeof row.reason === 'string' && row.reason.length <= 500 && typeof row.created_at === 'string' && validDate(row.created_at)
}
function validRequest(value: unknown): value is InterventionRerunRequest {
  if (!value || typeof value !== 'object') return false
  const row = value as InterventionRerunRequest
  return typeof row.id === 'string' && UUID.test(row.id) && row.case_id === props.caseRecord.id &&
    typeof row.data_revision_id === 'string' && UUID.test(row.data_revision_id) && typeof row.data_revision_digest === 'string' && SHA256.test(row.data_revision_digest) &&
    row.original_snapshot_id === props.caseRecord.snapshot_id && row.original_snapshot_digest === props.caseRecord.snapshot_digest &&
    typeof row.reason === 'string' && row.reason.length <= 500 && typeof row.status === 'string' && typeof row.execution === 'string' &&
    Number.isSafeInteger(row.case_version) && typeof row.created_at === 'string' && validDate(row.created_at)
}
function validPage<T>(value: unknown, validate: (item: unknown) => item is T, cursor: string): value is InterventionWaitDataPage<T> {
  if (!value || typeof value !== 'object') return false
  const page = value as InterventionWaitDataPage<T>
  if (!Array.isArray(page.items) || page.items.length > PAGE_SIZE || !page.items.every(validate) || typeof page.has_more !== 'boolean' || typeof page.next_cursor !== 'string') return false
  if (page.has_more) return page.items.length > 0 && !!page.next_cursor && page.next_cursor !== cursor
  return page.next_cursor === ''
}
function timeAndIdOrder<T extends {created_at: string; id: string}>(rows: T[]) {
  for (let i=1;i<rows.length;i++) {
    const prior = Date.parse(rows[i-1].created_at), current = Date.parse(rows[i].created_at)
    if (current < prior || (current === prior && rows[i].id.localeCompare(rows[i-1].id) <= 0)) return false
  }
  return true
}
function current(mine: number, key: string, signal: AbortSignal) {
  return active && mine === generation && key === props.lifetimeKey && signal === controller.signal && externalSignal === props.signal && !signal.aborted && !props.signal?.aborted
}
function setSafeError(err: unknown, operation: Kind | WriteKind | 'load') {
  if (!(err instanceof InterventionHttpError)) { error.value = t(operation === 'load' ? 'iv_followup_load_failed' : 'iv_followup_failed'); return }
  if (err.code === 'completion_unknown_reload' || err.status === 0) {
    error.value = t('iv_followup_unknown'); reloadRequired.value = true; return
  }
  if (err.status === 401 || err.status === 403 || err.status === 404) {
    error.value = err.status === 401 ? t('iv_session') : err.status === 403 ? t('iv_denied') : t('iv_followup_missing')
    reloadRequired.value = true; historyUnavailable = true; clearHistory(); savedCapture.value=null; savedCaptureCaseVersion.value=null; savedRequest.value=null
    captureReason.value=''; requestReason.value=''; return
  }
  if (err.status === 400 || err.status === 422 || err.code === 'invalid_request') { error.value = t('iv_followup_invalid'); return }
  if (err.status === 409) {
    if (err.code === 'source_unchanged') { error.value = t('iv_followup_unchanged'); return }
    if (err.code === 'request_pending') { error.value = t('iv_followup_pending'); reloadRequired.value = true; return }
    error.value = t('iv_followup_conflict'); reloadRequired.value = true; return
  }
  if (err.status >= 500 || err.code === 'source_unavailable') { error.value = t('iv_followup_unavailable_error'); reloadRequired.value = true; return }
  error.value = t(operation === 'load' ? 'iv_followup_load_failed' : 'iv_followup_failed')
}
function clearHistory() {
  revisions.value=[]; requests.value=[]; revisionCursor.value=''; requestCursor.value=''
  revisionHasMore.value=false; requestHasMore.value=false; revisionPageCount.value=0; requestPageCount.value=0
  revisionLoaded.value=false; requestLoaded.value=false; revisionComplete.value=false; requestComplete.value=false
  revisionCursors=new Set<string>(); requestCursors=new Set<string>()
}
function clearState() {
  clearHistory(); error.value=''; reloadRequired.value=false; captureReason.value=''; requestReason.value=''
  savedCapture.value=null; savedCaptureCaseVersion.value=null; savedRequest.value=null
}
function setupSignal() {
  controller.abort(); detachExternal?.(); detachExternal=null
  controller=new AbortController(); externalSignal=props.signal
  if (externalSignal) {
    const currentController=controller
    const abort = () => currentController.abort()
    if (externalSignal.aborted) abort()
    else { externalSignal.addEventListener('abort',abort,{once:true}); detachExternal=()=>externalSignal?.removeEventListener('abort',abort) }
  }
}
async function readPage(kind: Kind, mine: number, key: string, signal: AbortSignal, reset = false): Promise<boolean> {
  if (reset) {
    if (kind === 'revision') { revisions.value=[]; revisionCursor.value=''; revisionHasMore.value=false; revisionPageCount.value=0; revisionLoaded.value=false; revisionComplete.value=false; revisionCursors=new Set<string>() }
    else { requests.value=[]; requestCursor.value=''; requestHasMore.value=false; requestPageCount.value=0; requestLoaded.value=false; requestComplete.value=false; requestCursors=new Set<string>() }
  }
  const cursor = reset ? '' : kind === 'revision' ? revisionCursor.value : requestCursor.value
  const seen = kind === 'revision' ? revisionCursors : requestCursors
  if (cursor && seen.has(cursor)) { error.value=t('iv_followup_load_failed'); reloadRequired.value=true; return false }
  const readToken=Symbol(kind); activeReads.add(readToken); historyLoading.value=true
  try {
    const page = kind === 'revision'
      ? await listInterventionDataRevisions(props.tenantId,props.caseRecord.id,props.token,signal,cursor)
      : await listInterventionRerunRequests(props.tenantId,props.caseRecord.id,props.token,signal,cursor)
    if (!current(mine,key,signal) || historyUnavailable) return false
    const pageIsValid = kind === 'revision'
      ? validPage<InterventionDataRevision>(page,(item): item is InterventionDataRevision => validRevision(item),cursor)
      : validPage<InterventionRerunRequest>(page,(item): item is InterventionRerunRequest => validRequest(item),cursor)
    if (!pageIsValid) { error.value=t('iv_followup_load_failed'); reloadRequired.value=true; return false }
    const previous = kind === 'revision' ? revisions.value : requests.value
    const combined = [...previous,...page.items]
    const ids = new Set<string>()
    if (combined.some(row => ids.has(row.id) || (ids.add(row.id),false)) || !timeAndIdOrder(combined)) { error.value=t('iv_followup_load_failed'); reloadRequired.value=true; return false }
    if (kind === 'revision') {
      revisions.value=combined as InterventionDataRevision[]; revisionHasMore.value=page.has_more; revisionCursor.value=page.next_cursor
      revisionPageCount.value++; revisionLoaded.value=true; revisionComplete.value=!page.has_more
      if (cursor) revisionCursors.add(cursor)
    } else {
      requests.value=combined as InterventionRerunRequest[]; requestHasMore.value=page.has_more; requestCursor.value=page.next_cursor
      requestPageCount.value++; requestLoaded.value=true; requestComplete.value=!page.has_more
      if (cursor) requestCursors.add(cursor)
    }
    return true
  } catch (err) {
    if (current(mine,key,signal)) setSafeError(err,'load')
    return false
  } finally {
    activeReads.delete(readToken)
    if (current(mine,key,signal)) historyLoading.value=activeReads.size>0
  }
}
async function loadInitial(mine: number, key: string, signal: AbortSignal) {
  await Promise.all([readPage('revision',mine,key,signal,true),readPage('request',mine,key,signal,true)])
}
async function loadMore(kind: Kind) {
  if (controlsDisabled.value || (kind === 'revision' ? !revisionMoreAvailable.value : !requestMoreAvailable.value)) return
  const mine=generation, key=props.lifetimeKey, signal=controller.signal
  error.value=''
  await readPage(kind,mine,key,signal)
}
const capturedIsCurrent = computed(() => !!savedCapture.value && savedCaptureCaseVersion.value === props.caseRecord.version &&
  savedCapture.value.case_id === props.caseRecord.id && savedCapture.value.original_snapshot_id === props.caseRecord.snapshot_id &&
  savedCapture.value.original_snapshot_digest === props.caseRecord.snapshot_digest)
const latestRevision = computed(() => {
  if (revisionComplete.value) return revisions.value.length ? revisions.value[revisions.value.length-1] : null
  if (capturedIsCurrent.value) return savedCapture.value
  return null
})
const hasRequestForLatest = computed(() => {
  const latest=latestRevision.value
  return !!latest && (requests.value.some(item => item.data_revision_id === latest.id) || savedRequest.value?.data_revision_id === latest.id)
})
const requestReady = computed(() => canRequest.value && !controlsDisabled.value && !reloadRequired.value && requestComplete.value &&
  !!latestRevision.value && latestRevision.value.coverage === 'complete' && !hasRequestForLatest.value &&
  latestRevision.value.original_snapshot_id === props.caseRecord.snapshot_id &&
  latestRevision.value.original_snapshot_digest === props.caseRecord.snapshot_digest)
const requestReadinessLabel = computed(() => {
  if (!latestRevision.value) {
    if (!revisionLoaded.value) return t('iv_followup_loading_latest')
    if (revisionTruncated.value) return t('iv_followup_truncated')
    if (revisionHasMore.value) return t('iv_followup_load_latest')
    if (revisionComplete.value && revisions.value.length && revisions.value[revisions.value.length-1].coverage !== 'complete') return t('iv_followup_incomplete')
    return t('iv_followup_no_complete_revision')
  }
  if (latestRevision.value.coverage !== 'complete') return t('iv_followup_incomplete')
  if (!requestLoaded.value || !requestComplete.value) return requestTruncated.value ? t('iv_followup_truncated') : t('iv_followup_load_request_history')
  return ''
})
const requestsWithSaved = computed(() => {
  const rows=requests.value.slice()
  if (savedRequest.value && !rows.some(item=>item.id===savedRequest.value?.id)) rows.push(savedRequest.value)
  return rows.sort((a,b)=>Date.parse(a.created_at)-Date.parse(b.created_at) || a.id.localeCompare(b.id))
})
function coverageLabel(value: string) {
  if (value === 'complete') return t('iv_followup_coverage_complete')
  if (value === 'partial' || value === 'empty') return t('iv_followup_coverage_incomplete')
  return t('iv_followup_coverage_unknown')
}
function observationLabel(value: string) {
  if (value === 'LOCAL_MATCH_CURRENTNESS_UNVERIFIED') return t('iv_followup_observation_unverified')
  if (value === 'INCOMPLETE') return t('iv_followup_observation_incomplete')
  return t('iv_followup_coverage_unknown')
}
function requestStatusLabel(value: string) { return value === 'AWAITING_EXECUTION_AUTHORITY' ? t('iv_followup_status_waiting') : t('iv_followup_status_unknown') }
function executionLabel(value: string) { return value === 'NOT_RUN' ? t('iv_followup_execution_not_started') : t('iv_followup_execution_unknown') }
function formatDate(value: string) { const d=new Date(value); return Number.isFinite(d.getTime()) ? d.toLocaleString(locale.value) : t('iv_not_set') }
function newIntentKey() {
  if (!globalThis.crypto?.randomUUID) throw new Error('secure random unavailable')
  return globalThis.crypto.randomUUID()
}
function startWrite(kind: WriteKind) { writingKind.value=kind; emit('write-busy',true); error.value='' }
function finishWrite(mine: number, key: string, signal: AbortSignal) {
  if (!current(mine,key,signal)) return
  writingKind.value=''; emit('write-busy',false)
}
async function refreshAfterWrite(mine: number, key: string, signal: AbortSignal) {
  try {
    const result=await props.refreshCase()
    if (!current(mine,key,signal) || !result || result.case.id !== props.caseRecord.id ||
      result.case.snapshot_id !== props.caseRecord.snapshot_id || result.case.snapshot_digest !== props.caseRecord.snapshot_digest) return false
    await nextTick()
    return current(mine,key,signal)
  } catch { return false }
}
async function capture() {
  if (!canCapture.value || controlsDisabled.value || !captureReason.value.trim() || captureReason.value.length > 500) return
  const mine=generation, key=props.lifetimeKey, signal=controller.signal
  startWrite('capture')
  try {
    const result=await createInterventionDataRevision(props.tenantId,props.caseRecord.id,props.token,signal,{
      expected_version:props.caseRecord.version,expected_snapshot_id:props.caseRecord.snapshot_id || '',
      expected_snapshot_digest:props.caseRecord.snapshot_digest,reason:captureReason.value.trim(),
    })
    if (!current(mine,key,signal)) return
    if (!validRevision(result?.revision) || result.revision.originating_case_version !== props.caseRecord.version ||
      !Number.isSafeInteger(result.case_version) || result.case_version !== props.caseRecord.version + 1 || result.case_version !== result.revision.case_version) {
      error.value=t('iv_followup_unknown'); reloadRequired.value=true; savedCapture.value=null; savedCaptureCaseVersion.value=null; return
    }
    savedCapture.value=result.revision; savedCaptureCaseVersion.value=result.case_version; captureReason.value=''
    clearHistory()
    const refreshed=await refreshAfterWrite(mine,key,signal)
    if (!refreshed) { error.value=t('iv_followup_refresh_failed'); reloadRequired.value=true; return }
    await loadInitial(mine,key,signal)
  } catch (err) {
    if (current(mine,key,signal)) { setSafeError(err,'capture'); if (reloadRequired.value) { savedCapture.value=null; savedCaptureCaseVersion.value=null } }
  } finally { finishWrite(mine,key,signal) }
}
async function requestRerun() {
  const latest=latestRevision.value
  if (!requestReady.value || !latest || !requestReason.value.trim() || requestReason.value.length > 500) return
  const mine=generation, key=props.lifetimeKey, signal=controller.signal
  startWrite('request')
  try {
    const requestKey=newIntentKey()
    const result=await createInterventionRerunRequest(props.tenantId,props.caseRecord.id,props.token,signal,{
      expected_version:props.caseRecord.version,expected_snapshot_id:props.caseRecord.snapshot_id || '',
      expected_snapshot_digest:props.caseRecord.snapshot_digest,data_revision_id:latest.id,
      data_revision_digest:latest.digest,reason:requestReason.value.trim(),request_key:requestKey,
    })
    if (!current(mine,key,signal)) return
    if (!validRequest(result?.request) || result.replayed !== false || !Number.isSafeInteger(result.case_version) ||
      result.case_version !== props.caseRecord.version + 1 || result.case_version !== result.request.case_version ||
      result.request.data_revision_id !== latest.id || result.request.data_revision_digest !== latest.digest) {
      error.value=t('iv_followup_unknown'); reloadRequired.value=true; savedRequest.value=null; return
    }
    savedRequest.value=result.request; requestReason.value=''
    const refreshed=await refreshAfterWrite(mine,key,signal)
    if (!refreshed) { error.value=t('iv_followup_refresh_failed'); reloadRequired.value=true; return }
    await readPage('request',mine,key,signal,true)
  } catch (err) {
    if (current(mine,key,signal)) setSafeError(err,'request')
  } finally { finishWrite(mine,key,signal) }
}
async function reloadExplicitly() {
  if (historyLoading.value || writing.value || parentBusy.value || !allowedToRead.value) return
  const mine=generation, key=props.lifetimeKey, signal=controller.signal
  error.value=''; reloadRequired.value=false; historyUnavailable=false; savedCapture.value=null; savedCaptureCaseVersion.value=null; savedRequest.value=null
  clearHistory(); captureReason.value=''; requestReason.value=''; historyLoading.value=true
  try {
    const refreshed=await props.refreshCase()
    if (!current(mine,key,signal)) return
    if (!refreshed || refreshed.case.id !== props.caseRecord.id || refreshed.case.snapshot_id !== props.caseRecord.snapshot_id || refreshed.case.snapshot_digest !== props.caseRecord.snapshot_digest) {
      error.value=t('iv_followup_refresh_failed'); reloadRequired.value=true; return
    }
    await Promise.all([readPage('revision',mine,key,signal,true),readPage('request',mine,key,signal,true)])
    if (!revisionLoaded.value || !requestLoaded.value) reloadRequired.value=true
  } catch { if (current(mine,key,signal)) { error.value=t('iv_followup_refresh_failed'); reloadRequired.value=true } }
  finally { if (current(mine,key,signal)) historyLoading.value=activeReads.size>0 }
}
function initialize() {
  if (writing.value) { writingKind.value=''; emit('write-busy',false) }
  generation++; active=true; historyUnavailable=false; activeReads.clear(); historyLoading.value=false; setupSignal(); clearState()
  if (!allowedToRead.value || controller.signal.aborted) return
  const mine=generation, key=props.lifetimeKey, signal=controller.signal
  void loadInitial(mine,key,signal)
}

watch(() => [props.lifetimeKey,props.signal,props.caseRecord.id,props.caseRecord.snapshot_id,props.caseRecord.snapshot_digest,props.role,props.canRead,props.canCapture,props.canRequest], initialize, {immediate:true})
onBeforeUnmount(() => { active=false; generation++; controller.abort(); detachExternal?.(); detachExternal=null; if (writing.value) emit('write-busy',false) })
</script>
