<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { message, type TablePaginationConfig } from 'ant-design-vue'
import { DownloadOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import dayjs from 'dayjs'
import { call, client, errorMessage, type Schemas } from '@/api/client'
import PageCard from '@/components/PageCard.vue'
import TableToolbar from '@/components/TableToolbar.vue'
import CodeText from '@/components/CodeText.vue'
import { downloadCsv, exportFilename, fetchAllPages } from '@/utils/export'

type Plan = Schemas['DeliveryPlan']
type Input = Schemas['DeliveryPlanInput']
const formFields: {
  key: keyof Pick<
    Input,
    | 'logisticsCompany'
    | 'contact'
    | 'phone'
    | 'shippingAddress'
    | 'receivingAddress'
    | 'carModel'
    | 'plateNo'
    | 'driver'
    | 'driverPhone'
    | 'remark'
  >
  label: string
}[] = [
  { key: 'logisticsCompany', label: '物流公司' },
  { key: 'contact', label: '联系人' },
  { key: 'phone', label: '联系电话' },
  { key: 'shippingAddress', label: '发货地址' },
  { key: 'receivingAddress', label: '收货地址' },
  { key: 'carModel', label: '车型' },
  { key: 'plateNo', label: '车牌' },
  { key: 'driver', label: '司机' },
  { key: 'driverPhone', label: '司机电话' },
  { key: 'remark', label: '备注' },
]
function blank(): Input {
  return {
    factoryId: 0,
    estimatedDepartureDate: dayjs().format('YYYY-MM-DD'),
    noteIds: [],
    logisticsCompany: '',
    contact: '',
    phone: '',
    shippingAddress: '',
    receivingAddress: '',
    carModel: '',
    plateNo: '',
    driver: '',
    driverPhone: '',
    cost: 0,
    extraCost: 0,
    remark: '',
  }
}
const filters = reactive({ created: [] as string[], estimated: [] as string[], confirmed: undefined as boolean | undefined, keyword: '' })
const applied = reactive({ created: [] as string[], estimated: [] as string[], confirmed: undefined as boolean | undefined, keyword: '' })
const rows = ref<Plan[]>([])
const summary = ref<Schemas['DeliveryPlanPage']['summary']>()
const factories = ref<Schemas['Factory'][]>([])
const notes = ref<Schemas['DeliveryNote'][]>([])
const selected = ref<Schemas['DeliveryPlanDetail'] | null>(null)
const pager = reactive({ current: 1, pageSize: 20, total: 0 })
const loading = ref(false)
const exporting = ref(false)
const editing = ref(false)
const detailVisible = ref(false)
const confirmVisible = ref(false)
const saving = ref(false)
const editId = ref<number | null>(null)
const departureDate = ref(dayjs().format('YYYY-MM-DD'))
const form = reactive<Input>(blank())
const columns = [
  { title: '计划号', dataIndex: 'planNo', width: 160, fixed: 'left' as const },
  { title: '物流公司', dataIndex: 'logisticsCompany', width: 140 },
  { title: '联系人/电话', dataIndex: 'contact', width: 170 },
  { title: '发货→收货地址', dataIndex: 'address', width: 240 },
  { title: '车型/车牌', dataIndex: 'vehicle', width: 150 },
  { title: '司机/电话', dataIndex: 'driver', width: 170 },
  { title: '运费/额外费用', dataIndex: 'cost', width: 150 },
  { title: '预计发车', dataIndex: 'estimatedDepartureDate', width: 120 },
  { title: '确认发车', dataIndex: 'confirmedDepartureDate', width: 120 },
  { title: '发货单/件/面积', dataIndex: 'noteCount', width: 170 },
  { title: '状态', dataIndex: 'state', width: 100 },
  { title: '操作', dataIndex: 'actions', width: 220, fixed: 'right' as const },
]
const noteColumns = [
  { title: '发货单号', dataIndex: 'noteNo' },
  { title: '批次', dataIndex: 'lotNo' },
  { title: '件数', dataIndex: 'qty' },
  { title: '面积', dataIndex: 'area' },
]
function fetchPage(page: number, size: number) {
  return call(
    client.GET('/delivery-plans', {
      params: {
        query: {
          createdFrom: applied.created[0],
          createdTo: applied.created[1],
          estimatedFrom: applied.estimated[0],
          estimatedTo: applied.estimated[1],
          confirmed: applied.confirmed,
          keyword: applied.keyword.trim() || undefined,
          page,
          size,
        },
      },
    }),
  )
}
async function load() {
  loading.value = true
  try {
    const result = await fetchPage(pager.current, pager.pageSize)
    rows.value = result.items
    summary.value = result.summary
    pager.total = result.total
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    loading.value = false
  }
}
function search() {
  Object.assign(applied, filters, { created: [...filters.created], estimated: [...filters.estimated] })
  pager.current = 1
  void load()
}
function reset() {
  filters.created = []
  filters.estimated = []
  filters.confirmed = undefined
  filters.keyword = ''
  search()
}
function onTableChange(p: TablePaginationConfig) {
  pager.current = p.current ?? 1
  pager.pageSize = p.pageSize ?? 20
  void load()
}
async function exportCsv() {
  exporting.value = true
  try {
    const all = await fetchAllPages(fetchPage)
    downloadCsv(
      exportFilename('发货计划'),
      [
        { title: '计划号', value: (p: Plan) => p.planNo },
        { title: '物流公司', value: (p: Plan) => p.logisticsCompany },
        { title: '联系人', value: (p: Plan) => p.contact },
        { title: '联系电话', value: (p: Plan) => p.phone },
        { title: '发货地址', value: (p: Plan) => p.shippingAddress },
        { title: '收货地址', value: (p: Plan) => p.receivingAddress },
        { title: '车型', value: (p: Plan) => p.carModel },
        { title: '车牌', value: (p: Plan) => p.plateNo },
        { title: '司机', value: (p: Plan) => p.driver },
        { title: '司机电话', value: (p: Plan) => p.driverPhone },
        { title: '运费', value: (p: Plan) => p.cost },
        { title: '额外费用', value: (p: Plan) => p.extraCost },
        { title: '预计发车', value: (p: Plan) => p.estimatedDepartureDate },
        { title: '确认发车', value: (p: Plan) => p.confirmedDepartureDate ?? '' },
        { title: '发货单数', value: (p: Plan) => p.noteCount },
        { title: '件数', value: (p: Plan) => p.qty },
        { title: '面积', value: (p: Plan) => p.area },
      ],
      all,
    )
    message.success(`已导出 ${all.length} 条`)
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    exporting.value = false
  }
}
async function show(p: Plan) {
  try {
    selected.value = await call(client.GET('/delivery-plans/{id}', { params: { path: { id: p.id } } }))
    detailVisible.value = true
  } catch (e) {
    message.error(errorMessage(e))
  }
}
async function loadNotes() {
  if (!form.factoryId) {
    notes.value = []
    return
  }
  try {
    const result = await call(
      client.GET('/delivery-notes', {
        params: { query: { factoryId: form.factoryId, unplanned: true, page: 1, size: 200 } },
      }),
    )
    notes.value = result.items
    if (editId.value) {
      const current = await call(client.GET('/delivery-plans/{id}', { params: { path: { id: editId.value } } }))
      notes.value = [...current.notes, ...notes.value.filter((n) => !current.notes.some((x) => x.id === n.id))]
    }
  } catch (e) {
    message.error(errorMessage(e))
  }
}
function create() {
  editId.value = null
  Object.assign(form, blank())
  notes.value = []
  editing.value = true
}
function edit(p: Plan) {
  editId.value = p.id
  Object.assign(form, {
    factoryId: p.factoryId,
    logisticsCompany: p.logisticsCompany,
    contact: p.contact,
    phone: p.phone,
    shippingAddress: p.shippingAddress,
    receivingAddress: p.receivingAddress,
    carModel: p.carModel,
    plateNo: p.plateNo,
    driver: p.driver,
    driverPhone: p.driverPhone,
    cost: p.cost,
    extraCost: p.extraCost,
    estimatedDepartureDate: p.estimatedDepartureDate,
    remark: p.remark,
    noteIds: [],
  })
  detailVisible.value = false
  editing.value = true
  void loadNotes().then(() => {
    form.noteIds = notes.value.filter((n) => n.planId === p.id).map((n) => n.id)
  })
}
function changeFactory() {
  form.noteIds = []
  void loadNotes()
}
async function save() {
  saving.value = true
  try {
    if (editId.value) {
      await call(client.PUT('/delivery-plans/{id}', { params: { path: { id: editId.value } }, body: form }))
    } else {
      await call(client.POST('/delivery-plans', { body: form }))
    }
    editing.value = false
    message.success('已保存')
    await load()
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    saving.value = false
  }
}
async function confirm() {
  if (!selected.value) return
  try {
    await call(
      client.POST('/delivery-plans/{id}/confirm', {
        params: { path: { id: selected.value.id } },
        body: { departureDate: departureDate.value },
      }),
    )
    confirmVisible.value = false
    detailVisible.value = false
    message.success('已确认发车')
    await load()
  } catch (e) {
    message.error(errorMessage(e))
  }
}
async function remove(p: Plan) {
  try {
    await call(client.DELETE('/delivery-plans/{id}', { params: { path: { id: p.id } } }))
    detailVisible.value = false
    message.success('已删除')
    await load()
  } catch (e) {
    message.error(errorMessage(e))
  }
}
function confirmFromRow(p: Plan) {
  void show(p).then(() => {
    if (selected.value?.id === p.id) confirmVisible.value = true
  })
}
onMounted(() => {
  search()
  void call(client.GET('/factories'))
    .then((result) => {
      factories.value = result
    })
    .catch((e) => message.error(errorMessage(e)))
})
</script>
<template>
  <div class="report">
    <div class="heading">
      <div>
        <h1>发货计划</h1>
        <span>管理计划物流信息、关联发货单并确认发车</span>
      </div>
    </div>
    <PageCard>
      <div class="filters">
        <label>创建日期<a-range-picker v-model:value="filters.created" value-format="YYYY-MM-DD" /></label>
        <label>预计发车<a-range-picker v-model:value="filters.estimated" value-format="YYYY-MM-DD" /></label>
        <label
          >确认状态<a-select
            v-model:value="filters.confirmed"
            allow-clear
            placeholder="全部"
            :options="[
              { label: '已确认', value: true },
              { label: '未确认', value: false },
            ]"
        /></label>
        <label
          >快速查询<a-input
            v-model:value="filters.keyword"
            placeholder="计划号 / 物流公司 / 车牌 / 司机 / 发货单号"
            allow-clear
            @press-enter="search"
        /></label>
        <div class="actions">
          <a-button type="primary" @click="search">查询</a-button>
          <a-button @click="reset">重置</a-button>
        </div>
      </div>
    </PageCard>
    <PageCard>
      <TableToolbar>
        <template #left>
          <div v-if="summary" class="summary" data-testid="plans-summary">
            <span
              >计划 <b>{{ summary.plans }}</b></span
            >
            <span
              >已确认 <b>{{ summary.confirmed }}</b></span
            >
            <span
              >运费 <b>{{ summary.cost.toFixed(2) }}</b></span
            >
            <span
              >额外费用 <b>{{ summary.extraCost.toFixed(2) }}</b></span
            >
            <span
              >发货 <b>{{ summary.qty }}</b> 件 / {{ summary.area.toFixed(2) }} m²</span
            >
          </div>
        </template>
        <template #right>
          <a-button type="primary" @click="create">新建计划</a-button>
          <a-button :loading="exporting" data-testid="export-csv" @click="exportCsv"><DownloadOutlined />导出 CSV</a-button>
          <a-button aria-label="刷新" @click="load"><ReloadOutlined /></a-button>
        </template>
      </TableToolbar>
      <a-table
        row-key="id"
        size="middle"
        bordered
        :columns="columns"
        :data-source="rows"
        :loading="loading"
        :scroll="{ x: 2000 }"
        :pagination="{
          ...pager,
          showSizeChanger: true,
          pageSizeOptions: ['20', '50', '100', '200'],
          showTotal: (n: number) => `共 ${n} 条`,
        }"
        data-testid="plans-table"
        @change="onTableChange"
      >
        <template #emptyText><a-empty description="没有符合条件的发货计划" /></template>
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'planNo'"
            ><a @click="show(record)"><CodeText :value="record.planNo" /></a
          ></template>
          <template v-else-if="column.dataIndex === 'contact'">{{ record.contact }} / {{ record.phone }}</template>
          <template v-else-if="column.dataIndex === 'address'">{{ record.shippingAddress }} → {{ record.receivingAddress }}</template>
          <template v-else-if="column.dataIndex === 'vehicle'">{{ record.carModel }} / {{ record.plateNo }}</template>
          <template v-else-if="column.dataIndex === 'driver'">{{ record.driver }} / {{ record.driverPhone }}</template>
          <template v-else-if="column.dataIndex === 'cost'">{{ record.cost }} / {{ record.extraCost }}</template>
          <template v-else-if="column.dataIndex === 'noteCount'">{{ record.noteCount }} / {{ record.qty }} / {{ record.area }}</template>
          <a-tag v-else-if="column.dataIndex === 'state'" :color="record.confirmedAt ? 'green' : 'orange'">{{
            record.confirmedAt ? '已确认' : '未确认'
          }}</a-tag>
          <a-space v-else-if="column.dataIndex === 'actions' && !record.confirmedAt">
            <a @click="edit(record)">编辑</a>
            <a @click="confirmFromRow(record)">确认发车</a>
            <a-popconfirm title="确定删除？" @confirm="remove(record)"><a>删除</a></a-popconfirm>
          </a-space>
        </template>
      </a-table>
    </PageCard>
    <a-modal v-model:open="editing" :title="editId ? '编辑发货计划' : '新建发货计划'" :confirm-loading="saving" width="760px" @ok="save">
      <div class="form">
        <label
          >厂区<a-select
            v-model:value="form.factoryId"
            :options="factories.map((f) => ({ label: f.name, value: f.id }))"
            style="width: 100%"
            @change="changeFactory"
        /></label>
        <label
          >发货单<a-select
            v-model:value="form.noteIds"
            mode="multiple"
            :options="notes.map((n) => ({ label: `${n.noteNo} (${n.lotNo})`, value: n.id }))"
            style="width: 100%"
            placeholder="选择未挂计划的发货单"
        /></label>
        <label>预计发车<a-date-picker v-model:value="form.estimatedDepartureDate" value-format="YYYY-MM-DD" style="width: 100%" /></label>
        <label v-for="field in formFields" :key="field.key">{{ field.label }}<a-input v-model:value="form[field.key]" /></label>
        <label>运费<a-input-number v-model:value="form.cost" :min="0" style="width: 100%" /></label>
        <label>额外费用<a-input-number v-model:value="form.extraCost" :min="0" style="width: 100%" /></label>
      </div>
    </a-modal>
    <a-drawer v-model:open="detailVisible" title="发货计划详情" width="760">
      <template v-if="selected">
        <a-descriptions bordered :column="2">
          <a-descriptions-item label="计划号"><CodeText :value="selected.planNo" /></a-descriptions-item>
          <a-descriptions-item label="物流公司">{{ selected.logisticsCompany }}</a-descriptions-item>
          <a-descriptions-item label="联系人/电话">{{ selected.contact }} / {{ selected.phone }}</a-descriptions-item>
          <a-descriptions-item label="发货地址">{{ selected.shippingAddress }}</a-descriptions-item>
          <a-descriptions-item label="收货地址">{{ selected.receivingAddress }}</a-descriptions-item>
          <a-descriptions-item label="车型/车牌">{{ selected.carModel }} / {{ selected.plateNo }}</a-descriptions-item>
          <a-descriptions-item label="司机/电话">{{ selected.driver }} / {{ selected.driverPhone }}</a-descriptions-item>
          <a-descriptions-item label="运费/额外费用">{{ selected.cost }} / {{ selected.extraCost }}</a-descriptions-item>
          <a-descriptions-item label="预计发车">{{ selected.estimatedDepartureDate }}</a-descriptions-item>
          <a-descriptions-item label="确认发车">{{ selected.confirmedDepartureDate ?? '未确认' }}</a-descriptions-item>
          <a-descriptions-item label="备注" :span="2">{{ selected.remark || '—' }}</a-descriptions-item>
        </a-descriptions>
        <h3>关联发货单</h3>
        <a-table row-key="id" size="middle" bordered :data-source="selected.notes" :pagination="false" :columns="noteColumns">
          <template #emptyText><a-empty description="暂无关联发货单" /></template>
          <template #bodyCell="{ column, record }">
            <CodeText v-if="column.dataIndex === 'noteNo'" :value="record.noteNo" />
            <CodeText v-else-if="column.dataIndex === 'lotNo'" :value="record.lotNo" />
          </template>
        </a-table>
      </template>
      <template v-if="selected && !selected.confirmedAt" #footer>
        <a-space>
          <a-button @click="edit(selected)">编辑</a-button>
          <a-button type="primary" @click="confirmVisible = true">确认发车</a-button>
          <a-popconfirm title="确定删除？" @confirm="remove(selected)"><a-button danger>删除</a-button></a-popconfirm>
        </a-space>
      </template>
    </a-drawer>
    <a-modal v-model:open="confirmVisible" title="确认发车" @ok="confirm">
      <a-date-picker v-model:value="departureDate" value-format="YYYY-MM-DD" />
    </a-modal>
  </div>
</template>
<style scoped>
.report {
  display: grid;
  gap: 16px;
}
.heading h1 {
  margin: 0 0 4px;
  font-size: 20px;
}
.heading span {
  color: var(--gz-muted);
  font-size: 12px;
}
.filters {
  display: flex;
  flex-wrap: wrap;
  align-items: end;
  gap: 12px;
}
.filters label {
  display: grid;
  gap: 6px;
  color: var(--gz-secondary);
  font-size: 12px;
  min-width: 150px;
}
.actions {
  display: flex;
  gap: 8px;
  margin-left: auto;
}
.summary {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 18px;
  font-size: 13px;
}
.summary b {
  color: var(--gz-primary);
}
.form {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}
.form label {
  display: grid;
  gap: 6px;
}
</style>
