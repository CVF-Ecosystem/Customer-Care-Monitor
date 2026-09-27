<!--
  CCMAI-UX-000: static design-system preview. No API call, synthetic data only.
  Query `?dialog=review` or `?dialog=confirm` opens a dialog on load (for screenshots).
-->
<template>
  <v-app>
    <v-main class="ds">
      <div class="ds__wrap">
        <header class="ds__header">
          <div>
            <h1 class="ds__h1">Design system</h1>
            <p class="ds__muted">CCMAI-UX-000 · {{ isDark ? 'dark' : 'light' }} · dữ liệu tổng hợp</p>
          </div>
          <v-btn variant="outlined" :prepend-icon="isDark ? 'mdi-weather-sunny' : 'mdi-weather-night'" @click="toggleTheme">
            {{ isDark ? 'Light' : 'Dark' }}
          </v-btn>
        </header>

        <section class="ds__section">
          <h2 class="ds__h2">Màu</h2>
          <div class="ds__swatches">
            <div v-for="name in swatchNames" :key="name" class="ds__swatch">
              <span class="ds__swatch-color" :style="{ background: `rgb(var(--v-theme-${name}))` }" />
              <span class="ds__swatch-name">{{ name }}</span>
            </div>
          </div>
        </section>

        <section class="ds__section">
          <h2 class="ds__h2">Chữ — Be Vietnam Pro</h2>
          <p style="font-size: 30px; font-weight: 700; margin: 0">Tiêu đề trang 30 — Kết quả đánh giá</p>
          <p style="font-size: 20px; font-weight: 600; margin: 4px 0 0">Tiêu đề phụ 20 — Hội thoại cần xem lại</p>
          <p style="font-size: 16px; font-weight: 600; margin: 4px 0 0">Tiêu đề khối 16 — Trạng thái nguồn dữ liệu</p>
          <p style="font-size: 14px; margin: 4px 0 0">Thân 14 — Nhân viên đã xác nhận đơn hàng và hẹn giao trong ngày; khách hỏi thêm về phí vận chuyển.</p>
          <p class="ds__muted" style="font-size: 12px; margin: 4px 0 0">Chú thích 12 — Cập nhật {{ formatDateTime(sampleDate) }} · <span class="tabular-nums">{{ formatNumber(1234567) }}</span> · {{ formatCurrency(125000, 'VND') }}</p>
        </section>

        <section class="ds__section">
          <h2 class="ds__h2">Nút và menu ⋯</h2>
          <div class="ds__row">
            <v-btn color="primary" variant="flat">Chạy đánh giá</v-btn>
            <v-btn color="primary" variant="outlined">Xuất CSV</v-btn>
            <v-btn variant="text">Bỏ lọc</v-btn>
            <v-btn color="danger" variant="flat">Xóa kết quả</v-btn>
            <ActionMenu :items="menuItems" @select="lastAction = $event" />
            <span class="ds__muted">Đã chọn: {{ lastAction || '—' }}</span>
          </div>
        </section>

        <section class="ds__section">
          <h2 class="ds__h2">Chip kết luận (nền đặc)</h2>
          <div class="ds__row">
            <VerdictChip verdict="pass" />
            <VerdictChip verdict="fail" />
            <VerdictChip verdict="skip" />
            <VerdictChip verdict="classified" />
            <VerdictChip verdict="fail" small />
          </div>
        </section>

        <section class="ds__section">
          <h2 class="ds__h2">Chip trạng thái nguồn (viền + biểu tượng)</h2>
          <div class="ds__row">
            <SourceStatusChip status="changed_since_analysis" />
            <SourceStatusChip status="verification_unavailable" />
            <SourceStatusChip status="legacy_unverified" />
            <SourceStatusChip status="bound_currentness_unverified" />
            <SourceStatusChip status="unexpected_value" />
          </div>
          <p class="ds__muted ds__caption">Giá trị lạ hoặc thiếu hiển thị là "Không xác minh được" (chip cuối).</p>
        </section>

        <section class="ds__section">
          <h2 class="ds__h2">Chip đồng bộ</h2>
          <div class="ds__row">
            <SyncStatusChip status="" />
            <SyncStatusChip status="syncing" />
            <SyncStatusChip status="success" />
            <SyncStatusChip status="partial" />
            <SyncStatusChip status="error" />
            <SyncStatusChip status="weird" />
          </div>
        </section>

        <section class="ds__section">
          <h2 class="ds__h2">Thanh lọc</h2>
          <FilterBar label="Lọc kết quả">
            <v-chip v-for="f in filters" :key="f.key" :variant="filter === f.key ? 'flat' : 'outlined'" :color="filter === f.key ? 'primary' : undefined" @click="filter = f.key">
              {{ f.label }}: <span class="tabular-nums ml-1">{{ f.count }}</span>
            </v-chip>
          </FilterBar>
        </section>

        <section class="ds__section">
          <h2 class="ds__h2">Khung trạng thái nguồn</h2>
          <div class="ds__grid2">
            <SourceStatusPanel :statuses="mixedStatuses" />
            <SourceStatusPanel :statuses="legacyStatuses" />
          </div>
          <SourceStatusPanel class="mt-3" :statuses="[]" />
        </section>

        <section class="ds__section">
          <h2 class="ds__h2">Thẻ số liệu</h2>
          <div class="ds__grid4">
            <MetricCard label="Hội thoại đã đánh giá" :value="385" hint="30 ngày qua" />
            <MetricCard label="Không đạt" :value="42" tone="fail" to="/wireframes/design-system?filter=fail" />
            <MetricCard label="Cần xem lại" :value="47" tone="warning" to="/wireframes/design-system?filter=review" />
            <MetricCard label="Điểm trung bình" :value="null" hint="Chưa có dữ liệu" />
          </div>
        </section>

        <section class="ds__section">
          <h2 class="ds__h2">Hàng bảng (desktop)</h2>
          <div class="ds__table-wrap">
            <v-table density="comfortable">
              <thead>
                <tr>
                  <th>Nguồn</th>
                  <th>Khách hàng</th>
                  <th>Kết luận</th>
                  <th class="text-right">Điểm</th>
                  <th>Vấn đề</th>
                  <th>Thời gian</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="r in sampleResults" :key="r.id">
                  <td><SourceStatusChip :status="r.source" small /></td>
                  <td class="font-weight-medium">{{ r.customer }}</td>
                  <td><VerdictChip :verdict="verdictFromSeverity(r.severity)" small /></td>
                  <td class="text-right tabular-nums">{{ r.score ?? '—' }}</td>
                  <td>{{ r.summary }}</td>
                  <td class="ds__nowrap">{{ formatDateTime(r.at) }}</td>
                </tr>
              </tbody>
            </v-table>
          </div>
        </section>

        <section class="ds__section">
          <h2 class="ds__h2">Thẻ kết quả (mobile)</h2>
          <div class="ds__cards">
            <ResultCard
              v-for="r in sampleResults"
              :key="r.id"
              :customer-name="r.customer"
              :time="formatRelative(r.at, 'vi', now)"
              :verdict="verdictFromSeverity(r.severity)"
              :source-status="r.source"
              :summary="r.summary"
              :score="r.score"
              meta="QC tổng đài · Kênh Zalo mẫu"
              @open="dialog = 'review'"
            />
          </div>
        </section>

        <section class="ds__section">
          <h2 class="ds__h2">Độ tin cậy và nhãn AI</h2>
          <div class="ds__stack">
            <ConfidenceText :confidence="null" basis="unavailable" />
            <ConfidenceText :confidence="0.72" basis="model_reported_uncalibrated" />
            <ConfidenceText :confidence="1" basis="unavailable" />
            <AiGeneratedLabel />
          </div>
        </section>

        <section class="ds__section">
          <h2 class="ds__h2">Hộp thoại</h2>
          <div class="ds__row">
            <v-btn variant="outlined" @click="dialog = 'review'">Mở chi tiết</v-btn>
            <v-btn variant="outlined" color="danger" @click="dialog = 'confirm'">Mở xác nhận xóa</v-btn>
          </div>
        </section>
      </div>

      <AppDialog
        :model-value="dialog === 'review'"
        title="Trần Minh Khoa — hội thoại 14/09/2026"
        subtitle="QC tổng đài · Kênh Zalo mẫu"
        ai-generated
        @update:model-value="dialog = $event ? 'review' : ''"
      >
        <template #meta>
          <VerdictChip verdict="fail" small />
          <SourceStatusChip status="changed_since_analysis" small />
        </template>
        <SourceStatusPanel :statuses="['changed_since_analysis']" class="mb-4" />
        <p class="mb-2" style="font-size: 14px; line-height: 1.55">
          Nhân viên chưa xác nhận lại địa chỉ giao hàng trước khi chốt đơn và không thông báo phí vận chuyển.
        </p>
        <ConfidenceText :confidence="null" basis="unavailable" />
        <template #actions>
          <v-btn variant="text" @click="dialog = ''">Đóng</v-btn>
          <v-btn color="primary" variant="flat">Mở hội thoại</v-btn>
        </template>
      </AppDialog>

      <ConfirmDialog
        :model-value="dialog === 'confirm'"
        title="Xóa 12 kết quả của lần chạy này?"
        message="Kết quả đã xóa không khôi phục được. Hội thoại gốc không bị ảnh hưởng."
        confirm-label="Xóa kết quả"
        @update:model-value="dialog = $event ? 'confirm' : ''"
        @confirm="dialog = ''"
      />
    </v-main>
  </v-app>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useTheme } from 'vuetify'
import VerdictChip from '../components/ui/VerdictChip.vue'
import SourceStatusChip from '../components/ui/SourceStatusChip.vue'
import SourceStatusPanel from '../components/ui/SourceStatusPanel.vue'
import SyncStatusChip from '../components/ui/SyncStatusChip.vue'
import MetricCard from '../components/ui/MetricCard.vue'
import AppDialog from '../components/ui/AppDialog.vue'
import ConfirmDialog from '../components/ui/ConfirmDialog.vue'
import ActionMenu from '../components/ui/ActionMenu.vue'
import AiGeneratedLabel from '../components/ui/AiGeneratedLabel.vue'
import ConfidenceText from '../components/ui/ConfidenceText.vue'
import FilterBar from '../components/ui/FilterBar.vue'
import ResultCard from '../components/ui/ResultCard.vue'
import type { ActionMenuItem } from '../components/ui/types'
import { formatCurrency, formatDateTime, formatNumber, formatRelative } from '../utils/format'
import { verdictFromSeverity } from '../utils/review'
import { storeTheme } from '../styles/tokens'

const route = useRoute()
const theme = useTheme()
const isDark = computed(() => theme.global.current.value.dark)

function toggleTheme() {
  theme.global.name.value = isDark.value ? 'light' : 'dark'
  storeTheme(theme.global.name.value)
}

const now = new Date('2026-09-28T09:00:00+07:00')
const sampleDate = new Date('2026-09-14T15:42:00+07:00')

const swatchNames = [
  'primary', 'background', 'surface', 'border', 'on-surface', 'text-muted',
  'pass', 'pass-bg', 'fail', 'fail-bg', 'skip', 'skip-bg',
  'src-changed', 'src-changed-bg', 'src-unavailable', 'src-unavailable-bg', 'src-legacy', 'danger',
]

const menuItems: ActionMenuItem[] = [
  { key: 'export', label: 'Xuất kết quả', icon: 'mdi-download-outline' },
  { key: 'rerun', label: 'Chạy lại', icon: 'mdi-refresh' },
  { key: 'delete', label: 'Xóa kết quả', icon: 'mdi-delete-outline', danger: true },
]
const lastAction = ref('')

const filters = [
  { key: 'review', label: 'Cần xem lại', count: 47 },
  { key: 'all', label: 'Tất cả', count: 385 },
  { key: 'fail', label: 'Không đạt', count: 42 },
  { key: 'pass', label: 'Đạt', count: 321 },
  { key: 'skip', label: 'Bỏ qua', count: 22 },
  { key: 'changed', label: 'Nguồn đã đổi', count: 3 },
]
const filter = ref('review')

const mixedStatuses = [
  'changed_since_analysis', 'changed_since_analysis', 'changed_since_analysis',
  'verification_unavailable', 'verification_unavailable',
  ...Array(40).fill('bound_currentness_unverified'),
  ...Array(12).fill('legacy_unverified'),
]
const legacyStatuses = Array(385).fill('legacy_unverified')

const sampleResults = [
  { id: 1, customer: 'Trần Minh Khoa', severity: 'CAN_CAI_THIEN', score: 62, source: 'changed_since_analysis', summary: 'Chưa xác nhận lại địa chỉ giao hàng; không báo phí vận chuyển.', at: '2026-09-28T08:10:00+07:00' },
  { id: 2, customer: 'Lê Thu Hà', severity: 'PASS', score: 94, source: 'bound_currentness_unverified', summary: 'Chào hỏi đúng mẫu, giải đáp đủ câu hỏi về bảo hành.', at: '2026-09-28T06:30:00+07:00' },
  { id: 3, customer: 'Phạm Quốc Bảo', severity: 'SKIP', score: null, source: 'legacy_unverified', summary: 'Hội thoại quá ngắn để đánh giá.', at: '2026-09-25T17:05:00+07:00' },
  { id: 4, customer: 'Nguyễn Hoàng Anh Thư', severity: 'NGHIEM_TRONG', score: 35, source: 'verification_unavailable', summary: 'Nhân viên hứa hoàn tiền ngoài chính sách và không chuyển cấp trên.', at: '2026-09-29T10:00:00+07:00' },
]

const initialDialog = route.query.dialog
const dialog = ref<string>(initialDialog === 'review' || initialDialog === 'confirm' ? initialDialog : '')
</script>

<style scoped>
.ds {
  background: rgb(var(--v-theme-background));
}
.ds__wrap {
  max-width: 1200px;
  margin: 0 auto;
  padding: 24px 16px 64px;
}
.ds__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
  margin-bottom: 16px;
}
.ds__h1 {
  font-size: 30px;
  font-weight: 700;
  margin: 0;
}
.ds__h2 {
  font-size: 16px;
  font-weight: 600;
  margin: 0 0 12px;
}
.ds__muted {
  color: rgb(var(--v-theme-text-muted));
  margin: 0;
}
.ds__caption {
  font-size: 12px;
  margin-top: 8px;
}
.ds__section {
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 12px;
  background: rgb(var(--v-theme-surface));
  padding: 16px;
  margin-bottom: 16px;
}
.ds__row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 12px;
}
.ds__stack {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.ds__swatches {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
  gap: 8px;
}
.ds__swatch {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
}
.ds__swatch-color {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  border: 1px solid rgb(var(--v-theme-border));
  flex-shrink: 0;
}
.ds__grid2 {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 12px;
}
.ds__grid4 {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 12px;
}
.ds__cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 12px;
}
.ds__table-wrap {
  overflow-x: auto;
}
.ds__nowrap {
  white-space: nowrap;
}
</style>
