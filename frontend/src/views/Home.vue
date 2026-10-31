<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { message } from 'ant-design-vue'
import { useRouter } from 'vue-router'
import dayjs from 'dayjs'
import type { EChartsOption } from 'echarts'
import { call, client, errorMessage, isDemo, type Schemas } from '@/api/client'
import { resetDemo } from '@/api/demo'
import PageCard from '@/components/PageCard.vue'
import SparkKpiCard from '@/components/SparkKpiCard.vue'
import StatusTag from '@/components/StatusTag.vue'
import CodeText from '@/components/CodeText.vue'
import EChart from '@/components/EChart.vue'
import ProcessPipeline from '@/components/ProcessPipeline.vue'
import { statusTone, type Tone } from '@/theme'
import { fmtH } from '@/utils/time'

const router = useRouter()
const dashboard = ref<Schemas['Dashboard']>()
const loading = ref(true)
const colors = ['#1677ff', '#52c41a', '#faad14', '#ff4d4f', '#722ed1', '#13c2c2', '#eb2f96', '#8c8c8c']
const statusLabels: Record<Schemas['BatchStatus'], string> = {
  pending: '待排产',
  scheduled: '已排产',
  in_production: '生产中',
  done: '已完工',
}
const columns = [
  { title: '批次号', dataIndex: 'lotNo', width: 150, fixed: 'left' as const },
  { title: '工程 / 客户', dataIndex: 'projectName', width: 180, ellipsis: true },
  { title: '规格 / 件数', dataIndex: 'thickness', width: 200 },
  { title: '当前工序', dataIndex: 'currentJobName', width: 105 },
  { title: '进度', dataIndex: 'progress', width: 120 },
  { title: '需求交期', dataIndex: 'requiredDate', width: 100 },
  { title: '紧急度', dataIndex: 'urgency', width: 70 },
]
async function load() {
  loading.value = true
  try {
    dashboard.value = await call(client.GET('/dashboard', { params: { query: { days: 14 } } }))
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    loading.value = false
  }
}
function reset() {
  resetDemo()
  message.success('演示数据已重置')
  void load()
}
function openBatch(lotNo: string) {
  void router.push({ path: '/scheduling/batches', query: { keyword: lotNo } })
}
function sum(values: number[]) {
  return values.reduce((total, value) => total + value, 0)
}
function pct(recent: number, prev: number): number | undefined {
  return prev === 0 ? undefined : ((recent - prev) / prev) * 100
}
const kpis = computed(() => {
  const data = dashboard.value
  if (!data) return []
  const k = data.kpi
  const trend = data.scanTrend
  const active = trend.map((day) => day.activeBatches)
  const qty = trend.map((day) => day.qty)
  const delivered = trend.map((day) => day.deliveredQty)
  const done = trend.map((day) => day.doneBatches)
  return [
    {
      title: '生产中批次',
      icon: '◈',
      color: '#1677ff',
      value: String(k.inProduction),
      unit: '批',
      series: active,
      delta: pct(sum(active.slice(-7)) / 7, sum(active.slice(-14, -7)) / 7),
      footer: [
        { label: '待排产', value: String(k.pendingSchedule) },
        { label: '今日完工', value: String(k.doneToday) },
      ],
    },
    {
      title: '今日报工',
      icon: '▣',
      color: '#52c41a',
      value: String(k.todayScanQty),
      unit: '件',
      series: qty,
      delta: qty.length >= 2 ? pct(qty[qty.length - 1]!, qty[qty.length - 2]!) : undefined,
      footer: [
        { label: '面积', value: `${k.todayScanArea.toFixed(1)} m²` },
        { label: '14 天合计', value: `${sum(qty)} 件` },
      ],
    },
    {
      title: '近 14 天发货',
      icon: '▤',
      color: '#fa8c16',
      value: String(k.deliveredQty),
      unit: '件',
      series: delivered,
      delta: pct(sum(delivered.slice(-7)), sum(delivered.slice(-14, -7))),
      footer: [
        { label: '面积', value: `${k.deliveredArea.toFixed(1)} m²` },
        { label: '未对账', value: `${k.unaccountedNotes} 单` },
      ],
    },
    {
      title: '加急/逾期',
      icon: '!',
      color: '#ff4d4f',
      value: String(k.urgentOpen),
      unit: '加急',
      series: done,
      footer: [
        { label: '逾期', value: String(k.overdueOpen) },
        { label: '14 天完工', value: `${sum(done)} 批` },
      ],
    },
  ]
})
const batchStatuses: Schemas['BatchStatus'][] = ['pending', 'scheduled', 'in_production', 'done']
const statusColors: Record<Schemas['BatchStatus'], string> = {
  pending: '#1677ff',
  scheduled: '#52c41a',
  in_production: '#faad14',
  done: '#ff4d4f',
}
const statusCounts = computed(() =>
  batchStatuses.map((status) => ({
    status,
    name: statusLabels[status],
    count: dashboard.value?.batchStatus.find((item) => item.status === status)?.count ?? 0,
  })),
)
const batchOption = computed<EChartsOption>(() => {
  const data = statusCounts.value.map((s) => ({ name: s.name, value: s.count }))
  return {
    color: colors,
    tooltip: { trigger: 'item' },
    title: {
      text: String(data.reduce((n, d) => n + d.value, 0)),
      subtext: '批次总数',
      left: 'center',
      top: '36%',
      textStyle: { fontSize: 22 },
      subtextStyle: { fontSize: 12 },
    },
    series: [{ type: 'pie', radius: ['52%', '72%'], center: ['50%', '50%'], label: { show: false }, data }],
  }
})
const customerTotal = computed(() => sum((dashboard.value?.customerArea ?? []).map((c) => c.area)))
const topCustomers = computed(() =>
  (dashboard.value?.customerArea ?? [])
    .filter((c) => c.customerId != null)
    .sort((a, b) => b.area - a.area)
    .slice(0, 3),
)
const customerOption = computed<EChartsOption>(() => ({
  color: colors,
  tooltip: { trigger: 'item', valueFormatter: (v) => `${Number(v).toFixed(2)} m²` },
  title: {
    text: `${customerTotal.value.toFixed(1)} m²`,
    subtext: '总在制',
    left: 'center',
    top: '38%',
    textStyle: { fontSize: 17 },
    subtextStyle: { fontSize: 11 },
  },
  series: [
    {
      type: 'pie',
      radius: ['48%', '72%'],
      center: ['50%', '50%'],
      label: { show: false },
      data: dashboard.value?.customerArea.map((c) => ({ name: c.customerName, value: c.area })) ?? [],
    },
  ],
}))
const pipelineLegend: { tone: Tone; label: string }[] = [
  { tone: 'processing', label: '在制' },
  { tone: 'warning', label: '堆积' },
  { tone: 'success', label: '待对账' },
]
const sortedJobs = computed(() => [...(dashboard.value?.jobLoad ?? [])].sort((a, b) => a.seq - b.seq))
const cycleSummary = computed(() => {
  const jobs = sortedJobs.value
  const bottleneck = jobs.reduce<Schemas['JobLoad'] | undefined>(
    (best, job) => (job.cycleHours != null && (best?.cycleHours == null || job.cycleHours > best.cycleHours) ? job : best),
    undefined,
  )
  const longestDwell = jobs.reduce<Schemas['JobLoad'] | undefined>(
    (best, job) => (job.dwellMaxHours != null && (best?.dwellMaxHours == null || job.dwellMaxHours > best.dwellMaxHours) ? job : best),
    undefined,
  )
  return { total: sum(jobs.map((job) => job.cycleHours ?? 0)), bottleneck, longestDwell }
})
const topCycleJobs = computed(() =>
  sortedJobs.value
    .filter((job) => job.cycleHours != null)
    .sort((a, b) => (b.cycleHours ?? 0) - (a.cycleHours ?? 0))
    .slice(0, 5),
)
const cycleOption = computed<EChartsOption>(() => ({
  grid: { left: 90, right: 60, top: 8, bottom: 26 },
  xAxis: {
    type: 'value',
    name: '单位：小时',
    nameLocation: 'end',
    nameTextStyle: { fontSize: 10 },
    axisLabel: { fontSize: 10 },
    splitLine: { show: false },
  },
  yAxis: {
    type: 'category',
    inverse: true,
    data: topCycleJobs.value.map((job) => job.jobName),
    axisTick: { show: false },
    axisLine: { show: false },
    axisLabel: { fontSize: 10 },
  },
  tooltip: {
    trigger: 'item',
    formatter: (param) => {
      const job = topCycleJobs.value[Array.isArray(param) ? (param[0]?.dataIndex ?? -1) : param.dataIndex]
      if (!job) return ''
      return `${job.jobName}<br/>周期：${job.cycleHours == null ? '—' : fmtH(job.cycleHours)}<br/>P90：${job.cycleP90Hours == null ? '—' : fmtH(job.cycleP90Hours)}<br/>样本数：${job.cycleSamples} 件`
    },
  },
  series: [
    {
      type: 'bar',
      barMaxWidth: 10,
      data: topCycleJobs.value.map((job) => ({
        value: job.cycleHours ?? 0,
        itemStyle: { color: job === cycleSummary.value.bottleneck ? '#faad14' : '#1677ff' },
      })),
      label: { show: true, position: 'right', fontSize: 10, formatter: (param) => fmtH(Number(param.value)) },
    },
  ],
}))
type BatchFilter = 'all' | 'urgent' | 'overdue'
const batchFilter = ref<BatchFilter>('all')
const today = dayjs().format('YYYY-MM-DD')
function isOverdue(batch: Schemas['ActiveBatch']) {
  return !!batch.requiredDate && batch.requiredDate < today
}
const activeBatches = computed(() => dashboard.value?.activeBatches ?? [])
const urgentBatches = computed(() => activeBatches.value.filter((batch) => batch.urgency === 'urgent'))
const overdueBatches = computed(() => activeBatches.value.filter(isOverdue))
const filteredBatches = computed(() =>
  batchFilter.value === 'urgent' ? urgentBatches.value : batchFilter.value === 'overdue' ? overdueBatches.value : activeBatches.value,
)
const recentActivity = computed(() => dashboard.value?.recentActivity ?? [])
const trendOption = computed<EChartsOption>(() => ({
  color: colors,
  tooltip: { trigger: 'axis' },
  legend: { top: 0 },
  grid: { left: 50, right: 55, top: 48, bottom: 30 },
  xAxis: { type: 'category', boundaryGap: true, data: dashboard.value?.scanTrend.map((d) => d.date.slice(5)) ?? [] },
  yAxis: [
    { type: 'value', name: '件' },
    { type: 'value', name: 'm²' },
  ],
  series: [
    {
      name: '报工件数',
      type: 'line',
      smooth: true,
      data: dashboard.value?.scanTrend.map((d) => d.qty) ?? [],
      areaStyle: {
        color: {
          type: 'linear',
          x: 0,
          y: 0,
          x2: 0,
          y2: 1,
          colorStops: [
            { offset: 0, color: 'rgba(22,119,255,0.35)' },
            { offset: 1, color: 'rgba(22,119,255,0)' },
          ],
        },
      },
    },
    { name: '发货件数', type: 'line', smooth: true, data: dashboard.value?.scanTrend.map((d) => d.deliveredQty) ?? [] },
    { name: '报工面积', type: 'bar', yAxisIndex: 1, data: dashboard.value?.scanTrend.map((d) => d.area) ?? [], barMaxWidth: 18 },
  ],
}))
onMounted(() => {
  void load()
})
</script>
<template>
  <div class="home">
    <PageCard class="title-card">
      <div class="title-row">
        <div>
          <h1><span class="dot" />铝单板生产执行看板</h1>
          <StatusTag v-if="dashboard" tone="processing" :label="`生产中 ${dashboard.kpi.inProduction} 个批次`" />
          <span v-if="dashboard" class="muted updated">更新于 {{ dayjs(dashboard.generatedAt).format('YYYY-MM-DD HH:mm:ss') }}</span>
        </div>
        <div class="buttons">
          <a-button type="primary" @click="router.push('/production/scan')">一键扫码报工</a-button>
          <a-button @click="router.push('/scheduling/batches')">新建生产批次</a-button>
          <a-button :loading="loading" @click="load">刷新</a-button>
          <a-button v-if="isDemo" size="small" @click="reset">重置演示数据</a-button>
        </div>
      </div>
    </PageCard>
    <a-skeleton v-if="loading && !dashboard" active :paragraph="{ rows: 12 }" />
    <template v-else-if="dashboard">
      <a-row :gutter="[16, 16]">
        <a-col v-for="item in kpis" :key="item.title" :xs="24" :sm="12" :xl="6"><SparkKpiCard v-bind="item" /></a-col>
      </a-row>
      <PageCard title="铝单板 11 大工序流转全景看板">
        <template #extra>
          <div class="pipeline-legend">
            <span v-for="item in pipelineLegend" :key="item.tone" class="legend-item">
              <i class="legend-dot" :style="{ background: statusTone[item.tone].text }" />{{ item.label }}
            </span>
          </div>
        </template>
        <ProcessPipeline :items="dashboard.jobLoad" />
        <div class="cycle-summary">
          <a-row :gutter="[16, 12]" align="middle">
            <a-col :xs="24" :xl="8" class="cycle-facts">
              <div>
                全流程平均周期 ≈ <strong>{{ fmtH(cycleSummary.total) }}</strong>
              </div>
              <div>
                瓶颈工序：<strong class="bottleneck"
                  >{{ cycleSummary.bottleneck?.jobName ?? '—' }}（ø
                  {{ cycleSummary.bottleneck?.cycleHours == null ? '—' : fmtH(cycleSummary.bottleneck.cycleHours) }}）</strong
                >
              </div>
              <div>
                最长停留：<strong
                  >{{ cycleSummary.longestDwell?.jobName ?? '—' }}（{{
                    cycleSummary.longestDwell?.dwellMaxHours == null ? '—' : fmtH(cycleSummary.longestDwell.dwellMaxHours)
                  }}）</strong
                >
              </div>
            </a-col>
            <a-col :xs="24" :xl="16">
              <div class="cycle-chart-title">各工序平均周期（近 14 天）</div>
              <EChart :option="cycleOption" :height="150" />
            </a-col>
          </a-row>
        </div>
      </PageCard>
      <a-row :gutter="[16, 16]">
        <a-col :xs="24" :xl="16"
          ><PageCard title="近 14 天报工与发货趋势"><EChart :option="trendOption" :height="300" /></PageCard
        ></a-col>
        <a-col :xs="24" :xl="8"
          ><PageCard title="批次状态分布"
            ><EChart :option="batchOption" :height="220" />
            <div class="status-grid">
              <div v-for="item in statusCounts" :key="item.status" class="status-cell">
                <span><i class="legend-dot" :style="{ background: statusColors[item.status] }" />{{ item.name }}</span>
                <strong>{{ item.count }}</strong>
              </div>
            </div></PageCard
          ></a-col
        >
      </a-row>
      <a-row :gutter="[16, 16]">
        <a-col :xs="24" :xl="16">
          <PageCard title="在制批次">
            <template #extra>
              <a-radio-group v-model:value="batchFilter" size="small" button-style="solid">
                <a-radio-button value="all">全部 ({{ activeBatches.length }})</a-radio-button>
                <a-radio-button value="urgent">加急 ({{ urgentBatches.length }})</a-radio-button>
                <a-radio-button value="overdue">逾期 ({{ overdueBatches.length }})</a-radio-button>
              </a-radio-group>
            </template>
            <a-table
              :columns="columns"
              :data-source="filteredBatches"
              :row-class-name="(record: Schemas['ActiveBatch']) => (isOverdue(record) ? 'overdue-row' : '')"
              row-key="id"
              size="small"
              :pagination="false"
              :scroll="{ x: 980 }"
            >
              <template #bodyCell="{ column, record }">
                <template v-if="column.dataIndex === 'lotNo'"
                  ><a @click="openBatch(record.lotNo)"><CodeText :value="record.lotNo" /></a
                ></template>
                <template v-else-if="column.dataIndex === 'projectName'"
                  ><strong>{{ record.projectName }}</strong>
                  <div class="muted">{{ record.customerName }}</div></template
                >
                <template v-else-if="column.dataIndex === 'thickness'"
                  >{{ record.thickness }} mm · {{ record.color || '—' }}
                  <div class="muted">{{ record.itemQty }} 件 · {{ record.itemArea.toFixed(2) }} m²</div></template
                >
                <template v-else-if="column.dataIndex === 'currentJobName'"
                  ><StatusTag :tone="record.currentSeq ? 'processing' : 'neutral'" :label="record.currentJobName || '未开工'"
                /></template>
                <template v-else-if="column.dataIndex === 'progress'"
                  ><div class="progress-cell">
                    <a-progress :percent="record.progress" size="small" :show-info="false" /><span>{{ record.progress }}%</span>
                  </div></template
                >
                <template v-else-if="column.dataIndex === 'requiredDate'"
                  ><span :class="{ overdue: isOverdue(record) }">{{ record.requiredDate || '—' }}</span></template
                >
                <template v-else-if="column.dataIndex === 'urgency'"
                  ><StatusTag
                    :tone="record.urgency === 'urgent' ? 'error' : 'neutral'"
                    :label="record.urgency === 'urgent' ? '加急' : '普通'"
                /></template>
              </template>
            </a-table>
          </PageCard>
        </a-col>
        <a-col :xs="24" :xl="8" class="side-cards">
          <PageCard title="在制订单面积·按客户">
            <EChart :option="customerOption" :height="200" />
            <div class="customer-list">
              <div v-for="(customer, index) in topCustomers" :key="customer.customerId ?? customer.customerName" class="customer-row">
                <span><i class="legend-dot" :style="{ background: colors[index] }" />{{ customer.customerName }}</span>
                <strong
                  >{{ customer.area.toFixed(1) }} m² ({{ customerTotal ? Math.round((customer.area / customerTotal) * 100) : 0 }}%)</strong
                >
              </div>
            </div>
          </PageCard>
          <PageCard title="最近报工动态">
            <a-empty v-if="!recentActivity.length" description="暂无报工" />
            <div
              v-for="scan in recentActivity"
              :key="`${scan.batchId}-${scan.jobId}-${scan.scannedBy}-${scan.lastScannedAt}`"
              class="scan-row"
            >
              <div class="scan-detail">
                <CodeText :value="scan.lotNo" />
                <div class="muted">{{ scan.jobName }} · {{ scan.scannedBy }}</div>
              </div>
              <div class="scan-qty">
                <strong>+{{ scan.qty }} 件</strong><span class="muted">{{ dayjs(scan.lastScannedAt).format('HH:mm') }}</span>
              </div>
            </div>
          </PageCard>
        </a-col>
      </a-row>
    </template>
  </div>
</template>
<style scoped>
.home {
  display: grid;
  gap: 16px;
}
.title-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}
.title-row h1 {
  display: inline-block;
  font-size: 20px;
  margin: 0 14px 0 0;
}
.dot {
  display: inline-block;
  width: 9px;
  height: 9px;
  margin-right: 10px;
  border-radius: 50%;
  background: var(--gz-primary);
}
.buttons {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  align-items: center;
}
.muted {
  color: var(--gz-muted);
  font-size: 12px;
}
.updated {
  margin-left: 10px;
}
.overdue {
  color: #ff4d4f;
  font-weight: 600;
}
.pipeline-legend {
  display: flex;
  gap: 14px;
  flex-wrap: wrap;
  font-size: 12px;
  font-weight: 400;
  color: var(--gz-secondary);
}
.legend-item {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}
.legend-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}
.cycle-summary {
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px solid var(--gz-border);
}
.cycle-facts {
  display: grid;
  gap: 10px;
  font-size: 12px;
  color: var(--gz-secondary);
}
.cycle-facts strong {
  color: var(--gz-primary);
}
.cycle-facts .bottleneck {
  color: #faad14;
}
.cycle-chart-title {
  font-size: 11px;
  color: var(--gz-secondary);
}
.side-cards {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.scan-row {
  display: flex;
  gap: 12px;
  padding: 10px 0;
  border-bottom: 1px solid var(--gz-border);
  overflow-wrap: anywhere;
}
.scan-row:last-child {
  border-bottom: 0;
}
.status-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 8px;
  margin-top: 8px;
}
.status-cell {
  background: var(--gz-fill, #f5f7fa);
  border-radius: var(--gz-radius);
  padding: 7px 10px;
  display: flex;
  justify-content: space-between;
  gap: 8px;
  font-size: 12px;
}
.status-cell span,
.customer-row span {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.status-cell strong {
  font-weight: 700;
}
.customer-list {
  display: grid;
  gap: 8px;
  margin-top: 8px;
}
.customer-row {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  font-size: 12px;
}
.customer-row span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.customer-row strong {
  white-space: nowrap;
  font-weight: 500;
}
.progress-cell {
  display: flex;
  align-items: center;
  gap: 4px;
  white-space: nowrap;
}
.progress-cell :deep(.ant-progress) {
  flex: 1;
  min-width: 40px;
}
:deep(.overdue-row > td) {
  background: #fff1f0 !important;
}
.scan-detail {
  min-width: 0;
  flex: 1;
}
.scan-qty {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  white-space: nowrap;
}
.scan-qty strong {
  color: var(--gz-primary);
}
</style>
