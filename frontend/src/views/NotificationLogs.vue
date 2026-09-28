<!--
  Lịch sử thông báo đã gửi.
  CCMAI-UX-015: trình bày lại; endpoint và phân trang giữ nguyên. Lỗi tải không còn bị hiển thị
  như "chưa có thông báo".
-->
<template>
  <div class="nl-page">
    <div class="nl-head">
      <h1 class="nl-title">{{ $t('nav_notification_logs') }}</h1>
      <p v-if="loaded" class="nl-muted nl-m0" data-testid="nl-subtitle">{{ $t('lg_notif_sub', { n: formatNumber(total, uiLocale) }) }}</p>
    </div>

    <div v-if="loadError" class="nl-error" role="alert" data-testid="nl-load-error">
      <span>{{ $t('lg_notif_error') }}</span>
      <v-btn variant="outlined" color="error" height="44" data-testid="nl-retry" @click="loadLogs">{{ $t('lg_retry') }}</v-btn>
    </div>

    <v-skeleton-loader v-else-if="!loaded" type="list-item-two-line@4" />

    <div v-else-if="!logs.length" class="nl-empty" data-testid="nl-empty">
      <v-icon size="40" class="nl-muted" aria-hidden="true">mdi-bell-outline</v-icon>
      <span>{{ $t('lg_notif_empty') }}</span>
    </div>

    <!-- Bảng desktop -->
    <div v-else-if="mdAndUp" class="nl-table-wrap">
      <v-table density="comfortable">
        <thead>
          <tr>
            <th class="nl-col-time">{{ $t('lg_col_time') }}</th>
            <th class="nl-col-channel">{{ $t('lg_col_channel') }}</th>
            <th>{{ $t('lg_col_recipient') }}</th>
            <th class="nl-col-status">{{ $t('lg_col_status') }}</th>
            <th class="nl-col-toggle"><span class="d-sr-only">{{ $t('lg_col_content') }}</span></th>
          </tr>
        </thead>
        <tbody>
          <template v-for="log in logs" :key="log.id">
            <tr data-testid="nl-row">
              <td class="nl-muted nl-small nl-nowrap">{{ formatDateTime(log.sent_at, uiLocale) }}</td>
              <td>{{ channelLabel(log.channel_type) }}</td>
              <td class="nl-break">
                {{ log.recipient }}
                <div v-if="log.error_message" class="nl-err-text">{{ $t('lg_error_line', { msg: log.error_message }) }}</div>
              </td>
              <td><span class="nl-chip" :class="statusClass(log.status)" data-testid="nl-status">{{ statusLabel(log.status) }}</span></td>
              <td class="nl-col-toggle">
                <v-btn
                  variant="text"
                  color="primary"
                  height="40"
                  :aria-expanded="expandedId === log.id ? 'true' : 'false'"
                  :aria-controls="`nl-body-${log.id}`"
                  data-testid="nl-toggle"
                  @click="toggle(log.id)"
                >{{ expandedId === log.id ? $t('lg_hide_body') : $t('lg_show_body') }}</v-btn>
              </td>
            </tr>
            <tr v-if="expandedId === log.id" class="nl-body-row">
              <td colspan="5">
                <div :id="`nl-body-${log.id}`" class="nl-body-wrap">
                  <div v-if="log.subject" class="nl-muted nl-small">{{ $t('lg_subject', { s: log.subject }) }}</div>
                  <div class="nl-body" data-testid="nl-body">{{ log.body }}</div>
                </div>
              </td>
            </tr>
          </template>
        </tbody>
      </v-table>
    </div>

    <!-- Thẻ mobile -->
    <div v-else class="nl-cards">
      <article
        v-for="log in logs"
        :key="log.id"
        class="nl-card"
        :class="{ 'nl-card--failed': log.status === 'failed' }"
        data-testid="nl-row"
      >
        <div class="nl-card__top">
          <span class="nl-chip" :class="statusClass(log.status)" data-testid="nl-status">{{ statusLabel(log.status) }}</span>
          <span class="nl-strong">{{ channelLabel(log.channel_type) }}</span>
          <span class="nl-grow" />
          <span class="nl-muted nl-small">{{ formatDateTime(log.sent_at, uiLocale) }}</span>
        </div>
        <span class="nl-muted nl-small nl-break">{{ $t('lg_recipient_line', { r: log.recipient }) }}</span>
        <span v-if="log.error_message" class="nl-err-text">{{ $t('lg_error_line', { msg: log.error_message }) }}</span>
        <v-btn
          variant="text"
          color="primary"
          height="40"
          class="nl-toggle"
          :aria-expanded="expandedId === log.id ? 'true' : 'false'"
          :aria-controls="`nl-body-${log.id}`"
          data-testid="nl-toggle"
          @click="toggle(log.id)"
        >{{ expandedId === log.id ? $t('lg_hide_body') : $t('lg_show_body') }}</v-btn>
        <div v-if="expandedId === log.id" :id="`nl-body-${log.id}`" class="nl-body-wrap">
          <div v-if="log.subject" class="nl-muted nl-small">{{ $t('lg_subject', { s: log.subject }) }}</div>
          <div class="nl-body" data-testid="nl-body">{{ log.body }}</div>
        </div>
      </article>
    </div>

    <div v-if="totalPages > 1 && !loadError" class="nl-pager">
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
const { t, locale } = useI18n()
const { mdAndUp } = useDisplay()
const uiLocale = computed(() => locale.value as UiLocale)
const tenantId = computed(() => route.params.tenantId as string)

const logs = ref<any[]>([])
const expandedId = ref('')
const page = ref(1)
const total = ref(0)
const perPage = 20
const loaded = ref(false)
const loadError = ref(false)
const totalPages = computed(() => Math.ceil(total.value / perPage))
let seq = 0

async function loadLogs() {
  const mine = ++seq
  loadError.value = false
  try {
    const { data } = await api.get(`/tenants/${tenantId.value}/notification-logs`, { params: { page: page.value, per_page: perPage } })
    if (mine !== seq) return
    logs.value = Array.isArray(data?.data) ? data.data : []
    total.value = data?.total || 0
    loaded.value = true
  } catch {
    if (mine !== seq) return
    loadError.value = true
  }
}

onMounted(loadLogs)
watch(page, loadLogs)

function toggle(id: string) {
  expandedId.value = expandedId.value === id ? '' : id
}

function channelLabel(type: string) {
  if (type === 'telegram') return 'Telegram'
  if (type === 'email') return 'Email'
  return type || '—'
}

function statusLabel(status: string) {
  if (status === 'sent') return t('lg_sent')
  if (status === 'failed') return t('lg_failed')
  return status || '—'
}

function statusClass(status: string) {
  if (status === 'sent') return 'nl-chip--ok'
  if (status === 'failed') return 'nl-chip--bad'
  return 'nl-chip--neutral'
}
</script>

<style scoped>
.nl-page {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.nl-title {
  margin: 0;
  font-size: 26px;
  font-weight: 700;
}
.nl-muted {
  color: rgb(var(--v-theme-text-muted));
}
.nl-m0 {
  margin: 0;
}
.nl-small {
  font-size: 13px;
}
.nl-strong {
  font-size: 13px;
  font-weight: 600;
}
.nl-nowrap {
  white-space: nowrap;
}
.nl-break {
  overflow-wrap: anywhere;
}
.nl-grow {
  flex-grow: 1;
}
.nl-table-wrap {
  overflow-x: auto;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 12px;
  background: rgb(var(--v-theme-surface));
}
.nl-table-wrap :deep(th) {
  font-size: 12px !important;
  font-weight: 600 !important;
  color: rgb(var(--v-theme-text-muted)) !important;
}
.nl-table-wrap :deep(td) {
  height: auto !important;
  padding-top: 8px !important;
  padding-bottom: 8px !important;
  font-size: 14px;
}
.nl-col-time {
  width: 150px;
}
.nl-col-channel {
  width: 120px;
}
.nl-col-status {
  width: 120px;
}
.nl-col-toggle {
  width: 150px;
  text-align: right;
}
.nl-chip {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  padding: 2px 8px;
  border-radius: 8px;
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
}
.nl-chip--ok {
  background: rgb(var(--v-theme-pass-bg));
  color: rgb(var(--v-theme-pass));
}
.nl-chip--bad {
  background: rgb(var(--v-theme-fail-bg));
  color: rgb(var(--v-theme-fail));
}
.nl-chip--neutral {
  background: rgb(var(--v-theme-skip-bg));
  color: rgb(var(--v-theme-skip));
}
.nl-err-text {
  color: rgb(var(--v-theme-fail));
  font-size: 13px;
  overflow-wrap: anywhere;
}
.nl-body-row > td {
  background: rgb(var(--v-theme-background));
}
.nl-body-wrap {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 4px 0 8px;
}
.nl-body {
  padding: 10px 12px;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 8px;
  background: rgb(var(--v-theme-surface));
  font-size: 13px;
  line-height: 1.5;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
.nl-cards {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.nl-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px 14px;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 12px;
  background: rgb(var(--v-theme-surface));
}
.nl-card--failed {
  border-left: 4px solid rgb(var(--v-theme-fail));
}
.nl-card__top {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.nl-card .nl-body {
  background: rgb(var(--v-theme-background));
}
.nl-toggle {
  align-self: flex-start;
  margin-left: -10px;
}
.nl-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 32px 16px;
  border: 1px dashed rgb(var(--v-theme-border));
  border-radius: 12px;
  text-align: center;
  color: rgb(var(--v-theme-text-muted));
}
.nl-error {
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
.nl-pager {
  display: flex;
  justify-content: center;
}
</style>
