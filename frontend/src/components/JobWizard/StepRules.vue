<!-- CCMAI-UX-013: bước 3 — quy tắc (chất lượng CSKH) hoặc nhãn (phân loại). Dữ liệu lưu giữ nguyên. -->
<template>
  <div class="ws-stack">
    <div>
      <h2 class="ws-h2">{{ $t('job_wizard_step_rules') }}</h2>
      <p class="ws-intro">{{ form.job_type === 'qc_analysis' ? $t('jw_rules_intro_qc') : $t('jw_rules_intro_class') }}</p>
    </div>

    <!-- QC Analysis: markdown rules -->
    <template v-if="form.job_type === 'qc_analysis'">
      <v-textarea
        v-model="form.rules_content"
        :label="$t('jw_rules_label')"
        :placeholder="$t('rules_placeholder')"
        rows="12"
        auto-grow
        variant="outlined"
        class="ws-mono"
      />
      <v-btn variant="outlined" height="40" prepend-icon="mdi-file-document-outline" class="align-self-start" @click="loadTemplate">
        {{ $t('jw_rules_template') }}
      </v-btn>

      <v-divider class="my-2" />
      <div>
        <h3 class="ws-h3">{{ $t('jw_skip_title') }}</h3>
        <p class="ws-intro">{{ $t('jw_skip_desc') }}</p>
      </div>
      <v-textarea
        v-model="form.skip_conditions"
        :placeholder="$t('jw_skip_placeholder')"
        :aria-label="$t('jw_skip_title')"
        rows="3"
        auto-grow
        variant="outlined"
      />
      <v-btn variant="text" height="40" prepend-icon="mdi-file-document-outline" class="align-self-start" @click="loadSkipTemplate">
        {{ $t('jw_skip_template') }}
      </v-btn>
    </template>

    <!-- Classification: danh sách nhãn -->
    <template v-else>
      <article v-for="(rule, idx) in rules" :key="idx" class="ws-item" data-testid="jw-tag">
        <div class="ws-item__head">
          <h3 class="ws-h3">{{ $t('jw_tag_n', { n: idx + 1 }) }}</h3>
          <v-btn icon variant="text" size="44" :aria-label="$t('jw_tag_remove', { n: idx + 1 })" @click="removeRule(idx)">
            <v-icon>mdi-close</v-icon>
          </v-btn>
        </div>
        <v-text-field v-model="rule.name" :label="$t('jw_tag_name')" variant="outlined" density="comfortable" hide-details="auto" />
        <v-textarea v-model="rule.description" :label="$t('jw_tag_desc')" rows="2" auto-grow variant="outlined" density="comfortable" hide-details="auto" />
        <v-select
          v-model="rule.severity"
          :label="$t('jw_tag_severity')"
          :items="[{ title: $t('severity_critical'), value: 'NGHIEM_TRONG' }, { title: $t('severity_warning'), value: 'CAN_CAI_THIEN' }]"
          variant="outlined"
          density="comfortable"
          hide-details
        />
      </article>
      <v-btn variant="outlined" color="primary" height="44" prepend-icon="mdi-plus" class="align-self-start" @click="addRule">
        {{ $t('jw_tag_add') }}
      </v-btn>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

const form = defineModel<Record<string, any>>('form', { required: true })

function parseRules() {
  try {
    const parsed = JSON.parse(form.value.rules_config || '[]')
    return Array.isArray(parsed) ? parsed : []
  } catch { return [] }
}

const rules = ref<Array<{ name: string; description: string; severity: string }>>(parseRules())

// Sync rules to form.rules_config
watch(rules, (val) => {
  form.value.rules_config = JSON.stringify(val)
}, { deep: true })

function addRule() {
  rules.value.push({ name: '', description: '', severity: 'CAN_CAI_THIEN' })
}

function removeRule(idx: number) {
  rules.value.splice(idx, 1)
}

// Nội dung mẫu là dữ liệu người dùng có thể sửa, không phải chữ giao diện.
const defaultTemplate = `# Quy định chất lượng CSKH

## 1. Thời gian phản hồi
- Phải trả lời khách trong vòng 5 phút
- Nghiêm trọng nếu không phản hồi sau 15 phút

## 2. Thái độ giao tiếp
- Luôn chào hỏi lịch sự
- Không dùng từ ngữ thô tục
- Phải xin lỗi khi khách phản ánh vấn đề

## 3. Chất lượng nội dung
- Trả lời đúng trọng tâm câu hỏi
- Cung cấp thông tin chính xác
- Hướng dẫn cụ thể, rõ ràng

## 4. Kết thúc hội thoại
- Hỏi khách còn cần hỗ trợ gì không
- Cảm ơn khách đã liên hệ`

function loadTemplate() {
  form.value.rules_content = defaultTemplate
}

const defaultSkipTemplate = `- Cuộc chat dưới 2 tin nhắn
- Khách chỉ gửi sticker hoặc hình ảnh mà không có nội dung text
- Cuộc chat chỉ có tin nhắn tự động từ hệ thống`

function loadSkipTemplate() {
  form.value.skip_conditions = defaultSkipTemplate
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
.ws-mono :deep(textarea) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 13px;
}
.ws-item {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 8px 12px 14px 16px;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 12px;
}
.ws-item__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
</style>
