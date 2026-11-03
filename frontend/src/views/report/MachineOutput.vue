<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import type { TablePaginationConfig } from 'ant-design-vue'
import { DownloadOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import dayjs from 'dayjs'
import { call, client, errorMessage, type Schemas } from '@/api/client'
import PageCard from '@/components/PageCard.vue'
import TableToolbar from '@/components/TableToolbar.vue'
import CodeText from '@/components/CodeText.vue'
import { downloadCsv, exportFilename, fetchAllPages } from '@/utils/export'

type Row = Schemas['MachineOutputRow']
type Period = 'today' | 'yesterday' | 'week' | 'month' | 'custom'
const router = useRouter()
function periodRange(period: Exclude<Period, 'custom'>): [string, string] {
  const today = dayjs().startOf('day')
  const start = period === 'yesterday' ? 1 : period === 'week' ? 6 : period === 'month' ? 29 : 0
  return [today.subtract(start, 'day').format('YYYY-MM-DD'), today.subtract(period === 'yesterday' ? 1 : 0, 'day').format('YYYY-MM-DD')]
}
function emptyFilters() {
  return {
    period: 'today' as Period,
    dates: periodRange('today') as string[],
    factoryId: undefined as number | undefined,
    jobId: undefined as number | undefined,
    machineId: undefined as number | undefined,
    keyword: '',
    mode: 'summary' as 'summary' | 'detail',
  }
}
const filters = reactive(emptyFilters())
const applied = reactive(emptyFilters())
const range = computed<[string, string]>(() =>
  applied.period === 'custom' && applied.dates.length === 2
    ? [applied.dates[0]!, applied.dates[1]!]
    : periodRange(applied.period === 'custom' ? 'today' : applied.period),
)
const factories = ref<Schemas['Factory'][]>([])
const jobs = ref<Schemas['Job'][]>([])
const machines = ref<Schemas['Machine'][]>([])
const rows = ref<Row[]>([])
const summary = ref<Schemas['MachineOutputReport']['summary']>()
const pager = reactive({ current: 1, pageSize: 20, total: 0 })
const loading = ref(false)
const exporting = ref(false)
const factoryOptions = computed(() => factories.value.filter((f) => f.kind === 'own').map((f) => ({ value: f.id, label: f.name })))
const jobOptions = computed(() => jobs.value.map((j) => ({ value: j.id, label: j.name })))
const machineOptions = computed(() =>
  machines.value
    .filter((m) => (!filters.factoryId || m.factoryId === filters.factoryId) && (!filters.jobId || m.jobId === filters.jobId))
    .map((m) => ({ value: m.id, label: `${m.code} ${m.name}` })),
)
const columns = computed(() => [
  { title: '机台编码', dataIndex: 'code', width: 130 },
  { title: '名称', dataIndex: 'name', width: 150 },
  ...(applied.mode === 'summary' ? [{ title: '厂区', dataIndex: 'factoryName', width: 120 }] : []),
  { title: '工序', dataIndex: 'jobName', width: 120 },
  ...(applied.mode === 'detail'
    ? [
        { title: '批次', dataIndex: 'lotNo', width: 150 },
        { title: '工程', dataIndex: 'projectName', width: 180 },
      ]
    : []),
  { title: '数量', dataIndex: 'qty', width: 85 },
  { title: '展开面积 m²', dataIndex: 'pieceArea', width: 120 },
  { title: '喷涂面积 m²', dataIndex: 'sprayingArea', width: 120 },
  ...(applied.mode === 'summary'
    ? [
        { title: '产能数量负载', dataIndex: 'qtyLoad', width: 135 },
        { title: '产能面积负载', dataIndex: 'areaLoad', width: 135 },
        { title: '满载率', dataIndex: 'loadFactor', width: 150 },
      ]
    : []),
])
function fetchPage(page: number, size: number) {
  return call(
    client.GET('/reports/machine-output', {
      params: {
        query: {
          from: range.value[0],
          to: range.value[1],
          factoryId: applied.factoryId,
          machineId: applied.machineId,
          jobId: applied.jobId,
          keyword: applied.keyword.trim() || undefined,
          mode: applied.mode,
          page,
          size,
        },
      },
    }),
  )
}
async function load() {
  loading.value = true
  try {
    const result = await fetchPage(pager.current, pager.pageSize)
    rows.value = result.items
    pager.total = result.total
    summary.value = result.summary
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    loading.value = false
  }
}
function search() {
  Object.assign(applied, filters, { dates: [...filters.dates] })
  pager.current = 1
  void load()
}
function onPeriodChange() {
  if (filters.period !== 'custom') {
    filters.dates = periodRange(filters.period)
    search()
  }
}
function reset() {
  Object.assign(filters, emptyFilters())
  search()
}
function onTableChange(p: TablePaginationConfig) {
  pager.current = p.current ?? 1
  pager.pageSize = p.pageSize ?? 20
  void load()
}
async function exportCsv() {
  exporting.value = true
  try {
    const all = await fetchAllPages(fetchPage)
    downloadCsv(
      exportFilename('机台产量', range.value),
      [
        { title: '机台编码', value: (r: Row) => r.code },
        { title: '名称', value: (r: Row) => r.name },
        { title: '厂区', value: (r: Row) => r.factoryName },
        { title: '工序', value: (r: Row) => r.jobName },
        { title: '批次', value: (r: Row) => r.lotNo ?? '' },
        { title: '工程', value: (r: Row) => r.projectName ?? '' },
        { title: '数量', value: (r: Row) => r.qty },
        { title: '展开面积', value: (r: Row) => r.pieceArea },
        { title: '喷涂面积', value: (r: Row) => r.sprayingArea },
        { title: '产能数量负载', value: (r: Row) => r.qtyLoad ?? '' },
        { title: '产能面积负载', value: (r: Row) => r.areaLoad ?? '' },
        { title: '满载率', value: (r: Row) => r.loadFactor ?? '' },
      ],
      all,
    )
    message.success(`已导出 ${all.length} 条`)
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    exporting.value = false
  }
}
onMounted(async () => {
  search()
  try {
    ;[factories.value, jobs.value, machines.value] = await Promise.all([
      call(client.GET('/factories')),
      call(client.GET('/jobs')),
      call(client.GET('/machines')),
    ])
  } catch (e) {
    message.error(errorMessage(e))
  }
})
</script>
<template>
  <div class="report">
    <div class="heading">
      <div>
        <h1>机台产量</h1>
        <span>按机台统计扫描数量、面积及日产能负载</span>
      </div>
    </div>
    <PageCard
      ><div class="filters">
        <label
          >扫描日期<a-radio-group v-model:value="filters.period" class="period" button-style="solid" @change="onPeriodChange"
            ><a-radio-button value="today">今天</a-radio-button><a-radio-button value="yesterday">昨天</a-radio-button
            ><a-radio-button value="week">近 7 天</a-radio-button><a-radio-button value="month">近 30 天</a-radio-button
            ><a-radio-button value="custom">自定义</a-radio-button></a-radio-group
          ></label
        >
        <label v-if="filters.period === 'custom'">日期范围<a-range-picker v-model:value="filters.dates" value-format="YYYY-MM-DD" /></label>
        <label
          >厂区<a-select
            v-model:value="filters.factoryId"
            allow-clear
            placeholder="全部"
            :options="factoryOptions"
            @change="filters.machineId = undefined"
        /></label>
        <label
          >工序<a-select
            v-model:value="filters.jobId"
            allow-clear
            placeholder="全部"
            :options="jobOptions"
            @change="filters.machineId = undefined"
        /></label>
        <label
          >机台<a-select
            v-model:value="filters.machineId"
            allow-clear
            show-search
            option-filter-prop="label"
            placeholder="全部"
            :options="machineOptions"
        /></label>
        <label
          >快速查询<a-input v-model:value="filters.keyword" placeholder="机台编码 / 名称 / 批次 / 工程" allow-clear @press-enter="search"
        /></label>
        <div class="actions"><a-button type="primary" @click="search">查询</a-button><a-button @click="reset">重置</a-button></div>
      </div></PageCard
    >
    <PageCard>
      <TableToolbar
        ><template #left
          ><div class="summary" data-testid="machine-summary">
            <span v-if="summary"
              >机台 <b>{{ summary.machines }}</b> 台 · 数量 <b>{{ summary.qty }}</b> 件 · 展开面积 {{ summary.pieceArea.toFixed(2) }} m² ·
              喷涂面积 {{ summary.sprayingArea.toFixed(2) }} m²</span
            ><a-alert
              v-if="summary?.unspecifiedQty"
              type="warning"
              show-icon
              :message="`未指定机台：${summary.unspecifiedQty} 件`"
            /></div></template
        ><template #right
          ><a-radio-group v-model:value="filters.mode" data-testid="machine-mode" button-style="solid" @change="search"
            ><a-radio-button value="summary">汇总</a-radio-button><a-radio-button value="detail">明细</a-radio-button></a-radio-group
          ><a-button :loading="exporting" data-testid="export-csv" @click="exportCsv"><DownloadOutlined />导出</a-button
          ><a-button aria-label="刷新" @click="load"><ReloadOutlined /></a-button></template
      ></TableToolbar>
      <a-table
        :row-key="(row: Row) => `${row.machineId}-${row.batchId ?? 0}`"
        size="middle"
        bordered
        :columns="columns"
        :data-source="rows"
        :loading="loading"
        :scroll="{ x: columns.reduce((n, c) => n + c.width, 0) }"
        :pagination="{
          ...pager,
          showSizeChanger: true,
          pageSizeOptions: ['20', '50', '100', '200'],
          showTotal: (n: number) => `共 ${n} 条`,
        }"
        data-testid="machine-output-table"
        @change="onTableChange"
      >
        <template #emptyText><a-empty description="该时段没有机台产量" /></template>
        <template #bodyCell="{ column, record }"
          ><template v-if="column.dataIndex === 'code'"><CodeText :value="record.code" /></template
          ><template v-else-if="column.dataIndex === 'lotNo'"
            ><a @click="router.push({ path: '/report/scans', query: { keyword: record.lotNo, machineId: record.machineId } })"
              ><CodeText :value="record.lotNo" /></a></template
          ><template v-else-if="['qtyLoad', 'areaLoad'].includes(column.dataIndex)">{{
            record[column.dataIndex] == null ? '—' : `${record[column.dataIndex].toFixed(1)}%`
          }}</template
          ><template v-else-if="column.dataIndex === 'loadFactor'"
            ><a-progress
              v-if="record.loadFactor != null"
              :percent="Math.round(record.loadFactor)"
              :status="record.loadFactor > 100 ? 'exception' : 'normal'"
              size="small"
            /><span v-else>—</span></template
          ><template v-else-if="['pieceArea', 'sprayingArea'].includes(column.dataIndex)">{{
            record[column.dataIndex].toFixed(2)
          }}</template></template
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
.heading span {
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
.period {
  display: flex;
  white-space: nowrap;
}
.actions {
  display: flex;
  gap: 8px;
  margin-left: auto;
}
.summary {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 18px;
  font-size: 13px;
}
.summary b {
  color: var(--gz-primary);
}
</style>
