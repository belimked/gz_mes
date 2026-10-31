<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { message } from 'ant-design-vue'
import type { TablePaginationConfig } from 'ant-design-vue'
import dayjs from 'dayjs'
import { call, client, errorMessage, type Schemas } from '@/api/client'
import PageCard from '@/components/PageCard.vue'
import FilterCard from '@/components/FilterCard.vue'
import TableToolbar from '@/components/TableToolbar.vue'
import CodeText from '@/components/CodeText.vue'
import StatusTag from '@/components/StatusTag.vue'

type Account = Schemas['DeliverAccount']
type Note = Schemas['DeliveryNote']
const filters = reactive({ keyword: '', customerId: undefined as number | undefined, status: undefined as Account['status'] | undefined })
const applied = reactive({ ...filters })
const customers = ref<Schemas['Customer'][]>([])
const rows = ref<Account[]>([])
const pager = reactive({ current: 1, pageSize: 20, total: 0 })
const loading = ref(false)
const selected = ref<Schemas['DeliverAccountDetail']>()
const detailLoading = ref(false)
const createOpen = ref(false)
const saving = ref(false)
const notesLoading = ref(false)
const available = ref<Note[]>([])
const form = reactive({
  customerId: undefined as number | undefined,
  deliveryNoteIds: [] as number[],
  unitPrice: 0,
  extraAmount: 0,
  extraRemark: '',
  remark: '',
})
const picked = computed(() => available.value.filter((n) => form.deliveryNoteIds.includes(n.id)))
const totalArea = computed(() => picked.value.reduce((sum, n) => sum + Number(n.area), 0))
const amount = computed(() => totalArea.value * (Number(form.unitPrice) || 0) + (Number(form.extraAmount) || 0))
const columns = [
  { title: '对账单号', dataIndex: 'accountNo', width: 180 },
  { title: '客户', dataIndex: 'customerName', width: 160 },
  { title: '发货单数', dataIndex: 'noteCount', width: 100 },
  { title: '总件数', dataIndex: 'totalQty', width: 95 },
  { title: '总面积', dataIndex: 'totalArea', width: 105 },
  { title: '单价', dataIndex: 'unitPrice', width: 100 },
  { title: '附加费用', dataIndex: 'extraAmount', width: 105 },
  { title: '金额', dataIndex: 'amount', width: 120 },
  { title: '状态', dataIndex: 'status', width: 95 },
  { title: '创建人 / 时间', dataIndex: 'createdBy', width: 165 },
  { title: '操作', key: 'action', width: 130, fixed: 'right' as const },
]
const noteColumns = [
  { title: '发货单号', dataIndex: 'noteNo' },
  { title: '批次号', dataIndex: 'lotNo' },
  { title: '件数', dataIndex: 'qty' },
  { title: '面积', dataIndex: 'area' },
  { title: '发货日期', dataIndex: 'deliveredAt' },
]
async function load() {
  loading.value = true
  try {
    const result = await call(
      client.GET('/deliver-accounts', {
        params: {
          query: {
            page: pager.current,
            size: pager.pageSize,
            keyword: applied.keyword.trim() || undefined,
            customerId: applied.customerId,
            status: applied.status,
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
  Object.assign(filters, { keyword: '', customerId: undefined, status: undefined })
  search()
}
function onTableChange(p: TablePaginationConfig) {
  pager.current = p.current ?? 1
  pager.pageSize = p.pageSize ?? 20
  void load()
}
function openCreate() {
  Object.assign(form, { customerId: undefined, deliveryNoteIds: [], unitPrice: 0, extraAmount: 0, extraRemark: '', remark: '' })
  available.value = []
  createOpen.value = true
}
watch(
  () => form.customerId,
  async (customerId) => {
    form.deliveryNoteIds = []
    available.value = []
    if (!customerId || !createOpen.value) return
    notesLoading.value = true
    try {
      const result = await call(client.GET('/delivery-notes', { params: { query: { customerId, accounted: false, size: 200 } } }))
      if (form.customerId === customerId) available.value = result.items
    } catch (e) {
      message.error(errorMessage(e))
    } finally {
      notesLoading.value = false
    }
  },
)
function selectNotes(keys: (string | number)[]) {
  form.deliveryNoteIds = keys.map(Number)
}
async function save() {
  if (!form.customerId || !form.deliveryNoteIds.length || form.unitPrice === null || Number(form.unitPrice) < 0) {
    message.warning('请选择客户、发货单并填写非负单价')
    return
  }
  saving.value = true
  try {
    await call(
      client.POST('/deliver-accounts', {
        body: {
          customerId: form.customerId,
          deliveryNoteIds: form.deliveryNoteIds,
          unitPrice: Number(form.unitPrice),
          extraAmount: Number(form.extraAmount) || 0,
          extraRemark: form.extraRemark,
          remark: form.remark,
        },
      }),
    )
    message.success('对账单已创建')
    createOpen.value = false
    search()
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    saving.value = false
  }
}
async function openDetail(row: Account) {
  detailLoading.value = true
  try {
    selected.value = await call(client.GET('/deliver-accounts/{id}', { params: { path: { id: row.id } } }))
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    detailLoading.value = false
  }
}
function closeDetail() {
  selected.value = undefined
}
async function confirm(row: Account) {
  try {
    await call(client.POST('/deliver-accounts/{id}/confirm', { params: { path: { id: row.id } } }))
    message.success('已确认')
    await load()
    if (selected.value?.id === row.id) await openDetail(row)
  } catch (e) {
    message.error(errorMessage(e))
  }
}
onMounted(async () => {
  void load()
  try {
    customers.value = (await call(client.GET('/customers', { params: { query: { size: 200 } } }))).items
  } catch (e) {
    message.error(errorMessage(e))
  }
})
</script>
<template>
  <div class="page">
    <div class="heading">
      <div>
        <h1>发货对账查询</h1>
        <span>汇总发货单并确认对账</span>
      </div>
      <StatusTag tone="processing" :label="`共 ${pager.total} 张`" />
    </div>
    <FilterCard @search="search" @reset="reset">
      <label>关键字<a-input v-model:value="filters.keyword" placeholder="对账单号 / 客户" allow-clear @press-enter="search" /></label>
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
        >状态<a-select
          v-model:value="filters.status"
          allow-clear
          placeholder="全部"
          :options="[
            { value: 'draft', label: '草稿' },
            { value: 'confirmed', label: '已确认' },
          ]"
      /></label>
    </FilterCard>
    <PageCard
      ><TableToolbar
        ><template #left><a-button type="primary" @click="openCreate">新建对账单</a-button></template
        ><template #right><a-button @click="load">刷新</a-button></template></TableToolbar
      >
      <a-table
        row-key="id"
        :columns="columns"
        :data-source="rows"
        :loading="loading"
        :scroll="{ x: 1400 }"
        :pagination="{ ...pager, showSizeChanger: true }"
        data-testid="account-table"
        @change="onTableChange"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'accountNo'"
            ><a @click="openDetail(record)"><CodeText :value="record.accountNo" /></a
          ></template>
          <template v-else-if="column.dataIndex === 'totalArea'">{{ Number(record.totalArea).toFixed(2) }} m²</template>
          <template v-else-if="column.dataIndex === 'unitPrice' || column.dataIndex === 'extraAmount'"
            >¥{{ Number(record[column.dataIndex]).toFixed(2) }}</template
          >
          <template v-else-if="column.dataIndex === 'amount'"
            ><strong>¥{{ Number(record.amount).toFixed(2) }}</strong></template
          >
          <template v-else-if="column.dataIndex === 'status'"
            ><StatusTag
              :tone="record.status === 'confirmed' ? 'success' : 'warning'"
              :label="record.status === 'confirmed' ? '已确认' : '草稿'"
          /></template>
          <template v-else-if="column.dataIndex === 'createdBy'"
            >{{ record.createdBy }}<br />{{ dayjs(record.createdAt).format('YYYY-MM-DD HH:mm') }}</template
          >
          <template v-else-if="column.key === 'action'"
            ><a-popconfirm v-if="record.status === 'draft'" title="确认此对账单？" @confirm="confirm(record)"><a>确认</a></a-popconfirm
            ><a-divider v-if="record.status === 'draft'" type="vertical" /><a @click="openDetail(record)">详情</a></template
          >
        </template>
      </a-table>
    </PageCard>
    <a-drawer :open="!!selected || detailLoading" :width="800" :title="`对账单详情: ${selected?.accountNo ?? ''}`" @close="closeDetail"
      ><a-spin :spinning="detailLoading"
        ><template v-if="selected">
          <a-descriptions bordered size="small" :column="2">
            <a-descriptions-item label="对账单号">{{ selected.accountNo }}</a-descriptions-item
            ><a-descriptions-item label="状态"
              ><StatusTag
                :tone="selected.status === 'confirmed' ? 'success' : 'warning'"
                :label="selected.status === 'confirmed' ? '已确认' : '草稿'"
            /></a-descriptions-item>
            <a-descriptions-item label="客户">{{ selected.customerName }}</a-descriptions-item
            ><a-descriptions-item label="发货单数 / 件数">{{ selected.noteCount }} / {{ selected.totalQty }}</a-descriptions-item>
            <a-descriptions-item label="总面积">{{ Number(selected.totalArea).toFixed(2) }} m²</a-descriptions-item
            ><a-descriptions-item label="单价">¥{{ Number(selected.unitPrice).toFixed(2) }}</a-descriptions-item>
            <a-descriptions-item label="附加费用">¥{{ Number(selected.extraAmount).toFixed(2) }}</a-descriptions-item
            ><a-descriptions-item label="金额"
              ><strong>¥{{ Number(selected.amount).toFixed(2) }}</strong></a-descriptions-item
            >
            <a-descriptions-item label="附加说明">{{ selected.extraRemark || '—' }}</a-descriptions-item
            ><a-descriptions-item label="备注">{{ selected.remark || '—' }}</a-descriptions-item>
            <a-descriptions-item label="创建人 / 时间">{{ selected.createdBy }} / {{ selected.createdAt }}</a-descriptions-item
            ><a-descriptions-item label="确认人 / 时间"
              >{{ selected.confirmedBy || '—' }} / {{ selected.confirmedAt || '—' }}</a-descriptions-item
            >
          </a-descriptions>
          <h3>关联发货单</h3>
          <a-table row-key="id" size="small" :columns="noteColumns" :data-source="selected.notes" :pagination="false"
            ><template #bodyCell="{ column, record }"
              ><template v-if="column.dataIndex === 'area'">{{ Number(record.area).toFixed(2) }} m²</template></template
            ></a-table
          >
        </template></a-spin
      ></a-drawer
    >
    <a-modal v-model:open="createOpen" title="新建对账单" :width="850" :confirm-loading="saving" @ok="save"
      ><div class="form">
        <label
          >客户 *<a-select
            v-model:value="form.customerId"
            show-search
            option-filter-prop="label"
            placeholder="请选择客户"
            :options="customers.map((c) => ({ value: c.id, label: c.name }))"
        /></label>
        <div>选择未对账发货单</div>
        <a-table
          row-key="id"
          size="small"
          :columns="noteColumns"
          :data-source="available"
          :loading="notesLoading"
          :pagination="{ pageSize: 10 }"
          :row-selection="{ selectedRowKeys: form.deliveryNoteIds, onChange: selectNotes }"
          ><template #bodyCell="{ column, record }"
            ><template v-if="column.dataIndex === 'area'">{{ Number(record.area).toFixed(2) }} m²</template></template
          ></a-table
        >
        <div class="form-row">
          <label>单价 (元/m²) *<a-input-number v-model:value="form.unitPrice" :min="0" :precision="2" style="width: 100%" /></label
          ><label>附加费用<a-input-number v-model:value="form.extraAmount" :precision="2" style="width: 100%" /></label>
        </div>
        <label>附加说明<a-input v-model:value="form.extraRemark" /></label
        ><label>备注<a-textarea v-model:value="form.remark" :rows="2" /></label>
        <strong>已选 {{ picked.length }} 张 · 总面积 {{ totalArea.toFixed(2) }} m² · 金额预览 ¥{{ amount.toFixed(2) }}</strong>
      </div></a-modal
    >
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
  gap: 14px;
  max-height: 65vh;
  overflow: auto;
}
.form label {
  display: grid;
  gap: 6px;
}
.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}
h3 {
  margin-top: 24px;
}
</style>
