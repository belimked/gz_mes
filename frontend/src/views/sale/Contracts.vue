<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { message } from 'ant-design-vue'
import type { FormInstance, TablePaginationConfig } from 'ant-design-vue'
import { ReloadOutlined } from '@ant-design/icons-vue'
import dayjs from 'dayjs'
import { call, client, errorMessage, type Schemas } from '@/api/client'
import PageCard from '@/components/PageCard.vue'
import FilterCard from '@/components/FilterCard.vue'
import TableToolbar from '@/components/TableToolbar.vue'
import CodeText from '@/components/CodeText.vue'
import StatusTag from '@/components/StatusTag.vue'

type Contract = Schemas['Contract']
type AuditKind = Schemas['ValuationAudit']['kind']
const factories = ref<Schemas['Factory'][]>([])
const customers = ref<Schemas['Customer'][]>([])
const projects = ref<Schemas['Project'][]>([])
const factoryName = computed(() => new Map(factories.value.map((f) => [f.id, f.name])))
const filters = reactive({
  keyword: '',
  customerId: undefined as number | undefined,
  projectId: undefined as number | undefined,
  dates: [] as string[],
  customerAudited: undefined as boolean | undefined,
  outsourceAudited: undefined as boolean | undefined,
})
const applied = reactive({ ...filters })
const rows = ref<Contract[]>([])
const selected = ref<Contract>()
const loading = ref(false)
const auditing = ref(false)
const pager = reactive({ current: 1, pageSize: 20, total: 0 })
const visible = computed(() =>
  rows.value.filter(
    (c) =>
      (applied.customerAudited === undefined || c.customerValuationAudited === applied.customerAudited) &&
      (applied.outsourceAudited === undefined || c.outsourceValuationAudited === applied.outsourceAudited),
  ),
)
const columns = [
  { title: '合同单号', dataIndex: 'contractNo', width: 175 },
  { title: '客户名称', dataIndex: 'customerName', width: 180, ellipsis: true },
  { title: '工程名称', dataIndex: 'projectName', width: 210, ellipsis: true },
  { title: '业务员', dataIndex: 'salesman', width: 100 },
  { title: '订单数量', dataIndex: 'orderQty', width: 110, align: 'right' as const },
  { title: '订单面积', dataIndex: 'orderArea', width: 120, align: 'right' as const },
  { title: '计价方式', dataIndex: 'pricingManner', width: 110 },
  { title: '客户计价审核', dataIndex: 'customerValuationAudited', width: 130 },
  { title: '外协计价审核', dataIndex: 'outsourceValuationAudited', width: 130 },
  { title: '创建时间', dataIndex: 'createdAt', width: 170 },
  { title: '操作', key: 'action', width: 80, fixed: 'right' as const },
]
const options = (items: { id: number; name?: string; fullName?: string }[]) =>
  items.map((item) => ({ value: item.id, label: item.name ?? item.fullName ?? '' }))
async function load() {
  loading.value = true
  try {
    const r = await call(
      client.GET('/contracts', {
        params: {
          query: {
            page: pager.current,
            size: pager.pageSize,
            keyword: applied.keyword.trim() || undefined,
            customerId: applied.customerId,
            projectId: applied.projectId,
            beginDate: applied.dates?.[0],
            endDate: applied.dates?.[1],
          },
        },
      }),
    )
    rows.value = r.items
    pager.total = r.total
    if (selected.value) selected.value = r.items.find((c) => c.id === selected.value?.id) ?? selected.value
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
  Object.assign(filters, {
    keyword: '',
    customerId: undefined,
    projectId: undefined,
    dates: [],
    customerAudited: undefined,
    outsourceAudited: undefined,
  })
  search()
}
function onTableChange(p: TablePaginationConfig) {
  pager.current = p.current ?? 1
  pager.pageSize = p.pageSize ?? 20
  void load()
}
async function audit(kind: AuditKind) {
  if (!selected.value) return
  auditing.value = true
  try {
    const updated = await call(
      client.POST('/contracts/{id}/audit-valuation', { params: { path: { id: selected.value.id } }, body: { kind } }),
    )
    selected.value = updated
    message.success('已审核')
    await load()
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    auditing.value = false
  }
}
const open = ref(false)
const saving = ref(false)
const formRef = ref<FormInstance>()
const emptyForm = (): Schemas['ContractCreate'] => ({
  factoryId: factories.value[0]?.id ?? 0,
  contractNo: '',
  customerId: undefined as unknown as number,
  projectId: undefined as unknown as number,
  customerContractNo: '',
  innerContractNo: '',
  salesman: '',
  orderQty: 0,
  orderArea: 0,
  pricingManner: '',
  payment: '',
  paymentPoint: '',
  texture: '',
  paintType: '',
  packing: '',
  receiver: '',
  receiverPhone: '',
  receiverAddress: '',
  remark: '',
})
const form = ref<Schemas['ContractCreate']>(emptyForm())
const formCustomers = computed(() => customers.value.filter((c) => c.factoryId === form.value.factoryId))
const formProjects = computed(() => projects.value.filter((p) => p.factoryId === form.value.factoryId))
function changeFactory() {
  form.value.customerId = undefined as unknown as number
  form.value.projectId = undefined as unknown as number
}
function openCreate() {
  form.value = emptyForm()
  open.value = true
}
async function save() {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }
  saving.value = true
  try {
    await call(client.POST('/contracts', { body: form.value }))
    message.success('已创建')
    open.value = false
    search()
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    saving.value = false
  }
}
onMounted(async () => {
  try {
    const [f, c, p] = await Promise.all([
      call(client.GET('/factories')),
      call(client.GET('/customers', { params: { query: { size: 200 } } })),
      call(client.GET('/projects', { params: { query: { size: 200 } } })),
    ])
    factories.value = f
    customers.value = c.items
    projects.value = p.items
  } catch (e) {
    message.error(errorMessage(e))
  }
  await load()
})
</script>

<template>
  <div class="contracts">
    <div class="heading">
      <div>
        <h1>合同查询</h1>
        <span>查询合同档案与计价审核状态</span>
      </div>
      <a-tag>共 {{ pager.total }} 份合同</a-tag>
    </div>
    <FilterCard @search="search" @reset="reset"
      ><template #default="{ expanded }">
        <label
          >关键字<a-input
            v-model:value="filters.keyword"
            placeholder="合同号 / 客户合同号 / 内部合同号 / 客户 / 工程 / 业务员"
            allow-clear
            data-testid="contract-search"
            @press-enter="search"
        /></label>
        <label
          >客户<a-select
            v-model:value="filters.customerId"
            show-search
            option-filter-prop="label"
            allow-clear
            placeholder="全部客户"
            :options="options(customers)"
        /></label>
        <label
          >项目<a-select
            v-model:value="filters.projectId"
            show-search
            option-filter-prop="label"
            allow-clear
            placeholder="全部项目"
            :options="options(projects)"
        /></label>
        <label>创建日期<a-range-picker v-model:value="filters.dates" value-format="YYYY-MM-DD" /></label>
        <label v-if="expanded"
          >客户计价审核<a-select
            v-model:value="filters.customerAudited"
            allow-clear
            placeholder="全部状态"
            :options="[
              { value: true, label: '已审核' },
              { value: false, label: '未审核' },
            ]"
        /></label>
        <label v-if="expanded"
          >外协计价审核<a-select
            v-model:value="filters.outsourceAudited"
            allow-clear
            placeholder="全部状态"
            :options="[
              { value: true, label: '已审核' },
              { value: false, label: '未审核' },
            ]"
        /></label> </template
    ></FilterCard>
    <PageCard>
      <TableToolbar
        ><template #left
          ><a-button type="primary" data-testid="contract-new" @click="openCreate">新建合同</a-button
          ><a-button disabled>导出</a-button></template
        ><template #right
          ><a-button aria-label="刷新" @click="load"><ReloadOutlined /></a-button></template
      ></TableToolbar>
      <a-table
        row-key="id"
        size="middle"
        :columns="columns"
        :data-source="visible"
        :loading="loading"
        :scroll="{ x: 1520 }"
        :pagination="{ ...pager, showSizeChanger: true, showTotal: (t: number) => `共 ${t} 条` }"
        data-testid="contract-table"
        @change="onTableChange"
      >
        <template #emptyText
          ><a-empty description="暂无匹配的合同"><a-button type="primary" @click="reset">重置查询条件</a-button></a-empty></template
        >
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'contractNo'"><CodeText :value="record.contractNo" /></template>
          <template v-else-if="column.dataIndex === 'orderQty'"
            ><span class="numeric">{{ record.orderQty }}</span></template
          >
          <template v-else-if="column.dataIndex === 'orderArea'"
            ><span class="numeric">{{ Number(record.orderArea).toFixed(2) }}</span></template
          >
          <template v-else-if="column.dataIndex === 'customerValuationAudited' || column.dataIndex === 'outsourceValuationAudited'"
            ><StatusTag :tone="record[column.dataIndex] ? 'success' : 'warning'" :label="record[column.dataIndex] ? '已审核' : '未审核'"
          /></template>
          <template v-else-if="column.dataIndex === 'createdAt'">{{ dayjs(record.createdAt).format('YYYY-MM-DD HH:mm') }}</template>
          <template v-else-if="column.key === 'action'"><a @click="selected = record as Contract">详情</a></template>
        </template>
      </a-table>
    </PageCard>
    <a-drawer :open="!!selected" :width="720" placement="right" @close="selected = undefined">
      <template #title>合同详情: {{ selected?.contractNo }}</template>
      <template v-if="selected">
        <a-divider orientation="left">合同信息</a-divider>
        <a-descriptions bordered :column="2" size="small">
          <a-descriptions-item label="合同单号"><CodeText :value="selected.contractNo" /></a-descriptions-item>
          <a-descriptions-item label="厂区">{{ factoryName.get(selected.factoryId) ?? selected.factoryId }}</a-descriptions-item>
          <a-descriptions-item label="客户合同号">{{ selected.customerContractNo || '—' }}</a-descriptions-item>
          <a-descriptions-item label="内部合同号">{{ selected.innerContractNo || '—' }}</a-descriptions-item>
          <a-descriptions-item label="客户名称">{{ selected.customerName }}</a-descriptions-item>
          <a-descriptions-item label="工程名称">{{ selected.projectName }}</a-descriptions-item>
          <a-descriptions-item label="业务员">{{ selected.salesman || '—' }}</a-descriptions-item>
          <a-descriptions-item label="计价方式">{{ selected.pricingManner || '—' }}</a-descriptions-item>
          <a-descriptions-item label="订单数量">{{ selected.orderQty }}</a-descriptions-item>
          <a-descriptions-item label="订单面积">{{ Number(selected.orderArea).toFixed(2) }} m²</a-descriptions-item>
          <a-descriptions-item label="客户计价审核"
            ><StatusTag
              :tone="selected.customerValuationAudited ? 'success' : 'warning'"
              :label="selected.customerValuationAudited ? '已审核' : '未审核'"
          /></a-descriptions-item>
          <a-descriptions-item label="外协计价审核"
            ><StatusTag
              :tone="selected.outsourceValuationAudited ? 'success' : 'warning'"
              :label="selected.outsourceValuationAudited ? '已审核' : '未审核'"
          /></a-descriptions-item>
          <a-descriptions-item label="创建人">{{ selected.createdBy }}</a-descriptions-item>
          <a-descriptions-item label="创建时间">{{ dayjs(selected.createdAt).format('YYYY-MM-DD HH:mm') }}</a-descriptions-item>
        </a-descriptions>
        <a-divider orientation="left">商务条款</a-divider>
        <a-descriptions bordered :column="2" size="small">
          <a-descriptions-item label="付款方式">{{ selected.payment || '—' }}</a-descriptions-item>
          <a-descriptions-item label="付款节点">{{ selected.paymentPoint || '—' }}</a-descriptions-item>
          <a-descriptions-item label="材质">{{ selected.texture || '—' }}</a-descriptions-item>
          <a-descriptions-item label="喷涂类型">{{ selected.paintType || '—' }}</a-descriptions-item>
          <a-descriptions-item label="包装">{{ selected.packing || '—' }}</a-descriptions-item>
          <a-descriptions-item label="备注" :span="2">{{ selected.remark || '—' }}</a-descriptions-item>
        </a-descriptions>
        <a-divider orientation="left">收货信息</a-divider>
        <a-descriptions bordered :column="2" size="small">
          <a-descriptions-item label="收货人">{{ selected.receiver || '—' }}</a-descriptions-item>
          <a-descriptions-item label="收货电话">{{ selected.receiverPhone || '—' }}</a-descriptions-item>
          <a-descriptions-item label="收货地址" :span="2">{{ selected.receiverAddress || '—' }}</a-descriptions-item>
        </a-descriptions>
        <div class="drawer-footer">
          <a-space
            ><a-button type="primary" :disabled="selected.customerValuationAudited" :loading="auditing" @click="audit('customer')"
              >客户计价审核</a-button
            ><a-button :disabled="selected.outsourceValuationAudited" :loading="auditing" @click="audit('outsource')"
              >外协计价审核</a-button
            ></a-space
          >
        </div>
      </template>
    </a-drawer>
    <a-modal v-model:open="open" title="新建合同" :width="720" :confirm-loading="saving" ok-text="保存" destroy-on-close @ok="save">
      <a-form ref="formRef" :model="form" :label-col="{ span: 5 }">
        <a-form-item label="厂区" name="factoryId" :rules="[{ required: true, message: '请选择厂区' }]"
          ><a-select
            v-model:value="form.factoryId"
            :options="factories.map((f) => ({ value: f.id, label: f.name }))"
            @change="changeFactory"
        /></a-form-item>
        <a-form-item label="合同单号" name="contractNo" :rules="[{ required: true, whitespace: true, message: '请输入合同单号' }]"
          ><a-input v-model:value="form.contractNo" :maxlength="64"
        /></a-form-item>
        <a-form-item label="客户" name="customerId" :rules="[{ required: true, message: '请选择客户' }]"
          ><a-select v-model:value="form.customerId" show-search option-filter-prop="label" :options="options(formCustomers)"
        /></a-form-item>
        <a-form-item label="项目" name="projectId" :rules="[{ required: true, message: '请选择项目' }]"
          ><a-select v-model:value="form.projectId" show-search option-filter-prop="label" :options="options(formProjects)"
        /></a-form-item>
        <a-form-item label="客户合同号"><a-input v-model:value="form.customerContractNo" /></a-form-item>
        <a-form-item label="内部合同号"><a-input v-model:value="form.innerContractNo" /></a-form-item>
        <a-form-item label="业务员"><a-input v-model:value="form.salesman" /></a-form-item>
        <a-form-item label="订单数量"
          ><a-input-number v-model:value="form.orderQty" :min="0" :precision="0" style="width: 100%"
        /></a-form-item>
        <a-form-item label="订单面积"
          ><a-input-number v-model:value="form.orderArea" :min="0" :precision="2" style="width: 100%"
        /></a-form-item>
        <a-form-item label="计价方式"><a-input v-model:value="form.pricingManner" /></a-form-item>
        <a-form-item label="付款方式"><a-input v-model:value="form.payment" /></a-form-item>
        <a-form-item label="付款节点"><a-input v-model:value="form.paymentPoint" /></a-form-item>
        <a-form-item label="材质"><a-input v-model:value="form.texture" /></a-form-item>
        <a-form-item label="喷涂类型"><a-input v-model:value="form.paintType" /></a-form-item>
        <a-form-item label="包装"><a-input v-model:value="form.packing" /></a-form-item>
        <a-form-item label="收货人"><a-input v-model:value="form.receiver" /></a-form-item>
        <a-form-item label="收货电话"><a-input v-model:value="form.receiverPhone" /></a-form-item>
        <a-form-item label="收货地址"><a-input v-model:value="form.receiverAddress" /></a-form-item>
        <a-form-item label="备注"><a-textarea v-model:value="form.remark" :rows="3" /></a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<style scoped>
.contracts {
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
.contracts label {
  display: grid;
  gap: 6px;
  min-width: 160px;
  color: var(--gz-secondary);
  font-size: 12px;
}
.numeric {
  font-variant-numeric: tabular-nums;
}
.drawer-footer {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  padding: 12px 20px;
  background: var(--gz-card);
  border-top: 1px solid var(--gz-border);
  text-align: right;
}
</style>
