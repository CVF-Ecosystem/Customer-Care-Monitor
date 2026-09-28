<!--
  Trang Tin nhắn — danh sách hội thoại và nội dung từng hội thoại.
  CCMAI-UX-012: dựng lại phần trình bày trên API hội thoại hiện có. Số đếm là số
  hội thoại; "PASS" của lần phân tích gần nhất có thể là Đạt (chất lượng) hoặc
  Đã phân loại, nên không bao giờ hiện là "Đạt" trơn.
-->
<template>
  <div class="mg-page">
    <div class="mg-head">
      <div class="mg-head__titles">
        <h1 class="mg-title">{{ $t('nav_messages') }}</h1>
        <p class="mg-muted mg-m0" data-testid="msgs-subtitle">{{ $t('msgs_subtitle', { n: conversationStore.total }) }}</p>
      </div>
      <v-btn
        v-if="authStore.canEdit('messages')"
        variant="outlined"
        :height="mdAndUp ? 40 : 44"
        prepend-icon="mdi-download"
        data-testid="msgs-export-open"
        @click="showExportDialog = true"
      >{{ mdAndUp ? $t('msgs_export_open') : $t('msgs_export_short') }}</v-btn>
    </div>

    <div class="mg-layout" :class="{ 'mg-layout--split': mdAndUp }">
      <!-- Danh sách hội thoại -->
      <section v-if="mdAndUp || !selectedConvId" class="mg-list" :aria-label="$t('nav_messages')">
        <!-- Bộ lọc -->
        <div class="mg-filters">
          <v-text-field
            v-model="searchQuery"
            :placeholder="$t('msgs_search')"
            :aria-label="$t('msgs_search')"
            prepend-inner-icon="mdi-magnify"
            clearable
            density="comfortable"
            variant="outlined"
            hide-details
            class="mg-filters__search"
            @update:model-value="debouncedSearch"
          />
          <template v-if="mdAndUp">
            <v-select v-model="filterChannelType" :items="channelTypes" :label="$t('msgs_filter_channel_type')" clearable density="compact" variant="outlined" hide-details />
            <v-select v-model="filterChannelId" :items="channelOptions" :label="$t('msgs_filter_channel')" clearable density="compact" variant="outlined" hide-details />
            <v-select v-model="filterEvaluation" :items="evaluationFilterOptions" :label="$t('msgs_filter_eval')" clearable density="compact" variant="outlined" hide-details class="mg-filters__wide" />
          </template>
          <v-btn v-else variant="outlined" height="48" :color="soLoc ? 'primary' : undefined" class="mg-btn" @click="moLocMobile = true">
            <v-icon start size="small">mdi-filter-variant</v-icon>{{ $t('msgs_filters') }}<span v-if="soLoc" class="ml-1">· {{ soLoc }}</span>
          </v-btn>
        </div>
        <div class="d-flex align-center justify-space-between ga-2">
          <span v-if="conversationStore.conversations.length && !listError" class="mg-muted mg-small" data-testid="msgs-range">
            {{ $t('msgs_range', { from: tuDong, to: denDong, total: conversationStore.total }) }}
          </span>
          <v-btn v-if="coLoc" variant="text" color="primary" class="mg-btn ml-auto" data-testid="msgs-clear" @click="xoaLoc">{{ $t('msgs_clear') }}</v-btn>
        </div>

        <div v-if="listError" class="mg-error" role="alert" data-testid="msgs-list-error">
          <span>{{ $t('msgs_load_error') }}</span>
          <v-btn variant="outlined" color="error" height="44" @click="loadConversations">{{ $t('msgs_retry') }}</v-btn>
        </div>

        <v-skeleton-loader v-else-if="loading && !conversationStore.conversations.length" type="list-item-avatar-two-line@5" />

        <div v-else-if="!conversationStore.conversations.length && !coLoc" class="mg-empty" data-testid="msgs-empty">
          <h2 class="mg-empty__title">{{ $t('msgs_empty_title') }}</h2>
          <p class="mg-muted mg-m0">{{ $t('msgs_empty_desc') }}</p>
          <v-btn color="primary" height="44" :to="`/${tenantId}/channels`">{{ $t('msgs_go_channels') }}</v-btn>
        </div>

        <div v-else-if="!conversationStore.conversations.length" class="mg-empty" data-testid="msgs-no-match">
          <p class="mg-muted mg-m0">{{ $t('msgs_no_match') }}</p>
          <v-btn variant="outlined" height="44" @click="xoaLoc">{{ $t('msgs_clear') }}</v-btn>
        </div>

        <div v-else class="mg-rows" :class="{ 'mg-rows--busy': loading }">
          <button
            v-for="conv in conversationStore.conversations"
            :key="conv.id"
            type="button"
            class="mg-row"
            :class="{ 'mg-row--active': selectedConvId === conv.id }"
            :aria-current="selectedConvId === conv.id ? 'true' : undefined"
            data-testid="msgs-row"
            @click="selectConversation(conv.id)"
          >
            <span class="mg-badge" :title="channelTypeInfo(conv.channel_type).label">{{ channelTypeInfo(conv.channel_type).short }}</span>
            <span class="mg-row__body">
              <span class="mg-row__top">
                <span class="mg-row__name">{{ conv.customer_name || $t('msg_unknown_customer') }}</span>
                <span class="mg-muted mg-small mg-nowrap">{{ relative(conv.last_message_at) }}</span>
              </span>
              <span class="mg-row__meta">
                <!-- UX-012 R1: chỉ kết luận "chưa phân tích" khi đã tải được trạng thái; lỗi thì ghi "không rõ", đang tải thì chưa hiện -->
                <span v-if="mapState === 'ok'" class="mg-chip" :class="`mg-chip--${chipKind(evaluationMap[conv.id])}`" data-testid="msgs-chip">{{ chipLabel(evaluationMap[conv.id]) }}</span>
                <span v-else-if="mapState === 'error'" class="mg-chip mg-chip--none" data-testid="msgs-chip">{{ $t('msgs_chip_unknown') }}</span>
                <span class="mg-muted mg-small">{{ $t('msgs_message_count', { n: conv.message_count }) }}</span>
              </span>
            </span>
          </button>
        </div>

        <v-pagination v-if="totalPages > 1 && !listError" v-model="currentPage" :length="totalPages" :total-visible="mdAndUp ? 5 : 3" density="comfortable" />
      </section>

      <!-- Nội dung hội thoại -->
      <section v-if="selectedConvId" class="mg-conv" :aria-label="currentName">
        <header class="mg-conv__head">
          <v-btn v-if="!mdAndUp" icon variant="text" size="44" :aria-label="$t('msgs_back')" @click="selectedConvId = null">
            <v-icon>mdi-arrow-left</v-icon>
          </v-btn>
          <div class="mg-conv__titles">
            <h2 class="mg-conv__title">{{ currentName }}</h2>
            <p class="mg-muted mg-small mg-m0">
              {{ selectedConvChannelName }}<template v-if="conversationStore.currentConversation"> · {{ $t('msgs_message_count', { n: conversationStore.currentConversation.message_count }) }}</template>
            </p>
          </div>
          <template v-if="mdAndUp">
            <v-btn variant="outlined" prepend-icon="mdi-link-variant" class="mg-btn" @click="shareConversation">{{ $t('msgs_copy_link') }}</v-btn>
            <v-btn variant="outlined" prepend-icon="mdi-download" class="mg-btn" :disabled="!conversationStore.messages.length" @click="downloadConversation">{{ $t('msgs_download_txt') }}</v-btn>
          </template>
          <template v-else>
            <v-btn icon variant="text" size="44" color="primary" :aria-label="$t('msgs_copy_link')" @click="shareConversation"><v-icon>mdi-link-variant</v-icon></v-btn>
            <v-btn icon variant="text" size="44" color="primary" :aria-label="$t('msgs_download_txt')" :disabled="!conversationStore.messages.length" @click="downloadConversation"><v-icon>mdi-download</v-icon></v-btn>
          </template>
        </header>

        <v-tabs v-model="detailTab" :grow="!mdAndUp" class="mg-tabs">
          <v-tab value="messages">{{ $t('msgs_tab_messages') }}</v-tab>
          <v-tab value="qc">{{ $t(mdAndUp ? 'msgs_tab_qc' : 'msgs_tab_qc_short', { n: qcGroups.length }) }}</v-tab>
          <v-tab value="classification">{{ $t('msgs_tab_class', { n: classGroups.length }) }}</v-tab>
        </v-tabs>

        <div v-if="convError" class="mg-error ma-4" role="alert" data-testid="msgs-conv-error">
          <span>{{ $t('msgs_conv_load_error') }}</span>
          <v-btn variant="outlined" color="error" height="44" @click="selectConversation(selectedConvId!, detailTab)">{{ $t('msgs_retry') }}</v-btn>
        </div>

        <!-- Tin nhắn -->
        <div v-else-if="detailTab === 'messages'" ref="messagesContainer" class="mg-pane mg-transcript">
          <div v-if="loadingMessages" class="text-center py-8"><v-progress-circular indeterminate /></div>
          <template v-else>
            <div
              v-for="msg in conversationStore.messages"
              :key="msg.id"
              class="mg-msg"
              :class="msg.sender_type === 'agent' ? 'mg-msg--agent' : 'mg-msg--customer'"
            >
              <div class="mg-msg__who">{{ msg.sender_name }} · {{ formatMessageTime(msg.sent_at) }}</div>
              <div class="mg-msg__bubble">
                <div v-if="msg.content" class="mg-msg__text">{{ msg.content }}</div>
                <div v-if="msg.content_type === 'sticker'" class="font-italic">[Sticker]</div>
                <div v-if="hasAttachments(msg)" class="mt-1">
                  <template v-for="(att, i) in parseAttachments(msg)" :key="i">
                    <div v-if="isImageAttachment(att)" class="mb-1">
                      <img
                        v-if="authImageCache[getAttachmentUrl(att)] && authImageCache[getAttachmentUrl(att)] !== 'loading'"
                        :src="authImageCache[getAttachmentUrl(att)]"
                        :alt="att.name || $t('msgs_image')"
                        class="mg-msg__img"
                        @click="lightboxSrc = authImageCache[getAttachmentUrl(att)]"
                        @error="onImageError($event, att)"
                      />
                      <v-progress-circular v-else-if="authImageCache[getAttachmentUrl(att)] === 'loading'" indeterminate size="24" width="2" class="ma-2" />
                      <v-chip v-else size="x-small" variant="tonal">
                        <v-icon start size="12">mdi-image</v-icon>{{ att.name || $t('msgs_image') }}
                      </v-chip>
                    </div>
                    <v-chip v-else size="x-small" variant="tonal" class="mr-1" :href="getAttachmentUrl(att)" target="_blank">
                      <v-icon start size="12">mdi-paperclip</v-icon>{{ att.name || att.type || 'File' }}
                    </v-chip>
                  </template>
                </div>
                <div v-if="!msg.content && msg.content_type === 'attachment' && !hasAttachments(msg)" class="font-italic">{{ $t('msgs_attachment') }}</div>
              </div>
            </div>
          </template>
        </div>

        <!-- Đánh giá chất lượng -->
        <div v-else-if="detailTab === 'qc'" class="mg-pane mg-stack" data-testid="msgs-qc">
          <p class="mg-note" data-testid="msgs-source-note">
            {{ $t('msgs_source_note') }} <router-link :to="`/${tenantId}/results`">{{ $t('msgs_source_link') }}</router-link>.
          </p>
          <div v-if="loadingEvaluation" class="text-center py-6"><v-progress-circular indeterminate /></div>
          <p v-else-if="!qcGroups.length" class="mg-muted mg-m0">{{ $t('msgs_qc_empty') }}</p>
          <article v-for="g in qcGroups" v-else :key="g.job_run_id" class="mg-card">
            <div class="mg-card__head">
              <VerdictChip v-if="getQcVerdict(g)" :verdict="verdictFromSeverity(getQcVerdict(g))" small />
              <b v-if="getQcScore(g) != null && getQcVerdict(g) !== 'SKIP'" class="tabular-nums">{{ getQcScore(g) }}/100</b>
              <span class="mg-card__job">{{ g.job_name }}</span>
              <span class="mg-muted mg-small">{{ $t('msgs_evaluated_at', { time: formatTime(g.evaluated_at) }) }}</span>
            </div>
            <template v-if="getQcReview(g)">
              <div class="d-flex align-center justify-space-between ga-2">
                <h3 class="mg-h3">{{ $t('msgs_review') }}</h3>
                <AiGeneratedLabel />
              </div>
              <p class="mg-body mg-m0">{{ getQcReview(g) }}</p>
            </template>
            <!-- Bỏ qua: không chấm nên không có mục vấn đề -->
            <template v-if="getQcVerdict(g) !== 'SKIP'">
              <h3 class="mg-h3">{{ $t('msgs_issues', { n: getQcViolations(g).length }) }}</h3>
              <p v-if="!getQcViolations(g).length" class="mg-muted mg-m0">{{ $t('msgs_no_issues') }}</p>
            </template>
            <div v-for="(v, idx) in getQcViolations(g)" :key="idx" class="mg-issue">
              <span class="d-flex align-center flex-wrap ga-2">
                <span :class="`mg-sev mg-sev--${v.severity === 'NGHIEM_TRONG' ? 'critical' : 'warning'}`">
                  {{ v.severity === 'NGHIEM_TRONG' ? $t('severity_critical') : $t('severity_warning') }}
                </span>
                <b>{{ v.rule_name }}</b>
              </span>
              <span v-if="v.evidence" class="mg-quote">{{ v.evidence }}</span>
            </div>
          </article>
        </div>

        <!-- Phân loại: nhãn, không bao giờ gọi là vấn đề -->
        <div v-else class="mg-pane mg-stack" data-testid="msgs-class">
          <p class="mg-note">
            {{ $t('msgs_source_note') }} <router-link :to="`/${tenantId}/results`">{{ $t('msgs_source_link') }}</router-link>.
          </p>
          <div v-if="loadingEvaluation" class="text-center py-6"><v-progress-circular indeterminate /></div>
          <p v-else-if="!classGroups.length" class="mg-muted mg-m0">{{ $t('msgs_class_empty') }}</p>
          <article v-for="g in classGroups" v-else :key="g.job_run_id" class="mg-card">
            <div class="mg-card__head">
              <span class="mg-card__job">{{ g.job_name }}</span>
              <span class="mg-muted mg-small">{{ $t('msgs_evaluated_at', { time: formatTime(g.evaluated_at) }) }}</span>
            </div>
            <div v-if="getClassTags(g).length" class="d-flex flex-wrap ga-2">
              <span v-for="tag in getClassTags(g)" :key="tag" class="mg-tag">{{ tag }}</span>
            </div>
            <span v-else class="mg-chip mg-chip--skip align-self-start">{{ $t('msgs_class_skip') }}</span>
            <template v-if="getClassSummary(g)">
              <div class="d-flex align-center justify-space-between ga-2">
                <h3 class="mg-h3">{{ $t('msgs_summary') }}</h3>
                <AiGeneratedLabel />
              </div>
              <p class="mg-body mg-m0">{{ getClassSummary(g) }}</p>
            </template>
          </article>
        </div>
      </section>
    </div>

    <!-- Bộ lọc trên mobile -->
    <v-bottom-sheet v-model="moLocMobile">
      <v-card>
        <v-card-title class="d-flex align-center text-subtitle-1">
          {{ $t('msgs_filters') }}
          <v-spacer />
          <v-btn icon variant="text" size="44" :aria-label="$t('ui_close')" @click="moLocMobile = false"><v-icon>mdi-close</v-icon></v-btn>
        </v-card-title>
        <v-divider />
        <v-card-text class="d-flex flex-column ga-3 pt-4">
          <v-select v-model="filterChannelType" :items="channelTypes" :label="$t('msgs_filter_channel_type')" clearable density="comfortable" variant="outlined" hide-details />
          <v-select v-model="filterChannelId" :items="channelOptions" :label="$t('msgs_filter_channel')" clearable density="comfortable" variant="outlined" hide-details />
          <v-select v-model="filterEvaluation" :items="evaluationFilterOptions" :label="$t('msgs_filter_eval')" clearable density="comfortable" variant="outlined" hide-details />
        </v-card-text>
        <v-divider />
        <v-card-actions>
          <v-btn variant="text" @click="xoaLoc">{{ $t('msgs_clear') }}</v-btn>
          <v-spacer />
          <v-btn color="primary" variant="flat" @click="moLocMobile = false">{{ $t('msgs_apply') }}</v-btn>
        </v-card-actions>
      </v-card>
    </v-bottom-sheet>

    <!-- Xuất tin nhắn -->
    <AppDialog v-model="showExportDialog" :title="$t('msgs_export_title')" :max-width="480">
      <div class="d-flex flex-column ga-3">
        <p class="mg-muted mg-m0 mg-small-body" data-testid="msgs-export-desc">{{ $t('msgs_export_desc') }}</p>
        <div class="d-flex ga-2">
          <v-text-field v-model="exportFrom" :label="$t('msgs_export_from')" type="date" density="comfortable" variant="outlined" hide-details />
          <v-text-field v-model="exportTo" :label="$t('msgs_export_to')" type="date" density="comfortable" variant="outlined" hide-details />
        </div>
        <v-select v-model="exportFormat" :items="exportFormats" :label="$t('msgs_export_format')" density="comfortable" variant="outlined" hide-details />
        <v-select v-model="exportChannelType" :items="exportChannelTypes" :label="$t('msgs_filter_channel_type')" density="comfortable" variant="outlined" hide-details />
      </div>
      <template #actions>
        <v-btn variant="text" @click="showExportDialog = false">{{ $t('msgs_cancel') }}</v-btn>
        <v-btn color="primary" variant="flat" :loading="exporting" :disabled="!exportFrom || !exportTo" data-testid="msgs-export-download" @click="doExport">
          <v-icon start>mdi-download</v-icon>{{ $t('msgs_export_download') }}
        </v-btn>
      </template>
    </AppDialog>

    <v-snackbar v-model="snackbar" :color="snackError ? 'error' : undefined" timeout="4000">{{ snackText }}</v-snackbar>

    <!-- Lightbox -->
    <div v-if="lightboxSrc" class="lightbox-overlay" @click="lightboxSrc = ''">
      <img :src="lightboxSrc" alt="" class="lightbox-img" @click.stop />
      <v-btn icon="mdi-close" variant="flat" color="white" size="small" class="lightbox-close" :aria-label="$t('ui_close')" @click="lightboxSrc = ''" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { channelTypeInfo, CHANNEL_TYPES } from '../composables/channelTypes'
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { useDisplay } from 'vuetify'
import { useI18n } from 'vue-i18n'
import { useConversationStore, type Message } from '../stores/conversations'
import { useChannelStore } from '../stores/channels'
import { useAuthStore } from '../stores/auth'
import api from '../api'
import AppDialog from '../components/ui/AppDialog.vue'
import AiGeneratedLabel from '../components/ui/AiGeneratedLabel.vue'
import VerdictChip from '../components/ui/VerdictChip.vue'
import { verdictFromSeverity } from '../utils/review'
import { formatRelative, type UiLocale } from '../utils/format'

const route = useRoute()
const { t, locale } = useI18n()
const { mdAndUp } = useDisplay()
const conversationStore = useConversationStore()
const channelStore = useChannelStore()
const authStore = useAuthStore()

const tenantId = computed(() => route.params.tenantId as string)

const loading = ref(false)
const listError = ref(false)
const loadingMessages = ref(false)
const convError = ref(false)
const selectedConvId = ref<string | null>(null)
const selectedConvChannelType = ref('')
const selectedConvChannelName = ref('')
const currentPage = ref(1)
const detailTab = ref('messages')
const moLocMobile = ref(false)

// Evaluation state
const loadingEvaluation = ref(false)
const evaluation = ref<any>(null)

// Map hội thoại -> severity của lần phân tích gần nhất, mọi loại tác vụ.
// UX-012 R1: tách "đã tải xong, không có mục" khỏi "đang tải / tải lỗi" để không bao giờ
// ghi "Chưa phân tích" cho hội thoại mà thực ra ta không biết trạng thái.
const evaluationMap = ref<Record<string, string>>({})
const mapState = ref<'pending' | 'ok' | 'error'>('pending')

async function loadEvaluationMap() {
  mapState.value = 'pending'
  try {
    const { data } = await api.get(`/tenants/${tenantId.value}/conversations/evaluated`)
    evaluationMap.value = data || {}
    mapState.value = 'ok'
  } catch {
    evaluationMap.value = {}
    mapState.value = 'error'
  }
}

// Phân loại cũng ghi PASS, nên PASS chỉ được gọi là "Đạt / đã phân loại".
function chipKind(severity: string | undefined): 'pass' | 'fail' | 'skip' | 'other' | 'none' {
  if (!severity) return 'none'
  if (severity === 'PASS') return 'pass'
  if (severity === 'FAIL') return 'fail'
  if (severity === 'SKIP') return 'skip'
  return 'other'
}
function chipLabel(severity: string | undefined) {
  return t(`msgs_chip_${chipKind(severity)}`)
}

const currentName = computed(() => conversationStore.currentConversation?.customer_name || t('msg_unknown_customer'))

function relative(s: string | null) {
  return formatRelative(s, locale.value as UiLocale)
}

function toast(text: string, error = false) {
  snackText.value = text
  snackError.value = error
  snackbar.value = true
}

async function shareConversation() {
  const url = `${window.location.origin}/${tenantId.value}/messages?conv=${selectedConvId.value}`
  try {
    await navigator.clipboard.writeText(url)
    toast(t('msgs_link_copied'))
  } catch {
    toast(t('msgs_copy_failed'), true)
  }
}

// Khi không có hội thoại nào, máy chủ trả 200 kèm JSON {error}; không được tải về như một file.
function serverError(data: unknown): string {
  if (typeof data !== 'string') return (data as { error?: string } | null)?.error || ''
  const text = data.trim()
  if (!text.startsWith('{')) return ''
  try {
    return JSON.parse(text)?.error || ''
  } catch {
    return ''
  }
}

async function doExport() {
  exporting.value = true
  try {
    let url = `/tenants/${tenantId.value}/conversations/export?from=${exportFrom.value}&to=${exportTo.value}&format=${exportFormat.value}`
    if (exportChannelType.value) url += `&channel_type=${exportChannelType.value}`
    const { data } = await api.get(url, { responseType: 'text' })
    const loi = serverError(data)
    if (loi) {
      toast(loi, true)
      return
    }
    const blob = new Blob([data], { type: exportFormat.value === 'csv' ? 'text/csv;charset=utf-8' : 'text/plain;charset=utf-8' })
    const a = document.createElement('a')
    a.href = URL.createObjectURL(blob)
    a.download = `messages_${exportFrom.value}_${exportTo.value}.${exportFormat.value}`
    a.click()
    URL.revokeObjectURL(a.href)
    showExportDialog.value = false
  } catch {
    toast(t('msgs_export_error'), true)
  } finally {
    exporting.value = false
  }
}

function downloadConversation() {
  const conv = conversationStore.currentConversation
  const msgs = conversationStore.messages
  if (!conv || !msgs.length) return

  let text = `Cuộc chat: ${conv.customer_name || 'Không rõ'}\n`
  text += `Số tin nhắn: ${msgs.length}\n`
  text += '─'.repeat(50) + '\n\n'
  for (const m of msgs) {
    const time = new Date(m.sent_at).toLocaleString('vi-VN')
    text += `[${time}] ${m.sender_name || m.sender_type}: ${m.content || ''}\n\n`
  }

  // Add evaluation if available
  if (qcGroups.value.length > 0) {
    text += '─'.repeat(50) + '\n'
    for (const g of qcGroups.value) {
      const verdict = getQcVerdict(g)
      text += `Đánh giá (${g.job_name}): ${verdict === 'PASS' ? 'Đạt' : verdict === 'SKIP' ? 'Bỏ qua' : 'Không đạt'}\n`
      const review = getQcReview(g)
      if (review) text += `Nhận xét: ${review}\n`
    }
  }

  const blob = new Blob([text], { type: 'text/plain;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `chat-${conv.customer_name || conv.id}.txt`
  a.click()
  URL.revokeObjectURL(url)
}

// Evaluation groups by type
const qcGroups = computed(() => {
  if (!evaluation.value?.groups) return []
  return evaluation.value.groups.filter((g: any) => g.job_type === 'qc_analysis')
})
const classGroups = computed(() => {
  if (!evaluation.value?.groups) return []
  return evaluation.value.groups.filter((g: any) => g.job_type === 'classification')
})

// QC group helpers
function getQcVerdict(g: any): string {
  const ev = g.results?.find((r: any) => r.result_type === 'conversation_evaluation')
  return ev?.severity || ''
}
function getQcScore(g: any): number | null {
  const ev = g.results?.find((r: any) => r.result_type === 'conversation_evaluation')
  if (!ev?.detail) return null
  try { return JSON.parse(ev.detail)?.score ?? null } catch { return null }
}
function getQcReview(g: any): string {
  const ev = g.results?.find((r: any) => r.result_type === 'conversation_evaluation')
  return ev?.evidence || ''
}
function getQcViolations(g: any): any[] {
  return (g.results || []).filter((r: any) => r.result_type === 'qc_violation')
}

// Classification group helpers
function getClassTags(g: any): string[] {
  return (g.results || [])
    .filter((r: any) => r.result_type === 'classification_tag')
    .map((r: any) => r.rule_name || r.evidence || '')
    .filter(Boolean)
}
function getClassSummary(g: any): string {
  const ev = g.results?.find((r: any) => r.result_type === 'conversation_evaluation')
  if (!ev?.detail) return ''
  try { return JSON.parse(ev.detail)?.summary ?? '' } catch { return ev?.evidence || '' }
}

const filterChannelType = ref<string | null>(null)
const filterChannelId = ref<string | null>(null)
const filterEvaluation = ref<string | null>(null)

// Export
const showExportDialog = ref(false)
const exporting = ref(false)
const exportFormat = ref('txt')
const exportChannelType = ref('')
const exportFrom = ref(new Date(Date.now() - 7 * 86400000).toISOString().slice(0, 10))
const exportTo = ref(new Date().toISOString().slice(0, 10))
const exportFormats = computed(() => [
  { title: t('msgs_export_txt'), value: 'txt' },
  { title: t('msgs_export_csv'), value: 'csv' },
])
const exportChannelTypes = computed(() => [
  { title: t('msgs_export_all_types'), value: '' },
  ...CHANNEL_TYPES.map(c => ({ title: c.label, value: c.value })),
])
const evaluationFilterOptions = computed(() => [
  { title: t('msgs_eval_evaluated'), value: 'evaluated' },
  { title: t('msgs_eval_not_evaluated'), value: 'not_evaluated' },
  { title: t('msgs_eval_pass'), value: 'PASS' },
  { title: t('msgs_eval_fail'), value: 'FAIL' },
])
const snackbar = ref(false)
const snackError = ref(false)
const lightboxSrc = ref('')
const snackText = ref('')
const searchQuery = ref('')
const messagesContainer = ref<HTMLElement | null>(null)

const perPage = 9

const channelTypes = CHANNEL_TYPES.map(c => ({ title: c.label, value: c.value }))

const channelOptions = computed(() => {
  let filtered = channelStore.channels
  if (filterChannelType.value) {
    filtered = filtered.filter(c => c.channel_type === filterChannelType.value)
  }
  return filtered.map(c => ({ title: c.name, value: c.id }))
})

const totalPages = computed(() => Math.ceil(conversationStore.total / perPage))
const tuDong = computed(() => (currentPage.value - 1) * perPage + 1)
const denDong = computed(() => (currentPage.value - 1) * perPage + conversationStore.conversations.length)
const soLoc = computed(() => (filterChannelType.value ? 1 : 0) + (filterChannelId.value ? 1 : 0) + (filterEvaluation.value ? 1 : 0))
const coLoc = computed(() => soLoc.value > 0 || !!searchQuery.value)

function xoaLoc() {
  const coTim = !!searchQuery.value
  searchQuery.value = ''
  filterChannelType.value = null
  filterChannelId.value = null
  filterEvaluation.value = null
  // Các watch tự tải lại khi bộ lọc đổi; chỉ còn ô tìm là phải tải tay
  if (coTim) {
    currentPage.value = 1
    loadConversations()
  }
}

let searchTimeout: ReturnType<typeof setTimeout> | null = null
function debouncedSearch() {
  if (searchTimeout) clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => {
    currentPage.value = 1
    loadConversations()
  }, 300)
}

async function loadConversations() {
  loading.value = true
  listError.value = false
  try {
    const params: Record<string, string | number> = {
      page: currentPage.value,
      per_page: perPage,
    }
    if (filterChannelType.value) params.channel_type = filterChannelType.value
    if (filterChannelId.value) params.channel_id = filterChannelId.value
    if (searchQuery.value) params.search = searchQuery.value
    if (filterEvaluation.value) params.evaluation = filterEvaluation.value

    await conversationStore.fetchConversations(tenantId.value, params)
  } catch {
    listError.value = true
  } finally {
    loading.value = false
  }
}

async function selectConversation(convId: string, tab?: string) {
  selectedConvId.value = convId
  convError.value = false
  detailTab.value = tab === 'evaluation' || tab === 'qc' ? 'qc' : tab === 'classification' ? 'classification' : 'messages'
  const conv = conversationStore.conversations.find(c => c.id === convId)
  if (conv) {
    selectedConvChannelType.value = conv.channel_type
    selectedConvChannelName.value = conv.channel_name || channelTypeInfo(conv.channel_type).label
  }

  loadingMessages.value = true
  try {
    await conversationStore.fetchMessages(tenantId.value, convId)
    await nextTick()
    scrollToBottom()
  } catch {
    convError.value = true
  } finally {
    loadingMessages.value = false
  }

  // Load evaluation in background
  loadingEvaluation.value = true
  evaluation.value = null
  try {
    const { data } = await api.get(`/tenants/${tenantId.value}/conversations/${convId}/evaluations`)
    evaluation.value = data
  } catch {
    evaluation.value = null
  } finally {
    loadingEvaluation.value = false
  }
}

function scrollToBottom() {
  if (messagesContainer.value) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
  }
}

const dayNamesShort = ['CN', 'T2', 'T3', 'T4', 'T5', 'T6', 'T7']

function formatTime(dateStr: string | null) {
  if (!dateStr) return '—'
  const d = new Date(dateStr)
  if (Number.isNaN(d.getTime())) return '—'
  const dd = String(d.getDate()).padStart(2, '0')
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const hh = String(d.getHours()).padStart(2, '0')
  const mi = String(d.getMinutes()).padStart(2, '0')
  return `${dd}/${mm}/${d.getFullYear()} ${hh}:${mi}`
}

function formatMessageTime(dateStr: string) {
  const d = new Date(dateStr)
  if (Number.isNaN(d.getTime())) return '—'
  const day = dayNamesShort[d.getDay()]
  const dd = String(d.getDate()).padStart(2, '0')
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const hh = String(d.getHours()).padStart(2, '0')
  const mi = String(d.getMinutes()).padStart(2, '0')
  return `${day} ${dd}/${mm} ${hh}:${mi}`
}

function hasAttachments(msg: Message) {
  if (!msg.attachments || msg.attachments === '[]' || msg.attachments === 'null') return false
  try {
    const arr = JSON.parse(msg.attachments)
    return Array.isArray(arr) && arr.length > 0
  } catch {
    return false
  }
}

function getAttachmentUrl(att: any): string {
  if (att.local_path) return `/api/v1/files/${att.local_path}`
  return att.url || ''
}

// Auth image loading: fetch with JWT header, create blob URL
const authImageCache = ref<Record<string, string>>({})

async function loadAuthImage(url: string) {
  if (!url || authImageCache.value[url]) return
  // For external URLs (not /api/), load directly
  if (!url.startsWith('/api/')) {
    authImageCache.value[url] = url
    return
  }
  authImageCache.value[url] = 'loading'
  try {
    const token = localStorage.getItem('cqa_access_token')
    const resp = await fetch(url, {
      headers: token ? { 'Authorization': `Bearer ${token}` } : {},
    })
    if (resp.ok) {
      const blob = await resp.blob()
      authImageCache.value[url] = URL.createObjectURL(blob)
    } else {
      delete authImageCache.value[url] // allow retry
    }
  } catch {
    delete authImageCache.value[url] // allow retry
  }
}

// Load auth images when messages change
watch(() => conversationStore.messages, () => {
  const msgs = conversationStore.messages
  if (!msgs || !Array.isArray(msgs)) return
  for (const msg of msgs) {
    if (msg.attachments) {
      try {
        const atts = typeof msg.attachments === 'string' ? JSON.parse(msg.attachments) : msg.attachments
        if (!Array.isArray(atts)) continue
        for (const att of atts) {
          if (isImageAttachment(att)) {
            const url = getAttachmentUrl(att)
            if (url) loadAuthImage(url)
          }
        }
      } catch { continue }
    }
  }
}, { immediate: true })

onUnmounted(() => {
  if (searchTimeout) clearTimeout(searchTimeout)
  // Cleanup blob URLs to prevent memory leaks
  for (const blobUrl of Object.values(authImageCache.value)) {
    if (blobUrl && blobUrl.startsWith('blob:')) URL.revokeObjectURL(blobUrl)
  }
})

function isImageAttachment(att: any): boolean {
  if (!att.type) return false
  const t = att.type.toLowerCase()
  return t.startsWith('image') || t === 'photo' || t === 'gif' || t === 'sticker'
}

// Ảnh hỏng: bỏ đường dẫn khỏi bộ nhớ đệm để hiện chip tên file thay cho ảnh
function onImageError(_event: Event, att: any) {
  const url = getAttachmentUrl(att)
  if (url) authImageCache.value[url] = ''
}

function parseAttachments(msg: Message) {
  try {
    return JSON.parse(msg.attachments) || []
  } catch {
    return []
  }
}

watch(currentPage, () => loadConversations())
watch(filterChannelType, () => {
  currentPage.value = 1
  filterChannelId.value = null
  loadConversations()
})
watch(filterChannelId, () => {
  currentPage.value = 1
  loadConversations()
})

watch(filterEvaluation, () => {
  currentPage.value = 1
  loadConversations()
})

onMounted(async () => {
  // Reset pagination state
  currentPage.value = 1
  conversationStore.total = 0

  // Pre-fill filter from query params (e.g. from channels page)
  if (route.query.channel_id) {
    filterChannelId.value = route.query.channel_id as string
  }

  try {
    await channelStore.fetchChannels(tenantId.value)
  } catch { /* danh sách kênh chỉ để lọc; thiếu thì vẫn xem được hội thoại */ }
  await loadConversations()
  await loadEvaluationMap()

  // Auto-select conversation from query param (deep link)
  if (route.query.conv) {
    const convId = route.query.conv as string
    try {
      // Find which page this conversation is on
      const { data } = await api.get(`/tenants/${tenantId.value}/conversations/${convId}/page`, { params: { per_page: perPage } })
      if (data?.page && data.page !== currentPage.value) {
        currentPage.value = data.page
        await loadConversations()
      }
    } catch { /* fallback: stay on page 1 */ }
    selectConversation(convId, route.query.tab as string)
  }
})
</script>

<style scoped>
.mg-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.mg-head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 12px;
}
.mg-head__titles {
  flex: 1 1 240px;
}
.mg-title {
  margin: 0;
  font-size: 26px;
  font-weight: 700;
  line-height: 1.25;
}
.mg-muted {
  color: rgb(var(--v-theme-text-muted));
}
.mg-m0 {
  margin: 0;
}
.mg-small {
  font-size: 12px;
}
.mg-small-body {
  font-size: 13px;
  line-height: 1.5;
}
.mg-nowrap {
  white-space: nowrap;
}
.mg-body {
  font-size: 14px;
  line-height: 1.55;
}
.mg-h3 {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
}
.mg-btn {
  text-transform: none;
  letter-spacing: 0;
}
.mg-layout {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.mg-layout--split {
  display: grid;
  grid-template-columns: 420px minmax(0, 1fr);
  align-items: start;
}
.mg-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
}
.mg-filters {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 8px;
}
.mg-layout--split .mg-filters {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}
.mg-layout--split .mg-filters__search,
.mg-layout--split .mg-filters__wide {
  grid-column: span 2;
}
.mg-rows {
  display: flex;
  flex-direction: column;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 12px;
  background: rgb(var(--v-theme-surface));
  overflow: hidden;
}
.mg-rows--busy {
  opacity: 0.6;
}
.mg-row {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 64px;
  padding: 10px 14px;
  border: 0;
  border-bottom: 1px solid rgb(var(--v-theme-border));
  background: transparent;
  color: rgb(var(--v-theme-on-surface));
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.mg-row:last-child {
  border-bottom: 0;
}
.mg-row:hover {
  background: rgba(var(--v-theme-on-surface), 0.03);
}
.mg-row:focus-visible {
  outline: 2px solid rgb(var(--v-theme-primary));
  outline-offset: -2px;
}
.mg-row--active {
  background: rgba(var(--v-theme-primary), 0.08);
  box-shadow: inset 3px 0 0 rgb(var(--v-theme-primary));
}
.mg-badge {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 44px;
  height: 28px;
  padding: 0 6px;
  border-radius: 8px;
  background: rgba(var(--v-theme-on-surface), 0.06);
  font-size: 11px;
  font-weight: 700;
}
.mg-row__body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.mg-row__top {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 8px;
}
.mg-row__name {
  font-size: 15px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.mg-row__meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
}
.mg-chip {
  display: inline-flex;
  align-items: center;
  height: 22px;
  padding: 0 8px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
}
.mg-chip--pass {
  background: rgba(var(--v-theme-primary), 0.1);
  color: rgb(var(--v-theme-primary));
}
.mg-chip--fail {
  background: rgb(var(--v-theme-fail-bg));
  color: rgb(var(--v-theme-fail));
}
.mg-chip--skip {
  background: rgb(var(--v-theme-skip-bg));
  color: rgb(var(--v-theme-skip));
}
.mg-chip--other {
  background: rgba(var(--v-theme-on-surface), 0.06);
  color: rgb(var(--v-theme-on-surface));
}
.mg-chip--none {
  border: 1px solid rgb(var(--v-theme-border));
  color: rgb(var(--v-theme-text-muted));
  font-weight: 500;
}
.mg-error {
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
.mg-empty {
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
.mg-empty__title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}
.mg-conv {
  display: flex;
  flex-direction: column;
  min-width: 0;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 12px;
  background: rgb(var(--v-theme-surface));
  overflow: hidden;
}
.mg-layout--split .mg-conv {
  position: sticky;
  top: 16px;
  height: calc(100vh - 140px);
  min-height: 480px;
}
.mg-conv__head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  border-bottom: 1px solid rgb(var(--v-theme-border));
}
.mg-conv__titles {
  flex: 1;
  min-width: 0;
}
.mg-conv__title {
  margin: 0;
  font-size: 18px;
  font-weight: 700;
  overflow-wrap: anywhere;
}
.mg-tabs {
  flex-shrink: 0;
  border-bottom: 1px solid rgb(var(--v-theme-border));
}
.mg-pane {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 16px;
}
.mg-transcript {
  display: flex;
  flex-direction: column;
  gap: 10px;
  background: rgba(var(--v-theme-on-surface), 0.02);
}
.mg-stack {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.mg-msg {
  display: flex;
  flex-direction: column;
  gap: 2px;
  max-width: 78%;
}
.mg-msg--customer {
  align-self: flex-start;
}
.mg-msg--agent {
  align-self: flex-end;
  align-items: flex-end;
}
.mg-msg__who {
  font-size: 11px;
  color: rgb(var(--v-theme-text-muted));
}
.mg-msg__bubble {
  padding: 8px 12px;
  border-radius: 12px;
  border: 1px solid rgb(var(--v-theme-border));
  background: rgb(var(--v-theme-surface));
  font-size: 14px;
  line-height: 1.45;
  overflow-wrap: anywhere;
}
.mg-msg--agent .mg-msg__bubble {
  background: rgba(var(--v-theme-primary), 0.08);
}
.mg-msg__text {
  white-space: pre-wrap;
}
.mg-msg__img {
  max-width: 200px;
  max-height: 200px;
  border-radius: 8px;
  cursor: pointer;
}
.mg-note {
  margin: 0;
  padding: 10px 12px;
  border: 1px solid rgb(var(--v-theme-border));
  border-left: 4px solid rgb(var(--v-theme-src-legacy));
  border-radius: 10px;
  font-size: 13px;
  line-height: 1.5;
}
.mg-note a {
  color: rgb(var(--v-theme-primary));
  font-weight: 600;
}
.mg-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px 16px;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 12px;
}
.mg-card__head {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.mg-card__job {
  flex: 1;
  min-width: 120px;
  font-size: 14px;
  font-weight: 600;
}
.mg-issue {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px 10px;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 10px;
  font-size: 14px;
}
.mg-quote {
  padding: 6px 10px;
  border-left: 3px solid rgb(var(--v-theme-src-changed));
  border-radius: 4px;
  background: rgba(var(--v-theme-src-changed), 0.06);
  font-size: 13px;
  line-height: 1.5;
}
.mg-sev {
  display: inline-flex;
  align-items: center;
  height: 22px;
  padding: 0 8px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
}
.mg-sev--critical {
  background: rgb(var(--v-theme-fail-bg));
  color: rgb(var(--v-theme-fail));
}
.mg-sev--warning {
  background: rgb(var(--v-theme-src-changed-bg));
  color: rgb(var(--v-theme-src-changed));
}
.mg-tag {
  display: inline-flex;
  align-items: center;
  height: 28px;
  padding: 0 10px;
  border-radius: 8px;
  background: rgba(var(--v-theme-primary), 0.1);
  color: rgb(var(--v-theme-primary));
  font-size: 13px;
  font-weight: 600;
}
.lightbox-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.85);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
  cursor: pointer;
}
.lightbox-img {
  max-width: 90vw;
  max-height: 90vh;
  object-fit: contain;
  border-radius: 8px;
  cursor: default;
}
.lightbox-close {
  position: fixed;
  top: 16px;
  right: 16px;
}
</style>
