<!--
  CCMAI-UX-000: confirmation for destructive actions. The parent performs the action
  on `confirm`; this dialog never acts on its own. While loading, both buttons are disabled.
-->
<template>
  <v-dialog :model-value="modelValue" max-width="440" :persistent="loading" @update:model-value="onUpdate">
    <v-card class="ccma-confirm" role="alertdialog" :aria-label="title">
      <div class="ccma-confirm__body">
        <v-icon class="ccma-confirm__icon" size="24" aria-hidden="true">mdi-alert-outline</v-icon>
        <div>
          <h2 class="ccma-confirm__title">{{ title }}</h2>
          <p class="ccma-confirm__message">{{ message }}</p>
        </div>
      </div>
      <v-card-actions class="ccma-confirm__actions">
        <v-btn variant="text" size="large" :disabled="loading" data-testid="confirm-cancel" @click="cancel">{{ t('ui_cancel') }}</v-btn>
        <v-btn
          variant="flat"
          color="danger"
          size="large"
          :loading="loading"
          :disabled="loading"
          data-testid="confirm-ok"
          @click="emit('confirm')"
        >{{ confirmLabel }}</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  modelValue: boolean
  title: string
  message: string
  confirmLabel: string
  loading?: boolean
}>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; confirm: []; cancel: [] }>()
const { t } = useI18n()

function cancel() {
  emit('cancel')
  emit('update:modelValue', false)
}

function onUpdate(v: boolean) {
  if (!v && !props.loading) cancel()
}
</script>

<style scoped>
.ccma-confirm__body {
  display: flex;
  gap: 12px;
  padding: 20px 20px 8px;
}
.ccma-confirm__icon {
  color: rgb(var(--v-theme-danger));
  margin-top: 2px;
}
.ccma-confirm__title {
  font-size: 16px;
  font-weight: 600;
  margin: 0 0 6px;
}
.ccma-confirm__message {
  margin: 0;
  font-size: 14px;
  line-height: 1.5;
  color: rgb(var(--v-theme-text-muted));
}
.ccma-confirm__actions {
  justify-content: flex-end;
  gap: 8px;
  padding: 12px 20px 16px;
}
</style>
