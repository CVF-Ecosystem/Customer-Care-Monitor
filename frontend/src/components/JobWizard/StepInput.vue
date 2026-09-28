<!-- CCMAI-UX-013: bước 2 — chọn kênh đầu vào. -->
<template>
  <div class="ws-stack">
    <div>
      <h2 class="ws-h2">{{ $t('jw_input_title') }}</h2>
      <p class="ws-intro">{{ $t('jw_input_intro') }}</p>
    </div>
    <div v-if="!channels.length" class="ws-empty">
      <span>{{ $t('jw_input_empty') }}</span>
      <v-btn variant="outlined" height="44" :to="`/${route.params.tenantId}/channels`">{{ $t('jw_input_go') }}</v-btn>
    </div>
    <template v-else>
      <span class="ws-muted ws-small" data-testid="jw-selected">{{ $t('jw_selected', { n: form.input_channel_ids?.length || 0 }) }}</span>
      <div class="ws-list">
        <label v-for="ch in channels" :key="ch.id" class="ws-row">
          <v-checkbox-btn v-model="form.input_channel_ids" :value="ch.id" :aria-label="ch.name" />
          <span class="ws-row__text">
            <span class="ws-row__name">{{ ch.name }}</span>
            <span class="ws-muted ws-small">
              {{ channelTypeInfo(ch.channel_type).label }} ·
              <span :class="ch.is_active ? 'ws-on' : 'ws-off'">{{ ch.is_active ? $t('jl_active') : $t('jl_inactive') }}</span>
            </span>
          </span>
        </label>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { channelTypeInfo } from '../../composables/channelTypes'
import { onMounted } from 'vue'
import { storeToRefs } from 'pinia'
import { useRoute } from 'vue-router'
import { useChannelStore } from '../../stores/channels'

const form = defineModel<Record<string, any>>('form', { required: true })
const route = useRoute()
const channelStore = useChannelStore()
const { channels } = storeToRefs(channelStore)

onMounted(() => {
  const tenantId = route.params.tenantId as string
  channelStore.fetchChannels(tenantId)
})
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
.ws-intro {
  margin: 4px 0 0;
  font-size: 14px;
  color: rgb(var(--v-theme-text-muted));
}
.ws-muted {
  color: rgb(var(--v-theme-text-muted));
}
.ws-small {
  font-size: 12px;
}
.ws-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 24px;
  border: 1px dashed rgb(var(--v-theme-border));
  border-radius: 12px;
  font-size: 14px;
  color: rgb(var(--v-theme-text-muted));
}
.ws-list {
  display: flex;
  flex-direction: column;
  border: 1px solid rgb(var(--v-theme-border));
  border-radius: 12px;
  overflow: hidden;
}
.ws-row {
  display: flex;
  align-items: center;
  gap: 6px;
  min-height: 56px;
  padding: 6px 12px 6px 4px;
  border-bottom: 1px solid rgb(var(--v-theme-border));
  cursor: pointer;
}
.ws-row:last-child {
  border-bottom: 0;
}
.ws-row__text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.ws-row__name {
  font-size: 14px;
  font-weight: 600;
}
.ws-on {
  color: rgb(var(--v-theme-pass));
  font-weight: 600;
}
.ws-off {
  font-weight: 600;
}
</style>
