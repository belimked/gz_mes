<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { message, type TablePaginationConfig } from 'ant-design-vue'
import { call, client, errorMessage, type Schemas } from '@/api/client'
import PageCard from '@/components/PageCard.vue'
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
const loading = ref(false), exporting = ref(false)
const columns = [
  { title: '批次', dataIndex: 'lotNo', width: 150 }, { title: '工程', dataIndex: 'projectName', width: 180 },
  { title: '客户', dataIndex: 'customerName', width: 140 }, { title: '订单号', dataIndex: 'clientOrderNo', width: 130 },
  { title: '制单时间', dataIndex: 'createdAt', width: 175 }, { title: '板厚/颜色/喷涂', dataIndex: 'spec', width: 180 },
  { title: '外协厂', dataIndex: 'outsourceFactoryName', width: 130 }, { title: '状态', dataIndex: 'status', width: 100 },
  { title: '计划件数/面积', dataIndex: 'planned', width: 150 }, { title: '开料进度', dataIndex: 'cutting', width: 220 },
  { title: '外协发货进度', dataIndex: 'shipping', width: 220 },
]
function fetchPage(page: number, size: number) {
  return call(client.GET('/reports/outsource-orders', { params: { query: { from: applied.dates[0], to: applied.dates[1], outsourceFactoryId: applied.factoryId, keyword: applied.keyword || undefined, page, size } } }))
}
async function load() { loading.value = true; try { const result = await fetchPage(pager.current, pager.pageSize); rows.value = result.items; pager.total = result.total; summary.value = result.summary } catch (e) { message.error(errorMessage(e)) } finally { loading.value = false } }
function search() { Object.assign(applied, filters, { dates: [...filters.dates] }); pager.current = 1; void load() }
function reset() { filters.dates = []; filters.factoryId = undefined; filters.keyword = ''; search() }
function change(p: TablePaginationConfig) { pager.current = p.current ?? 1; pager.pageSize = p.pageSize ?? 20; void load() }
async function exportCsv() { exporting.value = true; try { const all = await fetchAllPages(fetchPage); downloadCsv(exportFilename('外协单查看'), [
  { title: '批次', value: (r: Row) => r.lotNo }, { title: '工程', value: (r: Row) => r.projectName }, { title: '客户', value: (r: Row) => r.customerName },
  { title: '订单号', value: (r: Row) => r.clientOrderNo }, { title: '制单时间', value: (r: Row) => r.createdAt },
  { title: '外协厂', value: (r: Row) => r.outsourceFactoryName }, { title: '计划件数', value: (r: Row) => r.itemQty }, { title: '计划面积', value: (r: Row) => r.itemArea },
  ...(['cutting', 'shipping'] as const).flatMap((key) => [
    { title: `${key}已扫件数`, value: (r: Row) => r[key].scannedQty }, { title: `${key}已扫面积`, value: (r: Row) => r[key].scannedArea },
    { title: `${key}未扫件数`, value: (r: Row) => r[key].unscannedQty }, { title: `${key}未扫面积`, value: (r: Row) => r[key].unscannedArea },
    { title: `${key}进度%`, value: (r: Row) => r[key].percent },
  ]),
], all); message.success(`已导出 ${all.length} 条`) } catch (e) { message.error(errorMessage(e)) } finally { exporting.value = false } }
onMounted(() => { search(); void call(client.GET('/factories')).then((v) => { factories.value = v }).catch((e) => message.error(errorMessage(e))) })
</script>
<template>
  <div class="report"><h1>外协单查看</h1><PageCard>
    <div class="filters"><label>制单日期 <a-range-picker v-model:value="filters.dates" value-format="YYYY-MM-DD" /></label>
      <label>外协厂 <a-select v-model:value="filters.factoryId" :options="options" allow-clear style="width: 180px" /></label>
      <label>快速查询 <a-input v-model:value="filters.keyword" allow-clear placeholder="批次 / 工程 / 客户 / 订单号" @press-enter="search" /></label>
      <a-button type="primary" @click="search">查询</a-button><a-button @click="reset">重置</a-button>
    </div></PageCard>
    <PageCard><div class="summary">批次数 {{ summary?.batches ?? 0 }} · 计划 {{ summary?.plannedQty ?? 0 }} 件 / {{ summary?.plannedArea ?? 0 }} m² · 开料已扫 {{ summary?.cuttingScannedQty ?? 0 }} 件 · 外协发货已扫 {{ summary?.shippingScannedQty ?? 0 }} 件</div>
      <a-button :loading="exporting" @click="exportCsv">导出 CSV</a-button>
      <a-table row-key="batchId" :columns="columns" :data-source="rows" :loading="loading" :pagination="{ current: pager.current, pageSize: pager.pageSize, total: pager.total, showSizeChanger: true }" :scroll="{ x: 1800 }" @change="change">
        <template #bodyCell="{ column, record }"><a v-if="column.dataIndex === 'lotNo'" @click="router.push({ path: '/scheduling/batches', query: { keyword: record.lotNo } })">{{ record.lotNo }}</a>
          <span v-else-if="column.dataIndex === 'createdAt'">{{ record.createdAt.slice(0, 10) }}</span>
          <span v-else-if="column.dataIndex === 'spec'">{{ record.thickness }} / {{ record.color }} / {{ record.sprayingType }}</span>
          <span v-else-if="column.dataIndex === 'planned'">{{ record.itemQty }} / {{ record.itemArea }}</span>
          <div v-else-if="column.dataIndex === 'cutting' || column.dataIndex === 'shipping'"><a-progress :percent="Number(record[column.dataIndex].percent.toFixed(1))" size="small" /><small>已扫 {{ record[column.dataIndex].scannedQty }} 件 / {{ record[column.dataIndex].scannedArea }} m² · 未扫 {{ record[column.dataIndex].unscannedQty }} 件 / {{ record[column.dataIndex].unscannedArea }} m²</small></div>
        </template>
      </a-table></PageCard></div>
</template>
<style scoped>.report{display:grid;gap:16px}.filters{display:flex;align-items:end;flex-wrap:wrap;gap:12px}.filters label{display:grid;gap:6px}.summary{margin-bottom:12px;font-weight:600}small{display:block;color:#888}</style>
