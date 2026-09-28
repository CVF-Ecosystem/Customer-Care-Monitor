<!-- CCMAI-UX-014: thiết lập lần đầu. Dữ liệu gửi đi và kiểm tra giữ nguyên; chữ qua i18n. -->
<template>
  <section class="au-card" :aria-label="$t('au_setup_title')">
    <div>
      <h2 class="au-title">{{ $t('au_setup_title') }}</h2>
      <p class="au-desc">{{ $t('au_setup_desc') }}</p>
    </div>
    <p v-if="errorMsg" class="au-error" role="alert" data-testid="au-error">{{ errorMsg }}</p>
    <v-form class="au-form" @submit.prevent="handleSetup">
      <v-text-field v-model="workspaceName" :label="$t('au_workspace')" :error-messages="workspaceError" variant="outlined" hide-details="auto" required />
      <v-text-field
        v-model="email"
        :label="$t('email')"
        type="email"
        autocomplete="username"
        prepend-inner-icon="mdi-email-outline"
        variant="outlined"
        hide-details="auto"
        required
      />
      <v-text-field v-model="name" :label="$t('au_display_name')" prepend-inner-icon="mdi-account-outline" variant="outlined" hide-details="auto" />
      <v-text-field
        v-model="password"
        :label="$t('password')"
        :type="showPass ? 'text' : 'password'"
        autocomplete="new-password"
        prepend-inner-icon="mdi-lock-outline"
        :append-inner-icon="showPass ? 'mdi-eye-off' : 'mdi-eye'"
        :hint="$t('au_password_rule')"
        persistent-hint
        variant="outlined"
        required
        @click:append-inner="showPass = !showPass"
      />
      <v-text-field
        v-model="confirmPassword"
        :label="$t('au_confirm_password')"
        :type="showPass ? 'text' : 'password'"
        autocomplete="new-password"
        prepend-inner-icon="mdi-lock-check-outline"
        :error-messages="confirmError"
        variant="outlined"
        hide-details="auto"
        required
      />
      <v-btn type="submit" color="primary" block height="48" :loading="loading" data-testid="au-submit">{{ $t('au_create') }}</v-btn>
    </v-form>
  </section>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { markSetupComplete } from '../router'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const { t } = useI18n()
const authStore = useAuthStore()

const email = ref('')
const name = ref('')
const workspaceName = ref('')
const workspaceError = ref('')
const password = ref('')
const confirmPassword = ref('')
const showPass = ref(false)
const loading = ref(false)
const errorMsg = ref('')
const confirmError = ref('')

async function handleSetup() {
  confirmError.value = ''
  errorMsg.value = ''
  workspaceError.value = ''

  if (workspaceName.value.trim().length < 2) {
    workspaceError.value = t('au_workspace_min')
    return
  }

  if (password.value !== confirmPassword.value) {
    confirmError.value = t('au_mismatch')
    return
  }

  loading.value = true
  try {
    await authStore.setup(name.value, email.value, password.value, workspaceName.value.trim())
    markSetupComplete()
    router.push('/')
  } catch (err: any) {
    errorMsg.value = err.response?.data?.message || err.response?.data?.error || t('au_error')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.au-card {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 24px;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 16px;
  background: rgb(var(--v-theme-surface));
}
.au-title {
  margin: 0;
  font-size: 20px;
  font-weight: 700;
  text-align: center;
}
.au-desc {
  margin: 4px 0 0;
  font-size: 14px;
  text-align: center;
  color: rgb(var(--v-theme-text-muted));
}
.au-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.au-error {
  margin: 0;
  padding: 10px 12px;
  border: 1px solid rgb(var(--v-theme-fail));
  border-radius: 10px;
  background: rgb(var(--v-theme-fail-bg));
  color: rgb(var(--v-theme-fail));
  font-size: 14px;
  font-weight: 600;
}
</style>
