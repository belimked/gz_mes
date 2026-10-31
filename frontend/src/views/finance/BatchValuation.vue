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

type Valuation = Schemas['BatchValuation']
type Basis = Schemas['ValuationBasis']
const basisLabels: Record<Basis, string> = {
  order_area: '订单面积',
  item_area: '构件面积',
  delivered_area: '已发货面积',
}
const statusLabels: Record<Valuation['status'], string> = {
  pending: '待排产',
  scheduled: '已排产',
  in_production: '生产中',
  done: '已完工',
}
const statusTones = { pending: 'neutral', scheduled: 'processing', in_production: 'processing', done: 'success' } as const
const filters = reactive({
  keyword: '',
  customerId: undefined as number | undefined,
  contractId: undefined as number | undefined,
  audited: undefined as boolean | undefined,
  dates: [] as string[],
})
const applied = reactive({ ...filters })
const customers = ref<Schemas['Customer'][]>([])
const contracts = ref<Schemas['Contract'][]>([])
const rows = ref<Valuation[]>([])
const pager = reactive({ current: 1, pageSize: 20, total: 0 })
const totalAmount = ref(0)
const loading = ref(false)
const saving = ref(false)
const auditingId = ref<number>()
const selected = ref<Valuation>()
const form = reactive<Schemas['BatchValuationInput']>({ basis: 'order_area', unitPrice: 0, ingotPrice: 0, remark: '' })
const previewArea = computed(() => {
  const row = selected.value
  if (!row) return 0
  return form.basis === 'order_area' ? row.orderArea : form.basis === 'item_area' ? row.itemArea : row.deliveredArea
})
const previewAmount = computed(() => Math.round((previewArea.value * (Number(form.unitPrice) || 0) + Number.EPSILON) * 100) / 100)
const pageQty = computed(() => rows.value.reduce((sum, row) => sum + row.orderQty, 0))
const pageArea = computed(() => rows.value.reduce((sum, row) => sum + row.orderArea, 0))
const columns = [
  { title: '批次号', dataIndex: 'lotNo', width: 160 },
  { title: '下单日期', dataIndex: 'createdAt', width: 115 },
  { title: '工程名称 / 客户', dataIndex: 'projectName', width: 185 },
  { title: '合同号', dataIndex: 'contractNo', width: 150 },
  { title: '板厚 · 颜色', dataIndex: 'thickness', width: 120 },
  { title: '订单件数', dataIndex: 'orderQty', width: 95, align: 'right' as const },
  { title: '订单面积', dataIndex: 'orderArea', width: 105, align: 'right' as const },
  { title: '构件面积', dataIndex: 'itemArea', width: 105, align: 'right' as const },
  { title: '已发货', dataIndex: 'deliveredQty', width: 145 },
  { title: '生产状态', dataIndex: 'status', width: 100 },
  { title: '计价基准', dataIndex: 'basis', width: 115 },
  { title: '计价面积', dataIndex: 'basisArea', width: 105, align: 'right' as const },
  { title: '单价', dataIndex: 'unitPrice', width: 100, align: 'right' as const },
  { title: '铝锭价', dataIndex: 'ingotPrice', width: 105, align: 'right' as const },
  { title: '金额', dataIndex: 'amount', width: 125, align: 'right' as const },
  { title: '审核', dataIndex: 'audited', width: 95 },
  { title: '操作', key: 'action', width: 130, fixed: 'right' as const },
]
async function load() {
  loading.value = true
  try {
    const result = await call(
      client.GET('/batch-valuations', {
        params: {
          query: {
            page: pager.current,
            size: pager.pageSize,
            keyword: applied.keyword.trim() || undefined,
            customerId: applied.customerId,
            contractId: applied.contractId,
            audited: applied.audited,
            beginDate: applied.dates?.[0],
            endDate: applied.dates?.[1],
          },
        },
      }),
    )
    rows.value = result.items
    pager.total = result.total
    totalAmount.value = result.totalAmount
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
  Object.assign(filters, { keyword: '', customerId: undefined, contractId: undefined, audited: undefined, dates: [] })
  search()
}
function onTableChange(p: TablePaginationConfig) {
  pager.current = p.current ?? 1
  pager.pageSize = p.pageSize ?? 20
  void load()
}
function openPricing(row: Valuation) {
  if (row.audited) return
  Object.assign(form, { basis: row.basis, unitPrice: row.unitPrice, ingotPrice: row.ingotPrice, remark: row.remark })
  selected.value = row
}
function closePricing() {
  selected.value = undefined
}
async function save() {
  const row = selected.value
  if (!row || row.audited) return
  if (
    form.unitPrice == null ||
    !Number.isFinite(Number(form.unitPrice)) ||
    Number(form.unitPrice) < 0 ||
    form.ingotPrice == null ||
    !Number.isFinite(Number(form.ingotPrice)) ||
    Number(form.ingotPrice) < 0
  ) {
    message.warning('请输入非负的单价和铝锭价')
    return
  }
  saving.value = true
  try {
    await call(
      client.PUT('/batches/{id}/valuation', {
        params: { path: { id: row.batchId } },
        body: {
          basis: form.basis,
          unitPrice: Number(form.unitPrice),
          ingotPrice: Number(form.ingotPrice),
          remark: form.remark,
        },
      }),
    )
    message.success('计价已保存')
    closePricing()
    await load()
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    saving.value = false
  }
}
async function audit(row: Valuation) {
  if (row.audited || row.unitPrice <= 0) return
  auditingId.value = row.batchId
  try {
    await call(client.POST('/batches/{id}/valuation/audit', { params: { path: { id: row.batchId } } }))
    message.success('审核成功')
    await load()
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    auditingId.value = undefined
  }
}
onMounted(() => {
  void load()
  void (async () => {
    try {
      const [customerPage, contractPage] = await Promise.all([
        call(client.GET('/customers', { params: { query: { size: 200 } } })),
        call(client.GET('/contracts', { params: { query: { size: 200 } } })),
      ])
      customers.value = customerPage.items
      contracts.value = contractPage.items
    } catch (e) {
      message.error(errorMessage(e))
    }
  })()
})
</script>
<template>
  <div class="page">
    <div class="heading">
      <div>
        <h1>工程批次对账</h1>
        <span>按批次计价并审核工程订单</span>
      </div>
      <StatusTag tone="processing" :label="`共 ${pager.total} 个批次`" />
    </div>
    <FilterCard @search="search" @reset="reset">
      <label
        >关键字<a-input v-model:value="filters.keyword" placeholder="批次号 / 合同号 / 工程 / 客户" allow-clear @press-enter="search"
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
        >合同<a-select
          v-model:value="filters.contractId"
          show-search
          option-filter-prop="label"
          allow-clear
          placeholder="全部合同"
          :options="contracts.map((c) => ({ value: c.id, label: `${c.contractNo} · ${c.projectName}` }))"
      /></label>
      <label
        >审核状态<a-select
          v-model:value="filters.audited"
          allow-clear
          placeholder="全部状态"
          :options="[
            { value: true, label: '已审核' },
            { value: false, label: '未审核' },
          ]"
      /></label>
      <label>下单日期<a-range-picker v-model:value="filters.dates" value-format="YYYY-MM-DD" /></label>
    </FilterCard>
    <PageCard>
      <TableToolbar>
        <template #left
          ><a-button disabled>导出</a-button
          ><span
            >筛选结果总金额 <strong>¥{{ Number(totalAmount).toFixed(2) }}</strong> · 当前页 {{ pageQty }} 件 /
            {{ pageArea.toFixed(2) }} m²</span
          ></template
        >
        <template #right><a-button @click="load">刷新</a-button></template>
      </TableToolbar>
      <a-table
        row-key="batchId"
        :columns="columns"
        :data-source="rows"
        :loading="loading"
        :scroll="{ x: 2100 }"
        :pagination="{ ...pager, showSizeChanger: true }"
        data-testid="valuation-table"
        @change="onTableChange"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'lotNo'"><CodeText :value="record.lotNo" /></template>
          <template v-else-if="column.dataIndex === 'createdAt'">{{ dayjs(record.createdAt).format('YYYY-MM-DD') }}</template>
          <template v-else-if="column.dataIndex === 'projectName'"
            >{{ record.projectName }}<br /><span class="muted">{{ record.customerName }}</span></template
          >
          <template v-else-if="column.dataIndex === 'thickness'">{{ record.thickness }} mm · {{ record.color || '—' }}</template>
          <template v-else-if="column.dataIndex === 'orderArea' || column.dataIndex === 'itemArea' || column.dataIndex === 'basisArea'"
            >{{ Number(record[column.dataIndex]).toFixed(2) }} m²</template
          >
          <template v-else-if="column.dataIndex === 'deliveredQty'"
            >{{ record.deliveredQty }} 件 / {{ Number(record.deliveredArea).toFixed(2) }} m²</template
          >
          <template v-else-if="column.dataIndex === 'status'"
            ><StatusTag
              :tone="statusTones[record.status as Valuation['status']]"
              :label="statusLabels[record.status as Valuation['status']]"
          /></template>
          <template v-else-if="column.dataIndex === 'basis'">{{ basisLabels[record.basis as Basis] }}</template>
          <template v-else-if="column.dataIndex === 'unitPrice' || column.dataIndex === 'ingotPrice'"
            >¥{{ Number(record[column.dataIndex]).toFixed(2) }}</template
          >
          <template v-else-if="column.dataIndex === 'amount'"
            ><strong>¥{{ Number(record.amount).toFixed(2) }}</strong></template
          >
          <template v-else-if="column.dataIndex === 'audited'"
            ><StatusTag :tone="record.audited ? 'success' : 'warning'" :label="record.audited ? '已审核' : '未审核'"
          /></template>
          <template v-else-if="column.key === 'action'">
            <a-button type="link" size="small" :disabled="record.audited" @click="openPricing(record)">计价</a-button>
            <a-popconfirm v-if="!record.audited && record.unitPrice > 0" title="确定审核此批次计价？" @confirm="audit(record)">
              <a-button type="link" size="small" :loading="auditingId === record.batchId">审核</a-button>
            </a-popconfirm>
          </template>
        </template>
      </a-table>
    </PageCard>
    <a-modal :open="!!selected" :title="`批次计价 · ${selected?.lotNo ?? ''}`" :confirm-loading="saving" @ok="save" @cancel="closePricing">
      <div class="form">
        <label
          >计价基准<a-radio-group
            v-model:value="form.basis"
            :options="Object.entries(basisLabels).map(([value, label]) => ({ value, label }))"
        /></label>
        <label>单价 (元/m²) *<a-input-number v-model:value="form.unitPrice" :min="0" :step="1" :precision="2" style="width: 100%" /></label>
        <label>铝锭价 (元/吨)<a-input-number v-model:value="form.ingotPrice" :min="0" :precision="2" style="width: 100%" /></label>
        <label>备注<a-textarea v-model:value="form.remark" :rows="2" /></label>
        <strong>计价面积 {{ Number(previewArea).toFixed(2) }} m² · 金额预览 ¥{{ previewAmount.toFixed(2) }}</strong>
      </div>
    </a-modal>
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
.heading span,
.muted {
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
</style>
