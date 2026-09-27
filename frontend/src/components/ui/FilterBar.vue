<!--
  CCMAI-UX-000: filter chip row. On narrow screens it scrolls sideways instead of
  wrapping or clipping chip text.
-->
<template>
  <div class="ccma-filterbar" role="toolbar" :aria-label="label">
    <slot />
  </div>
</template>

<script setup lang="ts">
defineProps<{ label?: string }>()
</script>

<style scoped>
.ccma-filterbar {
  display: flex;
  gap: 8px;
  overflow-x: auto;
  overscroll-behavior-x: contain;
  scrollbar-width: thin;
  padding-bottom: 4px;
}
.ccma-filterbar > :deep(*) {
  flex-shrink: 0;
}
</style>
