<script setup lang="ts">
import { PROCESS_STEPS, statusTone, type Tone } from '@/theme'
import StatusTag from './StatusTag.vue'
defineProps<{ items: { count: string; label: string; tone: Tone }[] }>()
</script>
<template>
  <div class="pipeline">
    <div
      v-for="(step, i) in PROCESS_STEPS"
      :key="step"
      class="step"
      :style="{ background: statusTone[items[i]?.tone ?? 'neutral'].bg, borderColor: statusTone[items[i]?.tone ?? 'neutral'].border }"
    >
      <div class="step-name">{{ String(i + 1).padStart(2, '0') }}. {{ step }}</div>
      <b>{{ items[i]?.count ?? '—' }}</b>
      <StatusTag :tone="items[i]?.tone ?? 'neutral'" :label="items[i]?.label ?? '暂无数据'" />
    </div>
  </div>
</template>
<style scoped>
.pipeline {
  display: flex;
  gap: 8px;
  overflow-x: auto;
  padding-bottom: 4px;
}
.step {
  flex: 1 0 90px;
  min-height: 134px;
  border: 1px solid;
  border-radius: var(--gz-radius);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: space-between;
  text-align: center;
  padding: 10px 4px;
}
.step-name {
  font-size: 12px;
  color: var(--gz-secondary);
}
.step b {
  font-size: 18px;
  font-variant-numeric: tabular-nums;
}
</style>
