<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { message } from 'ant-design-vue'
import { useRouter } from 'vue-router'
import dayjs from 'dayjs'
import type { EChartsOption } from 'echarts'
import { call, client, errorMessage, isDemo, type Schemas } from '@/api/client'
import { resetDemo } from '@/api/demo'
import PageCard from '@/components/PageCard.vue'
import KpiCard from '@/components/KpiCard.vue'
import StatusTag from '@/components/StatusTag.vue'
import CodeText from '@/components/CodeText.vue'
import EChart from '@/components/EChart.vue'

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
  { title: '批次号', dataIndex: 'lotNo', width: 160 },
  { title: '工程 / 客户', dataIndex: 'projectName', width: 180 },
  { title: '厚度 · 颜色', dataIndex: 'thickness', width: 115 },
  { title: '件数 / 面积', dataIndex: 'itemQty', width: 110 },
  { title: '当前工序', dataIndex: 'currentJobName', width: 105 },
  { title: '进度', dataIndex: 'progress', width: 125 },
  { title: '需求交期', dataIndex: 'requiredDate', width: 115 },
  { title: '紧急', dataIndex: 'urgency', width: 70 },
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
const kpis = computed(() => {
  const k = dashboard.value?.kpi
  if (!k) return []
  return [
    { title: '生产中批次', value: k.inProduction, unit: '批', sub: `待排产 ${k.pendingSchedule} · 今日完工 ${k.doneToday}`, icon: '◈' },
    { title: '今日报工', value: k.todayScanQty, unit: '件', sub: `${k.todayScanArea.toFixed(2)} m²`, icon: '▣' },
    {
      title: '近 14 天发货',
      value: k.deliveredQty,
      unit: '件',
      sub: `${k.deliveredArea.toFixed(2)} m²`,
      note: `未对账 ${k.unaccountedNotes} 单 · ${k.unaccountedArea.toFixed(2)} m²`,
      icon: '▤',
    },
    {
      title: '加急/逾期',
      value: k.urgentOpen,
      unit: '加急',
      sub: `逾期 ${k.overdueOpen}`,
      icon: '!',
      tone: k.overdueOpen > 0 ? ('error' as const) : undefined,
    },
  ]
})
const batchOption = computed<EChartsOption>(() => {
  const data = dashboard.value?.batchStatus.map((s) => ({ name: statusLabels[s.status], value: s.count })) ?? []
  return {
    color: colors,
    tooltip: { trigger: 'item' },
    legend: { bottom: 0, type: 'scroll' },
    title: {
      text: String(data.reduce((n, d) => n + d.value, 0)),
      subtext: '批次总数',
      left: 'center',
      top: '36%',
      textStyle: { fontSize: 22 },
      subtextStyle: { fontSize: 12 },
    },
    series: [{ type: 'pie', radius: ['52%', '72%'], center: ['50%', '43%'], label: { show: false }, data }],
  }
})
const customerOption = computed<EChartsOption>(() => ({
  color: colors,
  tooltip: { trigger: 'item', valueFormatter: (v) => `${Number(v).toFixed(2)} m²` },
  legend: { bottom: 0, type: 'scroll' },
  series: [
    {
      type: 'pie',
      radius: ['43%', '68%'],
      center: ['50%', '42%'],
      label: { show: false },
      data: dashboard.value?.customerArea.map((c) => ({ name: c.customerName, value: c.area })) ?? [],
    },
  ],
}))
const jobOption = computed<EChartsOption>(() => ({
  color: colors,
  grid: { left: 48, right: 12, top: 24, bottom: 70 },
  tooltip: {
    trigger: 'axis',
    formatter: (params: unknown) => {
      const p = (params as { dataIndex: number }[])[0]
      const job = dashboard.value?.jobLoad[p?.dataIndex ?? -1]
      return job ? `${job.jobName}<br/>${job.qty} 件 · ${job.area.toFixed(2)} m²` : ''
    },
  },
  xAxis: { type: 'category', data: dashboard.value?.jobLoad.map((j) => j.jobName) ?? [], axisLabel: { rotate: 40, interval: 0 } },
  yAxis: { type: 'value', name: '件' },
  series: [{ type: 'bar', data: dashboard.value?.jobLoad.map((j) => j.qty) ?? [], barMaxWidth: 24 }],
}))
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
        <a-col v-for="item in kpis" :key="item.title" :xs="24" :sm="12" :xl="6"><KpiCard v-bind="item" /></a-col>
      </a-row>
      <a-row :gutter="[16, 16]">
        <a-col :xs="24" :xl="8"
          ><PageCard title="批次状态分布"><EChart :option="batchOption" /></PageCard
        ></a-col>
        <a-col :xs="24" :xl="8"
          ><PageCard title="在制订单面积·按客户"><EChart :option="customerOption" /></PageCard
        ></a-col>
        <a-col :xs="24" :xl="8"
          ><PageCard title="各工序在制件数"><EChart :option="jobOption" /></PageCard
        ></a-col>
      </a-row>
      <PageCard title="近 14 天报工与发货趋势"><EChart :option="trendOption" :height="320" /></PageCard>
      <a-row :gutter="[16, 16]">
        <a-col :xs="24" :xl="16">
          <PageCard title="在制批次">
            <a-table
              :columns="columns"
              :data-source="dashboard.activeBatches"
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
                  >{{ record.thickness }} mm
                  <div class="muted">{{ record.color || '—' }}</div></template
                >
                <template v-else-if="column.dataIndex === 'itemQty'"
                  >{{ record.itemQty }} 件
                  <div class="muted">{{ record.itemArea.toFixed(2) }} m²</div></template
                >
                <template v-else-if="column.dataIndex === 'currentJobName'"
                  ><StatusTag :tone="record.currentSeq ? 'processing' : 'neutral'" :label="record.currentJobName || '未开工'"
                /></template>
                <template v-else-if="column.dataIndex === 'progress'"><a-progress :percent="record.progress" size="small" /></template>
                <template v-else-if="column.dataIndex === 'requiredDate'"
                  ><span :class="{ overdue: record.requiredDate && record.requiredDate < dayjs().format('YYYY-MM-DD') }">{{
                    record.requiredDate || '—'
                  }}</span></template
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
        <a-col :xs="24" :xl="8">
          <PageCard title="最近报工">
            <a-empty v-if="!dashboard.recentScans.length" description="暂无报工" />
            <div v-for="scan in dashboard.recentScans" :key="scan.id" class="scan-row">
              <span class="muted">{{ dayjs(scan.scannedAt).format('HH:mm') }}</span>
              <div>
                <CodeText :value="scan.barCode" />
                <div class="muted">{{ scan.jobName }} · {{ scan.scannedBy }}</div>
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
.scan-row > span {
  flex: 0 0 38px;
}
</style>
