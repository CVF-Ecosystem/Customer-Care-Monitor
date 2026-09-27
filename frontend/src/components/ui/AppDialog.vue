<!--
  CCMAI-UX-000: dialog shell. Always a close button in the header corner, closes on Esc,
  full screen on phones. `aiGenerated` adds the "Nhận xét do AI tạo" label under the title.
-->
<template>
  <v-dialog
    :model-value="modelValue"
    :max-width="maxWidth ?? 720"
    :fullscreen="smAndDown"
    scrollable
    @update:model-value="emit('update:modelValue', $event)"
  >
    <v-card class="ccma-dialog">
      <header class="ccma-dialog__head">
        <div class="ccma-dialog__titles">
          <h2 class="ccma-dialog__title">{{ title }}</h2>
          <div v-if="subtitle || aiGenerated || $slots.meta" class="ccma-dialog__meta">
            <span v-if="subtitle">{{ subtitle }}</span>
            <AiGeneratedLabel v-if="aiGenerated" />
            <slot name="meta" />
          </div>
        </div>
        <v-btn
          icon
          variant="text"
          size="44"
          class="ccma-dialog__close"
          :aria-label="t('ui_close')"
          data-testid="dialog-close"
          @click="emit('update:modelValue', false)"
        >
          <v-icon>mdi-close</v-icon>
        </v-btn>
      </header>
      <v-divider />
      <v-card-text class="ccma-dialog__body">
        <slot />
      </v-card-text>
      <template v-if="$slots.actions">
        <v-divider />
        <v-card-actions class="ccma-dialog__actions">
          <slot name="actions" />
        </v-card-actions>
      </template>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import { useDisplay } from 'vuetify'
import { useI18n } from 'vue-i18n'
import AiGeneratedLabel from './AiGeneratedLabel.vue'

defineProps<{
  modelValue: boolean
  title: string
  subtitle?: string
  aiGenerated?: boolean
  maxWidth?: number | string
}>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()
const { t } = useI18n()
const { smAndDown } = useDisplay()
</script>

<style scoped>
.ccma-dialog__head {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 12px 8px 12px 20px;
}
.ccma-dialog__titles {
  flex: 1;
  min-width: 0;
  padding-top: 8px;
}
.ccma-dialog__title {
  font-size: 18px;
  font-weight: 600;
  line-height: 1.35;
  margin: 0;
  overflow-wrap: anywhere;
}
.ccma-dialog__meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px 12px;
  margin-top: 4px;
  font-size: 13px;
  color: rgb(var(--v-theme-text-muted));
}
.ccma-dialog__close {
  flex-shrink: 0;
}
.ccma-dialog__body {
  padding: 16px 20px !important;
}
.ccma-dialog__actions {
  padding: 12px 20px;
  gap: 8px;
  justify-content: flex-end;
  flex-wrap: wrap;
}
</style>
