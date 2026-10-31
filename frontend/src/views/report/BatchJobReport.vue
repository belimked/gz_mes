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
import StatusTag from '@/components/StatusTag.vue'
import { downloadCsv, exportFilename, fetchAllPages } from '@/utils/export'

type Mode = 'daily' | 'progress'
type Row = Schemas['BatchJobRow']
type Job = Schemas['Job']
type Summary = Schemas['BatchJobReport']['summary']
type Period = 'today' | 'yesterday' | 'week' | 'month' | 'all' | 'custom'

const props = defineProps<{ mode: Mode }>()
const router = useRouter()
const isDaily = computed(() => props.mode === 'daily')

const titles: Record<Mode, [string, string]> = {
  daily: ['工程日报', '按扫描日期统计每个批次各工序的报工数量，默认今天'],
  progress: ['工程进度', '每个批次各工序的累计报工进度，按客户交期排序'],
}
const STATUS_OPTIONS = [
  { value: 'pending', label: '待排产' },
  { value: 'scheduled', label: '已排产' },
  { value: 'in_production', label: '生产中' },
  { value: 'done', label: '已完工' },
]
const SCANNED_OPTIONS = [
  { value: 'all', label: '全部' },
  { value: 'yes', label: '有报工' },
  { value: 'no', label: '未报工' },
]
const WORKSHOPS = ['钣金一车间', '钣金二车间'].map((v) => ({ value: v, label: v }))
const statusLabel: Record<string, string> = { pending: '待排产', scheduled: '已排产', in_production: '生产中', done: '已完工' }
const statusTone: Record<string, 'neutral' | 'warning' | 'processing' | 'success'> = {
  pending: 'neutral',
  scheduled: 'warning',
  in_production: 'processing',
  done: 'success',
}

function periodRange(p: Exclude<Period, 'custom'>): [string, string] | undefined {
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
    case 'all':
      return undefined
  }
}

function emptyFilters() {
  return {
    period: (isDaily.value ? 'today' : 'all') as Period,
    dates: (periodRange(isDaily.value ? 'today' : 'week') ?? ['', '']) as string[],
    status: undefined as Schemas['BatchStatus'] | undefined,
    workshop: undefined as string | undefined,
    outsourceFactoryId: undefined as number | undefined,
    jobId: undefined as number | undefined,
    scanned: 'all' as 'all' | 'yes' | 'no',
    keyword: '',
  }
}
const filters = reactive(emptyFilters())
const applied = reactive(emptyFilters())
const range = computed<[string, string] | undefined>(() => {
  if (applied.period !== 'custom') return periodRange(applied.period)
  const [from, to] = applied.dates
  return from && to ? [from, to] : undefined
})
function onPeriodChange() {
  if (filters.period === 'custom') return
  filters.dates = periodRange(filters.period) ?? filters.dates
  search()
}

const rows = ref<Row[]>([])
const jobs = ref<Job[]>([])
const summary = ref<Summary>({ batches: 0, itemQty: 0, itemArea: 0, labelQty: 0, cells: [] })
const factories = ref<Schemas['Factory'][]>([])
const pager = reactive({ current: 1, pageSize: 20, total: 0 })
const loading = ref(false)
const exporting = ref(false)
const showArea = ref(false)
/** 工程汇总: collapse the per-job matrix into done-count / status (progress mode only). */
const summaryView = ref(false)

const jobOptions = computed(() => jobs.value.map((j) => ({ value: j.id, label: `${j.code} ${j.name}` })))
const outsourceOptions = computed(() => factories.value.filter((f) => f.kind === 'outsource').map((f) => ({ value: f.id, label: f.name })))
const stageSpan = computed(() => {
  const spans: { stage: string; label: string; count: number }[] = []
  const names: Record<string, string> = { sheet_metal: '钣金', spray: '喷涂', finished: '成品' }
  for (const j of jobs.value) {
    const last = spans[spans.length - 1]
    if (last && last.stage === j.stage) last.count++
    else spans.push({ stage: j.stage, label: names[j.stage] ?? j.stage, count: 1 })
  }
  return spans
})

const fixedColumns = computed(() => [
  { title: '批次号', dataIndex: 'lotNo', width: 150, fixed: 'left' as const },
  { title: '工程 / 客户', dataIndex: 'projectName', width: 190, ellipsis: true },
  { title: '规格', dataIndex: 'thickness', width: 110 },
  { title: '状态', dataIndex: 'status', width: 90 },
  ...(isDaily.value ? [] : [{ title: '交期', dataIndex: 'requiredDate', width: 100 }]),
  { title: '车间 / 外协', dataIndex: 'workshop', width: 120 },
  { title: '件数 / 标签', dataIndex: 'itemQty', width: 100 },
])
const jobColumns = computed(() =>
  jobs.value.map((j, k) => ({
    title: j.name,
    key: `job:${k}`,
    dataIndex: 'cells',
    width: 86,
    align: 'center' as const,
    className: `stage-${j.stage}`,
  })),
)
const summaryColumns = [
  { title: '完成工序', dataIndex: 'doneJobs', width: 110, align: 'center' as const },
  { title: '已完工件数', dataIndex: 'doneQty', width: 110, align: 'right' as const },
  { title: '完成率', dataIndex: 'donePct', width: 160 },
]
const columns = computed(() => [
  ...fixedColumns.value,
  ...(summaryView.value ? summaryColumns : jobColumns.value),
  { title: '当前工序', dataIndex: 'currentJobName', width: 96, fixed: 'right' as const },
])
/** jobs whose cell is full for the row (all active labels passed). */
function doneJobs(r: Row) {
  return r.labelQty > 0 ? r.cells.filter((c) => c.qty >= r.labelQty).length : 0
}
/** pieces that passed the last job. */
function doneQty(r: Row) {
  return r.cells[r.cells.length - 1]?.qty ?? 0
}
function donePct(r: Row) {
  const total = r.labelQty * Math.max(jobs.value.length, 1)
  return total ? Math.round((r.cells.reduce((n, c) => n + c.qty, 0) * 100) / total) : 0
}
const scrollX = computed(() => columns.value.reduce((n, c) => n + c.width, 0))

function fetchPage(page: number, size: number) {
  return call(
    client.GET('/reports/batch-jobs', {
      params: {
        query: {
          mode: props.mode,
          from: range.value?.[0],
          to: range.value?.[1],
          status: applied.status,
          workshop: applied.workshop,
          outsourceFactoryId: applied.outsourceFactoryId,
          jobId: applied.jobId,
          scanned: applied.scanned === 'all' ? undefined : applied.scanned === 'yes',
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
    jobs.value = result.jobs
    summary.value = result.summary
    pager.total = result.total
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
    const name = summaryView.value ? '工程汇总' : titles[props.mode][0]
    downloadCsv(
      exportFilename(name, range.value),
      [
        { title: '批次号', value: (r) => r.lotNo },
        { title: '工程名称', value: (r) => r.projectName },
        { title: '客户', value: (r) => r.customerName },
        { title: '客户单号', value: (r) => r.clientOrderNo },
        { title: '厚度', value: (r) => r.thickness },
        { title: '颜色', value: (r) => r.color },
        { title: '状态', value: (r) => statusLabel[r.status] ?? r.status },
        { title: '加急', value: (r) => (r.urgency === 'urgent' ? '是' : '') },
        { title: '交期', value: (r) => r.requiredDate ?? '' },
        { title: '钣金车间', value: (r) => r.workshop },
        { title: '外协厂', value: (r) => r.outsourceFactoryName ?? '' },
        { title: '计划数量', value: (r) => r.itemQty },
        { title: '计划面积', value: (r) => r.itemArea },
        { title: '标签数', value: (r) => r.labelQty },
        ...(summaryView.value
          ? [
              { title: '完成工序', value: (r: Row) => doneJobs(r) },
              { title: '已完工件数', value: (r: Row) => doneQty(r) },
              { title: '完成率%', value: (r: Row) => donePct(r) },
            ]
          : jobs.value.map((j, k) => ({
              title: showArea.value ? `${j.name}面积` : j.name,
              value: (r: Row) => (showArea.value ? (r.cells[k]?.area ?? 0) : (r.cells[k]?.qty ?? 0)),
            }))),
        { title: '当前工序', value: (r) => r.currentJobName },
        { title: '最近扫描', value: (r) => (r.lastScanAt ? dayjs(r.lastScanAt).format('YYYY-MM-DD HH:mm:ss') : '') },
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
function openBatch(r: Row) {
  void router.push({ path: '/scheduling/batches', query: { keyword: r.lotNo } })
}
function cellOf(r: Row, key: string) {
  return r.cells[Number(key.slice(4))]
}
function cellClass(cell: Schemas['JobCell'] | undefined, r: Row) {
  if (!cell || cell.qty === 0) return 'cell-empty'
  if (r.labelQty > 0 && cell.qty >= r.labelQty) return 'cell-full'
  return 'cell-part'
}
function fmtArea(v: number) {
  return Number(v).toFixed(2)
}
function fmtTime(v: string | null | undefined) {
  return v ? dayjs(v).format('MM-DD HH:mm') : '—'
}

onMounted(() => {
  void load()
  void (async () => {
    try {
      factories.value = await call(client.GET('/factories'))
    } catch {
      // outsource filter is optional; ignore
    }
  })()
})
</script>
<template>
  <div class="report">
    <div class="heading">
      <div>
        <h1>{{ titles[mode][0] }}</h1>
        <span>{{ titles[mode][1] }}</span>
      </div>
      <div class="heading-actions">
        <a-segmented
          v-if="!isDaily"
          v-model:value="summaryView"
          :options="[
            { value: false, label: '工序明细' },
            { value: true, label: '工程汇总' },
          ]"
          data-testid="progress-view"
        />
        <a-segmented
          v-if="!summaryView"
          v-model:value="showArea"
          :options="[
            { value: false, label: '按件数' },
            { value: true, label: '按面积' },
          ]"
        />
      </div>
    </div>
    <PageCard>
      <div class="filters">
        <label
          >{{ isDaily ? '扫描日期' : '下单日期'
          }}<a-radio-group v-model:value="filters.period" class="period" button-style="solid" @change="onPeriodChange">
            <a-radio-button v-if="!isDaily" value="all">全部</a-radio-button>
            <a-radio-button value="today">今天</a-radio-button>
            <a-radio-button value="yesterday">昨天</a-radio-button>
            <a-radio-button value="week">近 7 天</a-radio-button>
            <a-radio-button value="month">近 30 天</a-radio-button>
            <a-radio-button value="custom">自定义</a-radio-button>
          </a-radio-group></label
        >
        <label v-if="filters.period === 'custom'">日期范围<a-range-picker v-model:value="filters.dates" value-format="YYYY-MM-DD" /></label>
        <label>批次状态<a-select v-model:value="filters.status" allow-clear placeholder="全部" :options="STATUS_OPTIONS" /></label>
        <label>钣金车间<a-select v-model:value="filters.workshop" allow-clear placeholder="全部" :options="WORKSHOPS" /></label>
        <label
          >外协厂<a-select v-model:value="filters.outsourceFactoryId" allow-clear placeholder="全部" :options="outsourceOptions"
        /></label>
        <label>岗位<a-select v-model:value="filters.jobId" allow-clear placeholder="任一岗位有报工" :options="jobOptions" /></label>
        <label v-if="!isDaily">报工<a-select v-model:value="filters.scanned" :options="SCANNED_OPTIONS" /></label>
        <label
          >关键字<a-input v-model:value="filters.keyword" placeholder="批次号 / 工程 / 客户 / 客户单号" allow-clear @press-enter="search"
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
          <div class="summary" data-testid="report-summary">
            <span
              >批次 <b>{{ summary.batches }}</b></span
            >
            <span
              >件数 <b>{{ summary.itemQty }}</b></span
            >
            <span
              >面积 <b>{{ fmtArea(summary.itemArea) }}</b> m²</span
            >
            <span
              >标签 <b>{{ summary.labelQty }}</b></span
            >
            <span v-if="range" class="muted">{{ range[0] }} ~ {{ range[1] }}</span>
          </div>
        </template>
        <template #right>
          <a-button :loading="exporting" data-testid="export-csv" @click="exportCsv"><DownloadOutlined />导出</a-button>
          <a-button aria-label="刷新" @click="load"><ReloadOutlined /></a-button>
        </template>
      </TableToolbar>
      <a-table
        row-key="batchId"
        size="small"
        :columns="columns"
        :data-source="rows"
        :loading="loading"
        :scroll="{ x: scrollX }"
        :pagination="{ ...pager, showSizeChanger: true, pageSizeOptions: ['20', '50', '100'], showTotal: (n: number) => `共 ${n} 条` }"
        class="report-table"
        data-testid="report-table"
        @change="onTableChange"
      >
        <template #emptyText><a-empty :description="isDaily ? '该时段没有报工记录' : '没有匹配的批次'" /></template>
        <template #headerCell="{ column }">
          <template v-if="String(column.key ?? '').startsWith('job:')">
            <div class="job-head" :title="`累计 ${summary.cells[Number(String(column.key).slice(4))]?.qty ?? 0} 件`">
              {{ column.title }}
              <small>{{
                showArea
                  ? fmtArea(summary.cells[Number(String(column.key).slice(4))]?.area ?? 0)
                  : (summary.cells[Number(String(column.key).slice(4))]?.qty ?? 0)
              }}</small>
            </div>
          </template>
        </template>
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'lotNo'">
            <a @click="openBatch(record)"><CodeText :value="record.lotNo" /></a>
            <div v-if="record.urgency === 'urgent'"><StatusTag tone="error" label="加急" /></div>
          </template>
          <template v-else-if="column.dataIndex === 'projectName'">
            <strong class="truncate" :title="record.projectName">{{ record.projectName }}</strong>
            <div class="muted truncate" :title="record.customerName">
              {{ record.customerName }}<template v-if="record.clientOrderNo"> · {{ record.clientOrderNo }}</template>
            </div>
          </template>
          <template v-else-if="column.dataIndex === 'thickness'">
            {{ record.thickness }} mm
            <div class="muted">{{ record.color || '—' }}</div>
          </template>
          <template v-else-if="column.dataIndex === 'status'">
            <StatusTag :tone="statusTone[record.status] ?? 'neutral'" :label="statusLabel[record.status] ?? record.status" />
          </template>
          <template v-else-if="column.dataIndex === 'requiredDate'">
            <span
              :class="{
                danger: record.requiredDate && record.status !== 'done' && record.requiredDate < new Date().toISOString().slice(0, 10),
              }"
              >{{ record.requiredDate ?? '—' }}</span
            >
          </template>
          <template v-else-if="column.dataIndex === 'workshop'">
            {{ record.workshop || '—' }}
            <div v-if="record.outsourceFactoryName" class="muted">{{ record.outsourceFactoryName }}</div>
          </template>
          <template v-else-if="column.dataIndex === 'itemQty'">
            {{ record.itemQty }} 件
            <div class="muted">{{ record.labelQty }} 标签</div>
          </template>
          <template v-else-if="String(column.key ?? '').startsWith('job:')">
            <a-tooltip
              :title="`${cellOf(record, String(column.key))?.qty ?? 0} 件 · ${fmtArea(cellOf(record, String(column.key))?.area ?? 0)} m² · ${cellOf(record, String(column.key))?.pct ?? 0}%`"
            >
              <span class="cell" :class="cellClass(cellOf(record, String(column.key)), record)">
                {{ showArea ? fmtArea(cellOf(record, String(column.key))?.area ?? 0) : (cellOf(record, String(column.key))?.qty ?? 0) }}
              </span>
            </a-tooltip>
          </template>
          <template v-else-if="column.dataIndex === 'doneJobs'">
            <b>{{ doneJobs(record) }}</b
            ><span class="muted"> / {{ jobs.length }}</span>
          </template>
          <template v-else-if="column.dataIndex === 'doneQty'">
            {{ doneQty(record) }}<span class="muted"> / {{ record.labelQty }}</span>
          </template>
          <template v-else-if="column.dataIndex === 'donePct'">
            <a-progress :percent="donePct(record)" size="small" :status="record.status === 'done' ? 'success' : 'normal'" />
          </template>
          <template v-else-if="column.dataIndex === 'currentJobName'">
            {{ record.currentJobName || '—' }}
            <div class="muted">{{ fmtTime(record.lastScanAt) }}</div>
          </template>
        </template>
      </a-table>
      <div v-if="!summaryView" class="legend">
        <span v-for="s in stageSpan" :key="s.stage" class="legend-item" :class="`stage-${s.stage}`"
          >{{ s.label }}工段 · {{ s.count }} 岗位</span
        >
        <span class="legend-item"><i class="cell cell-full" />全部完成</span>
        <span class="legend-item"><i class="cell cell-part" />部分完成</span>
      </div>
    </PageCard>
  </div>
</template>
<style scoped>
.report {
  display: grid;
  gap: 16px;
}
.heading {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.heading h1 {
  margin: 0 0 4px;
  font-size: 20px;
}
.heading-actions {
  display: flex;
  gap: 8px;
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
.truncate {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.danger {
  color: var(--gz-error);
}
.job-head {
  display: grid;
  line-height: 1.2;
}
.job-head small {
  color: var(--gz-muted);
  font-weight: normal;
}
.cell {
  display: inline-block;
  min-width: 36px;
  padding: 2px 6px;
  border-radius: 4px;
  font-variant-numeric: tabular-nums;
}
.cell-empty {
  color: var(--gz-muted);
}
.cell-part {
  background: #e6f4ff;
  color: #0958d9;
}
.cell-full {
  background: #f6ffed;
  color: #389e0d;
}
.legend {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  margin-top: 12px;
  font-size: 12px;
  color: var(--gz-muted);
}
.legend-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.legend-item i.cell {
  min-width: 18px;
  height: 14px;
  padding: 0;
}
.report-table :deep(th.stage-sheet_metal) {
  border-top: 2px solid #1677ff;
}
.report-table :deep(th.stage-spray) {
  border-top: 2px solid #fa8c16;
}
.report-table :deep(th.stage-finished) {
  border-top: 2px solid #52c41a;
}
.legend-item.stage-sheet_metal {
  border-top: 2px solid #1677ff;
}
.legend-item.stage-spray {
  border-top: 2px solid #fa8c16;
}
.legend-item.stage-finished {
  border-top: 2px solid #52c41a;
}
</style>
