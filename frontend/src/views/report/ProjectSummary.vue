<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { message } from 'ant-design-vue'
import type { TablePaginationConfig } from 'ant-design-vue'
import { call, client, errorMessage, type Schemas } from '@/api/client'
import { downloadCsv, exportFilename, fetchAllPages } from '@/utils/export'
import PageCard from '@/components/PageCard.vue'
import TableToolbar from '@/components/TableToolbar.vue'
type Row = Schemas['ProjectSummaryRow']
const filters = reactive({ dates: [] as string[], keyword: '', status: 'all' })
const applied = reactive({ ...filters })
const selected = ref<number[]>([]), jobs = ref<Schemas['Job'][]>([])
const initialized = ref(false)
const items = ref<Row[]>([]), summary = ref<Schemas['ProjectSummaryReport']['summary']>()
const pager = reactive({ current: 1, pageSize: 20, total: 0 })
const loading = ref(false), exporting = ref(false)
const visibleJobs = computed(() => jobs.value.filter((j) => selected.value.includes(j.id)))
const columns = computed(() => [
  { title: '行号', dataIndex: 'rowNo' }, { title: '工程名称', dataIndex: 'projectName' },
  { title: '客户', dataIndex: 'customerName' }, { title: '批次数', dataIndex: 'batchCount' },
  { title: '计划数量', dataIndex: 'planQty' }, { title: '计划面积 m²', dataIndex: 'planArea' },
  { title: '工程状态', dataIndex: 'status' },
  ...visibleJobs.value.flatMap((j) => [{ title: `${j.name} 数量`, dataIndex: `qty-${j.id}` }, { title: `${j.name} 面积 m²`, dataIndex: `area-${j.id}` }]),
])
const statusName: Record<string, string> = { pending: '待生产', in_production: '生产中', done: '已完成' }
const display = computed(() => items.value.map((r, i) => ({ ...r, status: statusName[r.status] ?? r.status, rowNo: (pager.current - 1) * pager.pageSize + i + 1, ...Object.fromEntries(r.cells.flatMap((c) => [[`qty-${c.jobId}`, c.qty], [`area-${c.jobId}`, c.area.toFixed(2)]])) })))
function fetchPage(page: number, size: number) { return call(client.GET('/reports/project-summary', { params: { query: { from: applied.dates[0], to: applied.dates[1], keyword: applied.keyword.trim() || undefined, status: applied.status, page, size } } })) }
async function load() { loading.value = true; try { const result = await fetchPage(pager.current, pager.pageSize); items.value = result.items; jobs.value = result.jobs; summary.value = result.summary; pager.total = result.total; if (!initialized.value) { selected.value = result.jobs.map((j) => j.id); initialized.value = true } } catch (e) { message.error(errorMessage(e)) } finally { loading.value = false } }
function search() { Object.assign(applied, filters, { dates: [...filters.dates] }); pager.current = 1; void load() }
function reset() { filters.dates = []; filters.keyword = ''; filters.status = 'all'; search() }
function change(p: TablePaginationConfig) { pager.current = p.current ?? 1; pager.pageSize = p.pageSize ?? 20; void load() }
async function exportCsv() { exporting.value = true; try { const all = await fetchAllPages(fetchPage); downloadCsv(exportFilename('项目汇总'), [
  { title: '工程名称', value: (r: Row) => r.projectName }, { title: '客户', value: (r: Row) => r.customerName },
  { title: '批次数', value: (r: Row) => r.batchCount }, { title: '计划数量', value: (r: Row) => r.planQty },
  { title: '计划面积', value: (r: Row) => r.planArea }, { title: '工程状态', value: (r: Row) => statusName[r.status] ?? r.status },
  ...visibleJobs.value.flatMap((j) => [{ title: `${j.name}数量`, value: (r: Row) => r.cells.find((c) => c.jobId === j.id)?.qty ?? 0 }, { title: `${j.name}面积`, value: (r: Row) => r.cells.find((c) => c.jobId === j.id)?.area ?? 0 }]),
], all); message.success(`已导出 ${all.length} 条`) } catch (e) { message.error(errorMessage(e)) } finally { exporting.value = false } }
onMounted(load)
</script>
<template>
  <div class="report"><div class="heading"><h1>项目汇总</h1></div>
    <PageCard><div class="filters"><label>批次下单日期<a-range-picker v-model:value="filters.dates" value-format="YYYY-MM-DD" /></label><label>快速查询<a-input v-model:value="filters.keyword" placeholder="项目名称 / 编码" allow-clear @press-enter="search" /></label><label>发货状态<a-select v-model:value="filters.status" style="width:160px" :options="[{ value: 'all', label: '全部' }, { value: 'not_done', label: '未完成' }, { value: 'done', label: '已完成' }]" /></label><div class="actions"><a-button type="primary" @click="search">查询</a-button><a-button @click="reset">重置</a-button></div></div></PageCard>
    <PageCard><TableToolbar><template #left><span data-testid="project-summary-totals">批次 {{ summary?.batchCount ?? 0 }} · 计划 {{ summary?.planQty ?? 0 }} 件 / {{ (summary?.planArea ?? 0).toFixed(2) }} m² · 标签 {{ summary?.labelQty ?? 0 }}</span></template><template #right><a-select v-model:value="selected" mode="multiple" style="min-width:220px;max-width:450px" placeholder="选择工序列" :options="jobs.map(j => ({ value: j.id, label: j.name }))" /><a-button data-testid="export-csv" :loading="exporting" @click="exportCsv">导出 CSV</a-button></template></TableToolbar><a-table :row-key="(r: Row) => r.projectId" :columns="columns" :data-source="display" :loading="loading" :pagination="{ current: pager.current, pageSize: pager.pageSize, total: pager.total, showSizeChanger: true }" :scroll="{ x: 1300 }" @change="change"><template #bodyCell="{ column, record }"><router-link v-if="column.dataIndex === 'projectName'" :to="{ name: 'batches', query: { keyword: record.projectName } }">{{ record.projectName }}</router-link></template></a-table></PageCard>
  </div>
</template>
<style scoped>.filters{display:flex;gap:16px;align-items:end;flex-wrap:wrap}.filters label{display:flex;flex-direction:column;gap:6px}.actions{display:flex;gap:8px}.heading{margin-bottom:16px}</style>
