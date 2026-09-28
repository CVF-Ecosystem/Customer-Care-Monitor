<!-- CCMAI-UX-014: đăng nhập. Luồng xác thực và thông báo lỗi giữ nguyên; chỉ trình bày lại. -->
<template>
  <section class="au-card" :aria-label="$t('login_title')">
    <h2 class="au-title">{{ $t('login_title') }}</h2>
    <p v-if="errorMsg" class="au-error" role="alert" data-testid="au-error">{{ errorMsg }}</p>
    <v-form class="au-form" @submit.prevent="handleLogin">
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
      <v-text-field
        v-model="password"
        :label="$t('password')"
        :type="showPass ? 'text' : 'password'"
        autocomplete="current-password"
        prepend-inner-icon="mdi-lock-outline"
        :append-inner-icon="showPass ? 'mdi-eye-off' : 'mdi-eye'"
        variant="outlined"
        hide-details="auto"
        required
        @click:append-inner="showPass = !showPass"
      />
      <v-btn type="submit" color="primary" block height="48" :loading="loading" data-testid="au-submit">
        {{ $t('login') }}
      </v-btn>
    </v-form>
  </section>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const { t } = useI18n()
const authStore = useAuthStore()

const email = ref('')
const password = ref('')
const showPass = ref(false)
const loading = ref(false)
const errorMsg = ref('')

async function handleLogin() {
  loading.value = true
  errorMsg.value = ''
  try {
    await authStore.login(email.value, password.value)
    router.push('/')
  } catch (err: any) {
    const status = err?.response?.status
    if (status === 429) {
      // Backend đã soạn sẵn câu thông báo kèm thời gian mở khoá
      errorMsg.value = err.response?.data?.message || t('account_locked')
    } else if (status === 401) {
      errorMsg.value = t('invalid_credentials')
    } else {
      errorMsg.value = t('login_failed')
    }
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
