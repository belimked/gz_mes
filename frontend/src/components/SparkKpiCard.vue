<script setup lang="ts">
import { computed } from 'vue'
import type { EChartsOption } from 'echarts'
import EChart from './EChart.vue'
import PageCard from './PageCard.vue'

const props = withDefaults(
  defineProps<{
    title: string
    icon: string
    color?: string
    value: string
    unit?: string
    delta?: number
    series: number[]
    footer: { label: string; value: string }[]
  }>(),
  { color: '#1677ff', unit: undefined, delta: undefined },
)

const option = computed<EChartsOption>(() => ({
  grid: { left: 0, right: 0, top: 0, bottom: 0 },
  xAxis: { type: 'category', show: false, boundaryGap: false, data: props.series.map((_, i) => i) },
  yAxis: { type: 'value', show: false, scale: true },
  tooltip: { show: false },
  series: [
    {
      type: 'line',
      data: props.series,
      smooth: true,
      symbol: 'none',
      lineStyle: { width: 2, color: props.color },
      areaStyle: {
        color: {
          type: 'linear',
          x: 0,
          y: 0,
          x2: 0,
          y2: 1,
          colorStops: [
            { offset: 0, color: `${props.color}40` },
            { offset: 1, color: `${props.color}00` },
          ],
        },
      },
    },
  ],
}))
</script>
<template>
  <PageCard class="spark-card" :style="{ '--spark-color': color, '--spark-icon-bg': `${color}1a` }">
    <div class="main">
      <div class="details">
        <div class="heading">
          <span class="icon"
            ><slot name="icon">{{ icon }}</slot></span
          >
          <span class="title">{{ title }}</span>
        </div>
        <div class="number-row">
          <strong class="number"
            >{{ value }}<small v-if="unit">{{ unit }}</small></strong
          >
          <span v-if="delta !== undefined && delta !== 0" class="delta" :class="delta > 0 ? 'up' : 'down'">
            {{ delta > 0 ? '↑' : '↓' }} {{ Math.abs(delta).toFixed(1) }}%
          </span>
        </div>
      </div>
      <div class="spark"><EChart :option="option" :height="56" /></div>
    </div>
    <div class="footer">
      <div v-for="item in footer.slice(0, 2)" :key="item.label" class="footer-item">
        <span>{{ item.label }}</span
        ><strong>{{ item.value }}</strong>
      </div>
    </div>
  </PageCard>
</template>
<style scoped>
.spark-card {
  min-width: 0;
}
.main {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 8px;
  min-height: 84px;
}
.details {
  min-width: 0;
  flex: 1;
}
.heading,
.number-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.heading {
  color: var(--gz-secondary);
  font-size: 13px;
  white-space: nowrap;
}
.icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 32px;
  height: 32px;
  border-radius: 10px;
  background: var(--spark-icon-bg);
  color: var(--spark-color);
  font-size: 18px;
  font-weight: 700;
}
.number-row {
  margin-top: 12px;
  white-space: nowrap;
}
.number {
  font-size: 27px;
  line-height: 36px;
  font-variant-numeric: tabular-nums;
}
.number small {
  margin-left: 4px;
  color: var(--gz-muted);
  font-size: 12px;
  font-weight: 400;
}
.delta {
  font-size: 11px;
  font-weight: 600;
}
.up {
  color: #52c41a;
}
.down {
  color: #ff4d4f;
}
.spark {
  flex: 0 0 45%;
  min-width: 0;
}
.footer {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px solid var(--gz-border);
  font-size: 12px;
}
.footer-item {
  display: flex;
  gap: 4px;
  min-width: 0;
  white-space: nowrap;
  color: var(--gz-secondary);
}
.footer-item strong {
  color: var(--spark-color);
  font-weight: 600;
}
</style>
