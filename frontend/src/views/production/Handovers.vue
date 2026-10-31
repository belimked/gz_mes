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

type Handover = Schemas['Handover']
const filters = reactive({
  keyword: '',
  batchId: undefined as number | undefined,
  jobId: undefined as number | undefined,
  received: undefined as boolean | undefined,
})
const applied = reactive({ ...filters })
const batches = ref<Schemas['Batch'][]>([])
const jobs = ref<Schemas['Job'][]>([])
const rows = ref<Handover[]>([])
const pager = reactive({ current: 1, pageSize: 20, total: 0 })
const loading = ref(false)
const selected = ref<Schemas['HandoverDetail']>()
const detailLoading = ref(false)
const createOpen = ref(false)
const saving = ref(false)
const form = reactive({ batchId: undefined as number | undefined, jobCode: undefined as string | undefined, barCodes: '', remark: '' })
const codes = (text: string) => [
  ...new Set(
    text
      .split(/[\s,，]+/)
      .map((s) => s.trim())
      .filter(Boolean),
  ),
]
const columns = [
  { title: '交接单号', dataIndex: 'handoverNo', width: 170 },
  { title: '批次号', dataIndex: 'lotNo', width: 150 },
  { title: '工程名称', dataIndex: 'projectName', width: 180 },
  { title: '工序', dataIndex: 'jobName', width: 110 },
  { title: '标签数', dataIndex: 'qty', width: 90 },
  { title: '面积', dataIndex: 'area', width: 110 },
  { title: '打包人', dataIndex: 'packedBy', width: 100 },
  { title: '打包时间', dataIndex: 'packedAt', width: 170 },
  { title: '接收状态', dataIndex: 'received', width: 100 },
  { title: '操作', key: 'action', width: 130, fixed: 'right' as const },
]
const labelColumns = [
  { title: '条码', dataIndex: 'barCode' },
  { title: '编号', dataIndex: 'pieceNo' },
  { title: '第几件', dataIndex: 'pieceIndex' },
  { title: '颜色', dataIndex: 'color' },
  { title: '单件面积', dataIndex: 'pieceArea' },
]
async function load() {
  loading.value = true
  try {
    const result = await call(
      client.GET('/handovers', {
        params: {
          query: {
            page: pager.current,
            size: pager.pageSize,
            keyword: applied.keyword.trim() || undefined,
            batchId: applied.batchId,
            jobId: applied.jobId,
            received: applied.received,
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
  Object.assign(filters, { keyword: '', batchId: undefined, jobId: undefined, received: undefined })
  search()
}
function onTableChange(p: TablePaginationConfig) {
  pager.current = p.current ?? 1
  pager.pageSize = p.pageSize ?? 20
  void load()
}
function openCreate() {
  Object.assign(form, { batchId: undefined, jobCode: undefined, barCodes: '', remark: '' })
  createOpen.value = true
}
async function save() {
  if (!form.batchId || !form.jobCode || !codes(form.barCodes).length) {
    message.warning('请选择批次、工序并填写条码')
    return
  }
  saving.value = true
  try {
    await call(
      client.POST('/handovers', {
        body: { batchId: form.batchId, jobCode: form.jobCode, barCodes: codes(form.barCodes), remark: form.remark },
      }),
    )
    message.success('交接单已创建')
    createOpen.value = false
    search()
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    saving.value = false
  }
}
async function openDetail(row: Handover) {
  detailLoading.value = true
  try {
    selected.value = await call(client.GET('/handovers/{id}', { params: { path: { id: row.id } } }))
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    detailLoading.value = false
  }
}
function closeDetail() {
  selected.value = undefined
}
async function receive(row: Handover) {
  try {
    await call(client.POST('/handovers/{id}/receive', { params: { path: { id: row.id } } }))
    message.success('已接收')
    await load()
    if (selected.value?.id === row.id) await openDetail(row)
  } catch (e) {
    message.error(errorMessage(e))
  }
}
onMounted(async () => {
  void load()
  try {
    const [b, j] = await Promise.all([call(client.GET('/batches', { params: { query: { size: 200 } } })), call(client.GET('/jobs'))])
    batches.value = b.items
    jobs.value = j
  } catch (e) {
    message.error(errorMessage(e))
  }
})
</script>
<template>
  <div class="page">
    <div class="heading">
      <div>
        <h1>交接单管理</h1>
        <span>查询、创建及接收生产交接单</span>
      </div>
      <StatusTag tone="processing" :label="`共 ${pager.total} 张`" />
    </div>
    <FilterCard @search="search" @reset="reset">
      <label
        >关键字<a-input v-model:value="filters.keyword" placeholder="交接单号 / 批次号 / 工程" allow-clear @press-enter="search"
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
        >工序<a-select
          v-model:value="filters.jobId"
          allow-clear
          placeholder="全部工序"
          :options="jobs.map((j) => ({ value: j.id, label: j.name }))"
      /></label>
      <label
        >是否接收<a-select
          v-model:value="filters.received"
          allow-clear
          placeholder="全部"
          :options="[
            { value: true, label: '已接收' },
            { value: false, label: '待接收' },
          ]"
      /></label>
    </FilterCard>
    <PageCard
      ><TableToolbar
        ><template #left><a-button type="primary" @click="openCreate">新建交接单</a-button></template
        ><template #right><a-button @click="load">刷新</a-button></template></TableToolbar
      >
      <a-table
        row-key="id"
        :columns="columns"
        :data-source="rows"
        :loading="loading"
        :scroll="{ x: 1350 }"
        :pagination="{ ...pager, showSizeChanger: true }"
        data-testid="handover-table"
        @change="onTableChange"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'handoverNo'"
            ><a @click="openDetail(record)"><CodeText :value="record.handoverNo" /></a
          ></template>
          <template v-else-if="column.dataIndex === 'lotNo'"><CodeText :value="record.lotNo" /></template>
          <template v-else-if="column.dataIndex === 'area'">{{ Number(record.area).toFixed(2) }} m²</template>
          <template v-else-if="column.dataIndex === 'packedAt'">{{ dayjs(record.packedAt).format('YYYY-MM-DD HH:mm') }}</template>
          <template v-else-if="column.dataIndex === 'received'"
            ><StatusTag :tone="record.received ? 'success' : 'warning'" :label="record.received ? '已接收' : '待接收'"
          /></template>
          <template v-else-if="column.key === 'action'"
            ><a-popconfirm v-if="!record.received" title="确认接收此交接单？" @confirm="receive(record)"><a>接收</a></a-popconfirm
            ><a-divider v-if="!record.received" type="vertical" /><a @click="openDetail(record)">详情</a></template
          >
        </template>
      </a-table>
    </PageCard>
    <a-drawer :open="!!selected || detailLoading" :width="760" :title="`交接单详情: ${selected?.handoverNo ?? ''}`" @close="closeDetail"
      ><a-spin :spinning="detailLoading"
        ><template v-if="selected">
          <a-descriptions bordered size="small" :column="2">
            <a-descriptions-item label="交接单号">{{ selected.handoverNo }}</a-descriptions-item
            ><a-descriptions-item label="接收状态"
              ><StatusTag :tone="selected.received ? 'success' : 'warning'" :label="selected.received ? '已接收' : '待接收'"
            /></a-descriptions-item>
            <a-descriptions-item label="批次号">{{ selected.lotNo }}</a-descriptions-item
            ><a-descriptions-item label="工程名称">{{ selected.projectName }}</a-descriptions-item>
            <a-descriptions-item label="工序">{{ selected.jobName }}</a-descriptions-item
            ><a-descriptions-item label="标签数 / 面积">{{ selected.qty }} / {{ Number(selected.area).toFixed(2) }} m²</a-descriptions-item>
            <a-descriptions-item label="打包人 / 时间">{{ selected.packedBy }} · {{ selected.packedAt }}</a-descriptions-item
            ><a-descriptions-item label="接收人 / 时间"
              >{{ selected.receivedBy || '—' }} · {{ selected.receivedAt || '—' }}</a-descriptions-item
            >
            <a-descriptions-item label="备注" :span="2">{{ selected.remark || '—' }}</a-descriptions-item>
          </a-descriptions>
          <h3>标签明细</h3>
          <a-table row-key="id" size="small" :columns="labelColumns" :data-source="selected.labels" :pagination="false"
            ><template #bodyCell="{ column, record }"
              ><template v-if="column.dataIndex === 'pieceArea'">{{ Number(record.pieceArea).toFixed(4) }} m²</template
              ><template v-else-if="column.dataIndex === 'pieceIndex'">{{ record.pieceIndex }} / {{ record.qty }}</template></template
            ></a-table
          >
        </template></a-spin
      ></a-drawer
    >
    <a-modal v-model:open="createOpen" title="新建交接单" :confirm-loading="saving" @ok="save"
      ><div class="form">
        <label
          >批次 *<a-select
            v-model:value="form.batchId"
            show-search
            option-filter-prop="label"
            placeholder="请选择批次"
            :options="batches.map((b) => ({ value: b.id, label: `${b.lotNo} · ${b.projectName}` }))"
        /></label>
        <label
          >工序 *<a-select
            v-model:value="form.jobCode"
            placeholder="请选择工序"
            :options="jobs.map((j) => ({ value: j.code, label: j.name }))"
        /></label>
        <label>条码 *<a-textarea v-model:value="form.barCodes" :rows="5" placeholder="每行一个条码，也支持逗号或空格分隔" /></label>
        <label>备注<a-textarea v-model:value="form.remark" :rows="2" /></label></div
    ></a-modal>
  </div>
</template>
<style scoped>
.page {
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
  font-size: 22px;
}
.heading span {
  color: #8c8c8c;
}
.form {
  display: grid;
  gap: 16px;
}
.form label {
  display: grid;
  gap: 6px;
}
h3 {
  margin-top: 24px;
}
</style>
