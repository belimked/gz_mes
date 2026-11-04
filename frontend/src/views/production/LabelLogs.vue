<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import dayjs from 'dayjs'
import { call, client, errorMessage, type Schemas } from '@/api/client'
import PageCard from '@/components/PageCard.vue'
import FilterCard from '@/components/FilterCard.vue'
import { downloadCsv, exportFilename, fetchAllPages } from '@/utils/export'

type Row = Schemas['LabelLog']
const router = useRouter()
const filters = reactive({
  dates: [] as string[],
  opType: undefined as 'generate' | 'cancel' | 'reprint' | undefined,
  keyword: '',
  operator: '',
})
const applied = reactive({ ...filters })
const rows = ref<Row[]>([])
const totals = reactive({ generateQty: 0, cancelQty: 0, reprintQty: 0, area: 0 })
const pager = reactive({ current: 1, pageSize: 20, total: 0 })
const loading = ref(false)
const names = { generate: '生成', cancel: '作废', reprint: '重打' }
const columns = [
  '操作人',
  '工号',
  '操作时间',
  '条码号',
  '工程名称',
  '批次',
  '编号',
  '序号',
  '图号',
  '包装数量',
  '单件展开面积',
  '板型',
  '厚度',
  '颜色',
  '类型',
  '原因',
].map((title, i) => ({
  title,
  dataIndex: [
    'operatorName',
    'staffNo',
    'createdAt',
    'barCode',
    'projectName',
    'lotNo',
    'pieceNo',
    'pieceIndex',
    'drawingNo',
    'qty',
    'pieceArea',
    'pattern',
    'thickness',
    'color',
    'opType',
    'reason',
  ][i],
}))
function fetchPage(page: number, size: number) {
  return call(
    client.GET('/label-logs', {
      params: {
        query: {
          page,
          size,
          from: applied.dates[0],
          to: applied.dates[1],
          opType: applied.opType,
          keyword: applied.keyword || undefined,
          operator: applied.operator || undefined,
        },
      },
    }),
  )
}
async function load() {
  loading.value = true
  try {
    const r = await fetchPage(pager.current, pager.pageSize)
    rows.value = r.items
    pager.total = r.total
    Object.assign(totals, r.totals)
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
  Object.assign(filters, { dates: [], opType: undefined, keyword: '', operator: '' })
  search()
}
async function exportCsv() {
  try {
    const all = await fetchAllPages(fetchPage)
    downloadCsv(
      exportFilename('标签日志', [applied.dates[0], applied.dates[1]]),
      columns.map((c) => ({ title: c.title, value: (r: Row) => String(r[c.dataIndex as keyof Row] ?? '') })),
      all,
    )
  } catch (e) {
    message.error(errorMessage(e))
  }
}
onMounted(load)
</script>
<template>
  <h1>标签日志</h1>
  <FilterCard @search="search" @reset="reset">
    <label
      >日期范围<a-range-picker
        :value="filters.dates.length ? [dayjs(filters.dates[0]), dayjs(filters.dates[1])] : undefined"
        @change="(_date: unknown, dates: [string, string]) => (filters.dates = dates || [])"
    /></label>
    <label
      >类型<a-select
        v-model:value="filters.opType"
        allow-clear
        :options="Object.entries(names).map(([value, label]) => ({ value, label }))"
    /></label>
    <label>关键字<a-input v-model:value="filters.keyword" placeholder="条码 / 批次 / 工程 / 编号 / 图号" @press-enter="search" /></label>
    <label>操作人<a-input v-model:value="filters.operator" @press-enter="search" /></label>
  </FilterCard>
  <a-row :gutter="16" style="margin: 16px 0"
    ><a-col
      v-for="(n, key) in { 生成: totals.generateQty, 作废: totals.cancelQty, 重打: totals.reprintQty, '面积 m²': totals.area.toFixed(4) }"
      :key="key"
      :span="6"
      ><a-card><a-statistic :title="key" :value="n" /></a-card></a-col
  ></a-row>
  <PageCard
    ><a-button @click="exportCsv">导出 CSV</a-button
    ><a-table
      row-key="id"
      :columns="columns"
      :data-source="rows"
      :loading="loading"
      :scroll="{ x: 1800 }"
      :pagination="{ ...pager, showSizeChanger: true }"
      @change="
        (p: { current?: number; pageSize?: number }) => {
          pager.current = p.current ?? 1
          pager.pageSize = p.pageSize ?? 20
          load()
        }
      "
      ><template #bodyCell="{ column, record }"
        ><template v-if="column.dataIndex === 'barCode'"
          ><a @click="router.push({ path: '/production/labels', query: { keyword: record.barCode } })">{{ record.barCode }}</a></template
        ><template v-else-if="column.dataIndex === 'createdAt'">{{ dayjs(record.createdAt).format('YYYY-MM-DD HH:mm:ss') }}</template
        ><template v-else-if="column.dataIndex === 'opType'"
          ><a-tag :color="record.opType === 'cancel' ? 'red' : 'blue'">{{ names[record.opType as keyof typeof names] }}</a-tag></template
        ></template
      ></a-table
    ></PageCard
  >
</template>
