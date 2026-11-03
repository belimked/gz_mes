<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import dayjs from 'dayjs'
import type { EChartsOption } from 'echarts'
import { message } from 'ant-design-vue'
import { DownloadOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import { call, client, errorMessage, type Schemas } from '@/api/client'
import PageCard from '@/components/PageCard.vue'
import TableToolbar from '@/components/TableToolbar.vue'
import EChart from '@/components/EChart.vue'
import { downloadCsv, exportFilename } from '@/utils/export'

const props = defineProps<{ mode: 'day' | 'month' }>()
type Row = Schemas['WorkshopOutputRow']
const current = () => dayjs()
const period = ref('single')
const selected = ref(current().format(props.mode === 'day' ? 'YYYY-MM' : 'YYYY'))
const dates = ref<string[]>([])
const factoryId = ref<number>()
const workshop = ref<string>()
const factories = ref<Schemas['Factory'][]>([])
const data = ref<Schemas['WorkshopOutputReport']>()
const loading = ref(false)
const exporting = ref(false)
const filter = reactive({ from: '', to: '', factoryId: undefined as number | undefined, workshop: undefined as string | undefined })
const month = computed(() => props.mode === 'month')
function range(): [string, string] {
  if (period.value === 'range' && dates.value.length === 2) return [dates.value[0]!, dates.value[1]!]
  const start = dayjs(selected.value, month.value ? 'YYYY' : 'YYYY-MM')
  return month.value
    ? [start.startOf('year').format('YYYY-MM-DD'), start.endOf('year').format('YYYY-MM-DD')]
    : [start.startOf('month').format('YYYY-MM-DD'), start.endOf('month').format('YYYY-MM-DD')]
}
async function load() {
  loading.value = true
  try {
    data.value = await call(client.GET('/reports/workshop-output', { params: { query: { granularity: props.mode, ...filter } } }))
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    loading.value = false
  }
}
function search() {
  const [from, to] = range()
  Object.assign(filter, { from, to, factoryId: factoryId.value, workshop: workshop.value })
  void load()
}
function quick(previous: boolean) {
  period.value = 'single'
  selected.value = current()
    .subtract(previous ? 1 : 0, month.value ? 'year' : 'month')
    .format(month.value ? 'YYYY' : 'YYYY-MM')
  search()
}
watch(
  () => props.mode,
  () => {
    selected.value = current().format(month.value ? 'YYYY' : 'YYYY-MM')
    period.value = 'single'
    search()
  },
)
const groups = [
  {
    title: '录批次',
    fields: [
      ['单数', 'batchCount'],
      ['面积', 'batchArea'],
    ],
  },
  {
    title: '排产',
    fields: [
      ['批次', 'scheduledCount'],
      ['面积', 'scheduledArea'],
    ],
  },
  {
    title: '钣金',
    fields: [
      ['件数', 'sheetQty'],
      ['面积', 'sheetArea'],
    ],
  },
  {
    title: '喷涂',
    fields: [
      ['件数', 'sprayQty'],
      ['面积', 'sprayArea'],
    ],
  },
  {
    title: '发货',
    fields: [
      ['数量', 'deliveredQty'],
      ['面积', 'deliveredArea'],
    ],
  },
] as const
const columns = computed(() => [
  { title: '序号', dataIndex: 'index', width: 65, fixed: 'left' as const },
  { title: month.value ? '月份' : '日期', dataIndex: 'period', width: 120, fixed: 'left' as const },
  ...groups.map((g) => ({ title: g.title, children: g.fields.map(([title, dataIndex]) => ({ title, dataIndex, width: 105 })) })),
  {
    title: '钣金面积分布',
    children: [
      { title: '本厂', dataIndex: 'sheetOwnArea', width: 105 },
      { title: '外协', dataIndex: 'sheetOutsourceArea', width: 105 },
      ...(data.value?.workshops ?? []).map((name, index) => ({ title: name, dataIndex: `workshop-${index}`, width: 120 })),
    ],
  },
])
const flat = computed(() => columns.value.flatMap((c) => ('children' in c ? c.children! : [c])))
function value(row: Row, key: string): string | number {
  return key.startsWith('workshop-') ? (row.workshopSheetArea[Number(key.slice(9))] ?? 0) : (row[key as keyof Row] as string | number)
}
function display(row: Row, key: string) {
  const v = value(row, key)
  return !v
    ? '—'
    : key === 'period' || key === 'index' || ['batchCount', 'scheduledCount', 'sheetQty', 'sprayQty', 'deliveredQty'].includes(key)
      ? v
      : Number(v).toFixed(2)
}
const chart = computed<EChartsOption>(() => ({
  tooltip: { trigger: 'axis' },
  legend: { data: ['钣金面积', '喷涂面积', '发货面积'] },
  xAxis: { type: 'category', data: data.value?.periods.map((r) => r.period) ?? [] },
  yAxis: { type: 'value', name: 'm²' },
  series: [
    { name: '钣金面积', type: 'bar', data: data.value?.periods.map((r) => r.sheetArea) ?? [] },
    { name: '喷涂面积', type: 'bar', data: data.value?.periods.map((r) => r.sprayArea) ?? [] },
    { name: '发货面积', type: 'line', data: data.value?.periods.map((r) => r.deliveredArea) ?? [] },
  ],
}))
function exportCsv(comparison = false) {
  if (!data.value) return
  exporting.value = true
  try {
    if (comparison)
      downloadCsv(
        exportFilename('钣金车间环比'),
        [
          { title: '车间', value: (r: Schemas['WorkshopComparison']) => r.workshop },
          ...(['prevPrevArea', 'prevArea', 'currentArea', 'chainRate'] as const).map((key) => ({
            title: key,
            value: (r: Schemas['WorkshopComparison']) => r[key] ?? '',
          })),
        ],
        data.value.comparison ?? [],
      )
    else
      downloadCsv(
        exportFilename(month.value ? '车间产量月报' : '车间产量日报', [filter.from, filter.to]),
        flat.value.map((c) => ({ title: c.title, value: (r: Row) => value(r, c.dataIndex) })),
        [...data.value.periods, data.value.total],
      )
  } finally {
    exporting.value = false
  }
}
onMounted(async () => {
  search()
  try {
    factories.value = await call(client.GET('/factories'))
  } catch (e) {
    message.error(errorMessage(e))
  }
})
</script>
<template>
  <div class="report">
    <div class="heading">
      <h1>{{ month ? '车间产量月报' : '车间产量日报' }}</h1>
      <span>「制图」阶段暂无数据，未统计。录批次按创建日，排产按首次有效审核日，钣金 / 喷涂按工序 20 / 40 扫码日，发货按发货日统计。</span>
    </div>
    <PageCard
      ><div class="filters">
        <label
          >查询方式<a-radio-group v-model:value="period" button-style="solid"
            ><a-radio-button value="single">{{ month ? '年份' : '月份' }}</a-radio-button
            ><a-radio-button value="range">范围</a-radio-button></a-radio-group
          ></label
        >
        <label v-if="period === 'single'"
          >{{ month ? '年份' : '月份'
          }}<a-date-picker v-model:value="selected" :picker="month ? 'year' : 'month'" :value-format="month ? 'YYYY' : 'YYYY-MM'"
        /></label>
        <label v-else>日期范围<a-range-picker v-model:value="dates" :picker="month ? 'month' : 'date'" value-format="YYYY-MM-DD" /></label>
        <label
          >厂区<a-select
            v-model:value="factoryId"
            allow-clear
            placeholder="全部"
            :options="factories.filter((f) => f.kind === 'own').map((f) => ({ label: f.name, value: f.id }))"
        /></label>
        <label
          >钣金车间<a-select
            v-model:value="workshop"
            allow-clear
            placeholder="全部"
            :options="(data?.workshops ?? []).map((w) => ({ label: w, value: w }))"
        /></label>
        <div class="actions">
          <a-button @click="quick(false)">{{ month ? '今年' : '本月' }}</a-button
          ><a-button @click="quick(true)">{{ month ? '去年' : '上月' }}</a-button
          ><a-button type="primary" @click="search">查询</a-button>
        </div>
      </div></PageCard
    >
    <PageCard><EChart :option="chart" :height="280" data-testid="workshop-chart" /></PageCard>
    <PageCard>
      <TableToolbar
        ><template #left
          ><span data-testid="workshop-total"
            >合计：钣金 {{ data?.total.sheetQty ?? 0 }} 件 / {{ (data?.total.sheetArea ?? 0).toFixed(2) }} m²，喷涂
            {{ (data?.total.sprayArea ?? 0).toFixed(2) }} m²</span
          ></template
        ><template #right
          ><a-button :loading="exporting" @click="exportCsv()"><DownloadOutlined />导出</a-button
          ><a-button aria-label="刷新" @click="load"><ReloadOutlined /></a-button></template
      ></TableToolbar>
      <a-table
        data-testid="workshop-table"
        :row-key="(r: Row) => r.period"
        :columns="columns"
        :data-source="data?.periods ?? []"
        :loading="loading"
        :pagination="false"
        :scroll="{ x: 1600, y: 550 }"
        bordered
        size="middle"
      >
        <template #emptyText><a-empty description="该时段没有产量" /></template>
        <template #bodyCell="{ column, record, index }"
          ><span :class="{ zero: !value(record, column.dataIndex) && column.dataIndex !== 'index' }">{{
            column.dataIndex === 'index' ? index + 1 : display(record, column.dataIndex)
          }}</span></template
        >
        <template #summary
          ><a-table-summary fixed
            ><a-table-summary-row
              ><a-table-summary-cell v-for="(c, i) in flat" :key="c.dataIndex" :index="i"
                ><b v-if="i === 0">合计</b
                ><span v-else-if="i > 1" :class="{ zero: !value(data!.total, c.dataIndex) }">{{
                  display(data!.total, c.dataIndex)
                }}</span></a-table-summary-cell
              ></a-table-summary-row
            ></a-table-summary
          ></template
        >
      </a-table>
    </PageCard>
    <PageCard v-if="month"
      ><TableToolbar
        ><template #left><b>钣金车间环比</b></template
        ><template #right
          ><a-button @click="exportCsv(true)"><DownloadOutlined />导出环比</a-button></template
        ></TableToolbar
      >
      <a-table
        data-testid="workshop-comparison"
        :row-key="(r: Schemas['WorkshopComparison']) => r.workshop"
        :data-source="data?.comparison ?? []"
        :pagination="false"
        bordered
        size="middle"
        :columns="[
          { title: '车间', dataIndex: 'workshop' },
          { title: '前月', dataIndex: 'prevPrevArea' },
          { title: '上月', dataIndex: 'prevArea' },
          { title: '本月', dataIndex: 'currentArea' },
          { title: '环比%', dataIndex: 'chainRate' },
        ]"
      >
        <template #emptyText><a-empty description="暂无环比数据" /></template>
        <template #bodyCell="{ column, record }"
          ><template v-if="column.dataIndex === 'chainRate'"
            ><span :class="record.chainRate > 0 ? 'positive' : record.chainRate < 0 ? 'negative' : ''">{{
              record.chainRate == null ? '—' : `${record.chainRate.toFixed(2)}%`
            }}</span></template
          ><template v-else-if="column.dataIndex !== 'workshop'">{{ record[column.dataIndex].toFixed(2) }}</template></template
        >
      </a-table>
    </PageCard>
  </div>
</template>
<style scoped>
.report {
  display: grid;
  gap: 16px;
}
.heading h1 {
  margin: 0 0 4px;
  font-size: 20px;
}
.heading span,
.zero {
  color: var(--gz-muted);
  font-size: 12px;
}
.filters {
  display: flex;
  flex-wrap: wrap;
  align-items: end;
  gap: 12px;
}
.filters label {
  display: grid;
  gap: 6px;
  color: var(--gz-secondary);
  font-size: 12px;
  min-width: 150px;
}
.actions {
  display: flex;
  gap: 8px;
  margin-left: auto;
}
.positive {
  color: green;
}
.negative {
  color: red;
}
</style>
