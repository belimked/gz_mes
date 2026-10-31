<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import type { TablePaginationConfig } from 'ant-design-vue'
import { DownloadOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import { call, client, errorMessage, type Schemas } from '@/api/client'
import PageCard from '@/components/PageCard.vue'
import TableToolbar from '@/components/TableToolbar.vue'
import CodeText from '@/components/CodeText.vue'
import StatusTag from '@/components/StatusTag.vue'
import { downloadCsv, exportFilename, fetchAllPages } from '@/utils/export'

type Row = Schemas['Batch']
type Balance = 'all' | 'balanced' | 'over' | 'under' | 'noplan'

const router = useRouter()
const STATUS_OPTIONS = [
  { value: 'pending', label: '待排产' },
  { value: 'scheduled', label: '已排产' },
  { value: 'in_production', label: '生产中' },
  { value: 'done', label: '已完工' },
]
const BALANCE_OPTIONS: { value: Balance; label: string }[] = [
  { value: 'all', label: '全部' },
  { value: 'balanced', label: '平衡' },
  { value: 'over', label: '计划多于下单' },
  { value: 'under', label: '计划少于下单' },
  { value: 'noplan', label: '未录构件' },
]
const statusLabel: Record<string, string> = { pending: '待排产', scheduled: '已排产', in_production: '生产中', done: '已完工' }
const statusTone: Record<string, 'neutral' | 'warning' | 'processing' | 'success'> = {
  pending: 'neutral',
  scheduled: 'warning',
  in_production: 'processing',
  done: 'success',
}

function emptyFilters() {
  return { keyword: '', status: undefined as Schemas['BatchStatus'] | undefined, balance: 'all' as Balance }
}
const filters = reactive(emptyFilters())
const applied = reactive(emptyFilters())

const all = ref<Row[]>([])
const loading = ref(false)
const exporting = ref(false)
const pager = reactive({ current: 1, pageSize: 20 })

/** Balance is judged on quantity; area is informational (客户面积 vs 构件面积 often differ by rounding). */
function balanceOf(r: Row): Exclude<Balance, 'all'> {
  if (r.itemQty === 0) return 'noplan'
  if (r.itemQty === r.orderQty) return 'balanced'
  return r.itemQty > r.orderQty ? 'over' : 'under'
}
const balanceLabel: Record<Exclude<Balance, 'all'>, string> = { balanced: '平衡', over: '计划多', under: '计划少', noplan: '未录构件' }
const balanceTone: Record<Exclude<Balance, 'all'>, 'success' | 'warning' | 'error' | 'neutral'> = {
  balanced: 'success',
  over: 'warning',
  under: 'error',
  noplan: 'neutral',
}

const filtered = computed(() => all.value.filter((r) => applied.balance === 'all' || balanceOf(r) === applied.balance))
const rows = computed(() => filtered.value.slice((pager.current - 1) * pager.pageSize, pager.current * pager.pageSize))
const summary = computed(() => {
  const s = { batches: filtered.value.length, orderQty: 0, orderArea: 0, itemQty: 0, itemArea: 0, unbalanced: 0 }
  for (const r of filtered.value) {
    s.orderQty += r.orderQty
    s.orderArea += r.orderArea
    s.itemQty += r.itemQty
    s.itemArea += r.itemArea
    if (balanceOf(r) !== 'balanced') s.unbalanced++
  }
  return s
})

const columns = [
  { title: '批次号', dataIndex: 'lotNo', width: 150, fixed: 'left' as const },
  { title: '工程 / 客户', dataIndex: 'projectName', width: 200, ellipsis: true },
  { title: '客户单号', dataIndex: 'clientOrderNo', width: 120 },
  { title: '规格', dataIndex: 'thickness', width: 110 },
  { title: '状态', dataIndex: 'status', width: 90 },
  { title: '下单数量', dataIndex: 'orderQty', width: 100, align: 'right' as const },
  { title: '下单面积 m²', dataIndex: 'orderArea', width: 120, align: 'right' as const },
  { title: '计划数量', dataIndex: 'itemQty', width: 100, align: 'right' as const },
  { title: '计划面积 m²', dataIndex: 'itemArea', width: 120, align: 'right' as const },
  { title: '数量差', dataIndex: 'qtyDiff', width: 90, align: 'right' as const },
  { title: '面积差 m²', dataIndex: 'areaDiff', width: 110, align: 'right' as const },
  { title: '平衡', dataIndex: 'balance', width: 100 },
]
const scrollX = columns.reduce((n, c) => n + c.width, 0)

function fetchPage(page: number, size: number) {
  return call(
    client.GET('/batches', {
      params: { query: { keyword: applied.keyword.trim() || undefined, status: applied.status, page, size } },
    }),
  )
}
async function load() {
  loading.value = true
  try {
    all.value = await fetchAllPages(fetchPage)
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    loading.value = false
  }
}
function search() {
  Object.assign(applied, filters)
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
}
function exportCsv() {
  exporting.value = true
  try {
    downloadCsv(
      exportFilename('生产平衡表'),
      [
        { title: '批次号', value: (r) => r.lotNo },
        { title: '工程名称', value: (r) => r.projectName },
        { title: '客户', value: (r) => r.customerName },
        { title: '客户单号', value: (r) => r.clientOrderNo },
        { title: '厚度', value: (r) => r.thickness },
        { title: '颜色', value: (r) => r.color },
        { title: '状态', value: (r) => statusLabel[r.status] ?? r.status },
        { title: '下单数量', value: (r) => r.orderQty },
        { title: '下单面积', value: (r) => r.orderArea },
        { title: '计划数量', value: (r) => r.itemQty },
        { title: '计划面积', value: (r) => r.itemArea },
        { title: '数量差', value: (r) => r.itemQty - r.orderQty },
        { title: '面积差', value: (r) => +(r.itemArea - r.orderArea).toFixed(4) },
        { title: '平衡', value: (r) => balanceLabel[balanceOf(r)] },
      ],
      filtered.value,
    )
    message.success(`已导出 ${filtered.value.length} 条`)
  } finally {
    exporting.value = false
  }
}
function openBatch(r: Row) {
  void router.push({ path: '/scheduling/batches', query: { keyword: r.lotNo } })
}
function fmt(v: number) {
  return Number(v).toFixed(2)
}
function signed(v: number, digits = 0) {
  const s = v.toFixed(digits)
  return v > 0 ? `+${s}` : s
}

onMounted(() => void load())
</script>
<template>
  <div class="report">
    <div class="heading">
      <div>
        <h1>生产平衡表</h1>
        <span>按批次对比客户下单数量 / 面积与构件清单的计划数量 / 面积</span>
      </div>
    </div>
    <PageCard>
      <div class="filters">
        <label>批次状态<a-select v-model:value="filters.status" allow-clear placeholder="全部" :options="STATUS_OPTIONS" /></label>
        <label>平衡状态<a-select v-model:value="filters.balance" :options="BALANCE_OPTIONS" /></label>
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
          <div class="summary" data-testid="balance-summary">
            <span
              >批次 <b>{{ summary.batches }}</b></span
            >
            <span
              >下单 <b>{{ summary.orderQty }}</b> 件 / <b>{{ fmt(summary.orderArea) }}</b> m²</span
            >
            <span
              >计划 <b>{{ summary.itemQty }}</b> 件 / <b>{{ fmt(summary.itemArea) }}</b> m²</span
            >
            <span
              >不平衡 <b :class="{ danger: summary.unbalanced }">{{ summary.unbalanced }}</b></span
            >
          </div>
        </template>
        <template #right>
          <a-button :loading="exporting" data-testid="export-csv" @click="exportCsv"><DownloadOutlined />导出</a-button>
          <a-button aria-label="刷新" @click="load"><ReloadOutlined /></a-button>
        </template>
      </TableToolbar>
      <a-table
        row-key="id"
        size="small"
        :columns="columns"
        :data-source="rows"
        :loading="loading"
        :scroll="{ x: scrollX }"
        :pagination="{
          ...pager,
          total: filtered.length,
          showSizeChanger: true,
          pageSizeOptions: ['20', '50', '100'],
          showTotal: (n: number) => `共 ${n} 条`,
        }"
        data-testid="balance-table"
        @change="onTableChange"
      >
        <template #emptyText><a-empty description="没有匹配的批次" /></template>
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'lotNo'">
            <a @click="openBatch(record)"><CodeText :value="record.lotNo" /></a>
          </template>
          <template v-else-if="column.dataIndex === 'projectName'">
            <strong class="truncate" :title="record.projectName">{{ record.projectName }}</strong>
            <div class="muted truncate" :title="record.customerName">{{ record.customerName }}</div>
          </template>
          <template v-else-if="column.dataIndex === 'clientOrderNo'">{{ record.clientOrderNo || '—' }}</template>
          <template v-else-if="column.dataIndex === 'thickness'">
            {{ record.thickness }} mm
            <div class="muted">{{ record.color || '—' }}</div>
          </template>
          <template v-else-if="column.dataIndex === 'status'">
            <StatusTag :tone="statusTone[record.status] ?? 'neutral'" :label="statusLabel[record.status] ?? record.status" />
          </template>
          <template v-else-if="column.dataIndex === 'orderArea' || column.dataIndex === 'itemArea'">
            {{ fmt(record[column.dataIndex as 'orderArea']) }}
          </template>
          <template v-else-if="column.dataIndex === 'qtyDiff'">
            <span :class="{ danger: record.itemQty < record.orderQty, warn: record.itemQty > record.orderQty }">{{
              signed(record.itemQty - record.orderQty)
            }}</span>
          </template>
          <template v-else-if="column.dataIndex === 'areaDiff'">
            <span class="muted">{{ signed(record.itemArea - record.orderArea, 2) }}</span>
          </template>
          <template v-else-if="column.dataIndex === 'balance'">
            <StatusTag :tone="balanceTone[balanceOf(record)]" :label="balanceLabel[balanceOf(record)]" />
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
.warn {
  color: #d46b08;
}
</style>
