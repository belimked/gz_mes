<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { message } from 'ant-design-vue'
import type { TablePaginationConfig } from 'ant-design-vue'
import dayjs from 'dayjs'
import { call, client, errorMessage, type Schemas } from '@/api/client'
import PageCard from '@/components/PageCard.vue'
import FilterCard from '@/components/FilterCard.vue'
import TableToolbar from '@/components/TableToolbar.vue'
import CodeText from '@/components/CodeText.vue'
import StatusTag from '@/components/StatusTag.vue'

type Label = Schemas['Label']
const filters = reactive({ keyword: '', batchId: undefined as number | undefined, cancelled: undefined as boolean | undefined })
const applied = reactive({ ...filters })
const batches = ref<Schemas['Batch'][]>([])
const rows = ref<Label[]>([])
const pager = reactive({ current: 1, pageSize: 20, total: 0 })
const loading = ref(false)
const selected = ref<Schemas['LabelDetail']>()
const detailLoading = ref(false)
const generateOpen = ref(false)
const generateBatchId = ref<number>()
function closeDetail() {
  selected.value = undefined
  detailLoading.value = false
}
function openGenerate() {
  generateBatchId.value = undefined
  generateOpen.value = true
}
const generating = ref(false)
const columns = [
  { title: '条码', dataIndex: 'barCode', width: 165 },
  { title: '批次号', dataIndex: 'lotNo', width: 150 },
  { title: '工程名称', dataIndex: 'projectName', width: 180 },
  { title: '编号', dataIndex: 'pieceNo', width: 110 },
  { title: '第几件', dataIndex: 'pieceIndex', width: 90 },
  { title: '图号', dataIndex: 'drawingNo', width: 120 },
  { title: '厚度', dataIndex: 'thickness', width: 80 },
  { title: '颜色', dataIndex: 'color', width: 95 },
  { title: '单件面积', dataIndex: 'pieceArea', width: 100 },
  { title: '最近工序', dataIndex: 'lastJobName', width: 110 },
  { title: '最近扫描时间', dataIndex: 'lastScannedAt', width: 165 },
  { title: '状态', dataIndex: 'cancelled', width: 90 },
  { title: '操作', key: 'action', width: 130, fixed: 'right' as const },
]
async function load() {
  loading.value = true
  try {
    const result = await call(
      client.GET('/labels', {
        params: {
          query: {
            page: pager.current,
            size: pager.pageSize,
            keyword: applied.keyword.trim() || undefined,
            batchId: applied.batchId,
            cancelled: applied.cancelled,
          },
        },
      }),
    )
    rows.value = result.items
    pager.total = result.total
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
  Object.assign(filters, { keyword: '', batchId: undefined, cancelled: undefined })
  search()
}
function onTableChange(p: TablePaginationConfig) {
  pager.current = p.current ?? 1
  pager.pageSize = p.pageSize ?? 20
  void load()
}
async function openDetail(row: Label) {
  detailLoading.value = true
  try {
    selected.value = await call(client.GET('/labels/{barCode}', { params: { path: { barCode: row.barCode } } }))
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    detailLoading.value = false
  }
}
async function cancel(row: Label) {
  try {
    await call(client.POST('/labels/{barCode}/cancel', { params: { path: { barCode: row.barCode } } }))
    message.success('标签已作废')
    await load()
  } catch (e) {
    message.error(errorMessage(e))
  }
}
async function generate() {
  if (generateBatchId.value === undefined) return
  generating.value = true
  try {
    const result = await call(client.POST('/batches/{id}/labels/generate', { params: { path: { id: generateBatchId.value } } }))
    message.success(`新建 ${result.created} 张，共 ${result.total} 张`)
    generateOpen.value = false
    await load()
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    generating.value = false
  }
}
onMounted(async () => {
  void load()
  try {
    batches.value = (await call(client.GET('/batches', { params: { query: { size: 200 } } }))).items
  } catch (e) {
    message.error(errorMessage(e))
  }
})
</script>
<template>
  <div class="labels">
    <div class="heading">
      <div>
        <h1>标签管理</h1>
        <span>按批次查询标签及工序扫描记录</span>
      </div>
      <StatusTag tone="processing" :label="`共 ${pager.total} 张标签`" />
    </div>
    <FilterCard @search="search" @reset="reset"
      ><label
        >关键字<a-input
          v-model:value="filters.keyword"
          data-testid="label-search"
          placeholder="条码 / 编号 / 批次号 / 工程"
          allow-clear
          @press-enter="search"
      /></label>
      <label
        >批次<a-select
          v-model:value="filters.batchId"
          show-search
          option-filter-prop="label"
          allow-clear
          placeholder="全部批次"
          :options="batches.map((b) => ({ value: b.id, label: `${b.lotNo} · ${b.projectName}` }))"
      /></label>
      <label
        >是否作废<a-select
          v-model:value="filters.cancelled"
          allow-clear
          placeholder="全部"
          :options="[
            { value: false, label: '正常' },
            { value: true, label: '已作废' },
          ]"
      /></label>
    </FilterCard>
    <PageCard
      ><TableToolbar
        ><template #left><a-button type="primary" @click="openGenerate">生成标签</a-button><a-button disabled>导出</a-button></template
        ><template #right><a-button @click="load">刷新</a-button></template></TableToolbar
      >
      <a-table
        row-key="id"
        size="middle"
        :columns="columns"
        :data-source="rows"
        :loading="loading"
        :scroll="{ x: 1600 }"
        :pagination="{ ...pager, showSizeChanger: true, showTotal: (n: number) => `共 ${n} 条` }"
        data-testid="label-table"
        @change="onTableChange"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'barCode'"
            ><a @click="openDetail(record)"><CodeText :value="record.barCode" /></a
          ></template>
          <template v-else-if="column.dataIndex === 'lotNo'"><CodeText :value="record.lotNo" /></template>
          <template v-else-if="column.dataIndex === 'pieceIndex'">{{ record.pieceIndex }} / {{ record.qty }}</template>
          <template v-else-if="column.dataIndex === 'thickness'">{{ record.thickness }} mm</template>
          <template v-else-if="column.dataIndex === 'pieceArea'">{{ Number(record.pieceArea).toFixed(4) }} m²</template>
          <template v-else-if="column.dataIndex === 'lastJobName'">{{ record.lastJobName || '—' }}</template>
          <template v-else-if="column.dataIndex === 'lastScannedAt'">{{
            record.lastScannedAt ? dayjs(record.lastScannedAt).format('YYYY-MM-DD HH:mm:ss') : '—'
          }}</template>
          <template v-else-if="column.dataIndex === 'cancelled'"
            ><StatusTag :tone="record.cancelled ? 'error' : 'success'" :label="record.cancelled ? '作废' : '正常'"
          /></template>
          <template v-else-if="column.key === 'action'"
            ><a-popconfirm v-if="!record.cancelled && !record.lastJobId" title="确认作废此标签？" @confirm="cancel(record)"
              ><a>作废</a></a-popconfirm
            ><a-divider v-if="!record.cancelled && !record.lastJobId" type="vertical" /><a @click="openDetail(record)">详情</a></template
          >
        </template>
      </a-table>
    </PageCard>
    <a-drawer :open="!!selected || detailLoading" :width="640" :title="`标签详情: ${selected?.barCode ?? ''}`" @close="closeDetail">
      <a-spin :spinning="detailLoading"
        ><template v-if="selected"
          ><a-descriptions bordered size="small" :column="2">
            <a-descriptions-item label="条码"><CodeText :value="selected.barCode" /></a-descriptions-item
            ><a-descriptions-item label="状态"
              ><StatusTag :tone="selected.cancelled ? 'error' : 'success'" :label="selected.cancelled ? '作废' : '正常'"
            /></a-descriptions-item>
            <a-descriptions-item label="批次号">{{ selected.lotNo }}</a-descriptions-item
            ><a-descriptions-item label="工程">{{ selected.projectName }}</a-descriptions-item>
            <a-descriptions-item label="编号">{{ selected.pieceNo }}</a-descriptions-item
            ><a-descriptions-item label="第几件">{{ selected.pieceIndex }} / {{ selected.qty }}</a-descriptions-item>
            <a-descriptions-item label="图号">{{ selected.drawingNo || '—' }}</a-descriptions-item
            ><a-descriptions-item label="厚度">{{ selected.thickness }} mm</a-descriptions-item>
            <a-descriptions-item label="颜色">{{ selected.color || '—' }}</a-descriptions-item
            ><a-descriptions-item label="单件面积">{{ Number(selected.pieceArea).toFixed(4) }} m²</a-descriptions-item>
            <a-descriptions-item label="最近工序">{{ selected.lastJobName || '—' }}</a-descriptions-item
            ><a-descriptions-item label="最近扫描时间">{{
              selected.lastScannedAt ? dayjs(selected.lastScannedAt).format('YYYY-MM-DD HH:mm:ss') : '—'
            }}</a-descriptions-item>
          </a-descriptions>
          <h3>扫描历史</h3>
          <a-empty v-if="!selected.scans.length" description="暂无扫描记录" /><a-timeline v-else
            ><a-timeline-item v-for="scan in selected.scans" :key="scan.id"
              >{{ scan.jobName }} · {{ scan.scannedBy }} · {{ dayjs(scan.scannedAt).format('YYYY-MM-DD HH:mm:ss') }}</a-timeline-item
            ></a-timeline
          ></template
        ></a-spin
      >
    </a-drawer>
    <a-modal
      v-model:open="generateOpen"
      title="生成标签"
      ok-text="生成"
      :ok-button-props="{ disabled: generateBatchId === undefined }"
      :confirm-loading="generating"
      @ok="generate"
      ><label
        >选择批次<a-select
          v-model:value="generateBatchId"
          show-search
          option-filter-prop="label"
          placeholder="请选择批次"
          :options="batches.map((b) => ({ value: b.id, label: `${b.lotNo} · ${b.projectName}` }))" /></label
    ></a-modal>
  </div>
</template>
<style scoped>
.labels {
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
.heading span {
  color: var(--gz-muted);
  font-size: 12px;
}
label {
  display: grid;
  gap: 6px;
  min-width: 180px;
  color: var(--gz-secondary);
  font-size: 12px;
}
h3 {
  margin: 24px 0 16px;
}
</style>
