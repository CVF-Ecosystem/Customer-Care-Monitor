<template>
  <v-alert v-if="message" type="info" class="mb-4" role="status">{{ message }}</v-alert>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getInterventionCapability, type InterventionCapability } from '../api/interventionCapability'

const props = defineProps<{ tenantId: string; token: string }>()
const { t } = useI18n({ useScope: 'local', messages: {
  vi: {
    unknown: 'Chưa kiểm tra được khả năng tạo và đánh giá lại đề xuất. Vui lòng thử lại sau.',
    disabled: 'Tạo đề xuất và đánh giá lại bằng AI chưa được bật. Bạn vẫn có thể xem và xử lý các vụ việc hiện có theo quyền được cấp.',
    partial: 'Tính năng AI mới được cấu hình một phần và chỉ dành cho dữ liệu minh họa. Một số tác vụ tạo hoặc đánh giá lại đề xuất chưa được bật.',
    synthetic: 'Tạo và đánh giá lại đề xuất chỉ dành cho dữ liệu minh họa. Mỗi lượt vẫn cần đúng quyền và hạn mức được cấp.',
  },
  en: {
    unknown: 'Proposal generation and reassessment availability could not be checked. Please try again later.',
    disabled: 'AI proposal generation and reassessment are not enabled. You can still view and handle existing cases according to your permissions.',
    partial: 'AI is partly configured and limited to demonstration data. Some generation or reassessment actions are not enabled.',
    synthetic: 'Proposal generation and reassessment are limited to demonstration data. Each request still requires permission and an available allowance.',
  },
}})
const capability = ref<InterventionCapability | null>(null)
let request: AbortController | null = null
let generation = 0
const message = computed(() => {
  if (!props.tenantId || !props.token) return ''
  const c = capability.value
  if (!c) return t('unknown')
  if (!c.generation_configured && !c.rerun_configured) return t('disabled')
  if (!c.generation_configured || !c.rerun_configured) return t('partial')
  return t('synthetic')
})
watch(() => [props.tenantId, props.token], async ([tenant, token]) => {
  request?.abort()
  capability.value = null
  const current = ++generation
  if (!tenant || !token) return
  const controller = new AbortController(); request = controller
  try {
    const result = await getInterventionCapability(tenant, token, controller.signal)
    if (current === generation && !controller.signal.aborted) capability.value = result
  } catch { /* Unknown remains visible; never infer enabled from a failed read. */ }
}, { immediate: true })
onBeforeUnmount(() => { ++generation; request?.abort() })
</script>
