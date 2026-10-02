<!--
  Trang Kết quả — gom kết quả của mọi tác vụ trong công ty.
  Lọc và phân trang chạy dưới database (GET /results), không tải hết về máy khách.
  CCMAI-UX-011: dựng lại trên component UX-000; mọi số đếm ghi rõ phạm vi của nó
  (chip kết quả = mọi trang theo bộ lọc, trạng thái nguồn = chỉ trang đang xem).
-->
<template>
  <div class="rs-page">
    <div class="rs-head">
      <div class="rs-head__titles">
        <h1 class="rs-title">{{ $t('nav_results') }}</h1>
        <p class="rs-muted rs-subtitle">{{ $t('results_subtitle') }}</p>
      </div>
      <div v-if="!trangRong && !loiTai" class="rs-export">
        <div v-if="mdAndUp" class="d-flex ga-2">
          <v-btn variant="outlined" prepend-icon="mdi-download" :loading="dangXuat === 'csv'" data-testid="export-csv" @click="xuatFile('csv')">{{ $t('results_export_csv') }}</v-btn>
          <v-btn variant="outlined" prepend-icon="mdi-download" :loading="dangXuat === 'xlsx'" data-testid="export-xlsx" @click="xuatFile('xlsx')">{{ $t('results_export_xlsx') }}</v-btn>
        </div>
        <v-menu v-else>
          <template #activator="{ props }">
            <v-btn v-bind="props" variant="outlined" height="44" prepend-icon="mdi-download" append-icon="mdi-menu-down" :loading="!!dangXuat">{{ $t('results_export') }}</v-btn>
          </template>
          <v-list density="comfortable">
            <v-list-subheader class="rs-export__menu-caption">{{ $t('results_export_scope') }}</v-list-subheader>
            <v-list-item @click="xuatFile('csv')"><v-list-item-title>{{ $t('results_export_csv') }}</v-list-item-title></v-list-item>
            <v-list-item @click="xuatFile('xlsx')"><v-list-item-title>{{ $t('results_export_xlsx') }}</v-list-item-title></v-list-item>
          </v-list>
        </v-menu>
        <span v-if="mdAndUp" class="rs-muted rs-small" data-testid="export-scope">{{ $t('results_export_scope') }}</span>
      </div>
    </div>

    <v-skeleton-loader v-if="dangTaiFacets" type="article, table" />

    <!-- Không tải được danh sách tác vụ/kênh: không được coi là "chưa có tác vụ" -->
    <div v-else-if="loiTai === 'facets'" class="rs-error" role="alert" data-testid="results-load-error">
      <span class="rs-error__text"><v-icon size="20" aria-hidden="true">mdi-alert-circle-outline</v-icon>{{ $t('results_load_error') }}</span>
      <v-btn variant="outlined" color="error" height="44" @click="thuLai">{{ $t('results_retry') }}</v-btn>
    </div>

    <!-- Công ty chưa có tác vụ nào -->
    <div v-else-if="trangRong" class="rs-empty">
      <v-icon size="40" class="rs-muted" aria-hidden="true">mdi-clipboard-text-search-outline</v-icon>
      <h2 class="rs-empty__title">{{ $t('results_empty_title') }}</h2>
      <p class="rs-muted">{{ $t('results_empty_desc') }}</p>
      <v-btn color="primary" height="44" prepend-icon="mdi-robot" :to="`/${tenantId}/jobs`">{{ $t('nav_jobs') }}</v-btn>
    </div>

    <template v-else>
      <!-- Tab chỉ hiện khi công ty có cả hai loại tác vụ -->
      <v-tabs v-if="hienTab" v-model="jobType" class="rs-tabs" :grow="!mdAndUp">
        <v-tab value="qc_analysis">{{ $t('results_tab_qc') }}</v-tab>
        <v-tab value="classification">{{ $t('results_tab_classification') }}</v-tab>
      </v-tabs>

      <!-- Thanh lọc trên mobile: ô tìm + một nút mở bảng lọc -->
      <div v-if="!mdAndUp" class="d-flex align-center ga-2">
        <v-text-field
          v-model="tuKhoa"
          :placeholder="$t('results_search')"
          :aria-label="$t('results_search')"
          density="comfortable"
          variant="outlined"
          prepend-inner-icon="mdi-magnify"
          hide-details
          clearable
          class="flex-grow-1"
        />
        <v-btn variant="outlined" height="48" class="loc-nut" :color="soLoc ? 'primary' : undefined" @click="moLocMobile = true">
          <v-icon start size="small">mdi-filter-variant</v-icon>{{ $t('results_filters') }}<span v-if="soLoc" class="ml-1 tabular-nums">· {{ soLoc }}</span>
        </v-btn>
      </div>

      <!-- Thanh lọc trên desktop: mỗi bộ lọc là một nút nhỏ mở menu riêng -->
      <div v-else class="d-flex align-center flex-wrap ga-2">
        <v-text-field
          v-model="tuKhoa"
          :placeholder="$t('results_search')"
          :aria-label="$t('results_search')"
          density="compact"
          variant="outlined"
          prepend-inner-icon="mdi-magnify"
          hide-details
          clearable
          class="loc-tim"
        />

        <v-menu :close-on-content-click="false">
          <template #activator="{ props }">
            <v-btn v-bind="props" variant="outlined" class="loc-nut" :color="jobIDs.length ? 'primary' : undefined">
              {{ $t('results_filter_job') }}<span v-if="jobIDs.length" class="ml-1 tabular-nums">· {{ jobIDs.length }}</span>
              <v-icon end size="small">mdi-menu-down</v-icon>
            </v-btn>
          </template>
          <v-card min-width="260" max-height="360" class="overflow-y-auto">
            <v-list density="compact">
              <v-list-item v-for="j in dsTacVu" :key="j.id" @click="doiChon(jobIDs, j.id)">
                <template #prepend>
                  <v-checkbox-btn :model-value="jobIDs.includes(j.id)" density="compact" />
                </template>
                <v-list-item-title class="text-body-2">{{ j.name }}</v-list-item-title>
              </v-list-item>
            </v-list>
          </v-card>
        </v-menu>

        <v-menu :close-on-content-click="false">
          <template #activator="{ props }">
            <v-btn v-bind="props" variant="outlined" class="loc-nut" :color="channelIDs.length ? 'primary' : undefined">
              {{ $t('results_filter_channel') }}<span v-if="channelIDs.length" class="ml-1 tabular-nums">· {{ channelIDs.length }}</span>
              <v-icon end size="small">mdi-menu-down</v-icon>
            </v-btn>
          </template>
          <v-card min-width="240" max-height="360" class="overflow-y-auto">
            <v-list density="compact">
              <v-list-item v-for="k in facets.channels" :key="k.id" @click="doiChon(channelIDs, k.id)">
                <template #prepend>
                  <v-checkbox-btn :model-value="channelIDs.includes(k.id)" density="compact" />
                </template>
                <v-list-item-title class="text-body-2">{{ k.name }}</v-list-item-title>
              </v-list-item>
            </v-list>
          </v-card>
        </v-menu>

        <!-- Phân loại: lọc theo nhãn. Chất lượng CSKH: lọc theo điểm -->
        <v-menu v-if="laPhanLoai" :close-on-content-click="false">
          <template #activator="{ props }">
            <v-btn v-bind="props" variant="outlined" class="loc-nut" :color="tags.length ? 'primary' : undefined">
              {{ $t('results_filter_tag') }}<span v-if="tags.length" class="ml-1 tabular-nums">· {{ tags.length }}</span>
              <v-icon end size="small">mdi-menu-down</v-icon>
            </v-btn>
          </template>
          <v-card min-width="220" max-height="360" class="overflow-y-auto">
            <v-list density="compact">
              <v-list-item v-for="n in facets.tags" :key="n" @click="doiChon(tags, n)">
                <template #prepend>
                  <v-checkbox-btn :model-value="tags.includes(n)" density="compact" />
                </template>
                <v-list-item-title class="text-body-2">{{ n }}</v-list-item-title>
              </v-list-item>
            </v-list>
          </v-card>
        </v-menu>

        <v-menu v-else :close-on-content-click="false">
          <template #activator="{ props }">
            <v-btn v-bind="props" variant="outlined" class="loc-nut" :color="locDiem ? 'primary' : undefined">
              {{ locDiem ? `${$t('results_filter_score')} ${khoangDiem[0]}–${khoangDiem[1]}` : $t('results_filter_score') }}
              <v-icon end size="small">mdi-menu-down</v-icon>
            </v-btn>
          </template>
          <v-card min-width="300" class="pa-4">
            <div class="text-body-2 font-weight-medium mb-3">
              {{ $t('results_score_range', { min: khoangDiem[0], max: khoangDiem[1] }) }}
            </div>
            <v-range-slider v-model="khoangDiem" :min="0" :max="100" :step="5" density="compact" thumb-label hide-details />
            <div class="d-flex justify-end mt-3">
              <v-btn size="small" variant="text" @click="khoangDiem = [0, 100]">{{ $t('reset') }}</v-btn>
            </div>
          </v-card>
        </v-menu>

        <v-menu :close-on-content-click="false">
          <template #activator="{ props }">
            <v-btn v-bind="props" variant="outlined" class="loc-nut" :color="preset !== 'all' ? 'primary' : undefined">
              <v-icon start size="small">mdi-calendar</v-icon>
              {{ nhanThoiGian }}
              <v-icon end size="small">mdi-menu-down</v-icon>
            </v-btn>
          </template>
          <v-card min-width="330" class="pa-4">
            <v-btn-toggle v-model="dateField" density="compact" variant="outlined" divided mandatory class="mb-3 w-100">
              <v-btn value="conv" size="small" class="flex-grow-1">{{ $t('results_date_conv') }}</v-btn>
              <v-btn value="eval" size="small" class="flex-grow-1">{{ $t('results_date_eval') }}</v-btn>
            </v-btn-toggle>
            <div class="d-flex flex-wrap ga-1 mb-3">
              <v-chip
                v-for="p in presets"
                :key="p.value"
                size="small"
                :variant="preset === p.value ? 'flat' : 'outlined'"
                :color="preset === p.value ? 'primary' : ''"
                @click="apDungPreset(p.value)"
              >{{ $t(p.label) }}</v-chip>
            </div>
            <div class="d-flex ga-2">
              <v-text-field v-model="tuNgay" type="date" density="compact" variant="outlined" :label="$t('from_date')" hide-details @update:model-value="preset = 'custom'" />
              <v-text-field v-model="denNgay" type="date" density="compact" variant="outlined" :label="$t('to_date')" hide-details @update:model-value="preset = 'custom'" />
              <span class="text-caption text-medium-emphasis" data-testid="vn-date-note">{{ $t('vn_date_note') }}</span>
            </div>
          </v-card>
        </v-menu>

        <v-menu>
          <template #activator="{ props }">
            <v-btn v-bind="props" variant="outlined" class="loc-nut">
              <v-icon start size="small">mdi-sort</v-icon>
              {{ $t(nhanSapXep) }}
              <v-icon end size="small">mdi-menu-down</v-icon>
            </v-btn>
          </template>
          <v-list density="compact">
            <v-list-item v-for="s in dsSapXep" :key="s.value" @click="sort = s.value">
              <v-list-item-title class="text-body-2">{{ $t(s.label) }}</v-list-item-title>
            </v-list-item>
          </v-list>
        </v-menu>

        <v-btn v-if="coLoc" variant="text" color="primary" class="loc-nut" @click="xoaLoc">
          {{ $t('results_clear_filter') }}
        </v-btn>

        <v-spacer />
        <v-btn-toggle v-model="cheDoXem" density="compact" variant="outlined" divided mandatory :aria-label="$t('results_view_label')">
          <v-btn value="table" size="small" prepend-icon="mdi-table" :aria-pressed="cheDoXem === 'table'">{{ $t('results_view_table') }}</v-btn>
          <v-btn value="card" size="small" prepend-icon="mdi-view-agenda-outline" :aria-pressed="cheDoXem === 'card'">{{ $t('results_view_card') }}</v-btn>
        </v-btn-toggle>
      </div>

      <!-- Chip kết quả: số đếm của máy chủ trên mọi trang, theo các bộ lọc còn lại -->
      <div class="rs-verdicts">
        <FilterBar :label="$t('results_verdict_filter_label')" class="rs-verdicts__bar" data-testid="verdict-chips">
          <v-chip
            v-for="c in chipKetQua"
            :key="c.value"
            :variant="verdict === c.value ? 'flat' : 'outlined'"
            :color="verdict === c.value ? 'primary' : undefined"
            :aria-pressed="verdict === c.value"
            @click="datVerdict(c.value)"
          >{{ c.label }} · <span class="tabular-nums ml-1">{{ c.count }}</span></v-chip>
        </FilterBar>
        <span class="rs-muted rs-small" data-testid="counts-scope">{{ $t('results_counts_scope') }}</span>
      </div>

      <v-skeleton-loader v-if="dangTai" type="table-row@5" />

      <div v-else-if="loiTai === 'results'" class="rs-error" role="alert" data-testid="results-load-error">
        <span class="rs-error__text"><v-icon size="20" aria-hidden="true">mdi-alert-circle-outline</v-icon>{{ $t('results_load_error') }}</span>
        <v-btn variant="outlined" color="error" height="44" @click="thuLai">{{ $t('results_retry') }}</v-btn>
      </div>

      <!-- Có tác vụ nhưng chưa chạy lần nào -->
      <div v-else-if="!items.length && !coLoc && !counts.all" class="rs-empty">
        <v-icon size="40" class="rs-muted" aria-hidden="true">mdi-play-circle-outline</v-icon>
        <h2 class="rs-empty__title">{{ $t('results_no_run_title') }}</h2>
        <p class="rs-muted">{{ $t('results_no_run_desc') }}</p>
        <v-btn color="primary" height="44" prepend-icon="mdi-play" :to="`/${tenantId}/jobs`">{{ $t('nav_jobs') }}</v-btn>
      </div>

      <div v-else-if="!items.length" class="rs-empty" data-testid="results-no-match">
        <p class="rs-muted">{{ $t('results_no_match') }}</p>
        <v-btn variant="outlined" height="44" data-testid="no-match-clear" @click="xoaLoc">{{ $t('results_clear_filter') }}</v-btn>
      </div>

      <template v-else>
        <!-- CCMAI-UX-002/UX-011: ghi chú "so sánh cục bộ" luôn ở trên danh sách; số đếm chỉ của trang này -->
        <SourceStatusPanel
          :statuses="items.map((r) => r.source_integrity_status)"
          :scope="$t('results_source_scope_page', { n: items.length })"
          data-testid="results-source-panel"
        />

        <!-- Bảng: cột Nguồn có chữ, không phải rê chuột mới đọc được -->
        <div v-if="xemBang" class="rs-table-wrap">
          <v-table density="comfortable" hover>
            <thead>
              <tr>
                <th data-testid="results-source-header">{{ $t('results_col_source') }}</th>
                <th class="rs-col-customer">{{ $t('results_col_customer') }}</th>
                <template v-if="!laPhanLoai">
                  <th>{{ $t('results_col_verdict') }}</th>
                  <th class="text-right">{{ $t('results_col_score') }}</th>
                  <th class="rs-col-wide">{{ $t('results_col_issues') }}</th>
                </template>
                <th v-else class="rs-col-wide">{{ $t('results_col_tags') }}</th>
                <th>{{ $t('results_col_date') }}</th>
                <th class="d-none d-lg-table-cell rs-col-job">{{ $t('results_col_job') }} · {{ $t('results_col_channel') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="r in items"
                :key="r.id"
                class="rs-row"
                :class="`rs-row--${normalizeSourceStatus(r.source_integrity_status)}`"
                tabindex="0"
                @click="moChiTiet(r)"
                @keydown.enter="moChiTiet(r)"
                @keydown.space.prevent="moChiTiet(r)"
              >
                <td><SourceStatusChip :status="r.source_integrity_status" small /></td>
                <td class="font-weight-medium">{{ r.customer_name || '—' }}</td>
                <template v-if="!laPhanLoai">
                  <td><VerdictChip :verdict="verdictOf(r)" small /></td>
                  <td class="text-right tabular-nums font-weight-medium">{{ r.severity === 'SKIP' || r.score === null ? '—' : r.score }}</td>
                  <td><span class="rs-clamp" :class="{ 'rs-muted': !r.issues.length }">{{ tomTatVanDe(r) }}</span></td>
                </template>
                <td v-else>
                  <div v-if="r.tags.length" class="d-flex flex-wrap ga-1">
                    <span v-for="t in r.tags" :key="t" class="rs-tag">{{ t }}</span>
                  </div>
                  <VerdictChip v-else-if="r.severity === 'SKIP'" verdict="skip" small />
                  <span v-else class="rs-muted">—</span>
                </td>
                <td class="rs-muted rs-nowrap">{{ hienNgayGio(r.conversation_at) }}</td>
                <td class="d-none d-lg-table-cell">
                  <span class="rs-cell-2">{{ r.job_name }}</span>
                  <span class="rs-cell-2 rs-muted rs-small">{{ r.channel_name }}</span>
                </td>
              </tr>
            </tbody>
          </v-table>
        </div>

        <!-- Thẻ: bấm để mở hộp thoại chi tiết (có diễn biến cuộc chat) -->
        <div v-else class="rs-cards">
          <ResultCard
            v-for="r in items"
            :key="r.id"
            :customer-name="r.customer_name"
            :time="hienNgayGio(r.conversation_at)"
            :verdict="verdictOf(r)"
            :source-status="r.source_integrity_status"
            :summary="tomTatThe(r)"
            :score="!laPhanLoai && r.severity !== 'SKIP' ? r.score : null"
            :meta="`${r.job_name} · ${r.channel_name}`"
            @open="moChiTiet(r)"
          />
        </div>

        <div class="rs-pager">
          <span class="rs-muted rs-small" data-testid="results-range">{{ $t('results_range', { from: tuDong, to: denDong, total }) }}</span>
          <v-pagination v-if="tongTrang > 1" v-model="page" :length="tongTrang" density="comfortable" :total-visible="mdAndUp ? 7 : 3" />
        </div>
      </template>
    </template>

    <!-- Bảng lọc trên mobile -->
    <v-bottom-sheet v-model="moLocMobile">
      <v-card>
        <v-card-title class="d-flex align-center text-subtitle-1">
          {{ $t('results_filters') }}
          <v-spacer />
          <v-btn icon variant="text" size="44" :aria-label="$t('close')" @click="moLocMobile = false">
            <v-icon>mdi-close</v-icon>
          </v-btn>
        </v-card-title>
        <v-divider />
        <v-card-text class="pt-4" style="max-height: 70vh; overflow-y: auto">
          <v-select
            v-model="jobIDs"
            :items="dsTacVu"
            item-title="name"
            item-value="id"
            :label="$t('results_filter_job')"
            density="compact"
            variant="outlined"
            multiple
            chips
            class="mb-3"
            hide-details
          />
          <v-select
            v-model="channelIDs"
            :items="facets.channels"
            item-title="name"
            item-value="id"
            :label="$t('results_filter_channel')"
            density="compact"
            variant="outlined"
            multiple
            chips
            class="mb-3"
            hide-details
          />
          <v-select
            v-if="laPhanLoai"
            v-model="tags"
            :items="facets.tags"
            :label="$t('results_filter_tag')"
            density="compact"
            variant="outlined"
            multiple
            chips
            class="mb-3"
            hide-details
          />
          <template v-else>
            <div class="text-body-2 mb-1">{{ $t('results_score_range', { min: khoangDiem[0], max: khoangDiem[1] }) }}</div>
            <v-range-slider v-model="khoangDiem" :min="0" :max="100" :step="5" density="compact" thumb-label hide-details class="mb-3 px-1" />
          </template>

          <div class="text-body-2 mb-1">{{ $t('results_time') }}</div>
          <v-btn-toggle v-model="dateField" density="compact" variant="outlined" divided mandatory class="mb-3 w-100">
            <v-btn value="conv" size="small" class="flex-grow-1">{{ $t('results_date_conv') }}</v-btn>
            <v-btn value="eval" size="small" class="flex-grow-1">{{ $t('results_date_eval') }}</v-btn>
          </v-btn-toggle>
          <div class="d-flex flex-wrap ga-1 mb-3">
            <v-chip
              v-for="p in presets"
              :key="p.value"
              size="small"
              :variant="preset === p.value ? 'flat' : 'outlined'"
              :color="preset === p.value ? 'primary' : ''"
              @click="apDungPreset(p.value)"
            >{{ $t(p.label) }}</v-chip>
          </div>
          <div class="d-flex ga-2 mb-3">
            <v-text-field v-model="tuNgay" type="date" density="compact" variant="outlined" :label="$t('from_date')" hide-details @update:model-value="preset = 'custom'" />
            <v-text-field v-model="denNgay" type="date" density="compact" variant="outlined" :label="$t('to_date')" hide-details @update:model-value="preset = 'custom'" />
            <span class="text-caption text-medium-emphasis" data-testid="vn-date-note-mobile">{{ $t('vn_date_note') }}</span>
          </div>

          <v-select
            v-model="sort"
            :items="dsSapXepMobile"
            item-title="title"
            item-value="value"
            :label="$t('results_sort')"
            density="compact"
            variant="outlined"
            hide-details
          />
        </v-card-text>
        <v-divider />
        <v-card-actions>
          <v-btn variant="text" @click="xoaLoc">{{ $t('results_clear_filter') }}</v-btn>
          <v-spacer />
          <v-btn color="primary" variant="flat" @click="moLocMobile = false">{{ $t('results_apply') }}</v-btn>
        </v-card-actions>
      </v-card>
    </v-bottom-sheet>

    <!-- Chi tiết một kết quả -->
    <AppDialog
      v-if="chiTiet"
      v-model="moChiTietDialog"
      :title="chiTiet.customer_name || '—'"
      :subtitle="`${chiTiet.job_name} · ${chiTiet.channel_name}`"
      :max-width="1100"
    >
      <template #meta>
        <VerdictChip :verdict="verdictOf(chiTiet)" small />
        <span v-if="!laPhanLoai && chiTiet.severity !== 'SKIP' && chiTiet.score !== null" class="tabular-nums font-weight-bold rs-on-surface">{{ chiTiet.score }}/100</span>
        <SourceStatusChip :status="chiTiet.source_integrity_status" small />
      </template>

      <div class="rs-detail" data-testid="results-dialog">
        <section class="rs-detail__side">
          <section
            class="rs-src-block"
            :class="`rs-src-block--${normalizeSourceStatus(chiTiet.source_integrity_status)}`"
            :aria-label="$t('ui_source_panel_title')"
          >
            <div class="d-flex align-center flex-wrap ga-2">
              <h3 class="rs-h3">{{ $t('ui_source_panel_title') }}</h3>
              <SourceStatusChip :status="chiTiet.source_integrity_status" small />
            </div>
            <p v-if="goiYNguon(chiTiet)" class="rs-src-block__hint" data-testid="source-hint">{{ goiYNguon(chiTiet) }}</p>
            <p class="rs-muted rs-small rs-m0" data-testid="results-dialog-source-note">{{ $t('results_source_note') }}</p>
          </section>

          <dl class="rs-dates">
            <div><dt class="rs-muted">{{ $t('results_date_conv') }}</dt><dd>{{ ngayGioDay(chiTiet.conversation_at) }}</dd></div>
            <div><dt class="rs-muted">{{ $t('results_date_eval') }}</dt><dd>{{ ngayGioDay(chiTiet.evaluated_at) }}</dd></div>
          </dl>

          <div v-if="chiTiet.review" class="rs-stack-sm">
            <div class="d-flex align-center justify-space-between ga-2">
              <h3 class="rs-h3">{{ $t('results_review') }}</h3>
              <AiGeneratedLabel />
            </div>
            <p class="rs-body rs-m0">{{ chiTiet.review }}</p>
          </div>

          <!-- Chất lượng CSKH: vấn đề. Phân loại: nhãn và căn cứ gắn nhãn — không bao giờ gọi là "vấn đề" -->
          <div v-if="!laPhanLoai" class="rs-stack-sm" data-testid="dialog-issues">
            <h3 class="rs-h3">{{ $t('results_issues_title', { n: chiTiet.issues.length }) }}</h3>
            <p v-if="!chiTiet.issues.length && chiTiet.severity !== 'SKIP'" class="rs-muted rs-m0">{{ $t('results_no_issues') }}</p>
            <div v-for="(i, idx) in chiTiet.issues" :key="idx" class="rs-issue">
              <span class="d-flex align-center flex-wrap ga-2">
                <span v-if="i.severity === 'NGHIEM_TRONG' || i.severity === 'CAN_CAI_THIEN'" :class="`rs-sev rs-sev--${i.severity === 'NGHIEM_TRONG' ? 'critical' : 'warning'}`">
                  {{ i.severity === 'NGHIEM_TRONG' ? $t('severity_critical') : $t('severity_warning') }}
                </span>
                <b>{{ i.rule_name }}</b>
              </span>
              <span v-if="i.evidence" class="rs-quote">{{ i.evidence }}</span>
            </div>
          </div>
          <template v-else>
            <div class="rs-stack-sm" data-testid="dialog-tags">
              <h3 class="rs-h3">{{ $t('results_tags_title', { n: chiTiet.tags.length }) }}</h3>
              <div v-if="chiTiet.tags.length" class="d-flex flex-wrap ga-2">
                <span v-for="t in chiTiet.tags" :key="t" class="rs-tag rs-tag--lg">{{ t }}</span>
              </div>
              <p v-else class="rs-muted rs-m0">—</p>
            </div>
            <div v-if="chiTiet.issues.length" class="rs-stack-sm" data-testid="dialog-tag-evidence">
              <h3 class="rs-h3">{{ $t('results_tag_evidence') }}</h3>
              <div v-for="(i, idx) in chiTiet.issues" :key="idx" class="rs-issue">
                <b>{{ i.rule_name }}</b>
                <span v-if="i.evidence" class="rs-quote rs-quote--tag">{{ i.evidence }}</span>
              </div>
            </div>
          </template>
        </section>

        <section class="rs-detail__chat" :aria-label="$t('results_transcript')">
          <div class="d-flex align-center justify-space-between flex-wrap ga-2">
            <h3 class="rs-h3">{{ $t('results_transcript') }}</h3>
            <v-btn :to="`/${tenantId}/messages?conv=${chiTiet.conversation_id}`" variant="text" color="primary" prepend-icon="mdi-open-in-new">
              {{ $t('results_open_messages') }}
            </v-btn>
          </div>
          <p v-if="chat.khongCoQuyen.value" class="rs-muted rs-m0">{{ $t('results_transcript_denied') }}</p>
          <div v-else-if="!chat.messages.value[chiTiet.conversation_id]" class="text-center pa-4">
            <v-progress-circular indeterminate size="24" />
          </div>
          <p v-else-if="!chat.messages.value[chiTiet.conversation_id].length" class="rs-muted rs-m0">{{ $t('results_transcript_empty') }}</p>
          <div v-else class="rs-transcript">
            <div
              v-for="msg in chat.messages.value[chiTiet.conversation_id]"
              :key="msg.id"
              class="rs-msg"
              :class="msg.sender_type === 'agent' ? 'rs-msg--agent' : 'rs-msg--customer'"
            >
              <div class="rs-msg__who">{{ msg.sender_name }} · {{ hienNgayGio(msg.sent_at) }}</div>
              <div class="rs-msg__bubble">
                <div v-if="msg.content">{{ msg.content }}</div>
                <div v-if="msg.content_type === 'sticker'" class="font-italic">[Sticker]</div>
                <div v-if="chat.hasAttachments(msg)" class="mt-1">
                  <template v-for="(att, ai) in chat.parseAttachments(msg)" :key="ai">
                    <div v-if="chat.isImageAttachment(att)" class="mb-1">
                      <img
                        v-if="anhSanSang(att)"
                        :src="chat.anhCache.value[chat.getAttachmentUrl(att)]"
                        alt=""
                        class="rs-msg__img"
                        @click="anhPhongTo = chat.anhCache.value[chat.getAttachmentUrl(att)]"
                      />
                      <v-progress-circular v-else indeterminate size="20" width="2" class="ma-2" />
                    </div>
                    <v-chip v-else size="x-small" variant="tonal" class="mr-1" :href="chat.getAttachmentUrl(att)" target="_blank">
                      <v-icon start size="12">mdi-paperclip</v-icon>{{ att.name || 'File' }}
                    </v-chip>
                  </template>
                </div>
              </div>
            </div>
          </div>
        </section>
      </div>

      <template #actions>
        <v-btn variant="outlined" :to="`/${tenantId}/jobs/${chiTiet.job_id}`">{{ $t('results_open_job') }}</v-btn>
        <v-btn variant="flat" color="primary" @click="moChiTietDialog = false">{{ $t('close') }}</v-btn>
      </template>
    </AppDialog>

    <v-dialog :model-value="!!anhPhongTo" max-width="900" @update:model-value="anhPhongTo = ''">
      <v-img :src="anhPhongTo" contain style="background: rgba(0, 0, 0, 0.9)" @click="anhPhongTo = ''" />
    </v-dialog>

    <v-snackbar v-model="baoLoi" color="error" timeout="6000">{{ noiDungLoi }}</v-snackbar>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import { useDisplay } from 'vuetify'
import { useI18n } from 'vue-i18n'
import api from '../api'
import { useChatTranscript } from '../composables/useChatTranscript'
import AppDialog from '../components/ui/AppDialog.vue'
import AiGeneratedLabel from '../components/ui/AiGeneratedLabel.vue'
import FilterBar from '../components/ui/FilterBar.vue'
import ResultCard from '../components/ui/ResultCard.vue'
import SourceStatusChip from '../components/ui/SourceStatusChip.vue'
import SourceStatusPanel from '../components/ui/SourceStatusPanel.vue'
import VerdictChip from '../components/ui/VerdictChip.vue'
import { normalizeSourceStatus, verdictFromSeverity, type Verdict } from '../utils/review'
import { formatDateTime, type UiLocale } from '../utils/format'
import { presetRange, type DatePreset } from '../utils/businessDay'

interface IssueItem {
  rule_name: string
  evidence: string
  severity: string
}

interface ResultItem {
  id: string
  conversation_id: string
  job_id: string
  job_name: string
  channel_id: string
  channel_name: string
  customer_name: string
  conversation_at: string | null
  evaluated_at: string
  severity: string
  review: string
  score: number | null
  issues: IssueItem[]
  tags: string[]
  source_integrity_status: string
}

interface JobFacet { id: string; name: string; job_type: string }
interface ChannelFacet { id: string; name: string }

const route = useRoute()
const { t, locale } = useI18n()
const { mdAndUp } = useDisplay()

const tenantId = computed(() => route.params.tenantId as string)

const dangTaiFacets = ref(true)
const dangTai = ref(false)
const dangXuat = ref('')
const baoLoi = ref(false)
const noiDungLoi = ref('')
// Lỗi tải hiện ngay tại chỗ có nút thử lại, không chỉ là snackbar thoáng qua
const loiTai = ref<'' | 'facets' | 'results'>('')

const facets = ref<{
  types: Record<string, { jobs: number; results: number }>
  jobs: JobFacet[]
  channels: ChannelFacet[]
  tags: string[]
}>({ types: {}, jobs: [], channels: [], tags: [] })

const jobType = ref<'qc_analysis' | 'classification'>('qc_analysis')
const verdict = ref('all')
const tuKhoa = ref('')
const jobIDs = ref<string[]>([])
const channelIDs = ref<string[]>([])
const tags = ref<string[]>([])
const khoangDiem = ref<[number, number]>([0, 100])
const dateField = ref<'conv' | 'eval'>('conv')
const preset = ref('all')
const tuNgay = ref('')
const denNgay = ref('')
const sort = ref('recent')
const page = ref(1)
const pageSize = 25
// Nhớ kiểu xem đã chọn để lần sau vào khỏi phải bấm lại
const KHOA_KIEU_XEM = 'cqa_results_view'
function docKieuXem(): 'card' | 'table' {
  try {
    return localStorage.getItem(KHOA_KIEU_XEM) === 'card' ? 'card' : 'table'
  } catch {
    return 'table'
  }
}
const cheDoXem = ref<'card' | 'table'>(docKieuXem())
const moLocMobile = ref(false)

const items = ref<ResultItem[]>([])
const total = ref(0)
const counts = ref({ all: 0, pass: 0, fail: 0, skip: 0, classified: 0 })

const chiTiet = ref<ResultItem | null>(null)
const moChiTietDialog = ref(false)

const anhPhongTo = ref('')
const chat = useChatTranscript()

const presets = [
  { label: 'results_preset_all', value: 'all' },
  { label: 'today', value: 'today' },
  { label: 'results_preset_7days', value: '7days' },
  { label: 'results_preset_28days', value: '28days' },
  { label: 'results_preset_month', value: 'month' },
]
const dsSapXep = [
  { label: 'results_sort_recent', value: 'recent' },
  { label: 'results_sort_score_asc', value: 'score_asc' },
  { label: 'results_sort_score_desc', value: 'score_desc' },
]
const dsSapXepMobile = computed(() => dsSapXep.map(s => ({ title: t(s.label), value: s.value })))

const laPhanLoai = computed(() => jobType.value === 'classification')
const coQC = computed(() => (facets.value.types.qc_analysis?.jobs || 0) > 0)
const coPhanLoai = computed(() => (facets.value.types.classification?.jobs || 0) > 0)
const hienTab = computed(() => coQC.value && coPhanLoai.value)
const trangRong = computed(() => !dangTaiFacets.value && !loiTai.value && !coQC.value && !coPhanLoai.value)
const xemBang = computed(() => mdAndUp.value && cheDoXem.value === 'table')
const tongTrang = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
// Khoảng dòng của trang đang xem, trên tổng số của máy chủ
const tuDong = computed(() => (items.value.length ? (page.value - 1) * pageSize + 1 : 0))
const denDong = computed(() => (page.value - 1) * pageSize + items.value.length)

const chipKetQua = computed(() =>
  laPhanLoai.value
    ? [
        { value: 'all', label: t('filter_all'), count: counts.value.all },
        { value: 'classified', label: t('results_classified'), count: counts.value.classified },
        { value: 'skip', label: t('verdict_skip'), count: counts.value.skip },
      ]
    : [
        { value: 'all', label: t('filter_all'), count: counts.value.all },
        { value: 'fail', label: t('filter_failed'), count: counts.value.fail },
        { value: 'pass', label: t('filter_passed'), count: counts.value.pass },
        { value: 'skip', label: t('verdict_skip'), count: counts.value.skip },
      ],
)

const locDiem = computed(() => khoangDiem.value[0] !== 0 || khoangDiem.value[1] !== 100)
const soLoc = computed(
  () => jobIDs.value.length + channelIDs.value.length + tags.value.length + (locDiem.value ? 1 : 0) + (preset.value !== 'all' ? 1 : 0)
)
const coLoc = computed(() => !!tuKhoa.value || soLoc.value > 0 || verdict.value !== 'all')

// Chỉ liệt kê tác vụ đúng loại tab đang mở
const dsTacVu = computed(() => facets.value.jobs.filter(j => j.job_type === jobType.value))

const nhanSapXep = computed(() => dsSapXep.find(s => s.value === sort.value)?.label || 'results_sort_recent')
const nhanThoiGian = computed(() => {
  const moc = dateField.value === 'conv' ? t('results_date_conv_short') : t('results_date_eval_short')
  const p = presets.find(x => x.value === preset.value)
  if (p) return `${t(p.label)} · ${moc}`
  return `${hienNgay(tuNgay.value)} – ${hienNgay(denNgay.value)} · ${moc}`
})

function hienNgay(s: string) {
  if (!s) return '—'
  const [, m, d] = s.split('-')
  return `${d}/${m}`
}

function hienNgayGio(s: string | null) {
  if (!s) return '—'
  const d = new Date(s)
  if (Number.isNaN(d.getTime())) return '—'
  const hai = (n: number) => String(n).padStart(2, '0')
  return `${hai(d.getDate())}/${hai(d.getMonth() + 1)} ${hai(d.getHours())}:${hai(d.getMinutes())}`
}

function ngayGioDay(s: string | null) {
  return formatDateTime(s, locale.value as UiLocale)
}

// Template đã tự bóc ref nên nhận thẳng mảng; sửa tại chỗ vẫn giữ tính phản ứng.
function doiChon(arr: string[], v: string) {
  const i = arr.indexOf(v)
  if (i >= 0) arr.splice(i, 1)
  else arr.push(v)
}

// CCMAI-RUNTIME-026: presets are Vietnam calendar dates (UTC+7), independent of the browser zone.
function apDungPreset(v: string) {
  preset.value = v
  if (v === 'all') {
    tuNgay.value = ''
    denNgay.value = ''
    return
  }
  const range = presetRange(v as DatePreset)
  tuNgay.value = range.from
  denNgay.value = range.to
}

function xoaLoc() {
  tuKhoa.value = ''
  jobIDs.value = []
  channelIDs.value = []
  tags.value = []
  khoangDiem.value = [0, 100]
  verdict.value = 'all'
  apDungPreset('all')
}

function datVerdict(v: string) {
  verdict.value = verdict.value === v ? 'all' : v
}

// Phân loại không chấm đạt/không đạt: chỉ "đã phân loại" hoặc "bỏ qua"
function verdictOf(r: ResultItem): Verdict {
  if (laPhanLoai.value) return r.severity === 'SKIP' ? 'skip' : 'classified'
  return verdictFromSeverity(r.severity)
}

// CCMAI-RUNTIME-004: chỉ giải thích hai trạng thái cần xem lại; không bao giờ nói "an toàn".
function goiYNguon(r: ResultItem) {
  const s = normalizeSourceStatus(r.source_integrity_status)
  if (s === 'changed_since_analysis') return t('results_source_changed_hint')
  if (s === 'verification_unavailable') return t('results_source_unavailable_hint')
  return ''
}

function tomTatVanDe(r: ResultItem) {
  if (!r.issues.length) return r.severity === 'SKIP' ? t('results_skip_reason', { reason: r.review || '—' }) : '—'
  return r.issues.map(i => (i.evidence ? `${i.rule_name}: ${i.evidence}` : i.rule_name)).join('; ')
}

function tomTatThe(r: ResultItem) {
  if (laPhanLoai.value) return r.tags.length ? t('results_tags_summary', { tags: r.tags.join(', ') }) : r.review
  if (r.issues.length) return t('results_issue_summary', { n: r.issues.length, names: r.issues.map(i => i.rule_name).join(', ') })
  return r.review
}

function anhSanSang(att: { url?: string; local_path?: string }) {
  const url = chat.getAttachmentUrl(att)
  const cache = chat.anhCache.value[url]
  return !!cache && cache !== 'loading'
}

function moChiTiet(r: ResultItem) {
  chiTiet.value = r
  moChiTietDialog.value = true
  chat.loadMessages(tenantId.value, r.conversation_id)
}

function thamSo() {
  const p: Record<string, string> = {
    job_type: jobType.value,
    verdict: verdict.value,
    date_field: dateField.value,
    sort: sort.value,
  }
  if (tuKhoa.value) p.q = tuKhoa.value
  if (jobIDs.value.length) p.job_ids = jobIDs.value.join(',')
  if (channelIDs.value.length) p.channel_ids = channelIDs.value.join(',')
  if (tags.value.length) p.tags = tags.value.join(',')
  if (tuNgay.value) p.from = tuNgay.value
  if (denNgay.value) p.to = denNgay.value
  if (locDiem.value) {
    p.score_min = String(khoangDiem.value[0])
    p.score_max = String(khoangDiem.value[1])
  }
  return p
}

async function taiFacets() {
  dangTaiFacets.value = true
  loiTai.value = ''
  try {
    const { data } = await api.get(`/tenants/${tenantId.value}/results/facets`)
    facets.value = data
    // Công ty chỉ chạy một loại tác vụ thì vào thẳng loại đó, không có tab rỗng
    if (!coQC.value && coPhanLoai.value) jobType.value = 'classification'
    else jobType.value = 'qc_analysis'
  } catch {
    loiTai.value = 'facets'
  } finally {
    dangTaiFacets.value = false
  }
}

async function taiKetQua() {
  if (trangRong.value || loiTai.value === 'facets') return
  dangTai.value = true
  loiTai.value = ''
  try {
    const { data } = await api.get(`/tenants/${tenantId.value}/results`, {
      params: { ...thamSo(), page: page.value, page_size: pageSize },
    })
    items.value = data.items || []
    total.value = data.total || 0
    counts.value = data.counts || { all: 0, pass: 0, fail: 0, skip: 0, classified: 0 }
  } catch {
    loiTai.value = 'results'
  } finally {
    dangTai.value = false
  }
}

async function thuLai() {
  if (loiTai.value === 'facets') {
    await taiFacets()
    if (loiTai.value) return
  }
  await taiKetQua()
}

async function xuatFile(format: 'csv' | 'xlsx') {
  dangXuat.value = format
  try {
    const { data } = await api.get(`/tenants/${tenantId.value}/results/export`, {
      params: { ...thamSo(), format },
      responseType: 'blob',
    })
    const loai = format === 'xlsx'
      ? 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
      : 'text/csv;charset=utf-8'
    const url = URL.createObjectURL(new Blob([data], { type: loai }))
    const a = document.createElement('a')
    a.href = url
    a.download = `${laPhanLoai.value ? 'phan-loai' : 'ket-qua'}.${format}`
    a.click()
    URL.revokeObjectURL(url)
  } catch (e: any) {
    // Lỗi trả về dạng blob khi responseType là blob nên phải đọc lại thành chữ
    let ma = ''
    let gioiHan = 0
    try {
      const text = await e?.response?.data?.text?.()
      const body = text ? JSON.parse(text) : null
      ma = body?.error || ''
      gioiHan = body?.limit || 0
    } catch { /* giữ thông báo chung */ }
    hienLoi(ma === 'export_too_large' ? t('results_export_too_large', { limit: gioiHan }) : t('results_export_error'))
  } finally {
    dangXuat.value = ''
  }
}

function hienLoi(msg: string) {
  noiDungLoi.value = msg
  baoLoi.value = true
}

let henGio: ReturnType<typeof setTimeout> | null = null
function taiLai(doiTrang = true) {
  if (doiTrang) page.value = 1
  if (henGio) clearTimeout(henGio)
  henGio = setTimeout(taiKetQua, 250)
}

watch(jobType, () => {
  // Bộ lọc của hai loại tác vụ không dùng chung được
  jobIDs.value = []
  tags.value = []
  verdict.value = 'all'
  khoangDiem.value = [0, 100]
  taiLai()
})
watch([verdict, dateField, sort, tuNgay, denNgay], () => taiLai())
watch([tuKhoa, jobIDs, channelIDs, tags, khoangDiem], () => taiLai(), { deep: true })
watch(page, () => taiKetQua())
watch(cheDoXem, (v) => {
  try {
    localStorage.setItem(KHOA_KIEU_XEM, v)
  } catch { /* trình duyệt chặn lưu thì bỏ qua, không ảnh hưởng gì */ }
})

onMounted(async () => {
  await taiFacets()
  await taiKetQua()
})
// Rời trang trong lúc đang chờ debounce thì không gửi thêm yêu cầu nữa
onBeforeUnmount(() => {
  if (henGio) clearTimeout(henGio)
})
</script>

<style scoped>
.rs-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.rs-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 12px 16px;
}
.rs-head__titles {
  flex: 1 1 280px;
  min-width: 0;
}
.rs-title {
  margin: 0;
  font-size: 26px;
  font-weight: 700;
  line-height: 1.25;
}
.rs-subtitle {
  margin: 4px 0 0;
  font-size: 14px;
}
.rs-export {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 6px;
}
.rs-export__menu-caption {
  white-space: normal;
  line-height: 1.4;
  max-width: 280px;
}
.rs-muted {
  color: rgb(var(--v-theme-text-muted));
}
.rs-on-surface {
  color: rgb(var(--v-theme-on-surface));
}
.rs-small {
  font-size: 12px;
}
.rs-m0 {
  margin: 0;
}
.rs-body {
  font-size: 14px;
  line-height: 1.55;
}
.rs-h3 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
}
.rs-stack-sm {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.rs-verdicts {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px 12px;
}
.rs-verdicts__bar {
  min-width: 0;
  max-width: 100%;
}
.rs-error {
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
}
.rs-error__text {
  display: inline-flex;
  align-items: flex-start;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
  line-height: 1.45;
}
.rs-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 32px 20px;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 12px;
  background: rgb(var(--v-theme-surface));
  text-align: center;
}
.rs-empty p {
  margin: 0 0 4px;
  font-size: 14px;
}
.rs-empty__title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}
.rs-table-wrap {
  overflow-x: auto;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 12px;
  background: rgb(var(--v-theme-surface));
}
.rs-table-wrap :deep(th) {
  font-size: 12px !important;
  font-weight: 600 !important;
  color: rgb(var(--v-theme-text-muted)) !important;
  white-space: nowrap;
}
.rs-table-wrap :deep(th),
.rs-table-wrap :deep(td) {
  padding-left: 12px !important;
  padding-right: 12px !important;
}
.rs-table-wrap :deep(td) {
  height: auto !important;
  padding-top: 10px !important;
  padding-bottom: 10px !important;
  font-size: 14px;
}
.rs-col-wide {
  width: 30%;
  min-width: 180px;
}
.rs-col-customer {
  min-width: 140px;
}
.rs-col-job {
  min-width: 150px;
}
.rs-cell-2 {
  display: block;
  line-height: 1.4;
}
.rs-row {
  cursor: pointer;
}
.rs-row:focus-visible {
  outline: 2px solid rgb(var(--v-theme-primary));
  outline-offset: -2px;
}
.rs-row--changed_since_analysis {
  background: rgba(var(--v-theme-src-changed), 0.06);
}
.rs-row--changed_since_analysis td:first-child {
  box-shadow: inset 4px 0 0 rgb(var(--v-theme-src-changed));
}
.rs-row--verification_unavailable td:first-child {
  box-shadow: inset 4px 0 0 rgb(var(--v-theme-src-unavailable));
}
.rs-nowrap {
  white-space: nowrap;
}
/* Vấn đề thường dài; cắt còn 2 dòng cho bảng dễ quét, bấm vào dòng xem đủ */
.rs-clamp {
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.rs-tag {
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
.rs-tag--lg {
  height: 28px;
  padding: 0 10px;
  font-size: 13px;
}
.rs-cards {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.rs-pager {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 8px 16px;
}
.rs-detail {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 20px;
}
@media (min-width: 960px) {
  .rs-detail {
    grid-template-columns: minmax(0, 7fr) minmax(0, 5fr);
  }
  /* Desktop: hội thoại bên trái như canvas; mobile: phần đánh giá đọc trước */
  .rs-detail__chat {
    order: -1;
  }
}
.rs-detail__side {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.rs-detail__chat {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}
.rs-src-block {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 12px 14px;
  border: 1px solid rgb(var(--v-theme-border));
  border-left: 4px solid rgb(var(--v-theme-src-legacy));
  border-radius: 12px;
}
.rs-src-block--changed_since_analysis {
  border-left-color: rgb(var(--v-theme-src-changed));
}
.rs-src-block--verification_unavailable {
  border-left-color: rgb(var(--v-theme-src-unavailable));
}
.rs-src-block__hint {
  margin: 0;
  font-size: 13px;
  line-height: 1.5;
}
.rs-dates {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px 16px;
  margin: 0;
  font-size: 13px;
}
.rs-dates dd {
  margin: 0;
  font-weight: 600;
}
.rs-issue {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px 12px;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 12px;
  font-size: 14px;
}
.rs-quote {
  padding: 6px 10px;
  border-left: 3px solid rgb(var(--v-theme-src-changed));
  border-radius: 4px;
  background: rgba(var(--v-theme-src-changed), 0.06);
  font-size: 13px;
  line-height: 1.5;
}
.rs-quote--tag {
  border-left-color: rgb(var(--v-theme-primary));
  background: rgba(var(--v-theme-primary), 0.06);
}
.rs-sev {
  display: inline-flex;
  align-items: center;
  height: 22px;
  padding: 0 8px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
}
.rs-sev--critical {
  background: rgb(var(--v-theme-fail-bg));
  color: rgb(var(--v-theme-fail));
}
.rs-sev--warning {
  background: rgb(var(--v-theme-src-changed-bg));
  color: rgb(var(--v-theme-src-changed));
}
.rs-transcript {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-height: 60vh;
  overflow-y: auto;
  padding: 12px;
  border-radius: 12px;
  background: rgba(var(--v-theme-on-surface), 0.03);
}
.rs-msg {
  display: flex;
  flex-direction: column;
  gap: 2px;
  max-width: 82%;
}
.rs-msg--customer {
  align-self: flex-start;
}
.rs-msg--agent {
  align-self: flex-end;
  align-items: flex-end;
}
.rs-msg__who {
  font-size: 11px;
  color: rgb(var(--v-theme-text-muted));
}
.rs-msg__bubble {
  padding: 8px 12px;
  border-radius: 12px;
  font-size: 14px;
  line-height: 1.45;
  border: 1px solid rgb(var(--v-theme-border));
  background: rgb(var(--v-theme-surface));
  overflow-wrap: anywhere;
}
.rs-msg--agent .rs-msg__bubble {
  background: rgba(var(--v-theme-primary), 0.08);
}
.rs-msg__img {
  max-width: 180px;
  max-height: 180px;
  border-radius: 8px;
  cursor: pointer;
}
/* Ô tìm kiếm cao bằng các nút lọc */
.loc-tim {
  max-width: 240px;
}
.loc-nut {
  text-transform: none;
  letter-spacing: 0;
  font-weight: 500;
}
</style>
