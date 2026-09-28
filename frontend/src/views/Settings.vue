<!--
  Cài đặt công ty: cấu hình AI, phân tích, lưu trữ file, thông tin chung.
  CCMAI-UX-014: trình bày lại; dữ liệu gửi đi giữ nguyên. Nếu không tải được cài đặt thì
  KHÔNG hiện biểu mẫu — trước đây biểu mẫu hiện giá trị mặc định và bấm Lưu sẽ ghi đè.
-->
<template>
  <div class="st-page">
    <h1 class="st-title">{{ $t('settings') }}</h1>

    <v-skeleton-loader v-if="loading" type="article, article" />

    <div v-else-if="loadError" class="st-error st-error--row" role="alert" data-testid="st-load-error">
      <span>{{ $t('st_load_error') }}</span>
      <v-btn variant="outlined" color="error" height="44" @click="init">{{ $t('st_retry') }}</v-btn>
    </div>

    <div v-else class="st-layout" :class="{ 'st-layout--split': mdAndUp }">
      <nav v-if="mdAndUp" class="st-nav" :aria-label="$t('st_sections')">
        <button
          v-for="tab in tabs"
          :key="tab.value"
          type="button"
          class="st-nav__item"
          :class="{ 'st-nav__item--on': activeTab === tab.value }"
          :aria-current="activeTab === tab.value ? 'page' : undefined"
          @click="activeTab = tab.value"
        >
          <v-icon size="20" aria-hidden="true">{{ tab.icon }}</v-icon>{{ $t(tab.label) }}
        </button>
      </nav>
      <v-tabs v-else v-model="activeTab" show-arrows class="st-tabs">
        <v-tab v-for="tab in tabs" :key="tab.value" :value="tab.value">{{ $t(tab.label) }}</v-tab>
      </v-tabs>

      <!-- Cấu hình AI -->
      <section v-if="activeTab === 'ai'" class="st-card" data-testid="st-ai">
        <header>
          <h2 class="st-h2">{{ $t('ai_config') }}</h2>
          <p class="st-intro">{{ $t('st_intro_ai') }}</p>
        </header>

        <v-select v-model="aiSettings.provider" :label="$t('ai_provider')" :items="providerOptions" variant="outlined" hide-details @update:model-value="onProviderChange" />

        <div class="d-flex align-start ga-2">
          <v-select v-model="aiSettings.model" :label="$t('ai_model')" :items="modelOptions" variant="outlined" hide-details="auto" class="flex-grow-1" />
          <v-btn icon variant="outlined" size="56" rounded="lg" :loading="refreshingModels" :aria-label="$t('refresh_model_list')" :title="$t('refresh_model_list')" @click="refreshModelList">
            <v-icon>mdi-refresh</v-icon>
          </v-btn>
        </div>
        <p v-if="modelsSource === 'static'" class="st-note">{{ $t('model_list_static') }}</p>

        <v-text-field
          v-model="aiSettings.apiKey"
          :label="$t('api_key')"
          :type="showKey ? 'text' : 'password'"
          :append-inner-icon="showKey ? 'mdi-eye-off' : 'mdi-eye'"
          autocomplete="off"
          variant="outlined"
          hide-details
          @click:append-inner="showKey = !showKey"
        />
        <p v-if="hasSavedKey" class="st-note">{{ $t('st_key_saved_note') }}</p>

        <v-switch v-model="useCustomBaseUrl" :label="$t('st_custom_url')" color="primary" density="compact" hide-details />
        <p class="st-note">{{ useCustomBaseUrl ? $t('st_custom_url_on') : $t('st_custom_url_off') }}</p>
        <v-text-field
          v-if="useCustomBaseUrl"
          v-model="aiSettings.baseUrl"
          :label="$t('st_base_url')"
          :placeholder="baseUrlPlaceholder"
          :hint="$t('st_base_url_hint')"
          persistent-hint
          clearable
          variant="outlined"
          :rules="[
            (v: string) => !!v || $t('st_base_url_required'),
            (v: string) => !v || v.startsWith('http://') || v.startsWith('https://') || $t('st_url_scheme'),
          ]"
        />

        <footer class="st-actions">
          <v-btn color="primary" height="44" :loading="savingAI" data-testid="st-save-ai" @click="saveAI">{{ $t('save_settings') }}</v-btn>
          <v-btn variant="outlined" height="44" :loading="testingKey" data-testid="st-test-key" @click="testKey">{{ $t('st_test_saved_key') }}</v-btn>
          <span class="st-note">{{ $t('st_test_saved_note') }}</span>
        </footer>
      </section>

      <!-- Phân tích -->
      <section v-if="activeTab === 'analysis'" class="st-card">
        <header>
          <h2 class="st-h2">{{ $t('analysis_settings') }}</h2>
          <p class="st-intro">{{ $t('st_intro_analysis') }}</p>
        </header>
        <h3 class="st-h3">{{ $t('st_batch_title') }}</h3>
        <v-switch v-model="aiSettings.batchMode" :label="$t('st_batch_toggle')" :hint="$t('st_batch_hint')" persistent-hint density="compact" color="primary" />
        <v-select v-if="aiSettings.batchMode" v-model="aiSettings.batchSize" :label="$t('st_batch_size')" :items="[3, 5, 10, 15, 20, 30]" variant="outlined" hide-details class="st-narrow" />
        <footer class="st-actions">
          <v-btn color="primary" height="44" :loading="savingAnalysis" @click="saveAnalysis">{{ $t('save_settings') }}</v-btn>
        </footer>
      </section>

      <!-- Lưu trữ file -->
      <section v-if="activeTab === 'storage'" class="st-card" data-testid="st-storage">
        <header>
          <h2 class="st-h2">{{ $t('storage_settings') }}</h2>
          <p class="st-intro">{{ $t('storage_intro') }}</p>
        </header>

        <div v-if="storageError" class="st-error st-error--row" role="alert">
          <span>{{ $t('st_storage_load_error') }}</span>
          <v-btn variant="outlined" color="error" height="44" @click="loadStorage">{{ $t('st_retry') }}</v-btn>
        </div>

        <template v-else>
          <p class="st-status" :class="{ 'st-status--s3': storage.backend === 's3' }">
            <template v-if="storage.backend === 's3'">{{ $t('storage_now_s3') }} <strong>{{ storage.bucket }}</strong> · {{ storageHost }}</template>
            <template v-else>{{ $t('storage_now_local') }}<template v-if="storageLocalUsage"> — {{ storageLocalUsage }}</template></template>
          </p>

          <v-switch v-model="dungS3" color="primary" density="compact" hide-details :label="$t('storage_use_s3')" @update:model-value="onToggleS3" />

          <v-expand-transition>
            <div v-if="dungS3" class="st-stack">
              <v-divider />
              <v-text-field v-model="storage.endpoint" :label="$t('storage_endpoint')" placeholder="https://s3.nha-cung-cap.vn" variant="outlined" hide-details="auto" @update:model-value="testResult = null" />
              <div class="st-grid">
                <v-text-field v-model="storage.bucket" :label="$t('storage_bucket')" variant="outlined" hide-details="auto" @update:model-value="testResult = null" />
                <v-text-field v-model="storage.region" :label="$t('storage_region')" variant="outlined" hide-details="auto" @update:model-value="testResult = null" />
              </div>
              <v-text-field v-model="storage.access_key" label="Access Key" autocomplete="off" variant="outlined" hide-details="auto" @update:model-value="testResult = null" />
              <v-text-field
                v-model="storage.secret_key"
                label="Secret Key"
                type="password"
                autocomplete="new-password"
                variant="outlined"
                :placeholder="storage.secret_key_da_luu ? '••••••••' : ''"
                :hint="storage.secret_key_da_luu ? $t('storage_secret_saved') : ''"
                persistent-hint
                @update:model-value="testResult = null"
              />

              <v-expansion-panels variant="accordion">
                <v-expansion-panel elevation="0">
                  <v-expansion-panel-title>{{ $t('advanced_options') }}</v-expansion-panel-title>
                  <v-expansion-panel-text>
                    <v-text-field v-model="storage.prefix" :label="$t('storage_prefix')" variant="outlined" hide-details="auto" @update:model-value="testResult = null" />
                    <v-switch v-model="storage.force_path_style" color="primary" density="compact" hide-details class="mt-2" :label="$t('storage_path_style')" @update:model-value="testResult = null" />
                  </v-expansion-panel-text>
                </v-expansion-panel>
              </v-expansion-panels>

              <p v-if="testResult" class="st-result" :class="testResult.ok ? 'st-result--ok' : 'st-result--bad'" role="status">{{ testResult.message }}</p>

              <footer class="st-actions">
                <v-btn variant="outlined" height="44" :loading="testingStorage" prepend-icon="mdi-lan-connect" @click="testStorage">{{ $t('storage_test') }}</v-btn>
                <v-btn color="primary" height="44" :disabled="!testResult?.ok" :loading="savingStorage" @click="saveStorage">{{ $t('save_settings') }}</v-btn>
                <span v-if="!testResult?.ok" class="st-note">{{ $t('storage_must_test') }}</span>
              </footer>

              <v-divider />
              <h3 class="st-h3">{{ $t('storage_old_files_title') }}</h3>
              <p class="st-intro">
                {{ $t('storage_old_files_desc') }}
                <template v-if="storageUsageParts"> {{ $t('storage_old_files_now', storageUsageParts) }}</template>
              </p>
              <v-btn
                variant="text"
                color="primary"
                prepend-icon="mdi-book-open-variant"
                class="align-self-start"
                href="https://cvf-ecosystem.github.io/Customer-Care-Monitor-AI/guide/s3-storage.html"
                target="_blank"
              >{{ $t('storage_guide') }}</v-btn>
            </div>
          </v-expand-transition>

          <footer v-if="!dungS3 && storage.backend === 's3'" class="st-actions">
            <v-btn color="primary" height="44" :loading="savingStorage" @click="saveStorage">{{ $t('save_settings') }}</v-btn>
          </footer>
        </template>
      </section>

      <!-- Chung -->
      <section v-if="activeTab === 'general'" class="st-card">
        <header>
          <h2 class="st-h2">{{ $t('general') }}</h2>
          <p class="st-intro">{{ $t('st_intro_general') }}</p>
        </header>
        <v-text-field v-model="generalSettings.companyName" :label="$t('company_name')" variant="outlined" hide-details />
        <v-select v-model="generalSettings.timezone" :label="$t('timezone')" :items="['Asia/Ho_Chi_Minh', 'Asia/Bangkok', 'UTC', 'America/New_York']" variant="outlined" hide-details />
        <v-select v-model="generalSettings.language" :label="$t('language')" :items="[{ title: 'Tiếng Việt', value: 'vi' }, { title: 'English', value: 'en' }]" variant="outlined" hide-details />
        <v-text-field v-model.number="generalSettings.exchangeRate" :label="$t('exchange_rate_vnd')" type="number" :suffix="$t('st_rate_suffix')" variant="outlined" hide-details />
        <v-text-field
          v-model="generalSettings.appUrl"
          :label="$t('st_app_url')"
          placeholder="https://cqa.yourdomain.com"
          :hint="$t('st_app_url_hint')"
          persistent-hint
          :rules="appUrlRules"
          variant="outlined"
        />
        <footer class="st-actions">
          <v-btn color="primary" height="44" :loading="savingGeneral" @click="saveGeneral">{{ $t('save_settings') }}</v-btn>
        </footer>
      </section>
    </div>

    <AppDialog v-model="confirmTatS3" :title="$t('storage_off_title')" :max-width="560">
      <div class="st-stack">
        <p class="st-warn">{{ $t('storage_off_warn') }}</p>
        <p class="st-m0">{{ $t('storage_off_keep') }}</p>
        <pre class="st-code"><code>docker exec cqa-app /app/cqa-server migrate-files -down -apply</code></pre>
        <p class="st-intro">{{ $t('storage_off_keys') }}</p>
      </div>
      <template #actions>
        <v-btn variant="text" @click="huyTatS3">{{ $t('cancel') }}</v-btn>
        <v-btn color="warning" variant="flat" @click="xacNhanTatS3">{{ $t('storage_off_confirm') }}</v-btn>
      </template>
    </AppDialog>

    <v-snackbar v-model="snackbar" :color="snackColor === 'error' ? 'error' : undefined" timeout="3000">{{ snackText }}</v-snackbar>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useDisplay } from 'vuetify'
import { useI18n } from 'vue-i18n'
import api from '../api'
import AppDialog from '../components/ui/AppDialog.vue'

const route = useRoute()
const { t } = useI18n()
const { mdAndUp } = useDisplay()
const tenantId = computed(() => route.params.tenantId as string)

const activeTab = ref('ai')
const showKey = ref(false)
const snackbar = ref(false)
const snackText = ref('')
const snackColor = ref('success')

// CCMAI-UX-014: chỉ hiện biểu mẫu sau khi tải cài đặt thành công
const loading = ref(true)
const loadError = ref(false)
const storageError = ref(false)

const savingAI = ref(false)
const testingKey = ref(false)
const savingGeneral = ref(false)
const savingStorage = ref(false)
const testingStorage = ref(false)

// Nơi cất file đính kèm, cấu hình riêng cho từng công ty.
const storage = reactive({
  backend: 'local',
  local_bytes: 0,
  local_files: 0,
  local_partial: false,
  endpoint: '',
  bucket: '',
  region: '',
  access_key: '',
  secret_key: '',
  prefix: '',
  force_path_style: false,
  secret_key_da_luu: false,
})
const dungS3 = ref(false)
const confirmTatS3 = ref(false)
const testResult = ref<{ ok: boolean; message: string } | null>(null)

// Tắt S3 là ảnh đã lưu trên đó thôi hiển thị, nên hỏi lại trước khi cho tắt.
// Chỉ hỏi khi công ty đang thật sự chạy trên S3, còn bật lên rồi đổi ý ngay
// thì không có gì để mất.
function onToggleS3(val: boolean | null) {
  testResult.value = null
  if (val === false && storage.backend === 's3') {
    confirmTatS3.value = true
  }
}

function huyTatS3() {
  confirmTatS3.value = false
  dungS3.value = true
}

async function xacNhanTatS3() {
  confirmTatS3.value = false
  await saveStorage()
}
const storageHost = computed(() => storage.endpoint.replace(/^https?:\/\//, '').replace(/\/$/, ''))

const tabs = [
  { label: 'ai_config', value: 'ai', icon: 'mdi-robot-outline' },
  { label: 'analysis_settings', value: 'analysis', icon: 'mdi-chart-bar' },
  { label: 'storage_settings', value: 'storage', icon: 'mdi-folder-multiple-image' },
  { label: 'general', value: 'general', icon: 'mdi-cog-outline' },
]

// Danh sách đối chiếu ngày 2026-09-15. Model thế hệ cũ vẫn giữ lại để ai đang
// dùng không bị mất lựa chọn, nhưng ghi rõ là cũ.
const claudeModels = [
  { title: 'Claude Sonnet 5 (Khuyến nghị)', value: 'claude-sonnet-5' },
  { title: 'Claude Haiku 4.5 (Nhanh & rẻ)', value: 'claude-haiku-4-5' },
  { title: 'Claude Opus 5 (Mạnh nhất)', value: 'claude-opus-5' },
  { title: 'Claude Sonnet 4.6 (Thế hệ cũ)', value: 'claude-sonnet-4-6' },
  { title: 'Claude Opus 4.6 (Thế hệ cũ)', value: 'claude-opus-4-6' },
]
const providerOptions = [
  { title: 'Claude (Anthropic)', value: 'claude' },
  { title: 'Gemini (Google)', value: 'gemini' },
  { title: 'ChatGPT (OpenAI)', value: 'openai' },
  { title: 'Grok (xAI)', value: 'xai' },
]
const openaiModels = [
  { title: 'GPT-5 (Khuyến nghị)', value: 'gpt-5' },
  { title: 'GPT-5 mini (Nhanh & rẻ)', value: 'gpt-5-mini' },
  { title: 'o3 (Suy luận sâu)', value: 'o3' },
]
const xaiModels = [
  { title: 'Grok 4 (Khuyến nghị)', value: 'grok-4' },
  { title: 'Grok 3 (Thế hệ cũ)', value: 'grok-3' },
]
const geminiModels = [
  { title: 'Gemini 3.8 Flash (Khuyến nghị)', value: 'gemini-3.8-flash' },
  { title: 'Gemini 3.1 Flash Lite (Nhanh & rẻ nhất)', value: 'gemini-3.1-flash-lite' },
  { title: 'Gemini 3.5 Flash Lite (Rẻ)', value: 'gemini-3.5-flash-lite' },
  { title: 'Gemini 3.5 Flash (Mạnh hơn)', value: 'gemini-3.5-flash' },
  { title: 'Gemini 2.5 Pro (Thế hệ cũ)', value: 'gemini-2.5-pro' },
  { title: 'Gemini 2.5 Flash (Thế hệ cũ)', value: 'gemini-2.5-flash' },
]

const useCustomBaseUrl = ref(false)
const hasSavedKey = ref(false)
const aiSettings = reactive({ provider: 'claude', model: 'claude-sonnet-5', apiKey: '', baseUrl: '', batchMode: true, batchSize: 5 })
const generalSettings = reactive({ companyName: '', timezone: 'Asia/Ho_Chi_Minh', language: 'vi', exchangeRate: 26000, appUrl: '' })

const appUrlRules = [
  (v: string) => !v || /^https?:\/\/.+/.test(v) || t('st_url_scheme'),
  (v: string) => !v || !v.endsWith('/') || t('st_url_trailing'),
]

const baseUrlPlaceholder = computed(() => {
  switch (aiSettings.provider) {
    case 'gemini': return 'https://generativelanguage.googleapis.com'
    case 'openai': return 'https://api.openai.com/v1'
    case 'xai': return 'https://api.x.ai/v1'
    default: return 'https://api.anthropic.com'
  }
})

function fallbackModels(provider: string) {
  switch (provider) {
    case 'gemini': return geminiModels
    case 'openai': return openaiModels
    case 'xai': return xaiModels
    default: return claudeModels
  }
}

function defaultModelFor(provider: string) {
  switch (provider) {
    case 'gemini': return 'gemini-3.8-flash'
    case 'openai': return 'gpt-5'
    case 'xai': return 'grok-4'
    default: return 'claude-sonnet-5'
  }
}

// Danh sách lấy từ nhà cung cấp nếu có, không thì dùng danh sách kèm sẵn.
const fetchedModels = ref<{ id: string; title: string }[]>([])
const modelsSource = ref('')
const refreshingModels = ref(false)

const modelOptions = computed(() => {
  if (fetchedModels.value.length) {
    return fetchedModels.value.map(m => ({ title: m.title, value: m.id }))
  }
  const fallback = fallbackModels(aiSettings.provider)
  // Model đang dùng phải luôn có mặt, nếu không ô chọn sẽ hiện trống.
  if (aiSettings.model && !fallback.some(m => m.value === aiSettings.model)) {
    return [{ title: t('st_model_current', { model: aiSettings.model }), value: aiSettings.model }, ...fallback]
  }
  return fallback
})

async function loadModelList() {
  try {
    const { data } = await api.get(`/tenants/${tenantId.value}/settings/ai/models`)
    fetchedModels.value = data.models || []
    modelsSource.value = data.source || ''
  } catch {
    // Không lấy được thì im lặng dùng danh sách kèm sẵn
    fetchedModels.value = []
    modelsSource.value = 'static'
  }
}

async function refreshModelList() {
  refreshingModels.value = true
  try {
    const { data } = await api.post(`/tenants/${tenantId.value}/settings/ai/models/refresh`)
    fetchedModels.value = data.models || []
    modelsSource.value = data.source || ''
    showSnack(t('st_models_updated', { n: fetchedModels.value.length }), 'success')
  } catch (err: any) {
    const res = err.response?.data
    showSnack(res?.message || res?.error || t('error'), 'error')
  } finally {
    refreshingModels.value = false
  }
}

function onProviderChange() {
  // Reset to default model when switching provider
  aiSettings.model = defaultModelFor(aiSettings.provider)
  // Danh sách model của nhà cung cấp cũ không còn đúng nữa
  fetchedModels.value = []
  modelsSource.value = ''
}

// Ném lỗi ra ngoài để init() biết mà chặn biểu mẫu.
async function loadSettings() {
  const { data } = await api.get(`/tenants/${tenantId.value}/settings`)
  if (data.settings.ai_provider) aiSettings.provider = data.settings.ai_provider
  if (data.settings.ai_model) aiSettings.model = data.settings.ai_model
  if (data.settings.ai_api_key) {
    aiSettings.apiKey = data.settings.ai_api_key
    hasSavedKey.value = true
  }
  if (data.settings.ai_base_url) {
    aiSettings.baseUrl = data.settings.ai_base_url
    useCustomBaseUrl.value = true
  }
  if (data.settings.ai_batch_mode) aiSettings.batchMode = data.settings.ai_batch_mode === 'true'
  if (data.settings.ai_batch_size) aiSettings.batchSize = parseInt(data.settings.ai_batch_size) || 5
  if (data.settings.exchange_rate_vnd) generalSettings.exchangeRate = parseFloat(data.settings.exchange_rate_vnd) || 26000
  if (data.settings.app_url) generalSettings.appUrl = data.settings.app_url
  if (data.tenant) {
    generalSettings.companyName = data.tenant.name || ''
    generalSettings.timezone = data.tenant.timezone || 'Asia/Ho_Chi_Minh'
    generalSettings.language = data.tenant.language || 'vi'
  }
}

const savingAnalysis = ref(false)

async function saveAnalysis() {
  savingAnalysis.value = true
  try {
    await api.put(`/tenants/${tenantId.value}/settings/analysis`, {
      batch_mode: aiSettings.batchMode ? 'true' : 'false',
      batch_size: String(aiSettings.batchSize),
    })
    showSnack(t('st_saved'), 'success')
  } catch (err: any) {
    showSnack(err.response?.data?.error || t('st_save_failed'), 'error')
  } finally {
    savingAnalysis.value = false
  }
}

async function saveAI() {
  // Ô key hiển thị dấu chấm khi đã có key lưu sẵn. Gửi chuỗi rỗng để backend
  // giữ nguyên key cũ, nhờ vậy đổi model hay cỡ lô không phải nhập lại key.
  const apiKeyToSend = aiSettings.apiKey === '••••••••' ? '' : aiSettings.apiKey
  if (!apiKeyToSend && !hasSavedKey.value) {
    showSnack(t('st_need_key'), 'error')
    return
  }
  if (useCustomBaseUrl.value && !aiSettings.baseUrl) {
    showSnack(t('st_base_url_required'), 'error')
    return
  }
  savingAI.value = true
  try {
    await api.put(`/tenants/${tenantId.value}/settings/ai`, {
      provider: aiSettings.provider,
      model: aiSettings.model,
      api_key: apiKeyToSend,
      base_url: useCustomBaseUrl.value ? (aiSettings.baseUrl || '') : '',
      batch_mode: aiSettings.batchMode ? 'true' : 'false',
      batch_size: String(aiSettings.batchSize),
    })
    showSnack(t('st_saved'), 'success')
  } catch (err: any) {
    showSnack(err.response?.data?.error || t('st_save_failed'), 'error')
  } finally {
    savingAI.value = false
  }
}

// Kiểm tra key ĐÃ LƯU trên máy chủ (gọi nhà cung cấp thật), không phải giá trị đang gõ.
async function testKey() {
  testingKey.value = true
  try {
    const { data } = await api.post(`/tenants/${tenantId.value}/settings/ai/test`)
    const detail = data.model ? `${data.provider} / ${data.model}` : data.provider
    showSnack(`${detail}: ${data.message}`, 'success')
  } catch (err: any) {
    const res = err.response?.data
    showSnack(res?.message || res?.error || t('error'), 'error')
  } finally {
    testingKey.value = false
  }
}

// Đổi số byte thành chuỗi người đọc được. Dùng bội số 1024 cho khớp với cách hệ
// điều hành báo dung lượng ổ đĩa.
function dinhDangDungLuong(bytes: number): string {
  if (!bytes || bytes < 0) return '0 MB'
  const don = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let v = bytes
  while (v >= 1024 && i < don.length - 1) {
    v /= 1024
    i++
  }
  // Dưới 10 thì giữ một chữ số thập phân cho đỡ mất thông tin (1,4 GB).
  const soChu = v < 10 && i > 0 ? 1 : 0
  return `${v.toFixed(soChu).replace('.', ',')} ${don[i]}`
}

// Hai cách diễn đạt cùng một con số: câu đầy đủ cho dòng trạng thái, và cụm
// ngắn để nhét vào giữa câu khác mà không lặp chữ.
const storageUsageParts = computed(() => {
  if (!storage.local_files) return null
  return {
    size: dinhDangDungLuong(storage.local_bytes),
    count: storage.local_files.toLocaleString('vi-VN'),
  }
})

const storageLocalUsage = computed(() => {
  const p = storageUsageParts.value
  if (!p) return ''
  const text = t('storage_local_usage', p)
  return storage.local_partial ? `${text} ${t('storage_usage_partial')}` : text
})

async function loadStorage() {
  storageError.value = false
  try {
    const { data } = await api.get(`/tenants/${tenantId.value}/settings/storage`)
    Object.assign(storage, data, { secret_key: '' })
    dungS3.value = data.backend === 's3'
    testResult.value = null
  } catch {
    // Không biết công ty đang lưu ở đâu thì không cho sửa phần này
    storageError.value = true
  }
}

function storagePayload() {
  return {
    backend: dungS3.value ? 's3' : 'local',
    endpoint: storage.endpoint,
    bucket: storage.bucket,
    region: storage.region,
    access_key: storage.access_key,
    secret_key: storage.secret_key,
    prefix: storage.prefix,
    force_path_style: storage.force_path_style,
  }
}

async function testStorage() {
  testingStorage.value = true
  testResult.value = null
  try {
    const { data } = await api.post(`/tenants/${tenantId.value}/settings/storage/test`, storagePayload())
    testResult.value = { ok: data.ok === true, message: data.message || '' }
  } catch (e: any) {
    testResult.value = { ok: false, message: e.response?.data?.message || t('connection_failed') }
  } finally {
    testingStorage.value = false
  }
}

async function saveStorage() {
  savingStorage.value = true
  try {
    await api.put(`/tenants/${tenantId.value}/settings/storage`, storagePayload())
    await loadStorage()
    showSnack(t('st_saved'), 'success')
  } catch (e: any) {
    showSnack(e.response?.data?.message || t('st_save_failed'), 'error')
  } finally {
    savingStorage.value = false
  }
}

async function saveGeneral() {
  savingGeneral.value = true
  try {
    await api.put(`/tenants/${tenantId.value}/settings/general`, {
      company_name: generalSettings.companyName,
      timezone: generalSettings.timezone,
      language: generalSettings.language,
      exchange_rate_vnd: generalSettings.exchangeRate,
      app_url: generalSettings.appUrl,
    })
    showSnack(t('st_saved'), 'success')
  } catch (err: any) {
    showSnack(err.response?.data?.error || t('st_save_failed'), 'error')
  } finally {
    savingGeneral.value = false
  }
}

function showSnack(text: string, color: string) {
  snackText.value = text
  snackColor.value = color
  snackbar.value = true
}

async function init() {
  loading.value = true
  loadError.value = false
  try {
    await loadSettings()
  } catch {
    loadError.value = true
    return
  } finally {
    loading.value = false
  }
  // Nạp sau khi đã biết nhà cung cấp và model đang chọn
  loadModelList()
  loadStorage()
}

onMounted(init)
</script>

<style scoped>
.st-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.st-title {
  margin: 0;
  font-size: 26px;
  font-weight: 700;
}
.st-layout {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.st-layout--split {
  display: grid;
  grid-template-columns: 240px minmax(0, 760px);
  gap: 20px;
  align-items: start;
}
.st-nav {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 8px;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 12px;
  background: rgb(var(--v-theme-surface));
}
.st-nav__item {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 44px;
  padding: 0 12px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: rgb(var(--v-theme-on-surface));
  font: inherit;
  font-size: 14px;
  text-align: left;
  cursor: pointer;
}
.st-nav__item:hover {
  background: rgba(var(--v-theme-on-surface), 0.04);
}
.st-nav__item:focus-visible {
  outline: 2px solid rgb(var(--v-theme-primary));
}
.st-nav__item--on {
  background: rgba(var(--v-theme-primary), 0.1);
  color: rgb(var(--v-theme-primary));
  font-weight: 600;
}
.st-tabs {
  border-bottom: 1px solid rgb(var(--v-theme-border));
}
.st-card {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 20px 24px;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 12px;
  background: rgb(var(--v-theme-surface));
  min-width: 0;
}
.st-h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}
.st-h3 {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
}
.st-intro {
  margin: 4px 0 0;
  font-size: 14px;
  line-height: 1.5;
  color: rgb(var(--v-theme-text-muted));
}
.st-m0 {
  margin: 0;
}
.st-note {
  margin: 0;
  font-size: 12px;
  line-height: 1.5;
  color: rgb(var(--v-theme-text-muted));
}
.st-narrow {
  max-width: 240px;
}
.st-stack {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.st-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 14px;
}
.st-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  padding-top: 12px;
  border-top: 1px solid rgb(var(--v-theme-border));
}
.st-status {
  margin: 0;
  padding: 10px 12px;
  border: 1px solid rgb(var(--v-theme-border));
  border-left: 4px solid rgb(var(--v-theme-src-legacy));
  border-radius: 10px;
  font-size: 14px;
}
.st-status--s3 {
  border-left-color: rgb(var(--v-theme-primary));
}
.st-result {
  margin: 0;
  padding: 10px 12px;
  border-radius: 10px;
  font-size: 14px;
  font-weight: 600;
}
.st-result--ok {
  background: rgb(var(--v-theme-pass-bg));
  color: rgb(var(--v-theme-pass));
}
.st-result--bad {
  background: rgb(var(--v-theme-fail-bg));
  color: rgb(var(--v-theme-fail));
}
.st-warn {
  margin: 0;
  padding: 10px 12px;
  border-left: 4px solid rgb(var(--v-theme-src-changed));
  border-radius: 6px;
  background: rgb(var(--v-theme-src-changed-bg));
  font-size: 14px;
  line-height: 1.5;
}
.st-code {
  margin: 0;
  padding: 10px 12px;
  border-radius: 8px;
  background: rgba(var(--v-theme-on-surface), 0.06);
  font-size: 12px;
  overflow-x: auto;
}
.st-error {
  padding: 14px 16px;
  border: 1px solid rgb(var(--v-theme-fail));
  border-radius: 12px;
  background: rgb(var(--v-theme-fail-bg));
  color: rgb(var(--v-theme-fail));
  font-size: 14px;
  font-weight: 600;
  line-height: 1.45;
}
.st-error--row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 12px;
}
</style>
