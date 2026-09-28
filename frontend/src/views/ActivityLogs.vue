<!--
  Nhật ký hệ thống.
  CCMAI-UX-015: trình bày lại; endpoint, tham số lọc (tiền tố action) và cỡ trang giữ nguyên.
  Chi tiết và lỗi do máy chủ ghi nên hiển thị nguyên văn.
-->
<template>
  <div class="lg-page">
    <div class="lg-head">
      <h1 class="lg-title">{{ $t('activity_logs') }}</h1>
      <p v-if="loaded" class="lg-muted lg-m0" data-testid="lg-subtitle">{{ $t('lg_events_sub', { n: formatNumber(total, uiLocale) }) }}</p>
    </div>

    <v-select
      v-model="filterAction"
      :items="actionOptions"
      :label="$t('lg_event_type')"
      variant="outlined"
      density="compact"
      hide-details
      class="lg-filter"
      data-testid="lg-filter"
      @update:model-value="onFilter"
    />

    <div v-if="loadError" class="lg-error" role="alert" data-testid="lg-load-error">
      <span>{{ $t('lg_events_error') }}</span>
      <v-btn variant="outlined" color="error" height="44" @click="loadLogs">{{ $t('lg_retry') }}</v-btn>
    </div>

    <v-skeleton-loader v-else-if="!loaded" type="table-row@5" />

    <div v-else-if="!logs.length" class="lg-empty" data-testid="lg-empty">
      {{ filterAction ? $t('lg_filtered_empty') : $t('lg_events_empty') }}
    </div>

    <!-- Bảng desktop -->
    <div v-else-if="mdAndUp" class="lg-table-wrap">
      <v-table density="comfortable">
        <thead>
          <tr>
            <th class="lg-col-time">{{ $t('lg_col_time') }}</th>
            <th class="lg-col-event">{{ $t('lg_col_event') }}</th>
            <th class="lg-col-actor">{{ $t('lg_col_actor') }}</th>
            <th>{{ $t('lg_col_detail') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="log in logs" :key="log.id" data-testid="lg-row">
            <td class="lg-muted lg-small lg-nowrap">{{ formatDateTime(log.created_at, uiLocale) }}</td>
            <td><span class="lg-chip" :class="`lg-chip--${actionTone(log.action)}`" data-testid="lg-action">{{ actionLabel(log.action) }}</span></td>
            <td class="lg-muted lg-break" data-testid="lg-actor">{{ actorLabel(log.user_email) }}</td>
            <td>
              <div class="lg-detail">
                <span>{{ log.detail || '—' }}</span>
                <span v-if="log.error_message" class="lg-err-text">{{ $t('lg_error_line', { msg: log.error_message }) }}</span>
              </div>
            </td>
          </tr>
        </tbody>
      </v-table>
    </div>

    <!-- Thẻ mobile -->
    <div v-else class="lg-cards">
      <article v-for="log in logs" :key="log.id" class="lg-card" data-testid="lg-row">
        <div class="lg-card__top">
          <span class="lg-chip" :class="`lg-chip--${actionTone(log.action)}`" data-testid="lg-action">{{ actionLabel(log.action) }}</span>
          <span class="lg-muted lg-small">{{ formatDateTime(log.created_at, uiLocale) }}</span>
        </div>
        <span class="lg-muted lg-small lg-break" data-testid="lg-actor">{{ actorLabel(log.user_email) }}</span>
        <span class="lg-detail-text">{{ log.detail || '—' }}</span>
        <span v-if="log.error_message" class="lg-err-text">{{ $t('lg_error_line', { msg: log.error_message }) }}</span>
      </article>
    </div>

    <div v-if="totalPages > 1 && !loadError" class="lg-pager">
      <v-pagination v-model="page" :length="totalPages" :total-visible="mdAndUp ? 7 : 3" density="comfortable" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useDisplay } from 'vuetify'
import { useI18n } from 'vue-i18n'
import api from '../api'
import { formatDateTime, formatNumber, type UiLocale } from '../utils/format'

const route = useRoute()
const { t, te, locale } = useI18n()
const { mdAndUp } = useDisplay()
const uiLocale = computed(() => locale.value as UiLocale)
const tenantId = computed(() => route.params.tenantId as string)

const logs = ref<any[]>([])
const page = ref(1)
const total = ref(0)
const perPage = 20
const filterAction = ref<string | null>('')
const loaded = ref(false)
const loadError = ref(false)
const totalPages = computed(() => Math.ceil(total.value / perPage))
let seq = 0

// Giá trị là tiền tố action mà máy chủ lọc bằng LIKE; chỉ liệt kê nhóm thực sự được ghi.
const actionOptions = computed(() => [
  { title: t('lg_all'), value: '' },
  { title: t('lg_f_job_run'), value: 'job.run' },
  { title: t('lg_f_job_delete'), value: 'job.delete' },
  { title: t('lg_f_job_clear'), value: 'job.clear' },
  { title: t('lg_f_channel'), value: 'channel' },
  { title: t('lg_f_sync'), value: 'sync' },
  { title: t('lg_f_notification'), value: 'notification' },
  { title: t('lg_f_login'), value: 'user.login' },
  { title: t('lg_f_settings'), value: 'settings' },
])

onMounted(loadLogs)
watch(page, loadLogs)

// Đổi bộ lọc thì quay về trang 1 (trước đây vẫn giữ trang cũ của bộ lọc trước).
function onFilter() {
  if (page.value !== 1) page.value = 1
  else loadLogs()
}

async function loadLogs() {
  const mine = ++seq
  loadError.value = false
  const params: Record<string, any> = { page: page.value, per_page: perPage }
  if (filterAction.value) params.action = filterAction.value
  try {
    const { data } = await api.get(`/tenants/${tenantId.value}/activity-logs`, { params })
    if (mine !== seq) return
    logs.value = Array.isArray(data?.data) ? data.data : []
    total.value = data?.total || 0
    loaded.value = true
  } catch {
    if (mine !== seq) return
    loadError.value = true
  }
}

function actionLabel(action: string) {
  const key = `lg_act_${String(action || '').replace(/\./g, '_')}`
  return te(key) ? t(key) : action
}

function actionTone(action: string) {
  if (/error|fail/.test(action)) return 'bad'
  if (/delete|clear|purge/.test(action)) return 'warn'
  return 'info'
}

function actorLabel(email: string | null | undefined) {
  return !email || email === 'system' ? t('lg_system') : email
}
</script>

<style scoped>
.lg-page {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.lg-title {
  margin: 0;
  font-size: 26px;
  font-weight: 700;
}
.lg-muted {
  color: rgb(var(--v-theme-text-muted));
}
.lg-m0 {
  margin: 0;
}
.lg-small {
  font-size: 13px;
}
.lg-nowrap {
  white-space: nowrap;
}
.lg-break {
  overflow-wrap: anywhere;
}
.lg-filter {
  max-width: 280px;
}
.lg-table-wrap {
  overflow-x: auto;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 12px;
  background: rgb(var(--v-theme-surface));
}
.lg-table-wrap :deep(th) {
  font-size: 12px !important;
  font-weight: 600 !important;
  color: rgb(var(--v-theme-text-muted)) !important;
}
.lg-table-wrap :deep(td) {
  height: auto !important;
  padding-top: 10px !important;
  padding-bottom: 10px !important;
  font-size: 14px;
  vertical-align: top;
}
.lg-col-time {
  width: 150px;
}
.lg-col-event {
  width: 220px;
}
.lg-col-actor {
  width: 220px;
}
.lg-detail {
  display: flex;
  flex-direction: column;
  gap: 4px;
  overflow-wrap: anywhere;
}
.lg-detail-text {
  font-size: 14px;
  overflow-wrap: anywhere;
}
.lg-err-text {
  color: rgb(var(--v-theme-fail));
  font-size: 13px;
  overflow-wrap: anywhere;
}
.lg-chip {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  padding: 2px 8px;
  border-radius: 8px;
  font-size: 12px;
  font-weight: 600;
}
.lg-chip--info {
  background: rgb(var(--v-theme-skip-bg));
  color: rgb(var(--v-theme-on-surface));
}
.lg-chip--warn {
  background: rgb(var(--v-theme-src-changed-bg));
  color: rgb(var(--v-theme-src-changed));
}
.lg-chip--bad {
  background: rgb(var(--v-theme-fail-bg));
  color: rgb(var(--v-theme-fail));
}
.lg-cards {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.lg-card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 12px 14px;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 12px;
  background: rgb(var(--v-theme-surface));
}
.lg-card__top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 8px;
}
.lg-empty {
  padding: 32px 16px;
  border: 1px dashed rgb(var(--v-theme-border));
  border-radius: 12px;
  text-align: center;
  color: rgb(var(--v-theme-text-muted));
}
.lg-error {
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
.lg-pager {
  display: flex;
  justify-content: center;
}
</style>
