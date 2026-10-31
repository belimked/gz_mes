<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { message } from 'ant-design-vue'
import type { FormInstance, TablePaginationConfig } from 'ant-design-vue'
import { ReloadOutlined } from '@ant-design/icons-vue'
import dayjs from 'dayjs'
import { call, client, errorMessage, type Schemas } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import PageCard from '@/components/PageCard.vue'
import FilterCard from '@/components/FilterCard.vue'
import TableToolbar from '@/components/TableToolbar.vue'
import CodeText from '@/components/CodeText.vue'
import StatusTag from '@/components/StatusTag.vue'

type Customer = Schemas['Customer']
const auth = useAuthStore()
const factories = ref<Schemas['Factory'][]>([])
const factoryName = computed(() => new Map(factories.value.map((f) => [f.id, f.name])))
const filters = reactive({
  keyword: '',
  factoryId: undefined as number | undefined,
  isInternal: undefined as boolean | undefined,
  confirmed: undefined as boolean | undefined,
  dates: [] as string[],
})
const applied = reactive({ ...filters })
const loading = ref(false)
const rows = ref<Customer[]>([])
const selected = ref<Customer>()
const pager = reactive({ current: 1, pageSize: 20, total: 0 })
const visible = computed(() =>
  rows.value.filter(
    (c) =>
      (applied.factoryId === undefined || c.factoryId === applied.factoryId) &&
      (applied.isInternal === undefined || c.isInternal === applied.isInternal) &&
      (applied.confirmed === undefined || c.confirmed === applied.confirmed) &&
      (!applied.dates?.length ||
        (dayjs(c.createdAt).format('YYYY-MM-DD') >= applied.dates[0]! && dayjs(c.createdAt).format('YYYY-MM-DD') <= applied.dates[1]!)),
  ),
)
const columns = [
  { title: '客户名称', dataIndex: 'name', ellipsis: true },
  { title: 'EAS客户编码', dataIndex: 'easCode', width: 150 },
  { title: 'EAS客户名称', dataIndex: 'easName', width: 160, ellipsis: true },
  { title: '联系人', dataIndex: 'contacts', width: 110 },
  { title: '电话', dataIndex: 'phone', width: 140 },
  { title: '厂区', dataIndex: 'factoryId', width: 110 },
  { title: '内部客户', dataIndex: 'isInternal', width: 100 },
  { title: '确认状态', dataIndex: 'confirmed', width: 100 },
  { title: '创建人', dataIndex: 'createdBy', width: 110 },
  { title: '创建时间', dataIndex: 'createdAt', width: 170 },
  { title: '操作', key: 'action', width: 120, fixed: 'right' as const },
]
async function load() {
  loading.value = true
  try {
    const r = await call(
      client.GET('/customers', {
        params: { query: { page: pager.current, size: pager.pageSize, keyword: applied.keyword.trim() || undefined } },
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
    factoryId: auth.user?.isAdmin ? undefined : factories.value[0]?.id,
    isInternal: undefined,
    confirmed: undefined,
    dates: [],
  })
  search()
}
function onTableChange(p: TablePaginationConfig) {
  pager.current = p.current ?? 1
  pager.pageSize = p.pageSize ?? 20
  void load()
}
async function confirm(c: Customer) {
  try {
    await call(client.POST('/customers/{id}/confirm', { params: { path: { id: c.id } } }))
    message.success('已确认')
    await load()
  } catch (e) {
    message.error(errorMessage(e))
  }
}
const open = ref(false)
const saving = ref(false)
const formRef = ref<FormInstance>()
const emptyForm = (): Schemas['CustomerCreate'] => ({
  factoryId: factories.value[0]?.id ?? 0,
  name: '',
  contacts: '',
  phone: '',
  address: '',
  bank: '',
  account: '',
  easCode: '',
  easName: '',
  isInternal: false,
})
const form = ref<Schemas['CustomerCreate']>(emptyForm())
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
    await call(client.POST('/customers', { body: form.value }))
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
    factories.value = await call(client.GET('/factories'))
    if (!auth.user?.isAdmin) {
      filters.factoryId = factories.value[0]?.id
      applied.factoryId = filters.factoryId
    }
  } catch (e) {
    message.error(errorMessage(e))
  }
  await load()
})
</script>

<template>
  <div class="customers">
    <div class="heading">
      <div>
        <h1>客户管理</h1>
        <span>维护客户档案，确认后锁定</span>
      </div>
      <a-tag>共 {{ pager.total }} 个客户</a-tag>
    </div>
    <FilterCard @search="search" @reset="reset">
      <template #default="{ expanded }">
        <label
          >关键字<a-input
            v-model:value="filters.keyword"
            placeholder="客户名 / EAS编码 / EAS名称"
            allow-clear
            data-testid="customer-search"
            @press-enter="search"
        /></label>
        <label
          >厂区<a-select
            v-model:value="filters.factoryId"
            allow-clear
            placeholder="全部厂区"
            :options="factories.map((f) => ({ value: f.id, label: f.name }))"
            :disabled="!auth.user?.isAdmin"
        /></label>
        <label
          >内部客户<a-select
            v-model:value="filters.isInternal"
            allow-clear
            placeholder="全部"
            :options="[
              { value: true, label: '是' },
              { value: false, label: '否' },
            ]"
        /></label>
        <label
          >确认状态<a-select
            v-model:value="filters.confirmed"
            allow-clear
            placeholder="全部状态"
            :options="[
              { value: true, label: '已确认' },
              { value: false, label: '未确认' },
            ]"
        /></label>
        <label v-if="expanded">创建日期<a-range-picker v-model:value="filters.dates" value-format="YYYY-MM-DD" /></label>
      </template>
    </FilterCard>
    <PageCard>
      <TableToolbar
        ><template #left
          ><a-button type="primary" data-testid="customer-new" @click="openCreate">新建客户</a-button
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
        :scroll="{ x: 1500 }"
        :pagination="{ ...pager, showSizeChanger: true, showTotal: (t: number) => `共 ${t} 条` }"
        data-testid="customer-table"
        @change="onTableChange"
      >
        <template #emptyText
          ><a-empty description="暂无匹配的客户"><a-button type="primary" @click="reset">重置查询条件</a-button></a-empty></template
        >
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'easCode'"><CodeText :value="record.easCode || '—'" /></template>
          <template v-else-if="column.dataIndex === 'factoryId'">{{ factoryName.get(record.factoryId) ?? record.factoryId }}</template>
          <template v-else-if="column.dataIndex === 'isInternal'"
            ><StatusTag :tone="record.isInternal ? 'processing' : 'neutral'" :label="record.isInternal ? '是' : '否'"
          /></template>
          <template v-else-if="column.dataIndex === 'confirmed'"
            ><StatusTag :tone="record.confirmed ? 'success' : 'neutral'" :label="record.confirmed ? '已确认' : '未确认'"
          /></template>
          <template v-else-if="column.dataIndex === 'createdAt'">{{ dayjs(record.createdAt).format('YYYY-MM-DD HH:mm') }}</template>
          <template v-else-if="column.key === 'action'"
            ><a-space
              ><a-popconfirm v-if="!record.confirmed" title="确认后客户将锁定，不能再修改。确定？" @confirm="confirm(record as Customer)"
                ><a :data-testid="`confirm-${record.id}`">确认</a></a-popconfirm
              ><a @click="selected = record as Customer">详情</a></a-space
            ></template
          >
        </template>
      </a-table>
    </PageCard>
    <a-drawer :open="!!selected" :width="560" placement="right" @close="selected = undefined">
      <template #title>客户详情: {{ selected?.name }}</template>
      <template v-if="selected"
        ><a-descriptions bordered :column="2" size="small">
          <a-descriptions-item label="客户名称">{{ selected.name }}</a-descriptions-item>
          <a-descriptions-item label="厂区">{{ factoryName.get(selected.factoryId) ?? selected.factoryId }}</a-descriptions-item>
          <a-descriptions-item label="EAS客户编码"><CodeText :value="selected.easCode || '—'" /></a-descriptions-item>
          <a-descriptions-item label="EAS客户名称">{{ selected.easName || '—' }}</a-descriptions-item>
          <a-descriptions-item label="联系人">{{ selected.contacts || '—' }}</a-descriptions-item>
          <a-descriptions-item label="电话">{{ selected.phone || '—' }}</a-descriptions-item>
          <a-descriptions-item label="地址" :span="2">{{ selected.address || '—' }}</a-descriptions-item>
          <a-descriptions-item label="开户行">{{ selected.bank || '—' }}</a-descriptions-item>
          <a-descriptions-item label="账号">{{ selected.account || '—' }}</a-descriptions-item>
          <a-descriptions-item label="内部客户"
            ><StatusTag :tone="selected.isInternal ? 'processing' : 'neutral'" :label="selected.isInternal ? '是' : '否'"
          /></a-descriptions-item>
          <a-descriptions-item label="确认状态"
            ><StatusTag :tone="selected.confirmed ? 'success' : 'neutral'" :label="selected.confirmed ? '已确认' : '未确认'"
          /></a-descriptions-item>
          <a-descriptions-item label="创建人">{{ selected.createdBy }}</a-descriptions-item>
          <a-descriptions-item label="创建时间">{{ dayjs(selected.createdAt).format('YYYY-MM-DD HH:mm') }}</a-descriptions-item>
        </a-descriptions>
        <div class="drawer-footer">
          <a-popconfirm v-if="!selected.confirmed" title="确认后客户将锁定，不能再修改。确定？" @confirm="confirm(selected!)"
            ><a-button type="primary">确认客户</a-button></a-popconfirm
          >
        </div></template
      >
    </a-drawer>
    <a-modal v-model:open="open" title="新建客户" :confirm-loading="saving" ok-text="保存" destroy-on-close @ok="save">
      <a-form ref="formRef" :model="form" :label-col="{ span: 5 }">
        <a-form-item label="厂区" name="factoryId" :rules="[{ required: true, message: '请选择厂区' }]"
          ><a-select v-model:value="form.factoryId" :options="factories.map((f) => ({ value: f.id, label: f.name }))"
        /></a-form-item>
        <a-form-item label="客户名称" name="name" :rules="[{ required: true, whitespace: true, message: '请输入客户名称' }]"
          ><a-input v-model:value="form.name" :maxlength="200"
        /></a-form-item>
        <a-form-item label="联系人"><a-input v-model:value="form.contacts" /></a-form-item>
        <a-form-item label="电话"><a-input v-model:value="form.phone" /></a-form-item>
        <a-form-item label="地址"><a-input v-model:value="form.address" /></a-form-item>
        <a-form-item label="开户行"><a-input v-model:value="form.bank" /></a-form-item>
        <a-form-item label="账号"><a-input v-model:value="form.account" /></a-form-item>
        <a-form-item label="EAS编码"><a-input v-model:value="form.easCode" /></a-form-item>
        <a-form-item label="EAS名称"><a-input v-model:value="form.easName" /></a-form-item>
        <a-form-item label="内部客户"><a-switch v-model:checked="form.isInternal" /></a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<style scoped>
.customers {
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
.customers label {
  display: grid;
  gap: 6px;
  min-width: 160px;
  color: var(--gz-secondary);
  font-size: 12px;
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
