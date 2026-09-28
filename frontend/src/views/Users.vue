<!--
  Người dùng của công ty.
  CCMAI-UX-014: trình bày lại; vai trò, giá trị phân quyền (r/rw) và thời điểm áp dụng
  giữ nguyên. Quy tắc mật khẩu khi tạo khớp với máy chủ (8 ký tự, chữ hoa, chữ số).
-->
<template>
  <div class="us-page">
    <div class="us-head">
      <div class="us-head__titles">
        <h1 class="us-title">{{ $t('nav_users') }}</h1>
        <p class="us-muted us-m0" data-testid="us-subtitle">{{ $t('us_subtitle', { n: userStore.users.length }) }}</p>
      </div>
      <v-btn color="primary" :height="mdAndUp ? 40 : 44" prepend-icon="mdi-account-plus-outline" @click="inviteDialog = true">{{ $t('create_user') }}</v-btn>
    </div>

    <div v-if="loadError" class="us-error" role="alert" data-testid="us-load-error">
      <span>{{ $t('us_load_error') }}</span>
      <v-btn variant="outlined" color="error" height="44" @click="load">{{ $t('st_retry') }}</v-btn>
    </div>

    <v-skeleton-loader v-else-if="loading && !userStore.users.length" type="table-row@3" />

    <!-- Bảng desktop -->
    <div v-else-if="mdAndUp" class="us-table-wrap">
      <v-table density="comfortable">
        <thead>
          <tr>
            <th>{{ $t('display_name') }}</th>
            <th>{{ $t('email') }}</th>
            <th>{{ $t('role') }}</th>
            <th class="us-col-actions"><span class="d-sr-only">{{ $t('actions') }}</span></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="u in userStore.users" :key="u.user_id" data-testid="us-row">
            <td class="font-weight-medium">{{ u.name }} <span v-if="isSelf(u)" class="us-muted">{{ $t('us_you') }}</span></td>
            <td class="us-muted">{{ u.email }}</td>
            <td class="us-col-role">
              <v-select
                :model-value="u.role"
                :items="roleOptions"
                :aria-label="$t('us_role_label', { name: u.name })"
                density="compact"
                variant="outlined"
                hide-details
                :disabled="isSelf(u)"
                data-testid="us-role"
                @update:model-value="changeRole(u.user_id, $event)"
              />
            </td>
            <td class="us-col-actions">
              <div class="d-flex align-center justify-end ga-1">
                <v-btn v-if="u.role === 'member' && !isSelf(u)" variant="outlined" size="small" height="36" @click="openPermissions(u)">{{ $t('us_perm_btn') }}</v-btn>
                <ActionMenu v-if="!isSelf(u)" :items="menuItems" :label="$t('us_more', { name: u.name })" @select="onAction($event, u)" />
              </div>
            </td>
          </tr>
        </tbody>
      </v-table>
    </div>

    <!-- Thẻ mobile -->
    <div v-else class="us-cards">
      <article v-for="u in userStore.users" :key="u.user_id" class="us-card" data-testid="us-row">
        <div class="us-card__top">
          <div class="us-card__titles">
            <span class="font-weight-medium">{{ u.name }} <span v-if="isSelf(u)" class="us-muted">{{ $t('us_you') }}</span></span>
            <span class="us-muted us-small">{{ u.email }}</span>
          </div>
          <ActionMenu v-if="!isSelf(u)" :items="menuItems" :label="$t('us_more', { name: u.name })" @select="onAction($event, u)" />
        </div>
        <v-select
          :model-value="u.role"
          :items="roleOptions"
          :label="$t('role')"
          density="comfortable"
          variant="outlined"
          hide-details
          :disabled="isSelf(u)"
          data-testid="us-role"
          @update:model-value="changeRole(u.user_id, $event)"
        />
        <v-btn v-if="u.role === 'member' && !isSelf(u)" variant="outlined" height="44" class="align-self-start" @click="openPermissions(u)">{{ $t('us_perm_btn') }}</v-btn>
      </article>
    </div>

    <!-- Tạo người dùng -->
    <AppDialog v-model="inviteDialog" :title="$t('create_user')" :max-width="480">
      <v-form ref="createFormRef" class="d-flex flex-column ga-3">
        <v-text-field v-model="inviteForm.name" :label="$t('display_name')" variant="outlined" :rules="[(v: string) => !!v || $t('validation_required')]" />
        <v-text-field v-model="inviteForm.email" :label="$t('email')" type="email" autocomplete="off" variant="outlined" :rules="[(v: string) => !!v || $t('validation_required'), (v: string) => /.+@.+\..+/.test(v) || $t('us_email_invalid')]" />
        <v-text-field
          v-model="inviteForm.password"
          :label="$t('password')"
          type="password"
          autocomplete="new-password"
          variant="outlined"
          :hint="$t('us_password_rule')"
          persistent-hint
          :rules="[(v: string) => !!v || $t('validation_required'), (v: string) => strongPassword(v) || $t('us_err_weak')]"
          data-testid="us-new-password"
        />
        <v-select v-model="inviteForm.role" :items="roleOptions" :label="$t('role')" variant="outlined" hide-details />
        <div v-if="inviteForm.role === 'member'">
          <h3 class="us-h3">{{ $t('permissions') }}</h3>
          <v-table density="compact">
            <thead><tr><th>{{ $t('feature') }}</th><th>{{ $t('view') }}</th><th>{{ $t('edit') }}</th></tr></thead>
            <tbody>
              <tr v-for="feat in permissionFeatures" :key="feat.key">
                <td>{{ feat.label }}</td>
                <td><v-checkbox-btn v-model="inviteForm.permissions[feat.key]" true-value="r" false-value="" density="compact" :aria-label="`${$t('view')} ${feat.label}`" /></td>
                <td><v-checkbox-btn v-model="inviteForm.permissions[feat.key]" true-value="rw" :false-value="inviteForm.permissions[feat.key] === 'rw' ? 'r' : ''" density="compact" :aria-label="`${$t('edit')} ${feat.label}`" /></td>
              </tr>
            </tbody>
          </v-table>
        </div>
      </v-form>
      <template #actions>
        <v-btn variant="text" @click="inviteDialog = false">{{ $t('cancel') }}</v-btn>
        <v-btn color="primary" variant="flat" :loading="inviting" data-testid="us-invite" @click="doInvite">{{ $t('create_user') }}</v-btn>
      </template>
    </AppDialog>

    <!-- Xóa khỏi công ty -->
    <ConfirmDialog
      v-model="removeDialog"
      :title="$t('us_remove_title', { email: removeTarget?.email ?? '' })"
      :message="$t('us_remove_msg')"
      :confirm-label="$t('us_remove')"
      :loading="removing"
      @confirm="doRemove"
    />

    <!-- Phân quyền -->
    <AppDialog v-model="permDialog" :title="$t('us_perm_title', { name: permTarget?.name ?? '' })" :max-width="480">
      <v-table density="compact">
        <thead><tr><th>{{ $t('feature') }}</th><th>{{ $t('view') }}</th><th>{{ $t('edit') }}</th></tr></thead>
        <tbody>
          <tr v-for="feat in permissionFeatures" :key="feat.key">
            <td>{{ feat.label }}</td>
            <td><v-checkbox-btn v-model="editPerms[feat.key]" true-value="r" false-value="" density="compact" :aria-label="`${$t('view')} ${feat.label}`" /></td>
            <td><v-checkbox-btn v-model="editPerms[feat.key]" true-value="rw" :false-value="editPerms[feat.key] === 'rw' ? 'r' : ''" density="compact" :aria-label="`${$t('edit')} ${feat.label}`" /></td>
          </tr>
        </tbody>
      </v-table>
      <template #actions>
        <v-btn variant="text" @click="permDialog = false">{{ $t('cancel') }}</v-btn>
        <v-btn color="primary" variant="flat" :loading="savingPerms" @click="savePermissions">{{ $t('save_settings') }}</v-btn>
      </template>
    </AppDialog>

    <!-- Đặt lại mật khẩu -->
    <AppDialog v-model="resetDialog" :title="$t('us_reset_title', { name: resetTarget?.name ?? '' })" :max-width="440">
      <v-text-field
        v-model="resetPassword"
        :label="$t('us_new_password')"
        type="password"
        autocomplete="new-password"
        variant="outlined"
        :hint="$t('us_password_rule')"
        persistent-hint
        :rules="[(v: string) => !!v || $t('validation_required'), (v: string) => strongPassword(v) || $t('us_err_weak')]"
      />
      <template #actions>
        <v-btn variant="text" @click="resetDialog = false">{{ $t('cancel') }}</v-btn>
        <v-btn color="primary" variant="flat" :loading="resettingPassword" @click="doResetPassword">{{ $t('us_reset_btn') }}</v-btn>
      </template>
    </AppDialog>

    <v-snackbar v-model="snack" :color="snackColor === 'error' ? 'error' : undefined" timeout="3000">{{ snackText }}</v-snackbar>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useDisplay } from 'vuetify'
import { useI18n } from 'vue-i18n'
import { useUserStore, type TenantUser } from '../stores/users'
import api from '../api'
import { useAuthStore } from '../stores/auth'
import AppDialog from '../components/ui/AppDialog.vue'
import ActionMenu from '../components/ui/ActionMenu.vue'
import ConfirmDialog from '../components/ui/ConfirmDialog.vue'
import type { ActionMenuItem } from '../components/ui/types'

const route = useRoute()
const { t } = useI18n()
const { mdAndUp } = useDisplay()
const userStore = useUserStore()
const authStore = useAuthStore()
const tenantId = computed(() => route.params.tenantId as string)

// Giá trị vai trò giữ nguyên; chỉ đổi chữ hiển thị sang tiếng Việt.
const roleOptions = computed(() => [
  { title: t('us_role_owner'), value: 'owner' },
  { title: t('us_role_admin'), value: 'admin' },
  { title: t('us_role_member'), value: 'member' },
])

const menuItems = computed<ActionMenuItem[]>(() => [
  { key: 'reset', label: t('us_reset'), icon: 'mdi-lock-reset' },
  { key: 'remove', label: t('us_remove'), icon: 'mdi-account-remove-outline', danger: true },
])

const loading = ref(false)
const loadError = ref(false)

const inviteDialog = ref(false)
const inviting = ref(false)
const inviteForm = ref({ name: '', email: '', password: '', role: 'member', permissions: { channels: 'r', messages: 'r', jobs: 'r', settings: '' } as Record<string, string> })
const createFormRef = ref<any>(null)
const permissionFeatures = computed(() => [
  { key: 'channels', label: t('us_feat_channels') },
  { key: 'messages', label: t('us_feat_messages') },
  { key: 'jobs', label: t('us_feat_jobs') },
  { key: 'settings', label: t('us_feat_settings') },
])

const removeDialog = ref(false)
const removing = ref(false)
const removeTarget = ref<TenantUser | null>(null)

const permDialog = ref(false)
const permTarget = ref<TenantUser | null>(null)
const editPerms = ref<Record<string, string>>({})
const savingPerms = ref(false)

const resetDialog = ref(false)
const resetTarget = ref<TenantUser | null>(null)
const resetPassword = ref('')
const resettingPassword = ref(false)

const snack = ref(false)
const snackText = ref('')
const snackColor = ref('success')

function isSelf(u: TenantUser) {
  return u.user_id === authStore.user?.id
}

// Cùng quy tắc với máy chủ (weak_password): ≥ 8 ký tự, có chữ hoa và chữ số.
function strongPassword(v: string) {
  return !!v && v.length >= 8 && /[A-Z]/.test(v) && /\d/.test(v)
}

async function load() {
  loading.value = true
  loadError.value = false
  try {
    await userStore.fetchUsers(tenantId.value)
  } catch {
    loadError.value = true
  } finally {
    loading.value = false
  }
}

onMounted(load)

function onAction(key: string, u: TenantUser) {
  if (key === 'reset') openResetPassword(u)
  if (key === 'remove') confirmRemove(u)
}

async function doInvite() {
  const { valid } = await createFormRef.value?.validate() || {}
  if (!valid) return

  inviting.value = true
  try {
    const payload = {
      ...inviteForm.value,
      permissions: inviteForm.value.role === 'member' ? JSON.stringify(inviteForm.value.permissions) : '',
    }
    await userStore.inviteUser(tenantId.value, payload)
    inviteDialog.value = false
    inviteForm.value = { name: '', email: '', password: '', role: 'member', permissions: { channels: 'r', messages: 'r', jobs: 'r', settings: '' } }
    showSnack(t('us_created'), 'success')
  } catch (err: any) {
    showSnack(friendlyError(err), 'error')
  } finally {
    inviting.value = false
  }
}

async function changeRole(userId: string, role: string) {
  try {
    await userStore.updateRole(tenantId.value, userId, role)
    showSnack(t('us_role_updated'), 'success')
  } catch (err: any) {
    showSnack(err.response?.data?.error || t('us_err_generic'), 'error')
    userStore.fetchUsers(tenantId.value) // reload to revert
  }
}

function confirmRemove(u: TenantUser) {
  removeTarget.value = u
  removeDialog.value = true
}

async function doRemove() {
  if (!removeTarget.value) return
  removing.value = true
  try {
    await userStore.removeUser(tenantId.value, removeTarget.value.user_id)
    removeDialog.value = false
    showSnack(t('us_removed'), 'success')
  } catch (err: any) {
    showSnack(err.response?.data?.error || t('us_err_generic'), 'error')
  } finally {
    removing.value = false
  }
}

function openPermissions(u: TenantUser) {
  permTarget.value = u
  try {
    editPerms.value = u.permissions ? JSON.parse(u.permissions) : { channels: 'r', messages: 'r', jobs: 'r', settings: '' }
  } catch {
    editPerms.value = { channels: 'r', messages: 'r', jobs: 'r', settings: '' }
  }
  permDialog.value = true
}

async function savePermissions() {
  if (!permTarget.value) return
  savingPerms.value = true
  try {
    await api.put(`/tenants/${tenantId.value}/users/${permTarget.value.user_id}/role`, {
      role: 'member',
      permissions: JSON.stringify(editPerms.value),
    })
    // Update local
    permTarget.value.permissions = JSON.stringify(editPerms.value)
    permDialog.value = false
    showSnack(t('us_perm_updated'), 'success')
  } catch (err: any) {
    showSnack(err.response?.data?.error || t('us_err_generic'), 'error')
  } finally {
    savingPerms.value = false
  }
}

function openResetPassword(u: TenantUser) {
  resetTarget.value = u
  resetPassword.value = ''
  resetDialog.value = true
}

async function doResetPassword() {
  if (!resetTarget.value || !strongPassword(resetPassword.value)) return
  resettingPassword.value = true
  try {
    await api.put(`/tenants/${tenantId.value}/users/${resetTarget.value.user_id}/reset-password`, {
      password: resetPassword.value,
    })
    resetDialog.value = false
    showSnack(t('us_password_reset'), 'success')
  } catch (err: any) {
    showSnack(friendlyError(err), 'error')
  } finally {
    resettingPassword.value = false
  }
}

function friendlyError(err: any): string {
  const key = err?.response?.data?.error
  const msg = err?.response?.data?.message
  if (msg) return msg
  const map: Record<string, string> = {
    weak_password: t('us_err_weak'),
    email_already_exists: t('us_err_exists'),
    invalid_request: t('us_err_invalid'),
    password_reset_failed: t('us_err_reset'),
  }
  return map[key] || key || t('us_err_generic')
}

function showSnack(text: string, color: string) {
  snackText.value = text
  snackColor.value = color
  snack.value = true
}
</script>

<style scoped>
.us-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.us-head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 12px;
}
.us-head__titles {
  flex: 1 1 240px;
}
.us-title {
  margin: 0;
  font-size: 26px;
  font-weight: 700;
}
.us-muted {
  color: rgb(var(--v-theme-text-muted));
}
.us-m0 {
  margin: 0;
}
.us-small {
  font-size: 12px;
}
.us-h3 {
  margin: 4px 0 6px;
  font-size: 14px;
  font-weight: 600;
}
.us-table-wrap {
  overflow-x: auto;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 12px;
  background: rgb(var(--v-theme-surface));
}
.us-table-wrap :deep(th) {
  font-size: 12px !important;
  font-weight: 600 !important;
  color: rgb(var(--v-theme-text-muted)) !important;
}
.us-table-wrap :deep(td) {
  height: auto !important;
  padding-top: 10px !important;
  padding-bottom: 10px !important;
  font-size: 14px;
}
.us-col-role {
  width: 200px;
}
.us-col-actions {
  width: 200px;
  text-align: right;
}
.us-cards {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.us-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px 8px 14px 16px;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 12px;
  background: rgb(var(--v-theme-surface));
}
.us-card__top {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}
.us-card__titles {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding-top: 4px;
  overflow-wrap: anywhere;
}
.us-error {
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
