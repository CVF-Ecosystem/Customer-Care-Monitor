<!--
  Kết nối MCP của người dùng đang đăng nhập (máy chủ lọc theo user, không theo công ty).
  CCMAI-UX-015: trình bày lại; payload tạo (name, redirect_uris, scopes) và luồng OAuth giữ nguyên.
  Secret chỉ có trong phản hồi tạo, được hiển thị một lần và xóa khỏi bộ nhớ khi đóng hộp thoại.
-->
<template>
  <div class="mc-page">
    <div class="mc-head">
      <div class="mc-head__titles">
        <h1 class="mc-title">{{ $t('mcp_connections') }}</h1>
        <p class="mc-muted mc-m0">{{ $t('mc_sub') }}</p>
      </div>
      <v-btn color="primary" :height="mdAndUp ? 40 : 44" prepend-icon="mdi-plus" data-testid="mc-open-create" @click="openCreate">{{ $t('mc_create') }}</v-btn>
    </div>

    <div v-if="loadError" class="mc-error" role="alert" data-testid="mc-load-error">
      <span>{{ $t('mc_error') }}</span>
      <v-btn variant="outlined" color="error" height="44" @click="loadClients">{{ $t('lg_retry') }}</v-btn>
    </div>

    <v-skeleton-loader v-else-if="!loaded" type="table-row@3" />

    <div v-else-if="!clients.length" class="mc-empty" data-testid="mc-empty">
      <v-icon size="40" class="mc-muted" aria-hidden="true">mdi-connection</v-icon>
      <span>{{ $t('mc_empty') }}</span>
    </div>

    <!-- Bảng desktop -->
    <div v-else-if="mdAndUp" class="mc-table-wrap">
      <v-table density="comfortable">
        <thead>
          <tr>
            <th>{{ $t('mc_col_name') }}</th>
            <th>{{ $t('mc_col_client') }}</th>
            <th>{{ $t('mc_col_redirect') }}</th>
            <th>{{ $t('mc_col_scopes') }}</th>
            <th>{{ $t('mc_col_created') }}</th>
            <th class="mc-col-actions"><span class="d-sr-only">{{ $t('actions') }}</span></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="client in clients" :key="client.id" data-testid="mc-row">
            <td class="font-weight-medium mc-break">{{ client.name }}</td>
            <td><code class="mc-code">{{ client.client_id }}</code></td>
            <td>
              <div v-if="parseList(client.redirect_uris).length" class="mc-uris">
                <span v-for="uri in parseList(client.redirect_uris)" :key="uri" class="mc-break mc-small">{{ uri }}</span>
              </div>
              <span v-else class="mc-muted mc-small">{{ $t('mc_no_redirect') }}</span>
            </td>
            <td>
              <div class="mc-scopes">
                <span v-for="scope in parseList(client.scopes)" :key="scope" class="mc-chip" data-testid="mc-scope">{{ scopeLabel(scope) }}</span>
              </div>
            </td>
            <td class="mc-muted mc-small mc-nowrap">{{ formatDateTime(client.created_at, uiLocale) }}</td>
            <td class="mc-col-actions">
              <v-btn
                variant="outlined"
                color="danger"
                size="small"
                height="36"
                :aria-label="$t('mc_revoke_label', { name: client.name })"
                data-testid="mc-revoke"
                @click="askRevoke(client)"
              >{{ $t('mc_revoke') }}</v-btn>
            </td>
          </tr>
        </tbody>
      </v-table>
    </div>

    <!-- Thẻ mobile -->
    <div v-else class="mc-cards">
      <article v-for="client in clients" :key="client.id" class="mc-card" data-testid="mc-row">
        <div class="mc-card__top">
          <span class="font-weight-medium mc-break">{{ client.name }}</span>
          <span class="mc-muted mc-small mc-nowrap">{{ $t('mc_created_at', { t: formatDateTime(client.created_at, uiLocale) }) }}</span>
        </div>
        <code class="mc-code">{{ client.client_id }}</code>
        <div class="mc-scopes">
          <span v-for="scope in parseList(client.scopes)" :key="scope" class="mc-chip" data-testid="mc-scope">{{ scopeLabel(scope) }}</span>
        </div>
        <div v-if="parseList(client.redirect_uris).length" class="mc-uris">
          <span v-for="uri in parseList(client.redirect_uris)" :key="uri" class="mc-break mc-small mc-muted">{{ uri }}</span>
        </div>
        <v-btn
          variant="outlined"
          color="danger"
          height="44"
          class="align-self-start"
          :aria-label="$t('mc_revoke_label', { name: client.name })"
          data-testid="mc-revoke"
          @click="askRevoke(client)"
        >{{ $t('mc_revoke') }}</v-btn>
      </article>
    </div>

    <!-- Tạo kết nối; sau khi tạo, cùng hộp thoại hiển thị secret một lần -->
    <AppDialog v-model="createDialog" :title="generatedSecret ? $t('mc_created_title') : $t('mc_create')" :max-width="560">
      <div v-if="generatedSecret" class="mc-secret" data-testid="mc-secret-block">
        <div class="mc-warn" role="note">{{ $t('mc_secret_warn') }}</div>
        <div class="mc-field">
          <span class="mc-muted mc-small">{{ $t('client_id') }}</span>
          <code class="mc-code mc-code--block">{{ generatedClientId }}</code>
        </div>
        <div class="mc-field">
          <span class="mc-muted mc-small">{{ $t('mc_secret') }}</span>
          <code class="mc-code mc-code--block" data-testid="mc-secret">{{ generatedSecret }}</code>
        </div>
        <v-btn variant="outlined" height="44" prepend-icon="mdi-content-copy" class="align-self-start" @click="copySecret">{{ $t('mc_copy') }}</v-btn>
      </div>
      <v-form v-else class="d-flex flex-column ga-4" @submit.prevent="generateClient">
        <v-text-field v-model="newName" :label="$t('mc_name')" :hint="$t('mc_name_hint')" persistent-hint variant="outlined" data-testid="mc-name" />
        <v-combobox
          v-model="newRedirectURIs"
          :label="$t('mc_redirect')"
          :hint="$t('mc_redirect_hint')"
          persistent-hint
          multiple
          chips
          closable-chips
          variant="outlined"
        />
        <v-select
          v-model="newScopes"
          :items="scopeOptions"
          :label="$t('mc_scopes')"
          multiple
          chips
          variant="outlined"
          hide-details
        />
      </v-form>
      <template #actions>
        <template v-if="generatedSecret">
          <v-btn color="primary" variant="flat" data-testid="mc-done" @click="createDialog = false">{{ $t('mc_done') }}</v-btn>
        </template>
        <template v-else>
          <v-btn variant="text" @click="createDialog = false">{{ $t('ui_cancel') }}</v-btn>
          <v-btn color="primary" variant="flat" :loading="creating" :disabled="!newName.trim()" data-testid="mc-create" @click="generateClient">{{ $t('mc_create') }}</v-btn>
        </template>
      </template>
    </AppDialog>

    <ConfirmDialog
      v-model="revokeDialog"
      :title="$t('mc_revoke_title', { name: revokeTarget?.name ?? '' })"
      :message="$t('mc_revoke_msg')"
      :confirm-label="$t('mc_revoke')"
      :loading="revoking"
      @confirm="doRevoke"
    />

    <v-snackbar v-model="snack" :color="snackError ? 'error' : undefined" timeout="3000">{{ snackText }}</v-snackbar>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useDisplay } from 'vuetify'
import { useI18n } from 'vue-i18n'
import api from '../api'
import AppDialog from '../components/ui/AppDialog.vue'
import ConfirmDialog from '../components/ui/ConfirmDialog.vue'
import { formatDateTime, type UiLocale } from '../utils/format'

interface McpClient {
  id: string
  client_id: string
  name: string
  redirect_uris: string
  scopes: string
  created_at: string
}

const { t, locale } = useI18n()
const { mdAndUp } = useDisplay()
const uiLocale = computed(() => locale.value as UiLocale)

const clients = ref<McpClient[]>([])
const loaded = ref(false)
const loadError = ref(false)

const createDialog = ref(false)
const newName = ref('')
const newRedirectURIs = ref<string[]>([])
const newScopes = ref<string[]>(['read', 'write'])
const generatedClientId = ref('')
const generatedSecret = ref('')
const creating = ref(false)

const revokeDialog = ref(false)
const revokeTarget = ref<McpClient | null>(null)
const revoking = ref(false)

const snack = ref(false)
const snackText = ref('')
const snackError = ref(false)

// Giá trị scope giữ nguyên (máy chủ chỉ nhận read/write); nhãn tiếng Việt.
const scopeOptions = computed(() => [
  { title: t('mc_scope_read'), value: 'read' },
  { title: t('mc_scope_write'), value: 'write' },
])

onMounted(loadClients)

// Đóng hộp thoại bằng bất kỳ cách nào (nút, ✕, Esc) đều xóa secret khỏi bộ nhớ trang.
watch(createDialog, (open) => {
  if (!open) {
    generatedSecret.value = ''
    generatedClientId.value = ''
  }
})

function parseList(val: string): string[] {
  try {
    const parsed = JSON.parse(val)
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

function scopeLabel(scope: string) {
  if (scope === 'read') return t('mc_scope_read')
  if (scope === 'write') return t('mc_scope_write')
  return scope
}

async function loadClients() {
  loadError.value = false
  try {
    const { data } = await api.get('/mcp/clients')
    clients.value = Array.isArray(data) ? data : []
    loaded.value = true
  } catch {
    loadError.value = true
  }
}

function openCreate() {
  newName.value = ''
  newRedirectURIs.value = []
  newScopes.value = ['read', 'write']
  createDialog.value = true
}

async function generateClient() {
  if (!newName.value.trim() || creating.value) return
  creating.value = true
  try {
    const { data } = await api.post('/mcp/clients', {
      name: newName.value,
      redirect_uris: newRedirectURIs.value,
      scopes: newScopes.value,
    })
    generatedClientId.value = data.client_id
    generatedSecret.value = data.client_secret
    await loadClients()
  } catch {
    showSnack(t('mc_create_failed'), true)
  } finally {
    creating.value = false
  }
}

function askRevoke(client: McpClient) {
  revokeTarget.value = client
  revokeDialog.value = true
}

async function doRevoke() {
  if (!revokeTarget.value) return
  revoking.value = true
  try {
    await api.delete(`/mcp/clients/${revokeTarget.value.id}`)
    revokeDialog.value = false
    showSnack(t('mc_revoked'), false)
    await loadClients()
  } catch {
    showSnack(t('mc_revoke_failed'), true)
  } finally {
    revoking.value = false
  }
}

async function copySecret() {
  try {
    if (!navigator.clipboard?.writeText) throw new Error('clipboard unavailable')
    await navigator.clipboard.writeText(generatedSecret.value)
    showSnack(t('mc_copied'), false)
  } catch {
    showSnack(t('mc_copy_failed'), true)
  }
}

function showSnack(text: string, isError: boolean) {
  snackText.value = text
  snackError.value = isError
  snack.value = true
}
</script>

<style scoped>
.mc-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.mc-head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 12px;
}
.mc-head__titles {
  flex: 1 1 280px;
}
.mc-title {
  margin: 0;
  font-size: 26px;
  font-weight: 700;
}
.mc-muted {
  color: rgb(var(--v-theme-text-muted));
}
.mc-m0 {
  margin: 0;
}
.mc-small {
  font-size: 13px;
}
.mc-nowrap {
  white-space: nowrap;
}
.mc-break {
  overflow-wrap: anywhere;
}
.mc-code {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 13px;
  overflow-wrap: anywhere;
}
.mc-code--block {
  display: block;
  padding: 10px 12px;
  border-radius: 8px;
  background: rgb(var(--v-theme-background));
  border: 1px solid rgb(var(--v-theme-border));
  user-select: all;
}
.mc-uris,
.mc-scopes {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 8px;
}
.mc-uris {
  flex-direction: column;
}
.mc-chip {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  padding: 2px 8px;
  border-radius: 8px;
  background: rgb(var(--v-theme-skip-bg));
  color: rgb(var(--v-theme-on-surface));
  font-size: 12px;
  font-weight: 600;
}
.mc-table-wrap {
  overflow-x: auto;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 12px;
  background: rgb(var(--v-theme-surface));
}
.mc-table-wrap :deep(th) {
  font-size: 12px !important;
  font-weight: 600 !important;
  color: rgb(var(--v-theme-text-muted)) !important;
}
.mc-table-wrap :deep(td) {
  height: auto !important;
  padding-top: 10px !important;
  padding-bottom: 10px !important;
  font-size: 14px;
}
.mc-col-actions {
  width: 120px;
  text-align: right;
}
.mc-cards {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.mc-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px 14px;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 12px;
  background: rgb(var(--v-theme-surface));
}
.mc-card__top {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 8px;
}
.mc-secret {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.mc-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.mc-warn {
  padding: 10px 12px;
  border-left: 4px solid rgb(var(--v-theme-src-changed));
  border-radius: 6px;
  background: rgb(var(--v-theme-src-changed-bg));
  color: rgb(var(--v-theme-src-changed));
  font-size: 14px;
  line-height: 1.5;
}
.mc-empty {
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
.mc-error {
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
</style>
