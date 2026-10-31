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

type Row = Schemas['DefectDailyRow']
type Period = 'today' | 'yesterday' | 'week' | 'month' | 'custom'

const router = useRouter()

function periodRange(p: Exclude<Period, 'custom'>): [string, string] {
  const today = dayjs().startOf('day')
  const f = (d: dayjs.Dayjs) => d.format('YYYY-MM-DD')
  switch (p) {
    case 'today':
      return [f(today), f(today)]
    case 'yesterday':
      return [f(today.subtract(1, 'day')), f(today.subtract(1, 'day'))]
    case 'week':
      return [f(today.subtract(6, 'day')), f(today)]
    case 'month':
      return [f(today.subtract(29, 'day')), f(today)]
  }
}
function emptyFilters() {
  return {
    period: 'today' as Period,
    dates: periodRange('today') as string[],
    outsourceFactoryId: undefined as number | undefined,
    keyword: '',
  }
}
const filters = reactive(emptyFilters())
const applied = reactive(emptyFilters())
const range = computed<[string, string]>(() => {
  if (applied.period !== 'custom') return periodRange(applied.period)
  const [from, to] = applied.dates
  return from && to ? [from, to] : periodRange('today')
})
function onPeriodChange() {
  if (filters.period === 'custom') return
  filters.dates = periodRange(filters.period)
  search()
}

const rows = ref<Row[]>([])
const qcJobs = ref<Schemas['Job'][]>([])
const factories = ref<Schemas['Factory'][]>([])
const summary = ref<Schemas['DefectDailyReport']['summary']>()
const pager = reactive({ current: 1, pageSize: 20, total: 0 })
const loading = ref(false)
const exporting = ref(false)
const factoryOptions = computed(() => factories.value.filter((f) => f.kind === 'outsource').map((f) => ({ value: f.id, label: f.name })))

const fixedColumns = [
  { title: '批次', dataIndex: 'lotNo', width: 150, fixed: 'left' as const },
  { title: '工程名称', dataIndex: 'projectName', width: 200, ellipsis: true },
  { title: '客户', dataIndex: 'customerName', width: 140, ellipsis: true },
  { title: '板厚 / 颜色', dataIndex: 'spec', width: 120 },
  { title: '车间 / 外协', dataIndex: 'workshop', width: 130, ellipsis: true },
  { title: '清单数', dataIndex: 'itemQty', width: 80, align: 'right' as const },
]
const tailColumns = [
  { title: '不合格合计', dataIndex: 'failedQty', width: 100, align: 'right' as const },
  { title: '不合格面积 m²', dataIndex: 'failedArea', width: 120, align: 'right' as const },
  { title: '待返工', dataIndex: 'pendingQty', width: 90, align: 'right' as const },
]
const columns = computed(() => [
  ...fixedColumns,
  ...qcJobs.value.map((job) => ({
    title: job.name,
    children: [
      { title: '不合格', dataIndex: `cell-${job.id}-failed`, width: 80, align: 'right' as const },
      { title: '已复检', dataIndex: `cell-${job.id}-re`, width: 80, align: 'right' as const },
      { title: '待返工', dataIndex: `cell-${job.id}-pending`, width: 80, align: 'right' as const },
    ],
  })),
  ...tailColumns,
])
const scrollX = computed(
  () => fixedColumns.reduce((n, c) => n + c.width, 0) + qcJobs.value.length * 240 + tailColumns.reduce((n, c) => n + c.width, 0),
)
function cellOf(r: Row, key: string) {
  const [, jobId, field] = key.split('-')
  const cell = r.cells.find((c) => String(c.jobId) === jobId)
  if (!cell) return 0
  return field === 'failed' ? cell.failedQty : field === 're' ? cell.reinspectedQty : cell.pendingQty
}

function fetchPage(page: number, size: number) {
  return call(
    client.GET('/reports/defect-daily', {
      params: {
        query: {
          from: range.value[0],
          to: range.value[1],
          outsourceFactoryId: applied.outsourceFactoryId,
          keyword: applied.keyword.trim() || undefined,
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
    qcJobs.value = result.jobs
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
    const jobCols = qcJobs.value.flatMap((job) => [
      { title: `${job.name} 不合格`, value: (r: Row) => cellOf(r, `cell-${job.id}-failed`) },
      { title: `${job.name} 已复检`, value: (r: Row) => cellOf(r, `cell-${job.id}-re`) },
      { title: `${job.name} 待返工`, value: (r: Row) => cellOf(r, `cell-${job.id}-pending`) },
    ])
    downloadCsv(
      exportFilename('不良品日报', range.value),
      [
        { title: '批次', value: (r) => r.lotNo },
        { title: '工程名称', value: (r) => r.projectName },
        { title: '客户', value: (r) => r.customerName },
        { title: '客户订单号', value: (r) => r.clientOrderNo },
        { title: '板厚', value: (r) => r.thickness },
        { title: '颜色', value: (r) => r.color },
        { title: '车间', value: (r) => r.workshop },
        { title: '外协厂', value: (r) => r.outsourceFactoryName ?? '' },
        { title: '清单数', value: (r) => r.itemQty },
        { title: '清单面积', value: (r) => r.itemArea },
        ...jobCols,
        { title: '不合格合计', value: (r) => r.failedQty },
        { title: '不合格面积', value: (r) => r.failedArea },
        { title: '待返工', value: (r) => r.pendingQty },
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
function openDetail(r: Row) {
  void router.push({ path: '/quality/qc-scans', query: { keyword: r.lotNo, result: 'fail' } })
}

onMounted(() => {
  search()
  void (async () => {
    try {
      factories.value = await call(client.GET('/factories'))
    } catch (e) {
      message.error(errorMessage(e))
    }
  })()
})
</script>
<template>
  <div class="report">
    <div class="heading">
      <div>
        <h1>不良品日报</h1>
        <span>按批次统计当日各质检岗位判定的不合格件，以及其中已复检合格 / 仍待返工的数量</span>
      </div>
    </div>
    <PageCard>
      <div class="filters">
        <label
          >检验日期<a-radio-group v-model:value="filters.period" class="period" button-style="solid" @change="onPeriodChange">
            <a-radio-button value="today">今天</a-radio-button>
            <a-radio-button value="yesterday">昨天</a-radio-button>
            <a-radio-button value="week">近 7 天</a-radio-button>
            <a-radio-button value="month">近 30 天</a-radio-button>
            <a-radio-button value="custom">自定义</a-radio-button>
          </a-radio-group></label
        >
        <label v-if="filters.period === 'custom'">日期范围<a-range-picker v-model:value="filters.dates" value-format="YYYY-MM-DD" /></label>
        <label
          >外协厂<a-select v-model:value="filters.outsourceFactoryId" allow-clear placeholder="全部" :options="factoryOptions"
        /></label>
        <label
          >快速查询<a-input v-model:value="filters.keyword" placeholder="批次 / 工程 / 客户 / 订单号" allow-clear @press-enter="search"
        /></label>
        <div class="actions">
          <a-button type="primary" @click="search">查询</a-button>
          <a-button @click="reset">重置</a-button>
        </div>
      </div>
    </PageCard>
    <PageCard>
      <TableToolbar>
        <template #left>
          <div v-if="summary" class="summary" data-testid="defect-summary">
            <span
              >涉及批次 <b>{{ summary.batches }}</b></span
            >
            <span
              >不合格 <b class="bad">{{ summary.failedQty }}</b> 件 / {{ summary.failedArea.toFixed(2) }} m²</span
            >
            <span
              >已复检 <b>{{ summary.reinspectedQty }}</b></span
            >
            <span
              >待返工 <b class="bad">{{ summary.pendingQty }}</b> 件 / {{ summary.pendingArea.toFixed(2) }} m²</span
            >
            <span class="muted">{{ range[0] }} ~ {{ range[1] }}</span>
          </div>
        </template>
        <template #right>
          <a-button :loading="exporting" data-testid="export-csv" @click="exportCsv"><DownloadOutlined />导出</a-button>
          <a-button aria-label="刷新" @click="load"><ReloadOutlined /></a-button>
        </template>
      </TableToolbar>
      <a-table
        row-key="batchId"
        size="middle"
        bordered
        :columns="columns"
        :data-source="rows"
        :loading="loading"
        :scroll="{ x: scrollX }"
        :pagination="{
          ...pager,
          showSizeChanger: true,
          pageSizeOptions: ['20', '50', '100', '200'],
          showTotal: (n: number) => `共 ${n} 条`,
        }"
        data-testid="defect-table"
        @change="onTableChange"
      >
        <template #emptyText><a-empty description="该时段没有不合格记录" /></template>
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'lotNo'">
            <a @click="openDetail(record)"><CodeText :value="record.lotNo" /></a>
          </template>
          <template v-else-if="column.dataIndex === 'spec'">{{ record.thickness || '—' }} / {{ record.color || '—' }}</template>
          <template v-else-if="column.dataIndex === 'workshop'">{{ record.outsourceFactoryName || record.workshop || '—' }}</template>
          <template v-else-if="String(column.dataIndex).startsWith('cell-')">
            <span
              :class="{
                bad: String(column.dataIndex).endsWith('-failed') || String(column.dataIndex).endsWith('-pending'),
                zero: !cellOf(record, String(column.dataIndex)),
              }"
              >{{ cellOf(record, String(column.dataIndex)) || '—' }}</span
            >
          </template>
          <template v-else-if="column.dataIndex === 'failedQty'"
            ><b class="bad">{{ record.failedQty }}</b></template
          >
          <template v-else-if="column.dataIndex === 'failedArea'">{{ record.failedArea.toFixed(2) }}</template>
          <template v-else-if="column.dataIndex === 'pendingQty'">
            <b :class="record.pendingQty ? 'bad' : ''">{{ record.pendingQty }}</b>
          </template>
        </template>
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
.muted {
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
.bad {
  color: #cf1322;
}
.zero {
  color: var(--gz-muted);
}
</style>
