<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { message } from 'ant-design-vue'
import type { TablePaginationConfig } from 'ant-design-vue'
import { call, client, errorMessage, type Schemas } from '@/api/client'
import { downloadCsv, exportFilename, fetchAllPages } from '@/utils/export'
import PageCard from '@/components/PageCard.vue'
import TableToolbar from '@/components/TableToolbar.vue'
const props = defineProps<{ kind: 'semi' | 'finished' }>()
type Row = Schemas['InventoryRow']
const title = computed(() => props.kind === 'semi' ? '半成品库存' : '成品库存')
const filters = reactive({ dates: [] as string[], keyword: '', hasStock: true })
const applied = reactive({ ...filters })
const items = ref<Row[]>([])
const totals = ref<Schemas['InventoryReport']['totals']>()
const pager = reactive({ current: 1, pageSize: 20, total: 0 })
const loading = ref(false), exporting = ref(false)
const columns = computed(() => [
  { title: '批次', dataIndex: 'lotNo' }, { title: '工程名称', dataIndex: 'projectName' },
  { title: '客户订单号', dataIndex: 'clientOrderNo' }, { title: '下单日期', dataIndex: 'orderDate' },
  { title: '厚度', dataIndex: 'thickness' }, { title: '颜色', dataIndex: 'color' },
  { title: '序号', dataIndex: 'seq' }, { title: '构件编号', dataIndex: 'pieceNo' }, { title: '图号', dataIndex: 'drawingNo' },
  { title: '计划数量', dataIndex: 'planQty' }, { title: '单件面积 m²', dataIndex: 'pieceArea' }, { title: '计划面积 m²', dataIndex: 'planArea' },
  { title: props.kind === 'semi' ? '本期入库(钣金下线)' : '本期入库(下挂)', dataIndex: 'inQty' },
  { title: props.kind === 'semi' ? '本期出库(下挂)' : '本期出库(发货)', dataIndex: 'outQty' },
  { title: '库存数量', dataIndex: 'stockQty' }, { title: '库存面积 m²', dataIndex: 'stockArea' },
])
function fetchPage(page: number, size: number) {
  return call(client.GET('/inventory', { params: { query: { kind: props.kind, from: applied.dates[0], to: applied.dates[1], keyword: applied.keyword.trim() || undefined, hasStock: applied.hasStock, page, size } } }))
}
async function load() { loading.value = true; try { const result = await fetchPage(pager.current, pager.pageSize); items.value = result.items; totals.value = result.totals; pager.total = result.total } catch (e) { message.error(errorMessage(e)) } finally { loading.value = false } }
function search() { Object.assign(applied, filters, { dates: [...filters.dates] }); pager.current = 1; void load() }
function reset() { filters.dates = []; filters.keyword = ''; filters.hasStock = true; search() }
function change(p: TablePaginationConfig) { pager.current = p.current ?? 1; pager.pageSize = p.pageSize ?? 20; void load() }
async function exportCsv() { exporting.value = true; try { const all = await fetchAllPages(fetchPage); downloadCsv(exportFilename(title.value), columns.value.map((c) => ({ title: c.title, value: (r: Row) => String(r[c.dataIndex as keyof Row] ?? '') })), all); message.success(`已导出 ${all.length} 条`) } catch (e) { message.error(errorMessage(e)) } finally { exporting.value = false } }
watch(() => props.kind, () => { pager.current = 1; void load() }, { immediate: true })
</script>
<template>
  <div class="report">
    <div class="heading"><h1>{{ title }}</h1></div>
    <PageCard><div class="filters"><label>扫描日期<a-range-picker v-model:value="filters.dates" value-format="YYYY-MM-DD" /></label><label>快速查询<a-input v-model:value="filters.keyword" placeholder="批次 / 工程 / 订单 / 构件 / 图号" allow-clear @press-enter="search" /></label><label>有库存<a-switch v-model:checked="filters.hasStock" /></label><div class="actions"><a-button type="primary" @click="search">查询</a-button><a-button @click="reset">重置</a-button></div></div></PageCard>
    <PageCard><TableToolbar><template #left><span data-testid="inventory-totals">库存 {{ totals?.stockQty ?? 0 }} 件 · {{ (totals?.stockArea ?? 0).toFixed(2) }} m² · 本期入库 {{ totals?.inQty ?? 0 }} · 本期出库 {{ totals?.outQty ?? 0 }}</span></template><template #right><a-button data-testid="export-csv" :loading="exporting" @click="exportCsv">导出 CSV</a-button></template></TableToolbar><a-table :row-key="(r: Row) => r.batchItemId" :columns="columns" :data-source="items" :loading="loading" :pagination="{ current: pager.current, pageSize: pager.pageSize, total: pager.total, showSizeChanger: true }" :scroll="{ x: 2000 }" @change="change" /></PageCard>
  </div>
</template>
<style scoped>.filters{display:flex;gap:16px;align-items:end;flex-wrap:wrap}.filters label{display:flex;flex-direction:column;gap:6px}.actions{display:flex;gap:8px}.heading{margin-bottom:16px}</style>
