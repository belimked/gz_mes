<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { message } from 'ant-design-vue'
import type { TablePaginationConfig } from 'ant-design-vue'
import dayjs from 'dayjs'
import { call, client, errorMessage, type Schemas } from '@/api/client'
import PageCard from '@/components/PageCard.vue'
import FilterCard from '@/components/FilterCard.vue'
import TableToolbar from '@/components/TableToolbar.vue'
import CodeText from '@/components/CodeText.vue'
import StatusTag from '@/components/StatusTag.vue'

type Note = Schemas['DeliveryNote']
const filters = reactive({
  keyword: '',
  customerId: undefined as number | undefined,
  batchId: undefined as number | undefined,
  accounted: undefined as boolean | undefined,
  dates: [] as string[],
})
const applied = reactive({ ...filters })
const customers = ref<Schemas['Customer'][]>([])
const batches = ref<Schemas['Batch'][]>([])
const rows = ref<Note[]>([])
const pager = reactive({ current: 1, pageSize: 20, total: 0 })
const loading = ref(false)
const selected = ref<Schemas['DeliveryNoteDetail']>()
const detailLoading = ref(false)
const createOpen = ref(false)
const saving = ref(false)
const form = reactive({
  batchId: undefined as number | undefined,
  barCodes: '',
  deliveryBatch: '',
  logisticsCompany: '',
  plateNo: '',
  driver: '',
  driverPhone: '',
  deliveredAt: dayjs().format('YYYY-MM-DD'),
  remark: '',
})
const codes = (text: string) => [
  ...new Set(
    text
      .split(/[\s,，]+/)
      .map((s) => s.trim())
      .filter(Boolean),
  ),
]
const totals = computed(() => ({
  qty: rows.value.reduce((n, r) => n + r.qty, 0),
  area: rows.value.reduce((n, r) => n + Number(r.area), 0).toFixed(2),
}))
const columns = [
  { title: '发货单号', dataIndex: 'noteNo', width: 170 },
  { title: '批次号 / 工程', dataIndex: 'lotNo', width: 190 },
  { title: '客户', dataIndex: 'customerName', width: 150 },
  { title: '合同号', dataIndex: 'contractNo', width: 140 },
  { title: '发货批次', dataIndex: 'deliveryBatch', width: 100 },
  { title: '件数', dataIndex: 'qty', width: 80 },
  { title: '面积', dataIndex: 'area', width: 100 },
  { title: '物流公司 / 车牌', dataIndex: 'logisticsCompany', width: 150 },
  { title: '司机', dataIndex: 'driver', width: 90 },
  { title: '发货日期', dataIndex: 'deliveredAt', width: 120 },
  { title: '对账状态', dataIndex: 'accounted', width: 100 },
  { title: '操作', key: 'action', width: 70, fixed: 'right' as const },
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
      client.GET('/delivery-notes', {
        params: {
          query: {
            page: pager.current,
            size: pager.pageSize,
            keyword: applied.keyword.trim() || undefined,
            customerId: applied.customerId,
            batchId: applied.batchId,
            accounted: applied.accounted,
            beginDate: applied.dates?.[0],
            endDate: applied.dates?.[1],
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
  Object.assign(filters, { keyword: '', customerId: undefined, batchId: undefined, accounted: undefined, dates: [] })
  search()
}
function onTableChange(p: TablePaginationConfig) {
  pager.current = p.current ?? 1
  pager.pageSize = p.pageSize ?? 20
  void load()
}
function openCreate() {
  Object.assign(form, {
    batchId: undefined,
    barCodes: '',
    deliveryBatch: '',
    logisticsCompany: '',
    plateNo: '',
    driver: '',
    driverPhone: '',
    deliveredAt: dayjs().format('YYYY-MM-DD'),
    remark: '',
  })
  createOpen.value = true
}
async function save() {
  if (!form.batchId || !codes(form.barCodes).length || !form.deliveredAt) {
    message.warning('请选择批次、填写条码和发货日期')
    return
  }
  saving.value = true
  try {
    await call(
      client.POST('/delivery-notes', {
        body: {
          batchId: form.batchId,
          barCodes: codes(form.barCodes),
          deliveryBatch: form.deliveryBatch,
          logisticsCompany: form.logisticsCompany,
          plateNo: form.plateNo,
          driver: form.driver,
          driverPhone: form.driverPhone,
          deliveredAt: form.deliveredAt,
          remark: form.remark,
        },
      }),
    )
    message.success('发货单已创建')
    createOpen.value = false
    search()
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    saving.value = false
  }
}
async function openDetail(row: Note) {
  detailLoading.value = true
  try {
    selected.value = await call(client.GET('/delivery-notes/{id}', { params: { path: { id: row.id } } }))
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    detailLoading.value = false
  }
}
function closeDetail() {
  selected.value = undefined
}
onMounted(async () => {
  void load()
  try {
    const [c, b] = await Promise.all([
      call(client.GET('/customers', { params: { query: { size: 200 } } })),
      call(client.GET('/batches', { params: { query: { size: 200 } } })),
    ])
    customers.value = c.items
    batches.value = b.items
  } catch (e) {
    message.error(errorMessage(e))
  }
})
</script>
<template>
  <div class="page">
    <div class="heading">
      <div>
        <h1>发货查询</h1>
        <span>查询发货记录与对账状态</span>
      </div>
      <StatusTag tone="processing" :label="`共 ${pager.total} 张`" />
    </div>
    <FilterCard @search="search" @reset="reset">
      <label
        >关键字<a-input v-model:value="filters.keyword" placeholder="发货单号 / 批次号 / 工程 / 车牌" allow-clear @press-enter="search"
      /></label>
      <label
        >客户<a-select
          v-model:value="filters.customerId"
          show-search
          option-filter-prop="label"
          allow-clear
          placeholder="全部客户"
          :options="customers.map((c) => ({ value: c.id, label: c.name }))"
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
        >是否已对账<a-select
          v-model:value="filters.accounted"
          allow-clear
          placeholder="全部"
          :options="[
            { value: true, label: '已对账' },
            { value: false, label: '未对账' },
          ]"
      /></label>
      <label>发货日期<a-range-picker v-model:value="filters.dates" value-format="YYYY-MM-DD" /></label>
    </FilterCard>
    <PageCard
      ><TableToolbar
        ><template #left><a-button type="primary" @click="openCreate">新建发货单</a-button></template
        ><template #right><a-button @click="load">刷新</a-button></template></TableToolbar
      >
      <a-table
        row-key="id"
        :columns="columns"
        :data-source="rows"
        :loading="loading"
        :scroll="{ x: 1550 }"
        :pagination="{ ...pager, showSizeChanger: true }"
        data-testid="note-table"
        @change="onTableChange"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'noteNo'"
            ><a @click="openDetail(record)"><CodeText :value="record.noteNo" /></a
          ></template>
          <template v-else-if="column.dataIndex === 'lotNo'"><CodeText :value="record.lotNo" /><br />{{ record.projectName }}</template>
          <template v-else-if="column.dataIndex === 'logisticsCompany'"
            >{{ record.logisticsCompany || '—' }}<br />{{ record.plateNo || '—' }}</template
          >
          <template v-else-if="column.dataIndex === 'area'">{{ Number(record.area).toFixed(2) }} m²</template>
          <template v-else-if="column.dataIndex === 'accounted'"
            ><StatusTag :tone="record.accounted ? 'success' : 'neutral'" :label="record.accounted ? '已对账' : '未对账'"
          /></template>
          <template v-else-if="column.key === 'action'"><a @click="openDetail(record)">详情</a></template>
        </template>
        <template #summary
          ><a-table-summary fixed
            ><a-table-summary-row
              ><a-table-summary-cell :index="0" :col-span="12"
                >当前页汇总：{{ totals.qty }} 件 / {{ totals.area }} m²</a-table-summary-cell
              ></a-table-summary-row
            ></a-table-summary
          ></template
        >
      </a-table>
    </PageCard>
    <a-drawer :open="!!selected || detailLoading" :width="780" :title="`发货单详情: ${selected?.noteNo ?? ''}`" @close="closeDetail"
      ><a-spin :spinning="detailLoading"
        ><template v-if="selected">
          <a-descriptions bordered size="small" :column="2">
            <a-descriptions-item label="发货单号">{{ selected.noteNo }}</a-descriptions-item
            ><a-descriptions-item label="对账状态"
              ><StatusTag :tone="selected.accounted ? 'success' : 'neutral'" :label="selected.accounted ? '已对账' : '未对账'"
            /></a-descriptions-item>
            <a-descriptions-item label="批次号">{{ selected.lotNo }}</a-descriptions-item
            ><a-descriptions-item label="工程">{{ selected.projectName }}</a-descriptions-item>
            <a-descriptions-item label="客户">{{ selected.customerName }}</a-descriptions-item
            ><a-descriptions-item label="合同号">{{ selected.contractNo }}</a-descriptions-item>
            <a-descriptions-item label="发货批次">{{ selected.deliveryBatch || '—' }}</a-descriptions-item
            ><a-descriptions-item label="件数 / 面积">{{ selected.qty }} / {{ Number(selected.area).toFixed(2) }} m²</a-descriptions-item>
            <a-descriptions-item label="物流公司 / 车牌"
              >{{ selected.logisticsCompany || '—' }} / {{ selected.plateNo || '—' }}</a-descriptions-item
            ><a-descriptions-item label="司机 / 电话">{{ selected.driver || '—' }} / {{ selected.driverPhone || '—' }}</a-descriptions-item>
            <a-descriptions-item label="发货日期">{{ selected.deliveredAt }}</a-descriptions-item
            ><a-descriptions-item label="创建人 / 时间">{{ selected.createdBy }} / {{ selected.createdAt }}</a-descriptions-item>
            <a-descriptions-item label="备注" :span="2">{{ selected.remark || '—' }}</a-descriptions-item>
          </a-descriptions>
          <h3>发货标签</h3>
          <a-table row-key="id" size="small" :columns="labelColumns" :data-source="selected.labels" :pagination="false"
            ><template #bodyCell="{ column, record }"
              ><template v-if="column.dataIndex === 'pieceArea'">{{ Number(record.pieceArea).toFixed(4) }} m²</template
              ><template v-else-if="column.dataIndex === 'pieceIndex'">{{ record.pieceIndex }} / {{ record.qty }}</template></template
            ></a-table
          >
        </template></a-spin
      ></a-drawer
    >
    <a-modal v-model:open="createOpen" title="新建发货单" :confirm-loading="saving" @ok="save"
      ><div class="form">
        <label
          >批次 *<a-select
            v-model:value="form.batchId"
            show-search
            option-filter-prop="label"
            placeholder="请选择批次"
            :options="batches.map((b) => ({ value: b.id, label: `${b.lotNo} · ${b.projectName}` }))"
        /></label>
        <label>条码 *<a-textarea v-model:value="form.barCodes" :rows="4" placeholder="每行一个条码，也支持逗号或空格分隔" /></label>
        <label>发货批次<a-input v-model:value="form.deliveryBatch" /></label
        ><label>物流公司<a-input v-model:value="form.logisticsCompany" /></label> <label>车牌<a-input v-model:value="form.plateNo" /></label
        ><label>司机<a-input v-model:value="form.driver" /></label> <label>司机电话<a-input v-model:value="form.driverPhone" /></label
        ><label>发货日期 *<a-date-picker v-model:value="form.deliveredAt" value-format="YYYY-MM-DD" style="width: 100%" /></label>
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
  gap: 12px;
  max-height: 65vh;
  overflow: auto;
}
.form label {
  display: grid;
  gap: 5px;
}
h3 {
  margin-top: 24px;
}
</style>
