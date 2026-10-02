<!--
  Job Detail — CCMAI-UX-010 redesign (SPEC docs/specs/JOB_DETAIL_SCREEN_UX010_2026-09-28.md,
  canvas version 1790540352-11c7). Data comes from the existing job, runs, results and messages
  endpoints only; scope, grouping and evidence logic live in ./job-detail/logic.ts.
-->
<template>
  <div class="jd">
    <!-- Header -->
    <header class="jd-header">
      <div class="jd-header__titles">
        <router-link :to="`/${tenantId}/jobs`" class="jd-back">
          <v-icon size="18" aria-hidden="true">mdi-chevron-left</v-icon>{{ $t('jd_back') }}
        </router-link>
        <h1 class="jd-title">{{ job?.name || '…' }}</h1>
        <div v-if="job" class="jd-meta">
          <span>{{ isClassification ? $t('job_classification') : $t('job_qc') }}</span>
          <span>· {{ formatSchedule(job.schedule_type, job.schedule_cron) }}</span>
          <span>· {{ $t('jd_meta_channels', { n: parsedChannelCount }) }}</span>
          <span v-if="tenantAIProvider">· {{ tenantAIProvider }}{{ tenantAIModel ? ' / ' + tenantAIModel : '' }}</span>
          <span v-if="job.last_run_at">
            · {{ $t('jd_meta_last_run', { time: fmtDateTime(job.last_run_at) }) }}
            <span :class="`jd-run-text jd-run-text--${runStatusKind(job.last_run_status)}`">{{ $t(`jd_run_${runStatusKind(job.last_run_status)}`) }}</span>
          </span>
        </div>
      </div>
      <div v-if="authStore.canEdit('jobs')" class="jd-actions">
        <v-btn
          v-if="isJobRunning"
          color="danger"
          variant="outlined"
          prepend-icon="mdi-stop"
          class="jd-action"
          :loading="cancelling"
          @click="cancelJob"
        >{{ $t('jd_stop') }}</v-btn>
        <v-btn variant="outlined" class="jd-action" :disabled="isJobRunning" @click="testRun">
          {{ mdAndUp ? $t('jd_run_test') : $t('jd_run_test_short') }}
        </v-btn>
        <v-btn color="primary" variant="flat" prepend-icon="mdi-play" class="jd-action" :disabled="isJobRunning" @click="openRunDialog">
          {{ $t('run_now') }}
        </v-btn>
        <ActionMenu :items="menuItems" @select="onMenu" />
      </div>
    </header>

    <!-- Run/cancel feedback (bounded copy; CCMAI-RUNTIME-028) -->
    <v-alert v-if="runNotice" :type="runNotice.type" variant="tonal" class="mb-4" density="compact" role="status" data-testid="run-notice">{{ runNotice.text }}</v-alert>

    <!-- Running progress, from the counters the analyzer writes into the run summary -->
    <v-alert v-if="progress" type="info" variant="tonal" class="mb-4" density="compact">
      <v-progress-linear :model-value="progressPercent" color="primary" height="8" rounded class="mb-2" />
      <div class="text-body-2">
        {{ $t('jd_progress', { analyzed: progress.analyzed, found: progress.found }) }}
        <span v-if="progress.errors"> · {{ $t('jd_progress_errors', { n: progress.errors }) }}</span>
      </div>
    </v-alert>

    <v-alert v-if="loadError" type="error" variant="tonal" class="mb-4" role="alert">
      <div class="d-flex align-center flex-wrap ga-3">
        <span class="flex-grow-1">{{ $t('jd_load_error') }}</span>
        <v-btn variant="outlined" color="error" class="jd-action" @click="reload">{{ $t('jd_retry') }}</v-btn>
      </div>
    </v-alert>

    <!-- Loading -->
    <div v-if="loading" class="jd-stack">
      <div class="jd-metrics">
        <v-skeleton-loader v-for="i in 4" :key="i" type="article" class="jd-skeleton" />
      </div>
      <v-skeleton-loader type="table-row@5" />
    </div>

    <!-- Never run -->
    <section v-else-if="!loadError && !jobStore.jobRuns.length && !jobStore.jobResults.length" class="jd-empty">
      <v-icon size="40" aria-hidden="true">mdi-play-circle-outline</v-icon>
      <h2 class="jd-empty__title">{{ $t('jd_never_run_title') }}</h2>
      <p class="jd-empty__desc">{{ $t('jd_never_run_desc') }}</p>
      <div v-if="authStore.canEdit('jobs')" class="d-flex ga-2">
        <v-btn variant="outlined" class="jd-action" @click="testRun">{{ $t('jd_run_test_short') }}</v-btn>
        <v-btn color="primary" variant="flat" class="jd-action" @click="openRunDialog">{{ $t('run_now') }}</v-btn>
      </div>
    </section>

    <template v-else-if="!loading">
      <!-- Scope: which run the numbers and list describe -->
      <div class="jd-scope">
        <p class="jd-scope__caption">{{ scopeCaption }}</p>
        <v-select
          v-model="scopeRunId"
          :items="scopeItems"
          :label="$t('jd_scope_label')"
          density="compact"
          hide-details
          class="jd-scope__select"
        />
      </div>

      <!-- Metrics -->
      <div class="jd-metrics">
        <template v-if="!isClassification">
          <MetricCard :label="$t('jd_m_evaluated')" :value="qc.evaluated" :hint="$t('jd_m_evaluated_hint', { total: qc.total, skipped: qc.skipped })" :to="filterLink('evaluated')" />
          <MetricCard :label="$t('jd_m_pass_rate')" :value="qc.passRate" suffix="%" :to="filterLink('pass')" />
          <MetricCard :label="$t('jd_m_issues')" :value="qc.issues" tone="fail" :to="filterLink('fail')" />
          <MetricCard :label="$t('jd_m_avg_score')" :value="qc.avgScore" suffix="/100" />
        </template>
        <template v-else>
          <MetricCard :label="$t('jd_m_total')" :value="cls.total" :to="filterLink('all')" />
          <MetricCard :label="$t('jd_m_classified')" :value="cls.classified" :to="filterLink('classified')" />
          <MetricCard :label="$t('jd_m_skipped')" :value="cls.skipped" :to="filterLink('skip')" />
          <MetricCard :label="$t('jd_m_top_tag')" :value="cls.topTag?.count ?? null" :hint="cls.topTag?.name" :to="cls.topTag ? filterLink(`tag:${cls.topTag.name}`) : undefined" />
        </template>
      </div>

      <!-- Trend (QC) -->
      <v-card v-if="!isClassification && scoped.length" class="jd-card">
        <h2 class="jd-card__title">
          <v-icon size="18" aria-hidden="true">mdi-chart-line</v-icon>{{ $t('job_trend') }}
        </h2>
        <div class="jd-trend">
          <Line :data="trendChartData" :options="chartOptions" />
        </div>
      </v-card>

      <!-- Tabs -->
      <v-card class="jd-card">
        <v-tabs v-model="activeTab" density="comfortable" class="mb-4">
          <v-tab value="results">{{ $t('tab_results') }}</v-tab>
          <v-tab value="history">{{ $t('run_history') }}</v-tab>
        </v-tabs>

        <!-- Results -->
        <div v-if="activeTab === 'results'" class="jd-stack">
          <SourceStatusPanel :statuses="panelStatuses" />
          <p v-if="allLegacy" class="jd-note">{{ $t('jd_all_legacy') }}</p>

          <div class="jd-toolbar">
            <FilterBar :label="$t('jd_filter_label')" class="jd-toolbar__filters">
              <v-chip
                v-for="f in filters"
                :key="f.key"
                :variant="resultFilter === f.key ? 'flat' : 'outlined'"
                :color="resultFilter === f.key ? 'primary' : undefined"
                @click="setFilter(f.key)"
              >{{ f.label }}: <span class="tabular-nums ml-1">{{ f.count }}</span></v-chip>
            </FilterBar>
            <div class="jd-toolbar__export">
              <v-btn variant="outlined" size="small" prepend-icon="mdi-file-delimited-outline" @click="exportResults('csv')">{{ $t('jd_export_all_csv') }}</v-btn>
              <v-btn variant="outlined" size="small" prepend-icon="mdi-file-excel-outline" @click="exportResults('xlsx')">{{ $t('jd_export_all_xlsx') }}</v-btn>
            </div>
          </div>

          <p v-if="!groups.length" class="jd-muted">{{ $t('jd_no_results_in_scope') }}</p>
          <p v-else-if="!filteredGroups.length" class="jd-muted">{{ $t('jd_no_match') }}</p>

          <!-- Desktop table -->
          <div v-else-if="mdAndUp" class="jd-table-wrap">
            <v-table density="comfortable" hover>
              <thead>
                <tr>
                  <th>{{ $t('job_source_col') }}</th>
                  <th>{{ $t('jd_col_customer') }}</th>
                  <th>{{ $t('jd_col_date') }}</th>
                  <template v-if="!isClassification">
                    <th>{{ $t('jd_col_verdict') }}</th>
                    <th>{{ $t('jd_col_review') }}</th>
                    <th class="text-right">{{ $t('jd_col_score') }}</th>
                    <th>{{ $t('jd_col_issues') }}</th>
                  </template>
                  <template v-else>
                    <th>{{ $t('jd_col_tags') }}</th>
                    <th>{{ $t('jd_col_summary') }}</th>
                  </template>
                </tr>
              </thead>
              <tbody>
                <tr v-for="g in pageGroups" :key="g.conversationId" class="jd-row" :class="{ 'jd-row--changed': isChanged(g) }" tabindex="0" @click="openDetail(g)" @keydown.enter="openDetail(g)">
                  <td>
                    <div class="d-flex flex-column ga-1 py-2">
                      <SourceStatusChip v-for="s in g.sourceStatuses" :key="s" :status="s" small />
                    </div>
                  </td>
                  <td class="font-weight-medium">{{ customerLabel(g) }}</td>
                  <td class="jd-nowrap">{{ fmtDateTime(g.conversationDate) }}</td>
                  <template v-if="!isClassification">
                    <td><VerdictChip :verdict="verdictOf(g)" small /></td>
                    <td class="jd-clamp-cell"><span class="jd-clamp">{{ g.review || '—' }}</span></td>
                    <td class="text-right tabular-nums">{{ g.verdict === 'SKIP' || g.score === null ? '—' : g.score }}</td>
                    <td class="jd-nowrap">{{ g.violations.length ? $t('jd_issue_count', { n: g.violations.length }) : '—' }}</td>
                  </template>
                  <template v-else>
                    <td>
                      <div v-if="g.tags.length" class="d-flex flex-wrap ga-1">
                        <span v-for="t in g.tags" :key="t" class="jd-tag">{{ t }}</span>
                      </div>
                      <VerdictChip v-else :verdict="verdictOf(g)" small />
                    </td>
                    <td class="jd-clamp-cell"><span class="jd-clamp">{{ classificationSummary(g) }}</span></td>
                  </template>
                </tr>
              </tbody>
            </v-table>
          </div>

          <!-- Mobile cards -->
          <div v-else class="jd-stack">
            <ResultCard
              v-for="g in pageGroups"
              :key="g.conversationId"
              :customer-name="customerLabel(g)"
              :time="fmtDateTime(g.conversationDate)"
              :verdict="verdictOf(g)"
              :source-status="g.sourceStatuses[0]"
              :summary="isClassification ? classificationSummary(g) : g.review"
              :score="isClassification || g.verdict === 'SKIP' ? null : g.score"
              :meta="countLabel(g)"
              @open="openDetail(g)"
            />
          </div>

          <v-pagination v-if="totalPages > 1" v-model="page" :length="totalPages" :total-visible="mdAndUp ? 7 : 3" density="comfortable" />
        </div>

        <!-- Run history -->
        <div v-else class="jd-stack">
          <p v-if="!jobStore.jobRuns.length" class="jd-muted">{{ $t('no_runs') }}</p>
          <div v-else class="jd-table-wrap">
            <v-table density="comfortable">
              <thead>
                <tr>
                  <th>{{ $t('jd_runs_start') }}</th>
                  <th>{{ $t('jd_runs_status') }}</th>
                  <th>{{ $t('jd_runs_duration') }}</th>
                  <th>{{ $t('jd_runs_conversations') }}</th>
                  <th>{{ $t('jd_runs_summary') }}</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="r in pageRuns" :key="r.id">
                  <td class="jd-nowrap tabular-nums">{{ fmtDateTime(r.started_at) }}</td>
                  <td><span :class="`jd-run-chip jd-run-chip--${runStatusKind(r.status)}`">{{ $t(`jd_run_${runStatusKind(r.status)}`) }}</span></td>
                  <td class="jd-nowrap tabular-nums">{{ durationLabel(r) }}</td>
                  <td class="tabular-nums">{{ runConversations(r) }}</td>
                  <td>
                    <span v-if="r.error_message" class="jd-run-error">{{ r.error_message }}</span>
                    <span v-else>{{ runSummary(r) }}</span>
                  </td>
                  <td class="text-right">
                    <v-btn v-if="runHasResults(r.id)" variant="text" color="primary" size="small" @click="viewRun(r.id)">{{ $t('jd_run_view') }}</v-btn>
                  </td>
                </tr>
              </tbody>
            </v-table>
          </div>
          <v-pagination v-if="totalRunPages > 1" v-model="runPage" :length="totalRunPages" :total-visible="5" density="comfortable" />
          <p class="jd-muted jd-small">{{ $t('jd_run_running_note') }}</p>
        </div>
      </v-card>
    </template>

    <!-- Run options dialog (behavior unchanged) -->
    <v-dialog v-model="runDialog" max-width="560">
      <v-card>
        <v-card-title>{{ $t('run_now') }}</v-card-title>
        <v-card-text>
          <v-radio-group v-model="runMode" class="mb-1">
            <v-radio value="unanalyzed">
              <template #label>
                <div>
                  <div class="font-weight-medium">Chạy cho những cuộc chat chưa được đánh giá</div>
                  <div class="text-caption jd-muted">Đánh giá tất cả cuộc chat chưa được công việc này phân tích lần nào, bất kể thời gian.</div>
                </div>
              </template>
            </v-radio>
            <v-radio value="since_last">
              <template #label>
                <div>
                  <div class="font-weight-medium">Chạy từ lần gần nhất</div>
                  <div class="text-caption jd-muted">Lấy cuộc chat gần nhất đã đánh giá làm mốc. Cuộc chat cũ hơn hoặc có thời điểm tin nhắn cuối bằng mốc sẽ không được đánh giá dù chưa phân tích.</div>
                </div>
              </template>
            </v-radio>
            <v-radio value="conditional">
              <template #label>
                <div>
                  <div class="font-weight-medium">Chạy theo điều kiện</div>
                  <div class="text-caption jd-muted">Đánh giá lại cả cuộc chat đã đánh giá. Ngày (giờ Việt Nam) chỉ chọn cuộc chat theo tin nhắn cuối; mỗi cuộc chat được chọn vẫn phân tích đủ ngữ cảnh, gồm cả tin nhắn trước ngày bắt đầu. Có thể chỉ chọn một đầu ngày và/hoặc giới hạn số cuộc chat; phải có ít nhất một điều kiện.</div>
                </div>
              </template>
            </v-radio>
          </v-radio-group>
          <template v-if="runMode === 'conditional'">
            <div class="d-flex ga-3 mt-2">
              <v-text-field v-model="runDateFrom" type="date" label="Từ ngày (giờ Việt Nam)" density="compact" :error-messages="runDateFromError" hide-details="auto" data-testid="run-date-from" />
              <v-text-field v-model="runDateTo" type="date" label="Đến ngày (giờ Việt Nam)" density="compact" :error-messages="runDateToError" hide-details="auto" data-testid="run-date-to" />
            </div>
          </template>
          <!-- The limit is only a count cap; it never changes the mode chosen above. -->
          <v-text-field v-model.number="runLimit" type="number" label="Giới hạn số cuộc chat" density="compact" hide-details="auto" class="mt-3" placeholder="Để trống nếu không muốn áp dụng" :min="1" :step="1" :error-messages="runLimitError" clearable data-testid="run-limit" />
          <v-alert v-if="runConditionalError" type="error" variant="tonal" density="compact" class="mt-3 text-caption">{{ runConditionalError }}</v-alert>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn @click="runDialog = false">{{ $t('cancel') }}</v-btn>
          <v-btn color="primary" :disabled="!!runConditionalError || !!runLimitError" data-testid="run-confirm" @click="confirmRun">{{ $t('confirm') }}</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Conversation detail -->
    <AppDialog
      v-if="dialogGroup"
      :model-value="detailDialog"
      :title="`${customerLabel(dialogGroup)} · ${fmtDateTime(dialogGroup.conversationDate)}`"
      :max-width="1100"
      @update:model-value="detailDialog = $event"
    >
      <template #meta>
        <VerdictChip :verdict="verdictOf(dialogGroup)" small />
        <span v-if="!isClassification && dialogGroup.score !== null && dialogGroup.verdict !== 'SKIP'" class="tabular-nums font-weight-bold">{{ dialogGroup.score }}/100</span>
        <SourceStatusChip v-for="s in dialogGroup.sourceStatuses" :key="s" :status="s" small />
      </template>

      <div class="jd-detail">
        <section class="jd-detail__chat" :aria-label="$t('jd_transcript')">
          <div class="jd-detail__head">
            <h3 class="jd-h3">{{ $t('jd_transcript') }}</h3>
            <span class="jd-muted jd-small">{{ $t('jd_transcript_hint') }}</span>
          </div>
          <p v-if="!dialogMessages" class="jd-muted">{{ $t('jd_loading_messages') }}</p>
          <div v-else class="jd-transcript">
            <div
              v-for="msg in dialogMessages"
              :id="`jd-msg-${msg.id}`"
              :key="msg.id"
              class="jd-msg"
              :class="[msg.sender_type === 'agent' ? 'jd-msg--agent' : 'jd-msg--customer', { 'jd-msg--quoted': highlightedIds.has(msg.id) }]"
            >
              <div class="jd-msg__who">{{ msg.sender_name }} · {{ fmtDateTime(msg.sent_at) }}</div>
              <div class="jd-msg__bubble">
                <div v-if="msg.content">{{ msg.content }}</div>
                <div v-if="msg.content_type === 'sticker'" class="font-italic">[Sticker]</div>
                <div v-if="hasAttachments(msg)" class="mt-1">
                  <template v-for="(att, ai) in parseAttachments(msg)" :key="ai">
                    <div v-if="isImageAttachment(att)" class="mb-1">
                      <img v-if="authImageCache[getAttachmentUrl(att)] && authImageCache[getAttachmentUrl(att)] !== 'loading'" :src="authImageCache[getAttachmentUrl(att)]" alt="" class="jd-msg__img" @click="lightboxSrc = authImageCache[getAttachmentUrl(att)]" />
                      <v-progress-circular v-else-if="authImageCache[getAttachmentUrl(att)] === 'loading'" indeterminate size="20" width="2" class="ma-2" />
                    </div>
                    <v-chip v-else size="x-small" variant="tonal" class="mr-1" :href="getAttachmentUrl(att)" target="_blank"><v-icon start size="12">mdi-paperclip</v-icon>{{ att.name || 'File' }}</v-chip>
                  </template>
                </div>
                <div v-if="!msg.content && !hasAttachments(msg) && msg.content_type !== 'text'" class="font-italic">[{{ msg.content_type || 'File' }}]</div>
              </div>
            </div>
          </div>
        </section>

        <section class="jd-detail__side">
          <SourceStatusPanel :statuses="dialogGroup.sourceStatuses" />

          <div class="jd-stack-sm">
            <div class="d-flex align-center justify-space-between ga-2">
              <h3 class="jd-h3">{{ $t('jd_review') }}</h3>
              <AiGeneratedLabel />
            </div>
            <p class="jd-review">{{ isClassification ? classificationSummary(dialogGroup) : dialogGroup.review || '—' }}</p>
            <ConfidenceText :confidence="dialogGroup.confidence" :basis="dialogGroup.confidenceBasis" />
          </div>

          <div class="jd-stack-sm">
            <h3 class="jd-h3">{{ countLabel(dialogGroup) }}</h3>
            <p v-if="!dialogGroup.violations.length && dialogGroup.verdict === 'PASS' && !isClassification" class="jd-muted">{{ $t('jd_no_issues_pass') }}</p>
            <button
              v-for="v in dialogGroup.violations"
              :key="v.id"
              type="button"
              class="jd-issue"
              :class="{ 'jd-issue--selected': selectedIssueId === v.id }"
              :aria-pressed="selectedIssueId === v.id"
              @click="selectIssue(v)"
            >
              <span class="d-flex align-center flex-wrap ga-2">
                <span v-if="!isClassification" :class="`jd-sev jd-sev--${v.severity === 'NGHIEM_TRONG' ? 'critical' : 'warning'}`">
                  {{ v.severity === 'NGHIEM_TRONG' ? $t('severity_critical') : $t('severity_warning') }}
                </span>
                <b>{{ v.rule_name }}</b>
              </span>
              <span v-if="v.evidence" class="jd-issue__evidence">{{ v.evidence }}</span>
              <span v-if="parseDetail(v.detail).explanation && parseDetail(v.detail).explanation !== v.evidence" class="jd-muted jd-small">{{ parseDetail(v.detail).explanation }}</span>
              <span v-if="parseDetail(v.detail).suggestion" class="jd-small"><b>{{ $t('jd_suggestion') }}:</b> {{ parseDetail(v.detail).suggestion }}</span>
              <span v-if="selectedIssueId === v.id" class="jd-issue__quote" :class="{ 'jd-issue__quote--missing': !highlightedIds.size }">
                {{ highlightedIds.size ? $t('jd_evidence_found') : $t('jd_evidence_missing') }}
              </span>
            </button>
          </div>
        </section>
      </div>

      <template #actions>
        <v-btn variant="outlined" :to="`/${tenantId}/messages?conv=${dialogGroup.conversationId}`" @click="detailDialog = false">{{ $t('jd_open_messages') }}</v-btn>
        <v-btn color="primary" variant="flat" :disabled="!nextReviewGroup" @click="nextReviewGroup && openDetail(nextReviewGroup)">{{ $t('jd_next_review') }}</v-btn>
      </template>
    </AppDialog>

    <ConfirmDialog
      v-model="clearResultsDialog"
      :title="$t('jd_confirm_clear_results_title')"
      :message="$t('jd_confirm_clear_results_msg')"
      :confirm-label="$t('jd_action_clear_results')"
      :loading="clearingResults"
      @confirm="clearResults"
    />
    <ConfirmDialog
      v-model="clearRunsDialog"
      :title="$t('jd_confirm_clear_runs_title')"
      :message="$t('jd_confirm_clear_runs_msg')"
      :confirm-label="$t('jd_action_clear_runs')"
      :loading="clearingRuns"
      @confirm="clearRuns"
    />

    <!-- AI not configured (unchanged) -->
    <v-dialog v-model="aiNotConfiguredDialog" max-width="450">
      <v-card class="pa-6">
        <v-card-title>
          <v-icon start color="warning">mdi-alert</v-icon>
          Chưa cấu hình AI Provider
        </v-card-title>
        <v-card-text>Bạn cần cấu hình API key của AI Provider (Claude hoặc Gemini) trước khi chạy tác vụ phân tích.</v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="aiNotConfiguredDialog = false">Đóng</v-btn>
          <v-btn color="primary" variant="flat" :to="`/${tenantId}/settings`" @click="aiNotConfiguredDialog = false">
            <v-icon start>mdi-cog</v-icon>Đi tới cài đặt
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <div v-if="lightboxSrc" class="lightbox-overlay" @click="lightboxSrc = ''">
      <img :src="lightboxSrc" alt="" class="lightbox-img" @click.stop />
      <v-btn icon="mdi-close" variant="flat" color="white" size="small" class="lightbox-close" :aria-label="$t('ui_close')" @click="lightboxSrc = ''" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useDisplay, useTheme } from 'vuetify'
import { useI18n } from 'vue-i18n'
import { useJobStore, type JobResult, type JobRun } from '../../stores/jobs'
import { useAuthStore } from '../../stores/auth'
import api from '../../api'
import { qualityTrendByDay } from '../../utils/trend'
import { formatDateTime, type UiLocale } from '../../utils/format'
import { verdictFromSeverity, type Verdict } from '../../utils/review'
import {
  type ConversationGroup,
  classificationMetrics,
  defaultScopeRunId,
  groupNeedsReview,
  groupResults,
  parseDetail,
  primarySourceStatus,
  qcMetrics,
  resolveEvidence,
  runDurationSeconds,
  runProgress,
  runStatusKind,
  scopeResults,
} from './job-detail/logic'
import ActionMenu from '../../components/ui/ActionMenu.vue'
import AppDialog from '../../components/ui/AppDialog.vue'
import ConfirmDialog from '../../components/ui/ConfirmDialog.vue'
import MetricCard from '../../components/ui/MetricCard.vue'
import SourceStatusPanel from '../../components/ui/SourceStatusPanel.vue'
import SourceStatusChip from '../../components/ui/SourceStatusChip.vue'
import VerdictChip from '../../components/ui/VerdictChip.vue'
import ResultCard from '../../components/ui/ResultCard.vue'
import FilterBar from '../../components/ui/FilterBar.vue'
import AiGeneratedLabel from '../../components/ui/AiGeneratedLabel.vue'
import ConfidenceText from '../../components/ui/ConfidenceText.vue'
import type { ActionMenuItem } from '../../components/ui/types'
import { Line } from 'vue-chartjs'
import { Chart as ChartJS, CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Filler, Legend } from 'chart.js'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Filler, Legend)

const route = useRoute()
const router = useRouter()
const { mdAndUp } = useDisplay()
const theme = useTheme()
const { t, locale } = useI18n()
const jobStore = useJobStore()
const authStore = useAuthStore()
const tenantId = computed(() => route.params.tenantId as string)
const jobId = computed(() => route.params.jobId as string)
const job = ref<Record<string, any> | null>(null)
const tenantAIProvider = ref('')
const tenantAIModel = ref('')
const isClassification = computed(() => job.value?.job_type === 'classification')

const loading = ref(true)
const loadError = ref(false)
const cancelling = ref(false)
// The run whose cancellation was accepted (202 means "requested", not "terminal"); polling follows it.
const cancelPendingRunId = ref<string | null>(null)
const TERMINAL_RUN_STATUSES = ['success', 'partial', 'error', 'cancelled']
function cancelOutcomeNotice(status: string): { type: 'info' | 'warning'; text: string } {
  if (status === 'cancelled') return { type: 'info', text: 'Lượt chạy đã dừng theo yêu cầu hủy.' }
  if (status === 'success') return { type: 'info', text: 'Lượt chạy đã hoàn tất thành công trước khi yêu cầu hủy có hiệu lực.' }
  if (status === 'partial') return { type: 'info', text: 'Lượt chạy đã hoàn tất một phần trước khi yêu cầu hủy có hiệu lực.' }
  return { type: 'warning', text: 'Lượt chạy đã kết thúc với lỗi trước khi yêu cầu hủy có hiệu lực.' }
}
const runNotice = ref<{ type: 'info' | 'warning' | 'error'; text: string } | null>(null)
function launchErrorNotice(e: unknown): { type: 'warning' | 'error'; text: string } {
  const code = (e as { response?: { data?: { error?: string } } })?.response?.data?.error
  if (code === 'job_already_running') return { type: 'warning', text: 'Công việc này đang có một lượt chạy khác; chưa thể bắt đầu lượt mới.' }
  return { type: 'error', text: 'Không khởi động được lượt chạy. Vui lòng thử lại.' }
}
const isJobRunning = computed(() => jobStore.jobRuns?.[0]?.status === 'running')
let pollTimer: ReturnType<typeof setTimeout> | null = null

function fmtDateTime(value: string | null | undefined) {
  return formatDateTime(value, locale.value as UiLocale)
}

// ---- Scope: one run (default: latest with results) or every run ----
const ALL_RUNS = '__all__'
const scopeRunId = ref<string>(ALL_RUNS)
const scopeInitialized = ref(false)
const scoped = computed(() => scopeResults(jobStore.jobResults, scopeRunId.value === ALL_RUNS ? null : scopeRunId.value))
const groups = computed(() => groupResults(scoped.value))
const runsWithResults = computed(() => new Set(jobStore.jobResults.map((r) => r.job_run_id)))
function runHasResults(id: string) {
  return runsWithResults.value.has(id)
}
const latestRunWithResults = computed(() => defaultScopeRunId(jobStore.jobRuns, jobStore.jobResults))

const scopeItems = computed(() => [
  ...jobStore.jobRuns
    .filter((r) => runHasResults(r.id))
    .map((r) => ({
      value: r.id,
      title: `${fmtDateTime(r.started_at)}${r.id === latestRunWithResults.value ? ` (${t('jd_scope_latest')})` : ''} · ${t(`jd_run_${runStatusKind(r.status)}`)}`,
    })),
  { value: ALL_RUNS, title: t('jd_scope_all') },
])
const scopeCaption = computed(() => {
  if (scopeRunId.value === ALL_RUNS) return t('jd_scope_caption_all')
  const run = jobStore.jobRuns.find((r) => r.id === scopeRunId.value)
  return t('jd_scope_caption_run', { time: fmtDateTime(run?.started_at) })
})
function initScope() {
  if (scopeInitialized.value) return
  scopeRunId.value = latestRunWithResults.value ?? ALL_RUNS
  scopeInitialized.value = true
}
// A new run finishing moves an untouched latest-run scope forward.
watch(latestRunWithResults, (id, prev) => {
  if (id && scopeRunId.value === prev) scopeRunId.value = id
})
function viewRun(runId: string) {
  scopeRunId.value = runId
  activeTab.value = 'results'
}

// ---- Metrics ----
const qc = computed(() => qcMetrics(groups.value))
const cls = computed(() => classificationMetrics(groups.value))

// ---- Filters (default "Cần xem lại" when there is anything to review) ----
const activeTab = ref<'results' | 'history'>('results')
const resultFilter = ref('all')
const filterTouched = ref(false)
const reviewCount = computed(() => groups.value.filter((g) => groupNeedsReview(g, isClassification.value)).length)

const filters = computed(() => {
  const gs = groups.value
  const out: { key: string; label: string; count: number }[] = [{ key: 'review', label: t('ui_needs_review'), count: reviewCount.value }]
  if (!isClassification.value) {
    out.push(
      { key: 'all', label: t('filter_all'), count: gs.length },
      { key: 'evaluated', label: t('jd_filter_evaluated'), count: gs.filter((g) => g.verdict !== 'SKIP').length },
      { key: 'fail', label: t('filter_failed'), count: gs.filter((g) => g.verdict !== 'SKIP' && verdictFromSeverity(g.verdict) === 'fail').length },
      { key: 'pass', label: t('filter_passed'), count: gs.filter((g) => g.verdict === 'PASS').length },
      { key: 'skip', label: t('verdict_skip'), count: gs.filter((g) => g.verdict === 'SKIP').length },
    )
  } else {
    out.push(
      { key: 'classified', label: t('ui_verdict_classified'), count: cls.value.classified },
      { key: 'all', label: t('filter_all'), count: gs.length },
      { key: 'skip', label: t('verdict_skip'), count: cls.value.skipped },
      ...cls.value.tagCounts.map((tc) => ({ key: `tag:${tc.name}`, label: tc.name, count: tc.count })),
    )
  }
  return out
})

const filteredGroups = computed(() => {
  const f = resultFilter.value
  const gs = groups.value
  if (f === 'review') return gs.filter((g) => groupNeedsReview(g, isClassification.value))
  if (f === 'pass') return gs.filter((g) => g.verdict === 'PASS')
  if (f === 'skip') return gs.filter((g) => g.verdict === 'SKIP')
  if (f === 'evaluated') return gs.filter((g) => g.verdict !== 'SKIP')
  if (f === 'fail') return gs.filter((g) => g.verdict !== 'SKIP' && verdictFromSeverity(g.verdict) === 'fail')
  if (f === 'classified') return gs.filter((g) => g.verdict !== 'SKIP')
  if (f.startsWith('tag:')) return gs.filter((g) => g.tags.includes(f.slice(4)))
  return gs
})

function setFilter(key: string) {
  filterTouched.value = true
  resultFilter.value = key
  page.value = 1
}
function applyDefaultFilter() {
  if (filterTouched.value) return
  resultFilter.value = reviewCount.value > 0 ? 'review' : isClassification.value ? 'classified' : 'all'
}
watch([groups, isClassification], applyDefaultFilter)

// Metric cards link to the list filtered within the current scope (route query ?filter=).
function filterLink(key: string) {
  return { query: { ...route.query, filter: key } }
}
watch(
  () => route.query.filter,
  (f) => {
    if (typeof f === 'string' && f) {
      setFilter(f)
      activeTab.value = 'results'
    }
  },
  { immediate: true },
)

// ---- Paging ----
const page = ref(1)
const perPage = 10
const totalPages = computed(() => Math.ceil(filteredGroups.value.length / perPage))
const pageGroups = computed(() => filteredGroups.value.slice((page.value - 1) * perPage, page.value * perPage))
watch([scopeRunId, resultFilter], () => (page.value = 1))

const runPage = ref(1)
const runsPerPage = 8
const totalRunPages = computed(() => Math.ceil(jobStore.jobRuns.length / runsPerPage))
const pageRuns = computed(() => jobStore.jobRuns.slice((runPage.value - 1) * runsPerPage, runPage.value * runsPerPage))

// ---- Source panel: each conversation counted once, under its most concerning status ----
const panelStatuses = computed(() => groups.value.map(primarySourceStatus))
const allLegacy = computed(() => groups.value.length > 0 && groups.value.every((g) => g.sourceStatuses.every((s) => s === 'legacy_unverified')))

// ---- Row helpers ----
function verdictOf(g: ConversationGroup): Verdict {
  if (isClassification.value) return g.verdict === 'SKIP' ? 'skip' : 'classified'
  return verdictFromSeverity(g.verdict)
}
function isChanged(g: ConversationGroup) {
  return g.sourceStatuses.includes('changed_since_analysis')
}
function customerLabel(g: ConversationGroup) {
  return g.customerName || g.conversationId.substring(0, 8) + '…'
}
function countLabel(g: ConversationGroup) {
  const n = g.violations.length
  return isClassification.value ? t('jd_tag_count', { n }) : t('jd_issue_count', { n })
}
function classificationSummary(g: ConversationGroup): string {
  const evidences = g.violations.map((v) => v.evidence).filter((e) => e && !e.startsWith('Cuộc chat được phân loại'))
  if (evidences.length) return evidences.join('; ')
  const explanations = g.violations.map((v) => parseDetail(v.detail).explanation).filter(Boolean)
  if (explanations.length) return explanations.join('; ')
  return g.review || '—'
}

// ---- Runs ----
const progress = computed(() => runProgress(jobStore.jobRuns[0]))
const progressPercent = computed(() => (progress.value ? Math.round((progress.value.analyzed / progress.value.found) * 100) : 0))
function durationLabel(r: JobRun) {
  const s = runDurationSeconds(r)
  if (s === null) return '—'
  return s >= 60 ? t('jd_duration_ms', { m: Math.floor(s / 60), s: s % 60 }) : t('jd_duration_s', { s })
}
function runConversations(r: JobRun) {
  if (r.status === 'running') {
    const p = runProgress(r)
    return p ? `${p.analyzed} / ${p.found}` : '—'
  }
  const n = parseDetail(r.summary).conversations_analyzed
  return typeof n === 'number' ? String(n) : '—'
}
function runSummary(r: JobRun) {
  if (r.status === 'running') return '—'
  const s = parseDetail(r.summary)
  if (typeof s.conversations_analyzed !== 'number') return '—'
  // Classification runs never set conversations_passed (engine/analyzer.go only
  // marks `passed` for qc_analysis), so showing it here would read as a false "0 đạt".
  // issues_found does hold a verified count for classification too — it is the
  // number of classification_tag rows the run created — but it means tags, not "vấn đề".
  if (isClassification.value) {
    return t('jd_run_summary_classification', { analyzed: s.conversations_analyzed, tags: s.issues_found ?? 0 })
  }
  return t('jd_run_summary_qc', { passed: s.conversations_passed ?? 0, analyzed: s.conversations_analyzed, issues: s.issues_found ?? 0 })
}

// ---- Trend (theme colors, Vietnam days) ----
const trendChartData = computed(() => {
  const points = qualityTrendByDay(scoped.value)
  const c = theme.current.value.colors as Record<string, string>
  return {
    labels: points.map((p) => p.label),
    datasets: [
      { label: t('verdict_pass'), data: points.map((p) => p.passed), borderColor: c.pass, backgroundColor: c.pass, fill: false, tension: 0.3, pointRadius: 4 },
      { label: t('verdict_fail'), data: points.map((p) => p.failed), borderColor: c.fail, backgroundColor: c.fail, fill: false, tension: 0.3, pointRadius: 4 },
    ],
  }
})
const chartOptions = computed(() => {
  const c = theme.current.value.colors as Record<string, string>
  return {
    responsive: true,
    maintainAspectRatio: false,
    plugins: { legend: { display: true, position: 'bottom' as const, labels: { color: c['text-muted'] } } },
    scales: {
      x: { grid: { display: false }, ticks: { color: c['text-muted'] } },
      y: { beginAtZero: true, ticks: { stepSize: 1, color: c['text-muted'] }, grid: { color: c.border } },
    },
  }
})

// ---- Header menu ----
const menuItems = computed<ActionMenuItem[]>(() => [
  { key: 'edit', label: t('jd_action_edit'), icon: 'mdi-pencil-outline' },
  { key: 'clear-results', label: t('jd_action_clear_results'), icon: 'mdi-delete-sweep-outline', danger: true },
  { key: 'clear-runs', label: t('jd_action_clear_runs'), icon: 'mdi-delete-clock-outline', danger: true },
])
function onMenu(key: string) {
  if (key === 'edit') router.push(`/${tenantId.value}/jobs/${jobId.value}/edit`)
  else if (key === 'clear-results') clearResultsDialog.value = true
  else if (key === 'clear-runs') clearRunsDialog.value = true
}

// ---- Detail dialog and evidence ----
const detailDialog = ref(false)
const dialogGroup = ref<ConversationGroup | null>(null)
const selectedIssueId = ref<string | null>(null)
const chatMessages = ref<Record<string, any[]>>({})
const dialogMessages = computed(() => (dialogGroup.value ? chatMessages.value[dialogGroup.value.conversationId] : undefined))
const highlightedIds = computed(() => {
  const g = dialogGroup.value
  const msgs = dialogMessages.value
  if (!g || !msgs || !selectedIssueId.value) return new Set<string>()
  const v = g.violations.find((x) => x.id === selectedIssueId.value)
  return new Set(v ? resolveEvidence(v, msgs).map((r) => r.message_id) : [])
})
const nextReviewGroup = computed(() => {
  const list = groups.value.filter((g) => groupNeedsReview(g, isClassification.value))
  const cur = dialogGroup.value
  if (!cur) return list[0] ?? null
  const i = list.findIndex((g) => g.conversationId === cur.conversationId)
  return list[i + 1] ?? null
})

async function openDetail(g: ConversationGroup) {
  dialogGroup.value = g
  selectedIssueId.value = null
  detailDialog.value = true
  if (!chatMessages.value[g.conversationId]) {
    try {
      const { data } = await api.get(`/tenants/${tenantId.value}/conversations/${g.conversationId}/messages`)
      chatMessages.value[g.conversationId] = data.messages || []
    } catch {
      chatMessages.value[g.conversationId] = []
    }
  }
}
async function selectIssue(v: JobResult) {
  selectedIssueId.value = selectedIssueId.value === v.id ? null : v.id
  await nextTick()
  const first = [...highlightedIds.value][0]
  if (first) document.getElementById(`jd-msg-${first}`)?.scrollIntoView({ block: 'center', behavior: 'smooth' })
}

// ---- Attachments (unchanged behavior) ----
const lightboxSrc = ref('')
const authImageCache = ref<Record<string, string>>({})
function hasAttachments(msg: any) {
  if (!msg.attachments || msg.attachments === '[]' || msg.attachments === 'null') return false
  try { const arr = JSON.parse(msg.attachments); return Array.isArray(arr) && arr.length > 0 } catch { return false }
}
function parseAttachments(msg: any) {
  try { return JSON.parse(msg.attachments) || [] } catch { return [] }
}
function isImageAttachment(att: any): boolean {
  if (!att.type) return false
  const ty = att.type.toLowerCase()
  return ty.startsWith('image') || ty === 'photo' || ty === 'gif' || ty === 'sticker'
}
function getAttachmentUrl(att: any): string {
  if (att.local_path) return `/api/v1/files/${att.local_path}`
  return att.url || ''
}
async function loadAuthImage(url: string) {
  if (!url || authImageCache.value[url]) return
  if (!url.startsWith('/api/')) { authImageCache.value[url] = url; return }
  authImageCache.value[url] = 'loading'
  try {
    const token = localStorage.getItem('cqa_access_token')
    const resp = await fetch(url, { headers: token ? { Authorization: `Bearer ${token}` } : {} })
    if (resp.ok) { const blob = await resp.blob(); authImageCache.value[url] = URL.createObjectURL(blob) }
    else { delete authImageCache.value[url] }
  } catch { delete authImageCache.value[url] }
}
watch(chatMessages, (val) => {
  for (const msgs of Object.values(val)) {
    for (const msg of msgs || []) {
      for (const att of parseAttachments(msg)) if (isImageAttachment(att)) { const u = getAttachmentUrl(att); if (u) loadAuthImage(u) }
    }
  }
}, { deep: true })
onUnmounted(() => {
  stopPolling()
  for (const url of Object.values(authImageCache.value)) { if (url?.startsWith('blob:')) URL.revokeObjectURL(url) }
})

// ---- Job fields ----
const parsedChannelCount = computed(() => {
  try { return JSON.parse(job.value?.input_channel_ids || '[]').length } catch { return 0 }
})
const dayNames = ['Chủ nhật', 'Thứ 2', 'Thứ 3', 'Thứ 4', 'Thứ 5', 'Thứ 6', 'Thứ 7']
function formatSchedule(type: string, cron: string) {
  if (type === 'manual' || !cron) return 'Thủ công'
  const parts = cron.trim().split(/\s+/)
  if (parts.length < 5) return cron
  const [min, hour, dom, , dow] = parts
  const time = `${hour.padStart(2, '0')}:${min.padStart(2, '0')}`
  if (dow === '*' && dom === '*') return `Hàng ngày lúc ${time}`
  if (dow === '1-5' && dom === '*') return `Thứ 2-6 lúc ${time}`
  if (dow === '0-6' && dom === '*') return `Hàng ngày lúc ${time}`
  if (dow !== '*' && dom === '*') return `${dow.split(',').map((d) => dayNames[parseInt(d)] || d).join(', ')} lúc ${time}`
  if (dom !== '*' && dow === '*') return `Ngày ${dom} hàng tháng lúc ${time}`
  return cron
}

// ---- Loading, polling and actions ----
async function reload() {
  loading.value = true
  loadError.value = false
  try {
    job.value = await jobStore.fetchJob(tenantId.value, jobId.value)
    await jobStore.fetchJobRuns(tenantId.value, jobId.value)
    await jobStore.fetchAllJobResults(tenantId.value, jobId.value)
    initScope()
    applyDefaultFilter()
  } catch {
    loadError.value = true
  } finally {
    loading.value = false
  }
  try {
    const { data } = await api.get(`/tenants/${tenantId.value}/settings`)
    tenantAIProvider.value = data?.settings?.ai_provider || 'claude'
    tenantAIModel.value = data?.settings?.ai_model || ''
  } catch { /* the model line is optional */ }
  if (isJobRunning.value) startPolling()
}
onMounted(reload)

function startPolling() {
  stopPolling()
  async function tick() {
    try {
      await jobStore.fetchJobRuns(tenantId.value, jobId.value)
      // With a pending cancellation the poll follows that exact run, not "the newest run", and it
      // settles only when that run is OBSERVED in a recognized terminal status. A missing run, an
      // empty/unknown status or a newer run never settles it (CCMAI-RUNTIME-028).
      const pendingId = cancelPendingRunId.value
      const observed = pendingId ? jobStore.jobRuns.find((r) => r.id === pendingId) : undefined
      const settled = pendingId ? !!observed && TERMINAL_RUN_STATUSES.includes(observed.status) : !isJobRunning.value
      if (settled) {
        await jobStore.fetchAllJobResults(tenantId.value, jobId.value)
        job.value = await jobStore.fetchJob(tenantId.value, jobId.value)
        if (pendingId && observed) {
          runNotice.value = cancelOutcomeNotice(observed.status)
          cancelPendingRunId.value = null
        }
        stopPolling()
        return
      }
    } catch { /* retry next tick */ }
    pollTimer = setTimeout(tick, 3000)
  }
  pollTimer = setTimeout(tick, 2000)
}
function stopPolling() {
  if (pollTimer) { clearTimeout(pollTimer); pollTimer = null }
}

const runDialog = ref(false)
const runMode = ref<'unanalyzed' | 'since_last' | 'conditional'>('since_last')
const runDateFrom = ref('')
const runDateTo = ref('')
const runLimit = ref<number | null>(null)
const runDateFromError = computed(() => {
  if (runMode.value !== 'conditional') return ''
  if (runDateFrom.value && runDateTo.value && runDateFrom.value > runDateTo.value) return 'Từ ngày phải nhỏ hơn đến ngày'
  return ''
})
// A single endpoint is a valid open range; only a reversed pair is rejected (on the from field).
const runDateToError = computed(() => '')
const runLimitError = computed(() => {
  const v = runLimit.value as unknown
  if (v === null || v === undefined || v === '') return ''
  return typeof v === 'number' && Number.isInteger(v) && v > 0 ? '' : 'Giới hạn phải là số nguyên dương'
})
const runConditionalError = computed(() => {
  if (runMode.value !== 'conditional') return ''
  if (!runDateFrom.value && !runDateTo.value && !runLimit.value) return 'Vui lòng chọn ít nhất một điều kiện (thời gian hoặc số lượng)'
  return runDateFromError.value || runDateToError.value || ''
})

const aiNotConfiguredDialog = ref(false)
async function checkAIConfigured(): Promise<boolean> {
  try {
    const { data } = await api.get(`/tenants/${tenantId.value}/settings`)
    if (data.settings?.ai_api_key) return true
  } catch { /* fall through */ }
  aiNotConfiguredDialog.value = true
  return false
}
async function openRunDialog() {
  if (!(await checkAIConfigured())) return
  runDialog.value = true
}
async function testRun() {
  if (!(await checkAIConfigured())) return
  try {
    await jobStore.testRunJob(tenantId.value, jobId.value)
    runNotice.value = null
    startPolling()
  } catch (e) {
    runNotice.value = launchErrorNotice(e)
    await jobStore.fetchJobRuns(tenantId.value, jobId.value)
    if (isJobRunning.value) startPolling()
  }
}
async function confirmRun() {
  if (runConditionalError.value || runLimitError.value) return
  runDialog.value = false
  try {
    const params: Record<string, string> = {}
    if (runMode.value === 'conditional') {
      if (runDateFrom.value) params.from = runDateFrom.value
      if (runDateTo.value) params.to = runDateTo.value
    }
    if (runLimit.value) params.limit = String(runLimit.value)
    await jobStore.triggerJob(tenantId.value, jobId.value, runMode.value, params)
    runNotice.value = null
    startPolling()
  } catch (e) {
    runNotice.value = launchErrorNotice(e)
    await jobStore.fetchJobRuns(tenantId.value, jobId.value)
    if (isJobRunning.value) startPolling()
  }
}
async function cancelJob() {
  // Capture the run the user is looking at NOW; a later refresh must never retarget the request.
  const target = jobStore.jobRuns?.[0]
  if (!target || target.status !== 'running') return
  const targetId = target.id
  cancelling.value = true
  try {
    await jobStore.cancelJobRun(tenantId.value, jobId.value, targetId)
    // 202 job_cancel_requested: the worker has not necessarily stopped. Keep polling this run.
    cancelPendingRunId.value = targetId
    runNotice.value = { type: 'info', text: 'Đã yêu cầu hủy; đang chờ lượt chạy kết thúc.' }
    startPolling()
  } catch (e) {
    const code = (e as { response?: { data?: { error?: string } } })?.response?.data?.error
    if (code === 'job_not_running') {
      runNotice.value = { type: 'info', text: 'Lượt chạy này đã kết thúc hoặc không còn là lượt đang chạy.' }
    } else if (code === 'job_run_not_owned') {
      runNotice.value = { type: 'warning', text: 'Lượt chạy này không do tiến trình hiện tại quản lý nên không thể hủy từ đây.' }
    } else {
      runNotice.value = { type: 'error', text: 'Không gửi được yêu cầu hủy; lượt chạy vẫn có thể đang chạy.' }
    }
    // Refresh the truth; keep following a run that is still running.
    try {
      await jobStore.fetchJobRuns(tenantId.value, jobId.value)
      if (isJobRunning.value || jobStore.jobRuns.some((r) => r.id === targetId && r.status === 'running')) startPolling()
    } catch { /* the next manual refresh shows the state */ }
  } finally { cancelling.value = false }
}

const clearResultsDialog = ref(false)
const clearRunsDialog = ref(false)
const clearingResults = ref(false)
const clearingRuns = ref(false)
async function clearResults() {
  clearingResults.value = true
  try {
    await api.delete(`/tenants/${tenantId.value}/jobs/${jobId.value}/results`)
    clearResultsDialog.value = false
    jobStore.jobResults = []
    await jobStore.fetchJobRuns(tenantId.value, jobId.value)
  } catch (err) {
    console.error('Clear results failed:', err)
  } finally {
    clearingResults.value = false
  }
}
async function clearRuns() {
  clearingRuns.value = true
  try {
    await api.delete(`/tenants/${tenantId.value}/jobs/${jobId.value}/runs`)
    clearRunsDialog.value = false
    jobStore.jobResults = []
    await jobStore.fetchJobRuns(tenantId.value, jobId.value)
    job.value = await jobStore.fetchJob(tenantId.value, jobId.value)
  } catch (err) {
    console.error('Clear runs failed:', err)
  } finally {
    clearingRuns.value = false
  }
}

// The export endpoint has no run filter, so the buttons say they export every run.
async function exportResults(format: 'csv' | 'xlsx') {
  try {
    const { data } = await api.get(`/tenants/${tenantId.value}/jobs/${jobId.value}/results/export?format=${format}`, {
      responseType: format === 'xlsx' ? 'blob' : 'text',
    })
    const blob = format === 'xlsx'
      ? new Blob([data], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' })
      : new Blob([data], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `results.${format}`
    a.click()
    URL.revokeObjectURL(url)
  } catch { /* ignore */ }
}
</script>

<style scoped>
.jd {
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-width: 1440px;
}
.jd-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 12px 16px;
  flex-wrap: wrap;
}
.jd-header__titles {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
  flex: 1 1 360px;
}
.jd-back {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  align-self: flex-start;
  min-height: 32px;
  font-size: 14px;
  color: rgb(var(--v-theme-primary));
  text-decoration: none;
}
.jd-title {
  margin: 0;
  font-size: 30px;
  line-height: 1.2;
  font-weight: 700;
  overflow-wrap: anywhere;
}
.jd-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 2px 6px;
  font-size: 14px;
  color: rgb(var(--v-theme-text-muted));
}
.jd-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.jd-action {
  min-height: 44px;
}
.jd-run-text--success { color: rgb(var(--v-theme-pass)); font-weight: 600; }
.jd-run-text--error { color: rgb(var(--v-theme-fail)); font-weight: 600; }
.jd-run-text--partial { color: rgb(var(--v-theme-src-changed)); font-weight: 600; }
.jd-run-text--running { color: rgb(var(--v-theme-primary)); font-weight: 600; }
.jd-stack {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.jd-stack-sm {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.jd-metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}
.jd-skeleton {
  border-radius: 12px;
}
.jd-scope {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px 16px;
  flex-wrap: wrap;
}
.jd-scope__caption {
  margin: 0;
  font-size: 13px;
  color: rgb(var(--v-theme-text-muted));
}
.jd-scope__select {
  flex: 0 1 340px;
  min-width: 240px;
}
.jd-card {
  padding: 16px 20px;
}
.jd-card__title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 0 8px;
  font-size: 16px;
  font-weight: 600;
}
.jd-trend {
  height: 200px;
}
.jd-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px 16px;
  flex-wrap: wrap;
}
.jd-toolbar__filters {
  flex: 1 1 420px;
  min-width: 0;
}
.jd-toolbar__export {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.jd-table-wrap {
  overflow-x: auto;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 12px;
}
.jd-row {
  cursor: pointer;
}
.jd-row:focus-visible {
  outline: 2px solid rgb(var(--v-theme-primary));
  outline-offset: -2px;
}
.jd-row--changed td:first-child {
  box-shadow: inset 4px 0 0 rgb(var(--v-theme-src-changed));
}
.jd-nowrap {
  white-space: nowrap;
}
.jd-clamp-cell {
  max-width: 420px;
  white-space: normal;
}
.jd-clamp {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.jd-tag {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 8px;
  border-radius: 8px;
  background: rgba(var(--v-theme-primary), 0.1);
  color: rgb(var(--v-theme-primary));
  font-size: 12px;
  font-weight: 600;
}
.jd-muted {
  margin: 0;
  color: rgb(var(--v-theme-text-muted));
}
.jd-small {
  font-size: 12px;
}
.jd-note {
  margin: 0;
  font-size: 13px;
  line-height: 1.5;
}
.jd-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 32px 20px;
  text-align: center;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 12px;
  background: rgb(var(--v-theme-surface));
  color: rgb(var(--v-theme-text-muted));
}
.jd-empty__title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: rgb(var(--v-theme-on-surface));
}
.jd-empty__desc {
  margin: 0 0 4px;
  max-width: 460px;
  font-size: 14px;
  line-height: 1.5;
}
.jd-run-chip {
  display: inline-flex;
  align-items: center;
  height: 26px;
  padding: 0 10px;
  border-radius: 8px;
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
}
.jd-run-chip--success { background: rgb(var(--v-theme-pass-bg)); color: rgb(var(--v-theme-pass)); }
.jd-run-chip--error { background: rgb(var(--v-theme-fail-bg)); color: rgb(var(--v-theme-fail)); }
.jd-run-chip--partial { background: rgb(var(--v-theme-src-changed-bg)); color: rgb(var(--v-theme-src-changed)); }
.jd-run-chip--running { background: rgba(var(--v-theme-primary), 0.1); color: rgb(var(--v-theme-primary)); }
.jd-run-chip--cancelled,
.jd-run-chip--unknown { background: rgb(var(--v-theme-skip-bg)); color: rgb(var(--v-theme-skip)); }
.jd-run-error {
  color: rgb(var(--v-theme-fail));
  font-size: 13px;
}
.jd-detail {
  display: grid;
  grid-template-columns: minmax(0, 7fr) minmax(0, 5fr);
  gap: 20px;
}
.jd-detail__head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 8px;
}
.jd-detail__side {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.jd-h3 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
}
.jd-transcript {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-height: 60vh;
  overflow-y: auto;
  padding: 12px;
  border-radius: 12px;
  background: rgba(var(--v-theme-on-surface), 0.03);
}
.jd-msg {
  display: flex;
  flex-direction: column;
  gap: 2px;
  max-width: 82%;
}
.jd-msg--customer { align-self: flex-start; }
.jd-msg--agent { align-self: flex-end; align-items: flex-end; }
.jd-msg__who {
  font-size: 11px;
  color: rgb(var(--v-theme-text-muted));
}
.jd-msg__bubble {
  padding: 8px 12px;
  border-radius: 12px;
  font-size: 14px;
  line-height: 1.45;
  border: 1px solid rgb(var(--v-theme-border));
  background: rgb(var(--v-theme-surface));
  overflow-wrap: anywhere;
}
.jd-msg--agent .jd-msg__bubble {
  background: rgba(var(--v-theme-primary), 0.08);
}
.jd-msg--quoted .jd-msg__bubble {
  outline: 2px solid rgb(var(--v-theme-primary));
  outline-offset: 2px;
}
.jd-msg__img {
  max-width: 180px;
  max-height: 180px;
  border-radius: 8px;
  cursor: pointer;
}
.jd-review {
  margin: 0;
  font-size: 14px;
  line-height: 1.55;
}
.jd-issue {
  display: flex;
  flex-direction: column;
  gap: 4px;
  width: 100%;
  min-height: 44px;
  padding: 10px 12px;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 12px;
  background: rgb(var(--v-theme-surface));
  color: rgb(var(--v-theme-on-surface));
  font: inherit;
  font-size: 14px;
  text-align: left;
  cursor: pointer;
}
.jd-issue--selected {
  border: 2px solid rgb(var(--v-theme-primary));
  background: rgba(var(--v-theme-primary), 0.06);
}
.jd-issue:focus-visible {
  outline: 2px solid rgb(var(--v-theme-primary));
  outline-offset: 2px;
}
.jd-issue__evidence {
  font-size: 13px;
  line-height: 1.45;
}
.jd-issue__quote {
  font-size: 12px;
  font-weight: 600;
  color: rgb(var(--v-theme-primary));
}
.jd-issue__quote--missing {
  color: rgb(var(--v-theme-src-unavailable));
}
.jd-sev {
  display: inline-flex;
  align-items: center;
  height: 22px;
  padding: 0 8px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
}
.jd-sev--critical { background: rgb(var(--v-theme-fail-bg)); color: rgb(var(--v-theme-fail)); }
.jd-sev--warning { background: rgb(var(--v-theme-src-changed-bg)); color: rgb(var(--v-theme-src-changed)); }
.lightbox-overlay { position: fixed; inset: 0; background: rgba(0, 0, 0, 0.85); display: flex; align-items: center; justify-content: center; z-index: 9999; cursor: pointer; }
.lightbox-img { max-width: 90vw; max-height: 90vh; object-fit: contain; border-radius: 8px; cursor: default; }
.lightbox-close { position: fixed; top: 16px; right: 16px; }

@media (max-width: 959px) {
  .jd-metrics { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .jd-detail { grid-template-columns: minmax(0, 1fr); }
  .jd-title { font-size: 24px; }
  .jd-scope__select { flex: 1 1 100%; min-width: 0; }
  .jd-transcript { max-height: none; }
}
</style>
