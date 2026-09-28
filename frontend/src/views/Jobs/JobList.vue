<!--
  Danh sách tác vụ AI.
  CCMAI-UX-013: trình bày lại trên dữ liệu GET /jobs sẵn có (loại, lịch chạy bằng lời,
  số kênh, trạng thái, lần chạy cuối). Xóa luôn qua hộp xác nhận nêu rõ phần bị xóa kèm.
-->
<template>
  <div class="jl-page">
    <div class="jl-head">
      <div class="jl-head__titles">
        <h1 class="jl-title">{{ $t('jl_title') }}</h1>
        <p class="jl-muted jl-m0" data-testid="jl-subtitle">{{ $t('jl_subtitle', { n: jobStore.jobs.length }) }}</p>
      </div>
      <v-btn v-if="authStore.canEdit('jobs')" color="primary" :height="mdAndUp ? 40 : 44" prepend-icon="mdi-plus" :to="`/${tenantId}/jobs/create`">
        {{ $t('jl_create') }}
      </v-btn>
    </div>

    <div v-if="loadError" class="jl-error" role="alert" data-testid="jl-load-error">
      <span>{{ $t('jl_load_error') }}</span>
      <v-btn variant="outlined" color="error" height="44" @click="load">{{ $t('jl_retry') }}</v-btn>
    </div>

    <v-skeleton-loader v-else-if="loading && !jobStore.jobs.length" type="table-row@3" />

    <div v-else-if="!jobStore.jobs.length" class="jl-empty" data-testid="jl-empty">
      <h2 class="jl-empty__title">{{ $t('jl_empty_title') }}</h2>
      <p class="jl-muted jl-m0">{{ $t('jl_empty_desc') }}</p>
      <v-btn color="primary" height="44" prepend-icon="mdi-plus" :to="`/${tenantId}/jobs/create`">{{ $t('jl_create') }}</v-btn>
    </div>

    <!-- Bảng desktop -->
    <div v-else-if="mdAndUp" class="jl-table-wrap">
      <v-table density="comfortable">
        <thead>
          <tr>
            <th>{{ $t('jl_col_job') }}</th>
            <th>{{ $t('jl_col_type') }}</th>
            <th>{{ $t('jl_col_schedule') }}</th>
            <th>{{ $t('jl_col_state') }}</th>
            <th>{{ $t('jl_col_last_run') }}</th>
            <th class="jl-col-actions"><span class="d-sr-only">{{ $t('actions') }}</span></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="job in jobStore.jobs" :key="job.id" data-testid="jl-row">
            <td class="jl-cell-main">
              <router-link :to="`/${tenantId}/jobs/${job.id}`" class="jl-name">{{ job.name }}</router-link>
              <span v-if="job.description" class="jl-muted jl-small">{{ job.description }}</span>
            </td>
            <td><span class="jl-type">{{ typeLabel(job.job_type) }}</span></td>
            <td>
              <span class="d-block jl-nowrap" data-testid="jl-schedule">{{ scheduleLabel(job) }}</span>
              <span class="jl-muted jl-small">{{ channelsLabel(job) }}</span>
            </td>
            <td class="jl-nowrap"><span class="jl-state" :class="{ 'jl-state--off': !job.is_active }">{{ job.is_active ? $t('jl_active') : $t('jl_inactive') }}</span></td>
            <td>
              <span class="d-flex align-center flex-wrap ga-2">
                <span class="jl-run" :class="`jl-run--${runKind(job)}`" data-testid="jl-run">{{ runLabel(job) }}</span>
                <span v-if="job.last_run_at" class="jl-muted jl-small tabular-nums" data-testid="jl-run-time">{{ fmt(job.last_run_at) }}</span>
              </span>
            </td>
            <td class="jl-col-actions">
              <ActionMenu :items="menuItems" :label="$t('jl_more', { name: job.name })" @select="onAction($event, job)" />
            </td>
          </tr>
        </tbody>
      </v-table>
    </div>

    <!-- Thẻ mobile -->
    <div v-else class="jl-cards">
      <article v-for="job in jobStore.jobs" :key="job.id" class="jl-card" data-testid="jl-row">
        <div class="jl-card__top">
          <div class="jl-card__titles">
            <router-link :to="`/${tenantId}/jobs/${job.id}`" class="jl-name">{{ job.name }}</router-link>
            <span class="jl-muted jl-small">{{ typeLabel(job.job_type) }} · {{ channelsLabel(job) }}</span>
          </div>
          <ActionMenu :items="menuItems" :label="$t('jl_more', { name: job.name })" @select="onAction($event, job)" />
        </div>
        <span class="jl-small-body">
          <span data-testid="jl-schedule">{{ scheduleLabel(job) }}</span> ·
          <span class="jl-state" :class="{ 'jl-state--off': !job.is_active }">{{ job.is_active ? $t('jl_active') : $t('jl_inactive') }}</span>
        </span>
        <span class="d-flex align-center flex-wrap ga-2">
          <span class="jl-run" :class="`jl-run--${runKind(job)}`" data-testid="jl-run">{{ runLabel(job) }}</span>
          <span v-if="job.last_run_at" class="jl-muted jl-small tabular-nums">{{ fmt(job.last_run_at) }}</span>
        </span>
      </article>
    </div>

    <ConfirmDialog
      v-model="confirmOpen"
      :title="$t('jl_confirm_title', { name: pendingDelete?.name ?? '' })"
      :message="$t('jl_confirm_msg')"
      :confirm-label="$t('jl_confirm_ok')"
      :loading="deleting"
      @confirm="doDelete"
    />

    <v-snackbar v-model="snackbar" :color="snackError ? 'error' : undefined" timeout="4000">{{ snackText }}</v-snackbar>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useDisplay } from 'vuetify'
import { useI18n } from 'vue-i18n'
import { useJobStore, type Job } from '../../stores/jobs'
import { useAuthStore } from '../../stores/auth'
import ActionMenu from '../../components/ui/ActionMenu.vue'
import ConfirmDialog from '../../components/ui/ConfirmDialog.vue'
import type { ActionMenuItem } from '../../components/ui/types'
import { runStatusKind } from './job-detail/logic'
import { channelCount, describeSchedule } from './job-list/schedule'
import { formatDateTime, type UiLocale } from '../../utils/format'

const route = useRoute()
const router = useRouter()
const { t, locale } = useI18n()
const { mdAndUp } = useDisplay()
const jobStore = useJobStore()
const authStore = useAuthStore()
const tenantId = computed(() => route.params.tenantId as string)

const loading = ref(false)
const loadError = ref(false)
const confirmOpen = ref(false)
const deleting = ref(false)
const pendingDelete = ref<Job | null>(null)
const snackbar = ref(false)
const snackError = ref(false)
const snackText = ref('')

// Sửa/Xóa hiển thị như trước (không đổi phân quyền trong tranche giao diện).
const menuItems = computed<ActionMenuItem[]>(() => [
  { key: 'edit', label: t('jl_action_edit'), icon: 'mdi-pencil-outline' },
  { key: 'delete', label: t('jl_action_delete'), icon: 'mdi-delete-outline', danger: true },
])

async function load() {
  loading.value = true
  loadError.value = false
  try {
    await jobStore.fetchJobs(tenantId.value)
  } catch {
    loadError.value = true
  } finally {
    loading.value = false
  }
}

onMounted(load)

function fmt(s: string | null) {
  return formatDateTime(s, locale.value as UiLocale)
}

function typeLabel(jobType: string) {
  return jobType === 'classification' ? t('jl_type_class') : t('jl_type_qc')
}

function scheduleLabel(job: Job) {
  const s = describeSchedule(job.schedule_type, job.schedule_cron)
  switch (s.kind) {
    case 'after_sync': return t('jl_sched_after_sync')
    case 'manual': return t('jl_sched_manual')
    case 'daily': return t('jl_sched_daily', { time: s.time })
    case 'weekly': return t('jl_sched_weekly', { days: s.days.map((d) => t(`jl_day_${d}`)).join(', '), time: s.time })
    case 'monthly': return t('jl_sched_monthly', { day: s.day, time: s.time })
    default: return t('jl_sched_raw', { cron: s.cron || '—' })
  }
}

function channelsLabel(job: Job) {
  const n = channelCount(job.input_channel_ids)
  return n === null ? t('jl_channels_unknown') : t('jl_channels', { n })
}

// Không có lần chạy nào → "Chưa chạy"; còn lại dùng đúng nhãn lần chạy của Job Detail (UX-010).
function runKind(job: Job) {
  if (!job.last_run_at && !job.last_run_status) return 'never'
  return runStatusKind(job.last_run_status)
}
function runLabel(job: Job) {
  const k = runKind(job)
  return k === 'never' ? t('jl_never_run') : t(`jd_run_${k}`)
}

function onAction(key: string, job: Job) {
  if (key === 'edit') {
    router.push(`/${tenantId.value}/jobs/${job.id}/edit`)
    return
  }
  if (key === 'delete') {
    pendingDelete.value = job
    confirmOpen.value = true
  }
}

async function doDelete() {
  if (!pendingDelete.value) return
  deleting.value = true
  try {
    await jobStore.deleteJob(tenantId.value, pendingDelete.value.id)
    confirmOpen.value = false
    snackText.value = t('jl_deleted')
    snackError.value = false
  } catch {
    confirmOpen.value = false
    snackText.value = t('jl_delete_error')
    snackError.value = true
  } finally {
    deleting.value = false
    snackbar.value = true
    pendingDelete.value = null
  }
}
</script>

<style scoped>
.jl-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.jl-head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 12px;
}
.jl-head__titles {
  flex: 1 1 260px;
}
.jl-title {
  margin: 0;
  font-size: 26px;
  font-weight: 700;
  line-height: 1.25;
}
.jl-muted {
  color: rgb(var(--v-theme-text-muted));
}
.jl-m0 {
  margin: 0;
}
.jl-small {
  font-size: 12px;
}
.jl-small-body {
  font-size: 13px;
}
.jl-table-wrap {
  overflow-x: auto;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 12px;
  background: rgb(var(--v-theme-surface));
}
.jl-table-wrap :deep(th) {
  font-size: 12px !important;
  font-weight: 600 !important;
  color: rgb(var(--v-theme-text-muted)) !important;
  white-space: nowrap;
}
.jl-table-wrap :deep(td) {
  height: auto !important;
  padding-top: 12px !important;
  padding-bottom: 12px !important;
  font-size: 14px;
}
.jl-nowrap {
  white-space: nowrap;
}
.jl-col-actions {
  width: 56px;
  text-align: right;
}
.jl-cell-main {
  min-width: 220px;
}
.jl-cell-main > * {
  display: block;
}
.jl-name {
  color: rgb(var(--v-theme-primary));
  font-weight: 600;
  text-decoration: none;
}
.jl-name:hover {
  text-decoration: underline;
}
.jl-type {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 8px;
  border-radius: 8px;
  background: rgba(var(--v-theme-primary), 0.1);
  color: rgb(var(--v-theme-primary));
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
}
.jl-state {
  font-weight: 600;
  color: rgb(var(--v-theme-pass));
}
.jl-state--off {
  color: rgb(var(--v-theme-text-muted));
}
.jl-run {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 8px;
  border-radius: 8px;
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
}
.jl-run--success {
  background: rgb(var(--v-theme-pass-bg));
  color: rgb(var(--v-theme-pass));
}
.jl-run--partial,
.jl-run--running {
  background: rgb(var(--v-theme-src-changed-bg));
  color: rgb(var(--v-theme-src-changed));
}
.jl-run--error {
  background: rgb(var(--v-theme-fail-bg));
  color: rgb(var(--v-theme-fail));
}
.jl-run--cancelled,
.jl-run--unknown,
.jl-run--never {
  background: rgb(var(--v-theme-skip-bg));
  color: rgb(var(--v-theme-skip));
}
.jl-cards {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.jl-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px 8px 14px 16px;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 12px;
  background: rgb(var(--v-theme-surface));
}
.jl-card__top {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}
.jl-card__titles {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding-top: 4px;
}
.jl-error {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 12px;
  padding: 14px 16px;
  border: 1px solid rgb(var(--v-theme-fail));
  border-radius: 12px;
  background: rgb(var(--v-theme-fail-bg));
  color: rgb(var(--v-theme-fail));
  font-size: 14px;
  font-weight: 600;
}
.jl-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 28px 20px;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 12px;
  background: rgb(var(--v-theme-surface));
  text-align: center;
  font-size: 14px;
}
.jl-empty__title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}
</style>
