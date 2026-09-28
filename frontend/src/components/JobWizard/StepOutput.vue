<!--
  CCMAI-UX-013: bước 4 — đầu ra. Cổng "mỗi đầu ra phải gửi thử thành công" giữ nguyên.
  /test-output hiện chỉ gửi được Telegram (BLOCKED_API_CONTRACT cho email), nên đầu ra email
  hiện lời giải thích thay vì một nút gửi thử chắc chắn thất bại.
-->
<template>
  <div class="ws-stack">
    <div>
      <h2 class="ws-h2">{{ $t('job_wizard_step_output') }}</h2>
      <p class="ws-intro">{{ $t('jw_output_intro') }}</p>
    </div>

    <article v-for="(output, idx) in outputs" :key="idx" class="ws-item" data-testid="jw-output">
      <div class="ws-item__head">
        <h3 class="ws-h3">{{ $t('jw_output_n', { n: idx + 1 }) }} · {{ output.type === 'telegram' ? $t('output_telegram') : $t('output_email') }}</h3>
        <span v-if="testPassed[idx]" class="ws-ok" data-testid="jw-output-tested">{{ $t('jw_output_tested') }}</span>
        <v-spacer />
        <v-btn icon variant="text" size="44" :aria-label="$t('jw_output_remove', { n: idx + 1 })" @click="removeOutput(idx)">
          <v-icon>mdi-close</v-icon>
        </v-btn>
      </div>

      <v-select
        v-model="output.type"
        :items="[{ title: $t('output_telegram'), value: 'telegram' }, { title: $t('output_email'), value: 'email' }]"
        :label="$t('jw_output_type')"
        variant="outlined"
        density="comfortable"
        hide-details
        @update:model-value="resetTest(idx)"
      />

      <!-- Telegram -->
      <template v-if="output.type === 'telegram'">
        <v-text-field
          v-model="output.bot_token"
          :label="$t('bot_token')"
          :rules="[v => !!v || $t('jw_required')]"
          :hint="$t('jw_bot_token_hint')"
          persistent-hint
          variant="outlined"
          density="comfortable"
          @update:model-value="resetTest(idx)"
        />
        <v-text-field
          v-model="output.chat_id"
          :label="$t('jw_group_id')"
          :rules="[v => !!v || $t('jw_required')]"
          :hint="$t('jw_group_id_hint')"
          persistent-hint
          variant="outlined"
          density="comfortable"
          @update:model-value="resetTest(idx)"
        />
      </template>

      <!-- Email -->
      <template v-else>
        <div class="ws-grid">
          <v-text-field v-model="output.smtp_host" :label="$t('smtp_host')" :rules="[v => !!v || $t('jw_required')]" variant="outlined" density="comfortable" class="ws-grid__wide" @update:model-value="resetTest(idx)" />
          <v-text-field v-model="output.smtp_port" :label="$t('smtp_port')" type="number" :rules="[v => !!v || $t('jw_required')]" variant="outlined" density="comfortable" @update:model-value="resetTest(idx)" />
          <v-text-field v-model="output.smtp_user" :label="$t('smtp_user')" variant="outlined" density="comfortable" />
          <v-text-field v-model="output.smtp_pass" :label="$t('smtp_pass')" type="password" variant="outlined" density="comfortable" />
        </div>
        <v-text-field v-model="output.from" :label="$t('email_from')" :rules="[v => !!v || $t('jw_required')]" variant="outlined" density="comfortable" @update:model-value="resetTest(idx)" />
        <v-text-field v-model="output.to" :label="$t('email_to')" :rules="[v => !!v || $t('jw_required')]" :hint="$t('jw_email_to_hint')" persistent-hint variant="outlined" density="comfortable" @update:model-value="resetTest(idx)" />
      </template>

      <!-- Template -->
      <v-select
        v-model="output.template"
        :items="[{ title: $t('jw_template_default'), value: 'default' }, { title: $t('jw_template_custom'), value: 'custom' }]"
        :label="$t('jw_template')"
        variant="outlined"
        density="comfortable"
        hide-details
        @update:model-value="(val: string) => { if (val === 'custom' && !output.custom_template) output.custom_template = getDefaultTemplate(output.type) }"
      />
      <template v-if="output.template === 'custom'">
        <v-textarea v-model="output.custom_template" :label="$t('jw_template_custom_label')" rows="6" variant="outlined" class="ws-mono" hide-details />
        <p class="ws-note">{{ templateHelpText }}</p>
        <v-btn variant="text" height="40" prepend-icon="mdi-restore" class="align-self-start" @click="output.custom_template = getDefaultTemplate(output.type)">
          {{ $t('jw_template_restore') }}
        </v-btn>
      </template>

      <!-- Gửi thử -->
      <p v-if="output.type === 'email'" class="ws-warn" role="note" data-testid="jw-email-unsupported">{{ $t('jw_output_email_unsupported') }}</p>
      <div v-else class="ws-test">
        <v-btn
          variant="outlined"
          color="primary"
          height="44"
          prepend-icon="mdi-send-check-outline"
          :loading="testingIdx === idx"
          :disabled="!isOutputValid(output)"
          data-testid="jw-output-test"
          @click="testSend(idx)"
        >{{ $t('jw_output_test') }}</v-btn>
        <span v-if="testResult && testResult.idx === idx" :class="testResult.success ? 'ws-ok' : 'ws-bad'" role="status">
          {{ testResult.success ? $t('jw_output_test_ok') : `${$t('jw_output_test_fail')}: ${testResult.message}` }}
        </span>
        <span v-else-if="!testPassed[idx] && isOutputValid(output)" class="ws-hint">{{ $t('jw_output_need_test') }}</span>
      </div>
    </article>

    <v-btn variant="outlined" color="primary" height="44" prepend-icon="mdi-plus" class="align-self-start" @click="addOutput">
      {{ $t('add_output') }}
    </v-btn>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import api from '../../api'

const route = useRoute()
const tenantId = computed(() => route.params.tenantId as string)
const form = defineModel<Record<string, any>>('form', { required: true })

interface OutputItem {
  type: string
  bot_token?: string
  chat_id?: string
  smtp_host?: string
  smtp_port?: number
  smtp_user?: string
  smtp_pass?: string
  from?: string
  to?: string
  template?: string
  custom_template?: string
}

// Tên biến trong mẫu gửi là cú pháp của máy chủ nên giữ nguyên, không đưa qua i18n.
const templateHelpText = `Biến có thể dùng:
{{job_name}} — Tên tác vụ | {{total}} — Tổng hội thoại | {{passed}} — Số đạt | {{failed}} — Số không đạt | {{issues}} — Số vấn đề
{{content}} — Nội dung đánh giá chi tiết | {{link}} — Link xem trên hệ thống`

function getDefaultTemplate(outputType: string) {
  const linkLine = outputType === 'email'
    ? `<a href="{{link}}">Xem chi tiết trên hệ thống</a>`
    : `Xem chi tiết: {{link}}`
  return `<b>Kết quả phân tích: {{job_name}}</b>

Tổng: {{total}} cuộc | Đạt: {{passed}} | Không đạt: {{failed}} | Vấn đề: {{issues}}

{{content}}

${linkLine}`
}

const outputs = ref<OutputItem[]>([])
const testingIdx = ref<number | null>(null)
const testResult = ref<{ idx: number; success: boolean; message: string } | null>(null)
const testPassed = ref<boolean[]>([])
const internalUpdate = ref(false)

function isOutputValid(output: OutputItem): boolean {
  if (output.type === 'telegram') {
    return !!(output.bot_token?.trim() && output.chat_id?.trim())
  }
  if (output.type === 'email') {
    return !!(output.smtp_host?.trim() && output.smtp_port && output.from?.trim() && output.to?.trim())
  }
  return false
}

function resetTest(idx: number) {
  testPassed.value[idx] = false
  syncValidation()
}

// Sync validation state to parent form
function syncValidation() {
  const allPassed = outputs.value.length === 0 || outputs.value.every((_, i) => testPassed.value[i])
  form.value.outputs_validated = allPassed
}

function tryParseOutputs() {
  if (outputs.value.length > 0) return
  if (form.value.outputs && form.value.outputs !== '[]') {
    try {
      const parsed = JSON.parse(form.value.outputs)
      if (Array.isArray(parsed) && parsed.length > 0) {
        internalUpdate.value = true
        outputs.value = parsed.map((o: any) => ({ ...o, template: o.template || 'default' }))
        // Mark existing outputs as tested (edit mode — they were saved before)
        testPassed.value = parsed.map(() => true)
        syncValidation()
      }
    } catch { /* ignore */ }
  }
}

onMounted(() => {
  tryParseOutputs()
  syncValidation()
})

watch(() => form.value.outputs, () => {
  if (!internalUpdate.value) tryParseOutputs()
  internalUpdate.value = false
})

watch(outputs, (val) => {
  internalUpdate.value = true
  form.value.outputs = JSON.stringify(val)
  // Ensure testPassed array matches length
  while (testPassed.value.length < val.length) testPassed.value.push(false)
  syncValidation()
}, { deep: true })

function addOutput() {
  outputs.value.push({ type: 'telegram', bot_token: '', chat_id: '', template: 'default' })
  testPassed.value.push(false)
  syncValidation()
}

function removeOutput(idx: number) {
  outputs.value.splice(idx, 1)
  testPassed.value.splice(idx, 1)
  syncValidation()
}

async function testSend(idx: number) {
  const output = outputs.value[idx]
  testingIdx.value = idx
  testResult.value = null
  try {
    await api.post(`/tenants/${tenantId.value}/test-output`, {
      type: output.type,
      bot_token: output.bot_token,
      chat_id: output.chat_id,
    })
    testResult.value = { idx, success: true, message: '' }
    testPassed.value[idx] = true
    syncValidation()
  } catch (err: any) {
    const msg = err.response?.data?.error || ''
    testResult.value = { idx, success: false, message: msg }
    testPassed.value[idx] = false
    syncValidation()
  } finally {
    testingIdx.value = null
  }
}
</script>

<style scoped>
.ws-stack {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.ws-h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}
.ws-h3 {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
}
.ws-intro {
  margin: 4px 0 0;
  font-size: 14px;
  line-height: 1.5;
  color: rgb(var(--v-theme-text-muted));
}
.ws-item {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 8px 12px 16px 16px;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 12px;
}
.ws-item__head {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.ws-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 0 12px;
}
.ws-mono :deep(textarea) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 13px;
}
.ws-note {
  margin: 0;
  padding: 10px 12px;
  border-radius: 8px;
  background: rgba(var(--v-theme-on-surface), 0.04);
  font-size: 12px;
  line-height: 1.5;
  white-space: pre-line;
}
.ws-warn {
  margin: 0;
  padding: 10px 12px;
  border: 1px solid rgb(var(--v-theme-border));
  border-left: 4px solid rgb(var(--v-theme-src-changed));
  border-radius: 10px;
  font-size: 13px;
  line-height: 1.5;
}
.ws-test {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px 12px;
}
.ws-ok {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 8px;
  border-radius: 8px;
  background: rgb(var(--v-theme-pass-bg));
  color: rgb(var(--v-theme-pass));
  font-size: 12px;
  font-weight: 600;
}
.ws-bad {
  font-size: 13px;
  font-weight: 600;
  color: rgb(var(--v-theme-fail));
}
.ws-hint {
  font-size: 13px;
  color: rgb(var(--v-theme-src-changed));
}
</style>
