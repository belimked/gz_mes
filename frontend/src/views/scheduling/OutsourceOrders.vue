<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { message, type TablePaginationConfig } from 'ant-design-vue'
import { DownloadOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import { call, client, errorMessage, type Schemas } from '@/api/client'
import PageCard from '@/components/PageCard.vue'
import TableToolbar from '@/components/TableToolbar.vue'
import CodeText from '@/components/CodeText.vue'
import { downloadCsv, exportFilename, fetchAllPages } from '@/utils/export'

type Row = Schemas['OutsourceOrder']
const router = useRouter()
const filters = reactive({ dates: [] as string[], factoryId: undefined as number | undefined, keyword: '' })
const applied = reactive({ dates: [] as string[], factoryId: undefined as number | undefined, keyword: '' })
const factories = ref<Schemas['Factory'][]>([])
const options = computed(() => factories.value.filter((f) => f.kind === 'outsource').map((f) => ({ label: f.name, value: f.id })))
const rows = ref<Row[]>([])
const summary = ref<Schemas['OutsourceOrderPage']['summary']>()
const pager = reactive({ current: 1, pageSize: 20, total: 0 })
const loading = ref(false)
const exporting = ref(false)
const columns = [
  { title: '批次', dataIndex: 'lotNo', width: 150, fixed: 'left' as const },
  { title: '工程', dataIndex: 'projectName', width: 180, ellipsis: true },
  { title: '客户', dataIndex: 'customerName', width: 140, ellipsis: true },
  { title: '订单号', dataIndex: 'clientOrderNo', width: 130 },
  { title: '制单时间', dataIndex: 'createdAt', width: 175 },
  { title: '板厚/颜色/喷涂', dataIndex: 'spec', width: 180 },
  { title: '外协厂', dataIndex: 'outsourceFactoryName', width: 130 },
  { title: '状态', dataIndex: 'status', width: 100 },
  { title: '计划件数/面积', dataIndex: 'planned', width: 150 },
  { title: '开料进度', dataIndex: 'cutting', width: 220 },
  { title: '外协发货进度', dataIndex: 'shipping', width: 220 },
]

function fetchPage(page: number, size: number) {
  return call(
    client.GET('/reports/outsource-orders', {
      params: {
        query: {
          from: applied.dates[0],
          to: applied.dates[1],
          outsourceFactoryId: applied.factoryId,
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
  filters.dates = []
  filters.factoryId = undefined
  filters.keyword = ''
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
      exportFilename('外协单查看'),
      [
        { title: '批次', value: (r: Row) => r.lotNo },
        { title: '工程', value: (r: Row) => r.projectName },
        { title: '客户', value: (r: Row) => r.customerName },
        { title: '订单号', value: (r: Row) => r.clientOrderNo },
        { title: '制单时间', value: (r: Row) => r.createdAt },
        { title: '状态', value: (r: Row) => r.status },
        { title: '板厚', value: (r: Row) => r.thickness },
        { title: '颜色', value: (r: Row) => r.color },
        { title: '喷涂类型', value: (r: Row) => r.sprayingType },
        { title: '外协厂', value: (r: Row) => r.outsourceFactoryName },
        { title: '计划件数', value: (r: Row) => r.itemQty },
        { title: '计划面积', value: (r: Row) => r.itemArea },
        ...(['cutting', 'shipping'] as const).flatMap((key) => {
          const name = key === 'cutting' ? '开料' : '外协发货'
          return [
            { title: `${name}已扫件数`, value: (r: Row) => r[key].scannedQty },
            { title: `${name}已扫面积`, value: (r: Row) => r[key].scannedArea },
            { title: `${name}未扫件数`, value: (r: Row) => r[key].unscannedQty },
            { title: `${name}未扫面积`, value: (r: Row) => r[key].unscannedArea },
            { title: `${name}进度%`, value: (r: Row) => r[key].percent },
          ]
        }),
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
onMounted(() => {
  search()
  void call(client.GET('/factories'))
    .then((result) => {
      factories.value = result
    })
    .catch((e) => message.error(errorMessage(e)))
})
</script>
<template>
  <div class="report">
    <div class="heading">
      <div>
        <h1>外协单查看</h1>
        <span>按外协厂查看分派批次、开料及外协发货进度</span>
      </div>
    </div>
    <PageCard>
      <div class="filters">
        <label>制单日期<a-range-picker v-model:value="filters.dates" value-format="YYYY-MM-DD" /></label>
        <label>外协厂<a-select v-model:value="filters.factoryId" :options="options" allow-clear placeholder="全部" /></label>
        <label>
          快速查询<a-input v-model:value="filters.keyword" allow-clear placeholder="批次 / 工程 / 客户 / 订单号" @press-enter="search" />
        </label>
        <div class="actions">
          <a-button type="primary" @click="search">查询</a-button>
          <a-button @click="reset">重置</a-button>
        </div>
      </div>
    </PageCard>
    <PageCard>
      <TableToolbar>
        <template #left>
          <div v-if="summary" class="summary" data-testid="outsource-summary">
            <span
              >批次数 <b>{{ summary.batches }}</b></span
            >
            <span
              >计划 <b>{{ summary.plannedQty }}</b> 件 / {{ summary.plannedArea.toFixed(2) }} m²</span
            >
            <span
              >开料已扫 <b>{{ summary.cuttingScannedQty }}</b> 件</span
            >
            <span
              >外协发货已扫 <b>{{ summary.shippingScannedQty }}</b> 件</span
            >
          </div>
        </template>
        <template #right>
          <a-button :loading="exporting" data-testid="export-csv" @click="exportCsv"><DownloadOutlined />导出 CSV</a-button>
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
        :pagination="{
          ...pager,
          showSizeChanger: true,
          pageSizeOptions: ['20', '50', '100', '200'],
          showTotal: (n: number) => `共 ${n} 条`,
        }"
        :scroll="{ x: 1800 }"
        data-testid="outsource-table"
        @change="onTableChange"
      >
        <template #emptyText><a-empty description="没有符合条件的外协批次" /></template>
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'lotNo'"
            ><a @click="openBatch(record)"><CodeText :value="record.lotNo" /></a
          ></template>
          <template v-else-if="column.dataIndex === 'createdAt'">{{ record.createdAt.slice(0, 10) }}</template>
          <template v-else-if="column.dataIndex === 'spec'"
            >{{ record.thickness }} / {{ record.color }} / {{ record.sprayingType }}</template
          >
          <template v-else-if="column.dataIndex === 'planned'">{{ record.itemQty }} / {{ record.itemArea.toFixed(2) }}</template>
          <div v-else-if="column.dataIndex === 'cutting' || column.dataIndex === 'shipping'">
            <a-progress :percent="Number(record[column.dataIndex].percent.toFixed(1))" size="small" />
            <small
              >已扫 {{ record[column.dataIndex].scannedQty }} 件 / {{ record[column.dataIndex].scannedArea }} m² · 未扫
              {{ record[column.dataIndex].unscannedQty }} 件 / {{ record[column.dataIndex].unscannedArea }} m²</small
            >
          </div>
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
small {
  display: block;
  color: var(--gz-muted);
}
</style>
