<script setup lang="ts">
import { PROCESS_STEPS } from '@/theme'
defineProps<{ current: number; quantity: number }>()
</script>
<template>
  <a-timeline class="timeline">
    <a-timeline-item
      v-for="(step, i) in PROCESS_STEPS.slice(0, Math.min(current + 2, 11))"
      :key="step"
      :color="i < current ? 'blue' : i === current ? 'blue' : 'gray'"
    >
      <div :class="{ active: i === current }">
        <strong>{{ String(i + 1).padStart(2, '0') }}. {{ step }}</strong>
        <span class="time">{{ i < current ? '已完成 · 08:30' : i === current ? '进行中' : '待接单' }}</span>
        <p>{{ i < current ? `演示一班 · 完成 ${quantity} 件` : i === current ? '演示工位 · 正在报工' : '等待上道工序交接' }}</p>
      </div>
    </a-timeline-item>
    <a-timeline-item v-if="current < 9" color="gray">后序：{{ PROCESS_STEPS.slice(current + 2).join(' → ') }}</a-timeline-item>
  </a-timeline>
</template>
<style scoped>
.timeline {
  padding: 12px 8px;
}
.active {
  padding: 8px;
  background: var(--gz-bg);
  color: var(--gz-primary);
  border-radius: 4px;
}
.time {
  float: right;
  font-size: 12px;
  color: var(--gz-muted);
}
p {
  margin: 4px 0;
  color: var(--gz-secondary);
  font-size: 12px;
}
</style>
