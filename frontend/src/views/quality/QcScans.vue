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

type Row = Schemas['QcRecord']
type Period = 'today' | 'yesterday' | 'week' | 'month' | 'all' | 'custom'
type Reinspect = 'all' | 'pending' | 'done'

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
    result: undefined as Schemas['QcResult'] | undefined,
    reinspect: 'all' as Reinspect,
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
const summary = ref<Schemas['QcRecordPage']['summary']>()
const pager = reactive({ current: 1, pageSize: 20, total: 0 })
const loading = ref(false)
const exporting = ref(false)
const jobOptions = computed(() => jobs.value.filter((j) => j.isQc).map((j) => ({ value: j.id, label: `${j.code} ${j.name}` })))

const columns = [
  { title: '检验时间', dataIndex: 'inspectedAt', width: 160 },
  { title: '质检岗位', dataIndex: 'jobName', width: 100 },
  { title: '判定', dataIndex: 'result', width: 90 },
  { title: '复检', dataIndex: 'reinspected', width: 150 },
  { title: '责任工序', dataIndex: 'defectJobName', width: 100 },
  { title: '不良类别 / 原因', dataIndex: 'defect', width: 200, ellipsis: true },
  { title: '检验员', dataIndex: 'inspectedBy', width: 140 },
  { title: '条码号', dataIndex: 'barCode', width: 150 },
  { title: '工程名称', dataIndex: 'projectName', width: 200, ellipsis: true },
  { title: '批次', dataIndex: 'lotNo', width: 150 },
  { title: '编号', dataIndex: 'pieceNo', width: 110 },
  { title: '序号', dataIndex: 'pieceIndex', width: 70, align: 'right' as const },
  { title: '单件面积 m²', dataIndex: 'pieceArea', width: 110, align: 'right' as const },
]
const scrollX = columns.reduce((n, c) => n + c.width, 0)

function fetchPage(page: number, size: number) {
  return call(
    client.GET('/qc-scans', {
      params: {
        query: {
          beginDate: range.value[0],
          endDate: range.value[1],
          jobId: applied.jobId,
          result: applied.result,
          reinspected: applied.reinspect === 'all' ? undefined : applied.reinspect === 'done',
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
const resultText = (r: Row) => (r.result === 'pass' ? '合格' : '不合格')
const reinspectText = (r: Row) => (r.result !== 'fail' ? '' : r.reinspected ? '已复检合格' : '待返工')
async function exportCsv() {
  exporting.value = true
  try {
    const all = await fetchAllPages(fetchPage)
    downloadCsv(
      exportFilename('质检扫描记录', range.value),
      [
        { title: '检验时间', value: (r) => dayjs(r.inspectedAt).format('YYYY-MM-DD HH:mm:ss') },
        { title: '质检岗位', value: (r) => r.jobName },
        { title: '判定', value: resultText },
        { title: '复检', value: reinspectText },
        { title: '复检时间', value: (r) => (r.reinspectedAt ? dayjs(r.reinspectedAt).format('YYYY-MM-DD HH:mm:ss') : '') },
        { title: '责任工序', value: (r) => r.defectJobName ?? '' },
        { title: '不良类别', value: (r) => r.defectCategory },
        { title: '不良原因', value: (r) => r.defectReason },
        { title: '检验员', value: (r) => r.inspectedBy },
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
  if (route.query.result === 'fail') filters.result = 'fail'
  if (route.query.pending === '1') {
    filters.result = 'fail'
    filters.reinspect = 'pending'
    filters.period = 'all'
  }
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
        <h1>质检扫描记录</h1>
        <span>钣金质检 / 喷涂质检每一次判定的明细；不合格件停留在上一道工序，复检合格后继续流转</span>
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
            <a-radio-button value="all">全部</a-radio-button>
            <a-radio-button value="custom">自定义</a-radio-button>
          </a-radio-group></label
        >
        <label v-if="filters.period === 'custom'">日期范围<a-range-picker v-model:value="filters.dates" value-format="YYYY-MM-DD" /></label>
        <label>质检岗位<a-select v-model:value="filters.jobId" allow-clear placeholder="全部岗位" :options="jobOptions" /></label>
        <label
          >判定<a-select
            v-model:value="filters.result"
            allow-clear
            placeholder="全部"
            :options="[
              { value: 'pass', label: '合格' },
              { value: 'fail', label: '不合格' },
            ]"
            data-testid="qc-result-filter"
        /></label>
        <label
          >复检状态<a-select
            v-model:value="filters.reinspect"
            :options="[
              { value: 'all', label: '全部' },
              { value: 'pending', label: '待返工（未复检）' },
              { value: 'done', label: '已复检合格' },
            ]"
        /></label>
        <label>检验员<a-input v-model:value="filters.staff" placeholder="姓名 / 工号" allow-clear @press-enter="search" /></label>
        <label
          >快速查询<a-input
            v-model:value="filters.keyword"
            placeholder="条码 / 批次 / 工程 / 编号 / 不良原因"
            allow-clear
            @press-enter="search"
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
          <div v-if="summary" class="summary" data-testid="qc-summary">
            <span
              >检验 <b>{{ summary.inspected }}</b> 件</span
            >
            <span
              >合格 <b class="ok">{{ summary.passed }}</b></span
            >
            <span
              >不合格 <b class="bad">{{ summary.failed }}</b> 件 / {{ summary.failedArea.toFixed(2) }} m²</span
            >
            <span
              >已复检 <b>{{ summary.reinspected }}</b></span
            >
            <span
              >待返工 <b class="bad">{{ summary.pending }}</b></span
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
        :row-class-name="(r: Row) => (r.result === 'fail' && !r.reinspected ? 'row-pending' : '')"
        :pagination="{
          ...pager,
          showSizeChanger: true,
          pageSizeOptions: ['20', '50', '100', '200'],
          showTotal: (n: number) => `共 ${n} 条`,
        }"
        data-testid="qc-table"
        @change="onTableChange"
      >
        <template #emptyText><a-empty description="该时段没有质检记录" /></template>
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'inspectedAt'">{{ dayjs(record.inspectedAt).format('YYYY-MM-DD HH:mm:ss') }}</template>
          <template v-else-if="column.dataIndex === 'jobName'">
            <a-tag color="blue">{{ record.jobName }}</a-tag>
          </template>
          <template v-else-if="column.dataIndex === 'result'">
            <a-tag :color="record.result === 'pass' ? 'success' : 'error'">{{ resultText(record) }}</a-tag>
          </template>
          <template v-else-if="column.dataIndex === 'reinspected'">
            <template v-if="record.result === 'fail'">
              <a-tag v-if="record.reinspected" color="green">已复检合格</a-tag>
              <a-tag v-else color="orange">待返工</a-tag>
              <span v-if="record.reinspectedAt" class="muted">{{ dayjs(record.reinspectedAt).format('MM-DD HH:mm') }}</span>
            </template>
            <template v-else>—</template>
          </template>
          <template v-else-if="column.dataIndex === 'defectJobName'">{{ record.defectJobName ?? '—' }}</template>
          <template v-else-if="column.dataIndex === 'defect'">
            <template v-if="record.result === 'fail'">
              <a-tag v-if="record.defectCategory">{{ record.defectCategory }}</a-tag
              >{{ record.defectReason || (record.defectCategory ? '' : '—') }}
            </template>
            <template v-else>—</template>
          </template>
          <template v-else-if="column.dataIndex === 'inspectedBy'">
            {{ record.inspectedBy }}
            <span v-if="record.staffNo" class="muted"> <CodeText :value="record.staffNo" /></span>
          </template>
          <template v-else-if="column.dataIndex === 'barCode'">
            <a @click="openLabel(record)"><CodeText :value="record.barCode" /></a>
          </template>
          <template v-else-if="column.dataIndex === 'lotNo'">
            <a @click="openBatch(record)"><CodeText :value="record.lotNo" /></a>
          </template>
          <template v-else-if="column.dataIndex === 'pieceArea'">{{ record.pieceArea.toFixed(4) }}</template>
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
.summary b.ok {
  color: #389e0d;
}
.summary b.bad {
  color: #cf1322;
}
:deep(.row-pending) td {
  background: #fff7e6;
}
</style>
