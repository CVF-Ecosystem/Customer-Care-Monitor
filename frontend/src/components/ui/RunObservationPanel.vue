<template>
  <details class="run-observation" :data-run-id="context.runId">
    <summary>{{ t('run_obs_title') }}</summary>
    <div class="run-observation__body">
      <p>{{ t('run_obs_prefix_note') }}</p>
      <section v-for="section in sections" :key="section.key" :data-section="section.key" :aria-label="t(`run_obs_${section.key}`)">
        <h4>{{ t(`run_obs_${section.key}`) }}</h4>
        <p v-if="!section.observation.available" data-testid="observation-unavailable">{{ t('run_obs_unavailable') }}</p>
        <template v-else>
          <dl>
            <div v-for="(value, key) in section.observation.values" :key="key">
              <dt>{{ t(`run_obs_field_${key}`) }}</dt>
              <dd :data-field="key" :class="{ 'run-observation__hash': key === 'fingerprint' }">{{ display(value, key) }}</dd>
            </div>
          </dl>
          <p v-if="section.key === 'preparation'">{{ t('run_obs_preparation_note') }}</p>
          <p v-if="section.key === 'rules'">{{ t('run_obs_rules_note') }}</p>
          <p v-if="section.key === 'usage'">{{ t('run_obs_usage_note') }}</p>
        </template>
      </section>
    </div>
  </details>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { projectRunObservation, type RunObservationContext } from '../../views/Jobs/job-detail/run-observation'
const props = defineProps<{ summary: string; context: RunObservationContext }>()
const { t } = useI18n()
const sections = computed(() => Object.entries(projectRunObservation(props.summary, props.context)).map(([key, observation]) => ({ key, observation })))
function display(value: number | string | boolean | null, key: string): string {
  // UNKNOWN_RENDER_GUARD: preserve unavailable totals, including unknown prices.
  if (value === null) return t('run_obs_unknown')
  if (typeof value === 'boolean') return t(value ? 'run_obs_yes' : 'run_obs_no')
  if (key === 'stop_reason') return t(`run_obs_stop_${value}`)
  if (key === 'local_estimate_usd') return `${value} USD`
  return String(value)
}
</script>

<style scoped>
.run-observation { margin-top: 8px; min-width: 220px; max-width: 520px; white-space: normal; color: rgb(var(--v-theme-text-primary)); font-size: 13px; line-height: 1.5; }
summary { cursor: pointer; font-weight: 600; border-radius: 4px; padding: 4px 0; }
summary:focus-visible { outline: 2px solid rgb(var(--v-theme-primary)); outline-offset: 3px; }
.run-observation__body { padding: 8px 12px; border: 1px solid rgb(var(--v-theme-border)); border-radius: 8px; background: rgb(var(--v-theme-surface)); }
p { margin: 4px 0 10px; color: rgb(var(--v-theme-text-muted)); }
section + section { border-top: 1px solid rgb(var(--v-theme-border)); margin-top: 12px; padding-top: 10px; }
h4 { margin: 0 0 6px; font-size: 14px; font-weight: 600; }
dl { margin: 0; }
dl > div { display: flex; justify-content: space-between; gap: 12px; padding: 3px 0; }
dt { flex: 1; overflow-wrap: anywhere; }
dd { margin: 0; text-align: right; font-variant-numeric: tabular-nums; max-width: 55%; overflow-wrap: anywhere; }
.run-observation__hash { font-family: monospace; font-size: 12px; }
@media (max-width: 600px) { .run-observation { min-width: 200px; max-width: 100%; } .run-observation__body { padding: 8px; } dl > div { gap: 8px; } }
</style>
