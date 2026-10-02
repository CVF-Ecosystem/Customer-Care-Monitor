<!--
  Nhật ký chi phí AI.
  CCMAI-UX-015: trình bày lại; endpoint và tham số lọc giữ nguyên. /cost-logs không trả tổng
  tiền theo bộ lọc, nên dòng tổng chỉ cộng trang đang xem và ghi rõ như vậy.
-->
<template>
  <div class="cl-page">
    <div class="cl-head">
      <h1 class="cl-title">{{ $t('cost_logs') }}</h1>
      <p v-if="loaded" class="cl-muted cl-m0" data-testid="cl-subtitle">
        {{ $t('lg_cost_sub', { n: formatNumber(total, uiLocale), rate: formatNumber(exchangeRate, uiLocale) }) }}
      </p>
    </div>

    <div class="cl-filters">
      <v-select
        v-model="filterProvider"
        :items="providerOptions"
        :label="$t('lg_provider')"
        variant="outlined"
        density="compact"
        hide-details
        class="cl-filter cl-filter--provider"
        data-testid="cl-provider"
        @update:model-value="onFilter"
      />
      <v-text-field
        v-model="dateFrom"
        type="date"
        :label="$t('lg_date_from')"
        variant="outlined"
        density="compact"
        hide-details
        class="cl-filter"
        @update:model-value="onFilter"
      />
      <v-text-field
        v-model="dateTo"
        type="date"
        :label="$t('lg_date_to')"
        variant="outlined"
        density="compact"
        hide-details
        class="cl-filter"
        @update:model-value="onFilter"
      />
      <span class="text-caption text-medium-emphasis" data-testid="vn-date-note">{{ $t('vn_date_note') }}</span>
    </div>

    <div v-if="loadError" class="cl-error" role="alert" data-testid="cl-load-error">
      <span>{{ $t('lg_cost_error') }}</span>
      <v-btn variant="outlined" color="error" height="44" @click="loadLogs">{{ $t('lg_retry') }}</v-btn>
    </div>

    <v-skeleton-loader v-else-if="!loaded" type="table-row@5" />

    <div v-else-if="!logs.length" class="cl-empty" data-testid="cl-empty">
      {{ hasFilter ? $t('lg_filtered_empty') : $t('lg_cost_empty') }}
    </div>

    <template v-else>
      <!-- Bảng desktop -->
      <div v-if="mdAndUp" class="cl-table-wrap">
        <v-table density="comfortable">
          <thead>
            <tr>
              <th>{{ $t('lg_col_time') }}</th>
              <th>{{ $t('lg_provider') }}</th>
              <th>{{ $t('lg_col_model') }}</th>
              <th class="cl-num">{{ $t('lg_col_in') }}</th>
              <th class="cl-num">{{ $t('lg_col_out') }}</th>
              <th class="cl-num">{{ $t('lg_col_usd') }}</th>
              <th class="cl-num">{{ $t('lg_col_vnd') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="log in logs" :key="log.id" data-testid="cl-row">
              <td class="cl-muted cl-small cl-nowrap">{{ formatDateTime(log.created_at, uiLocale) }}</td>
              <td>{{ providerLabel(log.provider) }}</td>
              <td class="cl-break">{{ log.model || '—' }}</td>
              <td class="cl-num">{{ formatNumber(log.input_tokens, uiLocale) }}</td>
              <td class="cl-num">{{ formatNumber(log.output_tokens, uiLocale) }}</td>
              <td class="cl-num">{{ usd(log.cost_usd) }}</td>
              <td class="cl-num">{{ vnd(log.cost_usd) }}</td>
            </tr>
          </tbody>
          <tfoot>
            <tr class="cl-total" data-testid="cl-total">
              <td colspan="5" class="cl-num">{{ $t('lg_page_total', { n: logs.length }) }}</td>
              <td class="cl-num">{{ usd(pageCostUSD) }}</td>
              <td class="cl-num">{{ vnd(pageCostUSD) }}</td>
            </tr>
          </tfoot>
        </v-table>
      </div>

      <!-- Thẻ mobile -->
      <div v-else class="cl-cards">
        <article v-for="log in logs" :key="log.id" class="cl-card" data-testid="cl-row">
          <div class="cl-card__top">
            <span class="cl-card__titles">
              <span class="font-weight-medium">{{ providerLabel(log.provider) }}</span>
              <span class="cl-muted cl-small cl-break">{{ log.model || '—' }}</span>
            </span>
            <span class="cl-muted cl-small cl-nowrap">{{ formatDateTime(log.created_at, uiLocale) }}</span>
          </div>
          <div class="cl-card__bottom">
            <span class="cl-muted cl-small">{{ $t('lg_tokens_line', { i: formatNumber(log.input_tokens, uiLocale), o: formatNumber(log.output_tokens, uiLocale) }) }}</span>
            <span class="cl-card__cost"><b>{{ vnd(log.cost_usd) }}</b><span class="cl-muted cl-small">{{ usd(log.cost_usd) }}</span></span>
          </div>
        </article>
        <div class="cl-card cl-total" data-testid="cl-total">
          <span>{{ $t('lg_page_total', { n: logs.length }) }}</span>
          <span class="cl-card__cost"><b>{{ vnd(pageCostUSD) }}</b><span class="cl-small">{{ usd(pageCostUSD) }}</span></span>
        </div>
      </div>

      <p v-if="total > logs.length" class="cl-muted cl-small cl-m0" data-testid="cl-total-note">
        {{ $t('lg_page_total_note', { n: formatNumber(total, uiLocale) }) }}
      </p>
    </template>

    <div v-if="totalPages > 1 && !loadError" class="cl-pager">
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
import { formatCurrency, formatDateTime, formatNumber, type UiLocale } from '../utils/format'

const route = useRoute()
const { t, locale } = useI18n()
const { mdAndUp } = useDisplay()
const uiLocale = computed(() => locale.value as UiLocale)
const tenantId = computed(() => route.params.tenantId as string)

const logs = ref<any[]>([])
const page = ref(1)
const total = ref(0)
const perPage = 20
const exchangeRate = ref(26000)
const filterProvider = ref<string | null>('')
const dateFrom = ref('')
const dateTo = ref('')
const loaded = ref(false)
const loadError = ref(false)
const totalPages = computed(() => Math.ceil(total.value / perPage))
const hasFilter = computed(() => !!(filterProvider.value || dateFrom.value || dateTo.value))
// Chỉ cộng các dòng của trang đang xem: máy chủ không trả tổng theo bộ lọc.
const pageCostUSD = computed(() => logs.value.reduce((sum: number, l: any) => sum + (Number(l.cost_usd) || 0), 0))
let seq = 0

// Giá trị provider đúng như máy chủ ghi; nhãn theo tên sản phẩm (giống màn Cài đặt).
const PROVIDER_LABEL: Record<string, string> = { claude: 'Claude', gemini: 'Gemini', openai: 'ChatGPT', xai: 'Grok' }
const providerOptions = computed(() => [
  { title: t('lg_all'), value: '' },
  ...Object.entries(PROVIDER_LABEL).map(([value, title]) => ({ title, value })),
])

onMounted(loadLogs)
watch(page, loadLogs)

function onFilter() {
  if (page.value !== 1) page.value = 1
  else loadLogs()
}

async function loadLogs() {
  const mine = ++seq
  loadError.value = false
  const params: Record<string, any> = { page: page.value, per_page: perPage }
  if (filterProvider.value) params.provider = filterProvider.value
  if (dateFrom.value) params.from = dateFrom.value
  if (dateTo.value) params.to = dateTo.value
  try {
    const { data } = await api.get(`/tenants/${tenantId.value}/cost-logs`, { params })
    if (mine !== seq) return
    logs.value = Array.isArray(data?.data) ? data.data : []
    total.value = data?.total || 0
    exchangeRate.value = data?.exchange_rate || 26000
    loaded.value = true
  } catch {
    if (mine !== seq) return
    loadError.value = true
  }
}

function providerLabel(p: string) {
  return PROVIDER_LABEL[p] || p || '—'
}

// Chi phí mỗi lần gọi thường dưới 1 xu nên US$ hiển thị 4 chữ số thập phân.
function usd(v: number | null | undefined) {
  const amount = formatNumber(Number(v) || 0, uiLocale.value, 4)
  return uiLocale.value === 'vi' ? `${amount} US$` : `$${amount}`
}

function vnd(v: number | null | undefined) {
  return formatCurrency((Number(v) || 0) * exchangeRate.value, 'VND', uiLocale.value)
}
</script>

<style scoped>
.cl-page {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.cl-title {
  margin: 0;
  font-size: 26px;
  font-weight: 700;
}
.cl-muted {
  color: rgb(var(--v-theme-text-muted));
}
.cl-m0 {
  margin: 0;
}
.cl-small {
  font-size: 13px;
}
.cl-nowrap {
  white-space: nowrap;
}
.cl-break {
  overflow-wrap: anywhere;
}
.cl-filters {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}
.cl-filter {
  flex: 0 1 180px;
  min-width: 150px;
}
.cl-filter--provider {
  flex-basis: 220px;
}
@media (max-width: 599px) {
  .cl-filter {
    flex: 1 1 0;
    min-width: 0;
  }
  .cl-filter--provider {
    flex-basis: 100%;
  }
}
.cl-card__titles {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.cl-table-wrap {
  overflow-x: auto;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 12px;
  background: rgb(var(--v-theme-surface));
}
.cl-table-wrap :deep(th) {
  font-size: 12px !important;
  font-weight: 600 !important;
  color: rgb(var(--v-theme-text-muted)) !important;
  white-space: nowrap;
}
.cl-table-wrap :deep(td) {
  height: auto !important;
  padding-top: 10px !important;
  padding-bottom: 10px !important;
  font-size: 14px;
}
.cl-num {
  text-align: right !important;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}
.cl-total {
  font-weight: 700;
}
.cl-table-wrap :deep(tfoot td) {
  border-top: 2px solid rgb(var(--v-theme-border));
}
.cl-cards {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.cl-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px 14px;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 12px;
  background: rgb(var(--v-theme-surface));
}
.cl-card.cl-total {
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.cl-card__top,
.cl-card__bottom {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
}
.cl-card__cost {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}
.cl-empty {
  padding: 32px 16px;
  border: 1px dashed rgb(var(--v-theme-border));
  border-radius: 12px;
  text-align: center;
  color: rgb(var(--v-theme-text-muted));
}
.cl-error {
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
.cl-pager {
  display: flex;
  justify-content: center;
}
</style>
