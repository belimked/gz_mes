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

type Row = Schemas['QualityReportRow']
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
    period: 'month' as Period,
    dates: periodRange('month') as string[],
    jobId: undefined as number | undefined,
    defectJobId: undefined as number | undefined,
    keyword: '',
  }
}
const filters = reactive(emptyFilters())
const applied = reactive(emptyFilters())
const range = computed<[string, string]>(() => {
  if (applied.period !== 'custom') return periodRange(applied.period)
  const [from, to] = applied.dates
  return from && to ? [from, to] : periodRange('month')
})
function onPeriodChange() {
  if (filters.period === 'custom') return
  filters.dates = periodRange(filters.period)
  search()
}

const rows = ref<Row[]>([])
const jobs = ref<Schemas['Job'][]>([])
const summary = ref<Schemas['QualityReport']['summary']>()
const pager = reactive({ current: 1, pageSize: 20, total: 0 })
const loading = ref(false)
const exporting = ref(false)
const qcJobOptions = computed(() => jobs.value.filter((j) => j.isQc).map((j) => ({ value: j.id, label: `${j.code} ${j.name}` })))
const jobOptions = computed(() => jobs.value.map((j) => ({ value: j.id, label: `${j.code} ${j.name}` })))

const columns = [
  { title: '日期', dataIndex: 'date', width: 110 },
  { title: '质检岗位', dataIndex: 'jobName', width: 100 },
  { title: '工程名称', dataIndex: 'projectName', width: 200, ellipsis: true },
  { title: '批次', dataIndex: 'lotNo', width: 150 },
  { title: '责任工序', dataIndex: 'defectJobName', width: 100 },
  { title: '不良类别', dataIndex: 'defectCategory', width: 100 },
  { title: '不良原因', dataIndex: 'defectReason', width: 180, ellipsis: true },
  { title: '不合格数', dataIndex: 'qty', width: 90, align: 'right' as const },
  { title: '面积 m²', dataIndex: 'area', width: 100, align: 'right' as const },
  { title: '已复检', dataIndex: 'reinspectedQty', width: 90, align: 'right' as const },
  { title: '待返工', dataIndex: 'pending', width: 90, align: 'right' as const },
  { title: '涉及编号', dataIndex: 'pieceNos', width: 160, ellipsis: true },
  { title: '检验员', dataIndex: 'inspectors', width: 120, ellipsis: true },
]
const scrollX = columns.reduce((n, c) => n + c.width, 0)

function fetchPage(page: number, size: number) {
  return call(
    client.GET('/reports/quality', {
      params: {
        query: {
          from: range.value[0],
          to: range.value[1],
          jobId: applied.jobId,
          defectJobId: applied.defectJobId,
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
    downloadCsv(
      exportFilename('质量报表', range.value),
      [
        { title: '日期', value: (r) => r.date },
        { title: '质检岗位', value: (r) => r.jobName },
        { title: '工程名称', value: (r) => r.projectName },
        { title: '批次', value: (r) => r.lotNo },
        { title: '外协厂', value: (r) => r.outsourceFactoryName ?? '' },
        { title: '责任工序', value: (r) => r.defectJobName ?? '' },
        { title: '不良类别', value: (r) => r.defectCategory },
        { title: '不良原因', value: (r) => r.defectReason },
        { title: '不合格数', value: (r) => r.qty },
        { title: '面积', value: (r) => r.area },
        { title: '已复检', value: (r) => r.reinspectedQty },
        { title: '待返工', value: (r) => r.qty - r.reinspectedQty },
        { title: '涉及编号', value: (r) => r.pieceNos },
        { title: '检验员', value: (r) => r.inspectors },
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
      jobs.value = await call(client.GET('/jobs'))
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
        <h1>质量报表</h1>
        <span>按日期 × 质检岗位 × 批次 × 不良原因汇总不合格件，默认近 30 天</span>
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
        <label>质检岗位<a-select v-model:value="filters.jobId" allow-clear placeholder="全部" :options="qcJobOptions" /></label>
        <label>责任工序<a-select v-model:value="filters.defectJobId" allow-clear placeholder="全部" :options="jobOptions" /></label>
        <label
          >快速查询<a-input v-model:value="filters.keyword" placeholder="批次 / 工程 / 不良原因" allow-clear @press-enter="search"
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
          <div v-if="summary" class="summary" data-testid="quality-summary">
            <span
              >检验 <b>{{ summary.inspected }}</b> 件</span
            >
            <span
              >合格率 <b :class="summary.passRate >= 95 ? 'ok' : 'bad'">{{ summary.passRate }}%</b></span
            >
            <span
              >不合格 <b class="bad">{{ summary.failed }}</b> 件 / {{ summary.failedArea.toFixed(2) }} m²</span
            >
            <span
              >已复检 <b>{{ summary.reinspected }}</b></span
            >
            <span
              >待返工 <b class="bad">{{ summary.failed - summary.reinspected }}</b></span
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
        :row-key="(r: Row) => [r.date, r.jobId, r.batchId, r.defectJobId, r.defectCategory, r.defectReason].join('|')"
        size="middle"
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
        data-testid="quality-table"
        @change="onTableChange"
      >
        <template #emptyText><a-empty description="该时段没有不合格记录" /></template>
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'jobName'">
            <a-tag color="blue">{{ record.jobName }}</a-tag>
          </template>
          <template v-else-if="column.dataIndex === 'lotNo'">
            <a @click="openDetail(record)"><CodeText :value="record.lotNo" /></a>
            <div v-if="record.outsourceFactoryName" class="muted">{{ record.outsourceFactoryName }}</div>
          </template>
          <template v-else-if="column.dataIndex === 'defectJobName'">{{ record.defectJobName ?? '—' }}</template>
          <template v-else-if="column.dataIndex === 'defectCategory'">{{ record.defectCategory || '—' }}</template>
          <template v-else-if="column.dataIndex === 'defectReason'">{{ record.defectReason || '—' }}</template>
          <template v-else-if="column.dataIndex === 'qty'"
            ><b class="bad">{{ record.qty }}</b></template
          >
          <template v-else-if="column.dataIndex === 'area'">{{ record.area.toFixed(2) }}</template>
          <template v-else-if="column.dataIndex === 'pending'">
            <span :class="{ bad: record.qty - record.reinspectedQty > 0 }">{{ record.qty - record.reinspectedQty }}</span>
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
.ok {
  color: #389e0d;
}
.bad {
  color: #cf1322;
}
</style>
