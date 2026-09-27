<!--
  CCMAI-UX-000: "⋯" menu for secondary and destructive actions. It only emits `select`;
  the parent decides what happens (destructive items must open a ConfirmDialog).
  Danger items are always listed last, after a divider.
-->
<template>
  <v-menu location="bottom end">
    <template #activator="{ props: menuProps }">
      <v-btn
        v-bind="menuProps"
        icon
        variant="text"
        size="44"
        :aria-label="label ?? t('ui_more_actions')"
        data-testid="action-menu"
      >
        <v-icon>mdi-dots-horizontal</v-icon>
      </v-btn>
    </template>
    <v-list density="comfortable" min-width="200">
      <v-list-item
        v-for="item in safeItems"
        :key="item.key"
        :prepend-icon="item.icon"
        :title="item.label"
        :data-key="item.key"
        @click="emit('select', item.key)"
      />
      <v-divider v-if="safeItems.length && dangerItems.length" class="my-1" />
      <v-list-item
        v-for="item in dangerItems"
        :key="item.key"
        :prepend-icon="item.icon"
        :title="item.label"
        :data-key="item.key"
        base-color="danger"
        @click="emit('select', item.key)"
      />
    </v-list>
  </v-menu>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ActionMenuItem } from './types'

const props = defineProps<{ items: ActionMenuItem[]; label?: string }>()
const emit = defineEmits<{ select: [key: string] }>()
const { t } = useI18n()

const safeItems = computed(() => props.items.filter((i) => !i.danger))
const dangerItems = computed(() => props.items.filter((i) => i.danger))
</script>
