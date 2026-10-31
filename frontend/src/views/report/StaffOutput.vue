<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import type { TablePaginationConfig } from 'ant-design-vue'
import { ReloadOutlined } from '@ant-design/icons-vue'
import dayjs from 'dayjs'
import { call, client, errorMessage, type Schemas } from '@/api/client'
import PageCard from '@/components/PageCard.vue'
import TableToolbar from '@/components/TableToolbar.vue'
import CodeText from '@/components/CodeText.vue'

type Row = Schemas['StaffOutputRow']
type Summary = Schemas['StaffOutputReport']['summary']
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
    jobId: undefined as number | undefined,
    staff: '',
    batchKeyword: '',
    detail: false,
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
function onDetailChange() {
  applied.detail = filters.detail
  pager.current = 1
  void load()
}

const rows = ref<Row[]>([])
const jobs = ref<Schemas['Job'][]>([])
const summary = ref<Summary>({ rows: 0, staff: 0, qty: 0, area: 0, sprayArea: 0, weldLine: 0 })
const pager = reactive({ current: 1, pageSize: 20, total: 0 })
const loading = ref(false)
const jobOptions = computed(() => jobs.value.map((j) => ({ value: j.id, label: `${j.code} ${j.name}` })))

const columns = computed(() => [
  { title: '工号', dataIndex: 'staffNo', width: 90 },
  { title: '员工', dataIndex: 'displayName', width: 120 },
  { title: '岗位', dataIndex: 'jobName', width: 100 },
  ...(applied.detail
    ? [
        { title: '批次号', dataIndex: 'lotNo', width: 150 },
        { title: '工程名称', dataIndex: 'projectName', width: 200, ellipsis: true },
      ]
    : []),
  { title: '已扫描数量', dataIndex: 'qty', width: 110, align: 'right' as const },
  { title: '展开面积 m²', dataIndex: 'area', width: 120, align: 'right' as const },
  { title: '喷涂面积 m²', dataIndex: 'sprayArea', width: 120, align: 'right' as const },
  { title: '焊缝 m', dataIndex: 'weldLine', width: 90, align: 'right' as const },
  { title: '首次 / 最近扫描', dataIndex: 'lastScanAt', width: 150 },
  { title: '占比', dataIndex: 'share', width: 150 },
])
const scrollX = computed(() => columns.value.reduce((n, c) => n + c.width, 0))

async function load() {
  loading.value = true
  try {
    const result = await call(
      client.GET('/reports/staff-output', {
        params: {
          query: {
            from: range.value[0],
            to: range.value[1],
            jobId: applied.jobId,
            staff: applied.staff.trim() || undefined,
            batchKeyword: applied.batchKeyword.trim() || undefined,
            detail: applied.detail,
            page: pager.current,
            size: pager.pageSize,
          },
        },
      }),
    )
    rows.value = result.items
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
function openBatch(r: Row) {
  if (r.lotNo) void router.push({ path: '/scheduling/batches', query: { keyword: r.lotNo } })
}
function rowKey(r: Row) {
  return `${r.userId}:${r.jobId}:${r.batchId ?? ''}`
}
function fmt(v: number, digits = 2) {
  return Number(v).toFixed(digits)
}
function share(r: Row) {
  return summary.value.qty ? Math.round((r.qty * 100) / summary.value.qty) : 0
}
function hm(v: string) {
  return dayjs(v).format('MM-DD HH:mm')
}

onMounted(() => {
  void load()
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
        <h1>员工产量</h1>
        <span>按员工与岗位统计扫码报工的数量与面积，默认今天</span>
      </div>
      <a-segmented
        v-model:value="filters.detail"
        :options="[
          { value: false, label: '汇总' },
          { value: true, label: '明细' },
        ]"
        data-testid="staff-mode"
        @change="onDetailChange"
      />
    </div>
    <PageCard>
      <div class="filters">
        <label
          >扫描日期<a-radio-group v-model:value="filters.period" class="period" button-style="solid" @change="onPeriodChange">
            <a-radio-button value="today">今天</a-radio-button>
            <a-radio-button value="yesterday">昨天</a-radio-button>
            <a-radio-button value="week">近 7 天</a-radio-button>
            <a-radio-button value="month">近 30 天</a-radio-button>
            <a-radio-button value="custom">自定义</a-radio-button>
          </a-radio-group></label
        >
        <label v-if="filters.period === 'custom'">日期范围<a-range-picker v-model:value="filters.dates" value-format="YYYY-MM-DD" /></label>
        <label>岗位<a-select v-model:value="filters.jobId" allow-clear placeholder="全部岗位" :options="jobOptions" /></label>
        <label>员工<a-input v-model:value="filters.staff" placeholder="姓名 / 工号" allow-clear @press-enter="search" /></label>
        <label>批次<a-input v-model:value="filters.batchKeyword" placeholder="批次号 / 工程" allow-clear @press-enter="search" /></label>
        <div class="actions">
          <a-button type="primary" @click="search">查询</a-button>
          <a-button @click="reset">重置</a-button>
        </div>
      </div>
    </PageCard>
    <PageCard>
      <TableToolbar>
        <template #left>
          <div class="summary" data-testid="staff-summary">
            <span
              >员工 <b>{{ summary.staff }}</b> 人</span
            >
            <span
              >扫描 <b>{{ summary.qty }}</b> 件</span
            >
            <span
              >展开面积 <b>{{ fmt(summary.area) }}</b> m²</span
            >
            <span
              >喷涂面积 <b>{{ fmt(summary.sprayArea) }}</b> m²</span
            >
            <span
              >焊缝 <b>{{ fmt(summary.weldLine) }}</b> m</span
            >
            <span class="muted">{{ range[0] }} ~ {{ range[1] }}</span>
          </div>
        </template>
        <template #right>
          <a-button aria-label="刷新" @click="load"><ReloadOutlined /></a-button>
        </template>
      </TableToolbar>
      <a-table
        :row-key="rowKey"
        size="middle"
        :columns="columns"
        :data-source="rows"
        :loading="loading"
        :scroll="{ x: scrollX }"
        :pagination="{ ...pager, showSizeChanger: true, pageSizeOptions: ['20', '50', '100'], showTotal: (n: number) => `共 ${n} 条` }"
        data-testid="staff-table"
        @change="onTableChange"
      >
        <template #emptyText><a-empty description="该时段没有报工记录" /></template>
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'staffNo'">
            <CodeText :value="record.staffNo || '—'" />
          </template>
          <template v-else-if="column.dataIndex === 'lotNo'">
            <a @click="openBatch(record)"><CodeText :value="record.lotNo ?? ''" /></a>
          </template>
          <template v-else-if="column.dataIndex === 'qty'">
            <b>{{ record.qty }}</b>
          </template>
          <template v-else-if="['area', 'sprayArea', 'weldLine'].includes(String(column.dataIndex))">
            {{ fmt(record[column.dataIndex as 'area']) }}
          </template>
          <template v-else-if="column.dataIndex === 'lastScanAt'">
            {{ hm(record.firstScanAt) }}
            <div class="muted">{{ hm(record.lastScanAt) }}</div>
          </template>
          <template v-else-if="column.dataIndex === 'share'">
            <a-progress :percent="share(record)" size="small" :format="(p: number) => `${p}%`" />
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
.heading {
  display: flex;
  justify-content: space-between;
  align-items: center;
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
