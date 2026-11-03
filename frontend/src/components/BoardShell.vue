<script setup lang="ts" generic="T extends { planId: number; urgency: string; dueToday: boolean; state: string }">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import type { EChartsOption } from 'echarts'
import EChart from './EChart.vue'

const props = defineProps<{
  title: string
  filterLabel: string
  rows: T[]
  kpis: { label: string; value: number; unit?: string }[]
  chartLabels: string[]
  chartPlanned: number[]
  chartDone: number[]
  refreshed: string
}>()
const emit = defineEmits<{ refresh: [date: string, filter: string] }>()
const route = useRoute()
const date = ref(new Date().toISOString().slice(0, 10))
const filter = ref('')
const clock = ref(new Date())
const page = ref(0)
const hovered = ref(false)
const fullscreen = ref(false)
const standalone = computed(() => route.query.fullscreen === '1' || fullscreen.value)
const count = computed(() => Math.max(1, Math.ceil(props.rows.length / 12)))
const visible = computed(() => props.rows.slice(page.value * 12, (page.value + 1) * 12))
const chart = computed<EChartsOption>(() => ({
  backgroundColor: 'transparent',
  tooltip: { trigger: 'axis' },
  legend: { textStyle: { color: '#b9d7f0' } },
  grid: { left: 44, right: 18, top: 38, bottom: 35 },
  xAxis: { type: 'category', data: props.chartLabels, axisLabel: { color: '#a8c5df' } },
  yAxis: { type: 'value', axisLabel: { color: '#a8c5df' }, splitLine: { lineStyle: { color: '#28415b' } } },
  series: [
    { name: '计划面积', type: 'bar', data: props.chartPlanned, itemStyle: { color: '#2389e6' } },
    { name: '当日完成', type: 'bar', data: props.chartDone, itemStyle: { color: '#25d4c8' } },
  ],
}))
let tick: ReturnType<typeof setInterval> | undefined
let refreshTimer: ReturnType<typeof setInterval> | undefined
let rotate: ReturnType<typeof setInterval> | undefined
function updateFullscreen() {
  fullscreen.value = !!document.fullscreenElement
}
function toggleFullscreen() {
  if (document.fullscreenElement) void document.exitFullscreen()
  else void document.documentElement.requestFullscreen()
}
watch(
  [date, filter],
  () => {
    page.value = 0
    emit('refresh', date.value, filter.value)
  },
  { immediate: true },
)
watch(
  () => props.rows.length,
  () => {
    page.value = 0
  },
)
onMounted(() => {
  document.addEventListener('fullscreenchange', updateFullscreen)
  tick = setInterval(() => {
    clock.value = new Date()
  }, 1000)
  refreshTimer = setInterval(() => emit('refresh', date.value, filter.value), 60000)
  rotate = setInterval(() => {
    if (!hovered.value && count.value > 1) page.value = (page.value + 1) % count.value
  }, 10000)
})
onUnmounted(() => {
  document.removeEventListener('fullscreenchange', updateFullscreen)
  clearInterval(tick)
  clearInterval(refreshTimer)
  clearInterval(rotate)
})
</script>
<template>
  <section class="board" :class="{ standalone }" @mouseenter="hovered = true" @mouseleave="hovered = false">
    <header class="board-header">
      <div>
        <small>GZ MES · 车间看板</small>
        <h1>{{ title }}</h1>
      </div>
      <div class="controls">
        <strong>{{ clock.toLocaleString('zh-CN', { hour12: false }) }}</strong>
        <input v-model="date" type="date" aria-label="看板日期" />
        <input v-model="filter" :placeholder="filterLabel" :aria-label="filterLabel" />
        <button @click="toggleFullscreen">{{ fullscreen ? '退出全屏' : '全屏' }}</button>
      </div>
    </header>
    <div class="refresh">最后刷新：{{ refreshed || '—' }}</div>
    <div class="kpis">
      <article v-for="k in kpis" :key="k.label" class="kpi">
        <span>{{ k.label }}</span
        ><b>{{ k.value.toLocaleString('zh-CN', { maximumFractionDigits: 2 }) }}</b
        ><small>{{ k.unit }}</small>
      </article>
    </div>
    <div class="chart-panel">
      <h2>产能分布</h2>
      <EChart :option="chart" :height="190" />
    </div>
    <div class="table-panel">
      <div class="table-title">
        <h2>今日计划明细</h2>
        <span>第 {{ page + 1 }} / {{ count }} 页 · {{ rows.length }} 条</span>
      </div>
      <div class="table-scroll">
        <table>
          <slot name="head" />
          <tbody v-if="visible.length">
            <slot name="rows" :rows="visible" />
          </tbody>
        </table>
        <div v-if="!visible.length" class="empty">当日暂无已审核计划</div>
      </div>
    </div>
  </section>
</template>
<style scoped>
.board {
  --panel: #10243d;
  --line: #284460;
  --text: #e8f5ff;
  --muted: #93b2ce;
  min-height: calc(100vh - 110px);
  padding: 24px;
  color: var(--text);
  background: radial-gradient(circle at 50% -20%, #17426c, #071426 65%);
  font-variant-numeric: tabular-nums;
}
.board.standalone {
  position: fixed;
  inset: 0;
  z-index: 9999;
  overflow: auto;
  min-height: 100vh;
}
.board-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 20px;
  border-bottom: 1px solid var(--line);
  padding-bottom: 16px;
}
.board-header small,
.refresh {
  color: var(--muted);
}
h1 {
  font-size: 30px;
  color: var(--text);
  margin: 4px 0 0;
  letter-spacing: 3px;
}
h2 {
  font-size: 17px;
  color: var(--text);
  margin: 0 0 10px;
}
.controls {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.controls input,
.controls button {
  color: var(--text);
  background: var(--panel);
  border: 1px solid var(--line);
  border-radius: 5px;
  padding: 8px 10px;
}
.controls input {
  color-scheme: dark;
}
.refresh {
  text-align: right;
  margin: 8px 0 16px;
  font-size: 12px;
}
.kpis {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 14px;
}
.kpi,
.chart-panel,
.table-panel {
  background: var(--panel);
  border: 1px solid var(--line);
  border-radius: 8px;
  padding: 16px;
}
.kpi {
  border-top: 3px solid #25d4c8;
}
.kpi span {
  display: block;
  color: var(--muted);
}
.kpi b {
  font-size: 30px;
  margin: 5px 8px 0 0;
  display: inline-block;
}
.chart-panel,
.table-panel {
  margin-top: 16px;
}
.table-title {
  display: flex;
  justify-content: space-between;
  color: var(--muted);
}
.table-scroll {
  overflow-x: auto;
}
table {
  width: 100%;
  border-collapse: collapse;
  white-space: nowrap;
}
:deep(th),
:deep(td) {
  text-align: left;
  padding: 10px 8px;
  border-bottom: 1px solid var(--line);
}
:deep(th) {
  color: var(--muted);
  font-weight: 500;
}
:deep(tr.urgent td:first-child) {
  border-left: 4px solid #ff4d4f;
}
:deep(tr.due td) {
  background: #58332766;
}
:deep(.ant-progress-text) {
  color: var(--text);
}
.empty {
  text-align: center;
  padding: 55px;
  color: var(--muted);
}
@media (max-width: 900px) {
  .kpis {
    grid-template-columns: repeat(2, 1fr);
  }
  .board-header {
    flex-wrap: wrap;
  }
}
</style>
