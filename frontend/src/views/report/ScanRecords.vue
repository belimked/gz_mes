<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import type { TablePaginationConfig } from 'ant-design-vue'
import { DownloadOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import dayjs from 'dayjs'
import { call, client, errorMessage, type Schemas } from '@/api/client'
import PageCard from '@/components/PageCard.vue'
import TableToolbar from '@/components/TableToolbar.vue'
import CodeText from '@/components/CodeText.vue'
import { downloadCsv, exportFilename, fetchAllPages } from '@/utils/export'

type Row = Schemas['Scan']
type Period = 'today' | 'yesterday' | 'week' | 'month' | 'all' | 'custom'

const route = useRoute()
const router = useRouter()

function periodRange(p: Exclude<Period, 'custom' | 'all'>): [string, string] {
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
    period: 'week' as Period,
    dates: periodRange('week') as string[],
    jobId: undefined as number | undefined,
    machineId: undefined as number | undefined,
    staff: '',
    keyword: '',
  }
}
const filters = reactive(emptyFilters())
const applied = reactive(emptyFilters())
const range = computed<[string | undefined, string | undefined]>(() => {
  if (applied.period === 'all') return [undefined, undefined]
  if (applied.period !== 'custom') return periodRange(applied.period)
  const [from, to] = applied.dates
  return from && to ? [from, to] : [undefined, undefined]
})
function onPeriodChange() {
  if (filters.period === 'custom') return
  if (filters.period !== 'all') filters.dates = periodRange(filters.period)
  search()
}

const rows = ref<Row[]>([])
const jobs = ref<Schemas['Job'][]>([])
const machines = ref<Schemas['Machine'][]>([])
const pager = reactive({ current: 1, pageSize: 20, total: 0 })
const loading = ref(false)
const exporting = ref(false)
const jobOptions = computed(() => jobs.value.map((j) => ({ value: j.id, label: `${j.code} ${j.name}` })))
const machineOptions = computed(() =>
  machines.value.filter((m) => !filters.jobId || m.jobId === filters.jobId).map((m) => ({ value: m.id, label: `${m.code} ${m.name}` })),
)
const pageArea = computed(() => rows.value.reduce((n, r) => n + r.pieceArea, 0))

const columns = [
  { title: '扫描时间', dataIndex: 'scannedAt', width: 160 },
  { title: '岗位', dataIndex: 'jobName', width: 100 },
  { title: '机台', dataIndex: 'machineName', width: 130 },
  { title: '经办人', dataIndex: 'scannedBy', width: 140 },
  { title: '条码号', dataIndex: 'barCode', width: 150 },
  { title: '工程名称', dataIndex: 'projectName', width: 200, ellipsis: true },
  { title: '批次', dataIndex: 'lotNo', width: 150 },
  { title: '编号', dataIndex: 'pieceNo', width: 110 },
  { title: '序号', dataIndex: 'pieceIndex', width: 70, align: 'right' as const },
  { title: '单件面积 m²', dataIndex: 'pieceArea', width: 110, align: 'right' as const },
  { title: '设备', dataIndex: 'device', width: 100 },
]
const scrollX = columns.reduce((n, c) => n + c.width, 0)

function fetchPage(page: number, size: number) {
  return call(
    client.GET('/scans', {
      params: {
        query: {
          beginDate: range.value[0],
          endDate: range.value[1],
          jobId: applied.jobId,
          machineId: applied.machineId,
          staff: applied.staff.trim() || undefined,
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
      exportFilename('扫描记录', range.value),
      [
        { title: '扫描时间', value: (r) => dayjs(r.scannedAt).format('YYYY-MM-DD HH:mm:ss') },
        { title: '岗位', value: (r) => r.jobName },
        { title: '机台', value: (r) => r.machineName ?? '' },
        { title: '经办人', value: (r) => r.scannedBy },
        { title: '工号', value: (r) => r.staffNo },
        { title: '条码号', value: (r) => r.barCode },
        { title: '工程名称', value: (r) => r.projectName },
        { title: '批次', value: (r) => r.lotNo },
        { title: '编号', value: (r) => r.pieceNo },
        { title: '序号', value: (r) => r.pieceIndex },
        { title: '单件面积', value: (r) => r.pieceArea },
        { title: '设备', value: (r) => r.device },
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
function openLabel(r: Row) {
  void router.push({ path: '/production/labels', query: { keyword: r.barCode } })
}

onMounted(() => {
  const kw = typeof route.query.keyword === 'string' ? route.query.keyword : ''
  if (kw) {
    filters.keyword = kw
    filters.period = 'all'
  }
  if (typeof route.query.machineId === 'string' && Number.isInteger(Number(route.query.machineId)))
    filters.machineId = Number(route.query.machineId)
  search()
  void (async () => {
    try {
      ;[jobs.value, machines.value] = await Promise.all([call(client.GET('/jobs')), call(client.GET('/machines'))])
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
        <h1>扫描记录</h1>
        <span>每一次扫码报工的明细，最新在前，默认近 7 天</span>
      </div>
    </div>
    <PageCard>
      <div class="filters">
        <label
          >扫描日期<a-radio-group v-model:value="filters.period" class="period" button-style="solid" @change="onPeriodChange">
            <a-radio-button value="today">今天</a-radio-button>
            <a-radio-button value="yesterday">昨天</a-radio-button>
            <a-radio-button value="week">近 7 天</a-radio-button>
            <a-radio-button value="month">近 30 天</a-radio-button>
            <a-radio-button value="all">全部</a-radio-button>
            <a-radio-button value="custom">自定义</a-radio-button>
          </a-radio-group></label
        >
        <label v-if="filters.period === 'custom'">日期范围<a-range-picker v-model:value="filters.dates" value-format="YYYY-MM-DD" /></label>
        <label>岗位<a-select v-model:value="filters.jobId" allow-clear placeholder="全部岗位" :options="jobOptions" /></label>
        <label
          >机台<a-select
            v-model:value="filters.machineId"
            allow-clear
            show-search
            option-filter-prop="label"
            placeholder="全部机台"
            :options="machineOptions"
        /></label>
        <label>经办人<a-input v-model:value="filters.staff" placeholder="姓名 / 工号" allow-clear @press-enter="search" /></label>
        <label
          >快速查询<a-input v-model:value="filters.keyword" placeholder="条码 / 批次 / 工程 / 编号" allow-clear @press-enter="search"
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
          <div class="summary" data-testid="scan-summary">
            <span
              >共 <b>{{ pager.total }}</b> 条</span
            >
            <span
              >本页面积 <b>{{ pageArea.toFixed(2) }}</b> m²</span
            >
            <span class="muted">{{ range[0] ? `${range[0]} ~ ${range[1]}` : '全部日期' }}</span>
          </div>
        </template>
        <template #right>
          <a-button :loading="exporting" data-testid="export-csv" @click="exportCsv"><DownloadOutlined />导出</a-button>
          <a-button aria-label="刷新" @click="load"><ReloadOutlined /></a-button>
        </template>
      </TableToolbar>
      <a-table
        row-key="id"
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
        data-testid="scan-table"
        @change="onTableChange"
      >
        <template #emptyText><a-empty description="该时段没有扫描记录" /></template>
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'machineName'">{{ record.machineName || '—' }}</template>
          <template v-else-if="column.dataIndex === 'scannedAt'">{{ dayjs(record.scannedAt).format('YYYY-MM-DD HH:mm:ss') }}</template>
          <template v-else-if="column.dataIndex === 'jobName'">
            <a-tag color="blue">{{ record.jobName }}</a-tag>
          </template>
          <template v-else-if="column.dataIndex === 'scannedBy'">
            {{ record.scannedBy }}
            <span v-if="record.staffNo" class="muted"> <CodeText :value="record.staffNo" /></span>
          </template>
          <template v-else-if="column.dataIndex === 'barCode'">
            <a @click="openLabel(record)"><CodeText :value="record.barCode" /></a>
          </template>
          <template v-else-if="column.dataIndex === 'lotNo'">
            <a @click="openBatch(record)"><CodeText :value="record.lotNo" /></a>
          </template>
          <template v-else-if="column.dataIndex === 'pieceArea'">{{ record.pieceArea.toFixed(4) }}</template>
          <template v-else-if="column.dataIndex === 'device'">{{ record.device || '—' }}</template>
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
</style>
