<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { message } from 'ant-design-vue'
import type { FormInstance, TablePaginationConfig } from 'ant-design-vue'
import { ReloadOutlined } from '@ant-design/icons-vue'
import dayjs from 'dayjs'
import { call, client, errorMessage, type Schemas } from '@/api/client'
import { PROCESS_STEPS } from '@/theme'
import PageCard from '@/components/PageCard.vue'
import FilterCard from '@/components/FilterCard.vue'
import TableToolbar from '@/components/TableToolbar.vue'
import CodeText from '@/components/CodeText.vue'
import StatusTag from '@/components/StatusTag.vue'
import ProcessTimeline from '@/components/ProcessTimeline.vue'

type Batch = Schemas['Batch']
type Schedule = Schemas['BatchSchedule']
const statusLabels: Record<Batch['status'], string> = {
  pending: '待排产',
  scheduled: '已排产',
  in_production: '生产中',
  done: '已完工',
}
const statusTones = { pending: 'neutral', scheduled: 'processing', in_production: 'processing', done: 'success' } as const
const filters = reactive({
  keyword: '',
  status: undefined as Batch['status'] | undefined,
  urgency: undefined as Batch['urgency'] | undefined,
  dates: [] as string[],
  projectId: undefined as number | undefined,
  contractId: undefined as number | undefined,
})
const applied = reactive({ ...filters })
const rows = ref<Batch[]>([])
const jobs = ref<Schemas['Job'][]>([])
const projects = ref<Schemas['Project'][]>([])
const contracts = ref<Schemas['Contract'][]>([])
const factories = ref<Schemas['Factory'][]>([])
const pager = reactive({ current: 1, pageSize: 20, total: 0 })
const loading = ref(false)
const updatedAt = ref('')
const selected = ref<Batch>()
const tab = ref('基本信息')
const items = ref<Schemas['BatchItem'][]>([])
const itemsLoading = ref(false)
const progress = ref<Schemas['BatchProgress']>()
const progressLoading = ref(false)
const generating = ref(false)
const currentStage = computed(() => Math.max(0, ...(progress.value?.jobs.filter((j) => j.scannedQty > 0).map((j) => j.seq) ?? [])))
async function loadProgress(id: number) {
  progressLoading.value = true
  progress.value = undefined
  try {
    const result = await call(client.GET('/batches/{id}/progress', { params: { path: { id } } }))
    if (selected.value?.id === id) progress.value = result
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    progressLoading.value = false
  }
}
async function generateLabels() {
  if (!selected.value) return
  const id = selected.value.id
  generating.value = true
  try {
    const result = await call(client.POST('/batches/{id}/labels/generate', { params: { path: { id } } }))
    message.success(`新建 ${result.created} 张，共 ${result.total} 张`)
    await loadProgress(id)
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    generating.value = false
  }
}
const totals = computed(() => ({
  qty: rows.value.reduce((n, b) => n + b.orderQty, 0),
  area: rows.value.reduce((n, b) => n + b.orderArea, 0).toFixed(2),
  itemArea: rows.value.reduce((n, b) => n + b.itemArea, 0).toFixed(2),
}))
const columns = [
  { title: '批次编号', dataIndex: 'lotNo', width: 155 },
  { title: '工程项目', dataIndex: 'projectName', width: 170 },
  { title: '合同号', dataIndex: 'contractNo', width: 150 },
  { title: '厚度 · 颜色', dataIndex: 'thickness', width: 120 },
  { title: '订单件数', dataIndex: 'orderQty', width: 90, align: 'right' as const },
  { title: '订单面积', dataIndex: 'orderArea', width: 100, align: 'right' as const },
  { title: '构件', dataIndex: 'itemQty', width: 120 },
  { title: '阶段', dataIndex: 'stage', width: 110 },
  { title: '需求交期', dataIndex: 'requiredDate', width: 115 },
  { title: '紧急', dataIndex: 'urgency', width: 70 },
  { title: '状态', dataIndex: 'status', width: 90 },
  { title: '操作', key: 'action', width: 105, fixed: 'right' as const },
]
async function load() {
  loading.value = true
  try {
    const result = await call(
      client.GET('/batches', {
        params: {
          query: {
            page: pager.current,
            size: pager.pageSize,
            keyword: applied.keyword.trim() || undefined,
            status: applied.status,
            urgency: applied.urgency,
            projectId: applied.projectId,
            contractId: applied.contractId,
            beginDate: applied.dates?.[0],
            endDate: applied.dates?.[1],
          },
        },
      }),
    )
    rows.value = result.items
    pager.total = result.total
    updatedAt.value = dayjs().format('YYYY-MM-DD HH:mm:ss')
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
  Object.assign(filters, { keyword: '', status: undefined, urgency: undefined, dates: [], projectId: undefined, contractId: undefined })
  search()
}
function onTableChange(p: TablePaginationConfig) {
  pager.current = p.current ?? 1
  pager.pageSize = p.pageSize ?? 20
  void load()
}
const scheduleKeys = [
  'planTechDate',
  'planDrawingDate',
  'planCutDate',
  'planSheetStart',
  'planSheetDelivery',
  'planSprayStart',
  'planSprayDelivery',
  'outsourceFactoryId',
  'workshop',
  'urgency',
  'remark',
] as const
const dateKeys = scheduleKeys.slice(0, 7)
const schedule = ref<Schedule>({})
function fillSchedule(b: Batch) {
  schedule.value = Object.fromEntries(
    scheduleKeys.map((key) => [key, b[key] ?? (dateKeys.includes(key) || key === 'outsourceFactoryId' ? null : '')]),
  ) as Schedule
}
async function refreshSelected(id: number) {
  const batch = await call(client.GET('/batches/{id}', { params: { path: { id } } }))
  if (selected.value?.id === id) {
    selected.value = batch
    fillSchedule(batch)
  }
  await load()
}
async function openBatch(b: Batch, target = '基本信息') {
  selected.value = b
  tab.value = target
  items.value = []
  fillSchedule(b)
  void loadProgress(b.id)
  try {
    await refreshSelected(b.id)
  } catch (e) {
    message.error(errorMessage(e))
  }
}
watch(tab, async (value) => {
  if (value !== '构件清单' || !selected.value) return
  itemsLoading.value = true
  try {
    items.value = await call(client.GET('/batches/{id}/items', { params: { path: { id: selected.value.id } } }))
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    itemsLoading.value = false
  }
})
const saving = ref(false)
async function saveSchedule() {
  const b = selected.value
  if (!b || b.status === 'done') return
  const changes: Schedule = {}
  for (const key of scheduleKeys) {
    const before = b[key] ?? (dateKeys.includes(key) || key === 'outsourceFactoryId' ? null : '')
    if (schedule.value[key] !== before) Object.assign(changes, { [key]: schedule.value[key] })
  }
  if (!Object.keys(changes).length) {
    message.info('排程未修改')
    return
  }
  saving.value = true
  try {
    await call(client.PUT('/batches/{id}/schedule', { params: { path: { id: b.id } }, body: changes }))
    await refreshSelected(b.id)
    message.success('排程已保存')
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    saving.value = false
  }
}
const createOpen = ref(false)
const createRef = ref<FormInstance>()
const createForm = ref<Schemas['BatchCreate']>({
  contractId: undefined as unknown as number,
  lotNo: '',
  thickness: 0,
  orderQty: 0,
  orderArea: 0,
  urgency: 'normal',
})
function openCreate() {
  createForm.value = { contractId: undefined as unknown as number, lotNo: '', thickness: 0, orderQty: 0, orderArea: 0, urgency: 'normal' }
  createOpen.value = true
}
async function saveCreate() {
  try {
    await createRef.value?.validate()
  } catch {
    return
  }
  saving.value = true
  try {
    await call(client.POST('/batches', { body: createForm.value }))
    message.success('批次已创建')
    createOpen.value = false
    search()
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    saving.value = false
  }
}
const importOpen = ref(false)
const importText = ref('')
const importing = ref(false)
function openImport() {
  importText.value = ''
  importOpen.value = true
}
function parseItems(text: string, batch: Batch): Schemas['BatchItemInput'][] {
  const lines = text.trim().split(/\r?\n/)
  if (!text.trim()) throw new Error('请粘贴构件清单')
  if (lines.length > 5000) throw new Error('最多导入 5000 行')
  const seen = new Set<string>()
  return lines.map((line, index) => {
    const fields = line.split(line.includes('\t') ? '\t' : ',').map((x) => x.trim())
    const [
      pieceNo = '',
      drawingNo = '',
      rawLength = '',
      rawWidth = '',
      rawQty = '',
      pattern = '',
      color = '',
      rawPackage = '',
      rawBending = '',
      remark = '',
    ] = fields
    const length = Number(rawLength),
      width = Number(rawWidth),
      qty = Number(rawQty)
    const packageQty = rawPackage ? Number(rawPackage) : 0,
      bendingTimes = rawBending ? Number(rawBending) : 0
    if (
      fields.length < 5 ||
      fields.length > 10 ||
      !pieceNo ||
      seen.has(pieceNo) ||
      !rawLength ||
      !rawWidth ||
      !rawQty ||
      !Number.isFinite(length) ||
      length < 0 ||
      !Number.isFinite(width) ||
      width < 0 ||
      !Number.isInteger(qty) ||
      qty < 1 ||
      !Number.isInteger(packageQty) ||
      packageQty < 0 ||
      !Number.isInteger(bendingTimes) ||
      bendingTimes < 0
    )
      throw new Error(`第 ${index + 1} 行格式错误或编号重复`)
    seen.add(pieceNo)
    const pieceArea = Number(((length * width) / 1e6).toFixed(4))
    return {
      pieceNo,
      drawingNo,
      length,
      width,
      qty,
      pattern,
      color: color || batch.color,
      thickness: batch.thickness,
      pieceArea,
      sprayingArea: pieceArea,
      packageQty,
      bendingTimes,
      weldLine: 0,
      remark,
    }
  })
}
async function importItems() {
  const b = selected.value
  if (!b || b.status === 'done') return
  let body: Schemas['BatchItemInput'][]
  try {
    body = parseItems(importText.value, b)
  } catch (e) {
    message.error(errorMessage(e))
    return
  }
  importing.value = true
  try {
    items.value = await call(client.PUT('/batches/{id}/items', { params: { path: { id: b.id } }, body }))
    await refreshSelected(b.id)
    importOpen.value = false
    message.success('构件清单已替换')
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    importing.value = false
  }
}
onMounted(async () => {
  await Promise.all([
    load(),
    (async () => {
      try {
        jobs.value = await call(client.GET('/jobs'))
      } catch (e) {
        message.error(errorMessage(e))
      }
    })(),
    (async () => {
      try {
        projects.value = (await call(client.GET('/projects', { params: { query: { size: 200 } } }))).items
      } catch (e) {
        message.error(errorMessage(e))
      }
    })(),
    (async () => {
      try {
        contracts.value = (await call(client.GET('/contracts', { params: { query: { size: 200 } } }))).items
      } catch (e) {
        message.error(errorMessage(e))
      }
    })(),
    (async () => {
      try {
        factories.value = await call(client.GET('/factories'))
      } catch (e) {
        message.error(errorMessage(e))
      }
    })(),
  ])
})
</script>
<template>
  <div class="batches">
    <div class="heading">
      <div>
        <h1>生产批次查询</h1>
        <span>查询排产与工序执行进度，追溯构件全生命周期</span>
      </div>
      <StatusTag tone="processing" :label="`共 ${pager.total} 个批次`" />
    </div>
    <FilterCard @search="search" @reset="reset"
      ><template #default="{ expanded }">
        <label
          >关键字<a-input
            v-model:value="filters.keyword"
            placeholder="批次号 / 客户单号 / 合同号 / 客户 / 工程"
            allow-clear
            @press-enter="search"
        /></label>
        <label
          >生产状态<a-select
            v-model:value="filters.status"
            allow-clear
            placeholder="全部状态"
            :options="Object.entries(statusLabels).map(([value, label]) => ({ value, label }))"
        /></label>
        <label
          >紧急程度<a-select
            v-model:value="filters.urgency"
            allow-clear
            placeholder="全部"
            :options="[
              { value: 'normal', label: '普通' },
              { value: 'urgent', label: '加急' },
            ]"
        /></label>
        <label>需求交期<a-range-picker v-model:value="filters.dates" value-format="YYYY-MM-DD" /></label>
        <template v-if="expanded">
          <label
            >项目<a-select
              v-model:value="filters.projectId"
              allow-clear
              show-search
              option-filter-prop="label"
              placeholder="全部项目"
              :options="projects.map((p) => ({ value: p.id, label: p.fullName }))"
          /></label>
          <label
            >合同<a-select
              v-model:value="filters.contractId"
              allow-clear
              show-search
              option-filter-prop="label"
              placeholder="全部合同"
              :options="contracts.map((c) => ({ value: c.id, label: `${c.contractNo} · ${c.customerName} · ${c.projectName}` }))"
          /></label>
        </template> </template
    ></FilterCard>
    <PageCard>
      <TableToolbar
        ><template #left
          ><a-button type="primary" data-testid="batch-new" @click="openCreate">新建生产批次</a-button
          ><a-button disabled>导出</a-button></template
        ><template #right
          ><a-button aria-label="刷新" @click="load"><ReloadOutlined /></a-button></template
      ></TableToolbar>
      <a-table
        row-key="id"
        size="middle"
        :columns="columns"
        :data-source="rows"
        :loading="loading"
        :scroll="{ x: 1440 }"
        :pagination="{ ...pager, showSizeChanger: true, pageSizeOptions: ['20', '50', '100'], showTotal: (n: number) => `共 ${n} 条` }"
        data-testid="batch-table"
        @change="onTableChange"
      >
        <template #emptyText
          ><a-empty description="暂无匹配的生产批次"><a-button type="primary" @click="reset">重置查询条件</a-button></a-empty></template
        >
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'lotNo'"
            ><a @click="openBatch(record)"><CodeText :value="record.lotNo" /></a
          ></template>
          <template v-else-if="column.dataIndex === 'projectName'"
            ><strong>{{ record.projectName }}</strong>
            <div class="muted">{{ record.customerName }}</div></template
          >
          <template v-else-if="column.dataIndex === 'contractNo'"><CodeText :value="record.contractNo" /></template>
          <template v-else-if="column.dataIndex === 'thickness'"
            >{{ record.thickness }} mm
            <div class="muted">{{ record.color || '—' }}</div></template
          >
          <template v-else-if="column.dataIndex === 'orderArea'">{{ Number(record.orderArea).toFixed(2) }}</template>
          <template v-else-if="column.dataIndex === 'itemQty'"
            >{{ record.itemQty }} 件
            <div class="muted">{{ Number(record.itemArea).toFixed(2) }} m²</div></template
          >
          <template v-else-if="column.dataIndex === 'stage'"
            ><StatusTag
              tone="processing"
              :label="
                PROCESS_STEPS[
                  Math.min(({ pending: 0, scheduled: 1, in_production: 5, done: 10 } as const)[record.status as Batch['status']], 10)
                ] ?? '—'
              "
          /></template>
          <template v-else-if="column.dataIndex === 'requiredDate'"
            ><span :class="{ overdue: record.urgency === 'urgent' }">{{ record.requiredDate || '—' }}</span></template
          >
          <template v-else-if="column.dataIndex === 'urgency'"
            ><StatusTag v-if="record.urgency === 'urgent'" tone="error" label="加急" /><span v-else> </span
          ></template>
          <template v-else-if="column.dataIndex === 'status'"
            ><StatusTag :tone="statusTones[record.status as Batch['status']]" :label="statusLabels[record.status as Batch['status']]"
          /></template>
          <template v-else-if="column.key === 'action'"
            ><a @click="openBatch(record)">查看</a><a-divider type="vertical" /><a @click="openBatch(record, '排程')">排程</a></template
          >
        </template>
      </a-table>
      <div class="summary">
        <b>当前页数据汇总合计:</b><span>{{ rows.length }} 个批次</span
        ><span
          >订单件数: <b>{{ totals.qty }}</b> 件</span
        ><span
          >订单面积: <b>{{ totals.area }}</b> m²</span
        ><span
          >构件面积: <b>{{ totals.itemArea }}</b> m²</span
        ><span v-for="(label, key) in statusLabels" :key="key">{{ label }} {{ rows.filter((b) => b.status === key).length }}</span
        ><span>数据更新: {{ updatedAt || '—' }}</span>
      </div>
    </PageCard>
    <a-drawer :open="!!selected" :width="720" placement="right" @close="selected = undefined">
      <template #title
        >批次详情: {{ selected?.lotNo }}
        <StatusTag v-if="selected" :tone="statusTones[selected.status]" :label="statusLabels[selected.status]"
      /></template>
      <template v-if="selected">
        <a-tabs v-model:active-key="tab"
          ><a-tab-pane key="基本信息" tab="基本信息" /><a-tab-pane key="排程" tab="排程" /><a-tab-pane
            key="构件清单"
            :tab="`构件清单(${selected.itemCount})`" /><a-tab-pane key="工序进度" tab="工序进度"
        /></a-tabs>
        <div v-if="tab === '基本信息'" class="drawer-body">
          <h3>批次工程要素信息</h3>
          <a-descriptions bordered :column="2" size="small">
            <a-descriptions-item
              v-for="[label, value] in [
                ['合同', selected.contractNo],
                ['客户', selected.customerName],
                ['工程', selected.projectName],
                ['批次号', selected.lotNo],
                ['客户单号', selected.clientOrderNo],
                ['厚度', `${selected.thickness} mm`],
                ['颜色', selected.color],
                ['色号', selected.colorNo],
                ['材质', selected.texture],
                ['喷涂类型', selected.sprayingType],
                ['订单件数', selected.orderQty],
                ['订单面积', `${Number(selected.orderArea).toFixed(2)} m²`],
                ['需求交期', selected.requiredDate],
                ['紧急', selected.urgency === 'urgent' ? '加急' : '普通'],
                ['发货地址', selected.deliveryAddress],
                ['特殊要求', selected.specialRequest],
                ['备注', selected.remark],
                ['创建人', selected.createdBy],
                ['创建时间', dayjs(selected.createdAt).format('YYYY-MM-DD HH:mm')],
              ]"
              :key="label"
              :label="label"
              >{{ value || '—' }}</a-descriptions-item
            >
          </a-descriptions>
          <h3>工序阶段</h3>
          <a-spin :spinning="progressLoading"
            ><ProcessTimeline v-if="progress" :current="currentStage" :quantity="progress.totalQty"
          /></a-spin>
        </div>
        <div v-else-if="tab === '排程'" class="drawer-body">
          <a-alert v-if="selected.status === 'done'" type="info" show-icon message="已完工批次不可修改排程" class="notice" />
          <a-form :model="schedule" :label-col="{ span: 7 }" :wrapper-col="{ span: 16 }"
            ><fieldset :disabled="selected.status === 'done'">
              <a-form-item
                v-for="[key, label] in [
                  ['planTechDate', '计划工艺日期'],
                  ['planDrawingDate', '计划出图日期'],
                  ['planCutDate', '计划开料日期'],
                  ['planSheetStart', '计划钣金开始'],
                  ['planSheetDelivery', '计划钣金交期'],
                  ['planSprayStart', '计划喷涂开始'],
                  ['planSprayDelivery', '计划喷涂交期'],
                ] as const"
                :key="key"
                :label="label"
                ><a-date-picker
                  v-model:value="schedule[key]"
                  value-format="YYYY-MM-DD"
                  style="width: 100%"
                  :disabled="selected.status === 'done'"
              /></a-form-item>
              <a-form-item label="外协钣金厂"
                ><a-select
                  v-model:value="schedule.outsourceFactoryId"
                  allow-clear
                  :disabled="selected.status === 'done'"
                  placeholder="未指定"
                  :options="factories.filter((f) => f.kind === 'outsource').map((f) => ({ value: f.id, label: f.name }))"
              /></a-form-item>
              <a-form-item label="计划钣金车间"><a-input v-model:value="schedule.workshop" /></a-form-item>
              <a-form-item label="紧急程度"
                ><a-select
                  v-model:value="schedule.urgency"
                  :disabled="selected.status === 'done'"
                  :options="[
                    { value: 'normal', label: '普通' },
                    { value: 'urgent', label: '加急' },
                  ]"
              /></a-form-item>
              <a-form-item label="备注"><a-textarea v-model:value="schedule.remark" :rows="3" /></a-form-item></fieldset
          ></a-form>
          <a-button type="primary" :disabled="selected.status === 'done'" :loading="saving" @click="saveSchedule">保存排程</a-button>
        </div>
        <div v-else-if="tab === '工序进度'" class="drawer-body">
          <a-spin :spinning="progressLoading">
            <div v-if="progress">
              <div class="item-toolbar">
                标签 {{ progress.labelQty }} 张 / 总件数 {{ progress.totalQty }} 件
                <a-button v-if="progress.labelQty === 0" type="primary" :loading="generating" style="float: right" @click="generateLabels"
                  >生成标签</a-button
                >
              </div>
              <a-table
                row-key="jobId"
                size="small"
                :data-source="progress.jobs"
                :pagination="false"
                :columns="[
                  { title: '工序', dataIndex: 'jobName' },
                  { title: '已扫件数 / 总件数', dataIndex: 'scannedQty' },
                  { title: '已扫面积', dataIndex: 'scannedArea' },
                  { title: '进度', dataIndex: 'percent' },
                ]"
              >
                <template #bodyCell="{ column, record }">
                  <template v-if="column.dataIndex === 'jobName'">{{ record.seq }}. {{ record.jobName }}</template>
                  <template v-else-if="column.dataIndex === 'scannedQty'">{{ record.scannedQty }} / {{ progress.totalQty }}</template>
                  <template v-else-if="column.dataIndex === 'scannedArea'">{{ Number(record.scannedArea).toFixed(2) }} m²</template>
                  <template v-else-if="column.dataIndex === 'percent'"
                    ><a-progress :percent="progress.totalQty ? Math.round((record.scannedQty / progress.totalQty) * 100) : 0" size="small"
                  /></template>
                </template>
              </a-table>
            </div>
          </a-spin>
        </div>
        <div v-else class="drawer-body">
          <div class="item-toolbar">
            <a-button type="primary" :disabled="selected.status === 'done'" @click="openImport">导入/替换清单</a-button>
          </div>
          <a-table
            row-key="id"
            size="small"
            :loading="itemsLoading"
            :data-source="items"
            :scroll="{ x: 1100 }"
            :pagination="false"
            :columns="[
              { title: '序号', dataIndex: 'seq' },
              { title: '编号', dataIndex: 'pieceNo' },
              { title: '图号', dataIndex: 'drawingNo' },
              { title: '板型', dataIndex: 'pattern' },
              { title: '厚度', dataIndex: 'thickness' },
              { title: '长×宽', dataIndex: 'dimension' },
              { title: '单件面积', dataIndex: 'pieceArea' },
              { title: '喷涂面积', dataIndex: 'sprayingArea' },
              { title: '数量', dataIndex: 'qty' },
              { title: '包装数', dataIndex: 'packageQty' },
              { title: '刀数', dataIndex: 'bendingTimes' },
              { title: '备注', dataIndex: 'remark' },
            ]"
            ><template #bodyCell="{ column, record }"
              ><template v-if="column.dataIndex === 'dimension'">{{ record.length }} × {{ record.width }}</template></template
            ></a-table
          >
        </div>
        <div class="drawer-footer"><a-button @click="selected = undefined">关闭</a-button></div>
      </template>
    </a-drawer>
    <a-modal
      v-model:open="createOpen"
      title="新建生产批次"
      :width="720"
      :confirm-loading="saving"
      ok-text="保存"
      destroy-on-close
      @ok="saveCreate"
      ><a-form ref="createRef" :model="createForm" :label-col="{ span: 6 }">
        <a-form-item label="合同" name="contractId" :rules="[{ required: true, message: '请选择合同' }]"
          ><a-select
            v-model:value="createForm.contractId"
            show-search
            option-filter-prop="label"
            :options="contracts.map((c) => ({ value: c.id, label: `${c.contractNo} · ${c.customerName} · ${c.projectName}` }))"
        /></a-form-item>
        <a-form-item label="批次号" name="lotNo" :rules="[{ required: true, whitespace: true, message: '请输入批次号' }]"
          ><a-input v-model:value="createForm.lotNo" :maxlength="64"
        /></a-form-item>
        <a-form-item label="客户单号"><a-input v-model:value="createForm.clientOrderNo" /></a-form-item>
        <a-form-item label="厚度 (mm)"
          ><a-input-number v-model:value="createForm.thickness" :min="0" :step="0.1" style="width: 100%"
        /></a-form-item>
        <a-form-item
          v-for="[key, label] in [
            ['color', '颜色'],
            ['colorNo', '色号'],
            ['texture', '材质'],
            ['sprayingType', '喷涂类型'],
          ] as const"
          :key="key"
          :label="label"
          ><a-input v-model:value="createForm[key]"
        /></a-form-item>
        <a-form-item label="订单件数"
          ><a-input-number v-model:value="createForm.orderQty" :min="0" :precision="0" style="width: 100%"
        /></a-form-item>
        <a-form-item label="订单面积"
          ><a-input-number v-model:value="createForm.orderArea" :min="0" :precision="2" style="width: 100%"
        /></a-form-item>
        <a-form-item label="需求交期"
          ><a-date-picker v-model:value="createForm.requiredDate" value-format="YYYY-MM-DD" style="width: 100%"
        /></a-form-item>
        <a-form-item label="紧急"
          ><a-switch :checked="createForm.urgency === 'urgent'" @change="(v: boolean) => (createForm.urgency = v ? 'urgent' : 'normal')"
        /></a-form-item>
        <a-form-item label="发货地址"><a-input v-model:value="createForm.deliveryAddress" /></a-form-item>
        <a-form-item label="特殊要求"><a-textarea v-model:value="createForm.specialRequest" /></a-form-item>
        <a-form-item label="备注"><a-textarea v-model:value="createForm.remark" /></a-form-item> </a-form
    ></a-modal>
    <a-modal v-model:open="importOpen" title="导入/替换构件清单" :confirm-loading="importing" ok-text="替换清单" @ok="importItems">
      <p>每行：编号,图号,长,宽,数量[,板型,颜色,包装数,刀数,备注]；支持制表符或逗号分隔，长宽单位 mm。</p>
      <a-textarea v-model:value="importText" :rows="10" placeholder="P001,D001,1200,600,5" />
    </a-modal>
  </div>
</template>
<style scoped>
.batches {
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
.heading span,
.muted {
  color: var(--gz-muted);
  font-size: 12px;
}
label {
  display: grid;
  gap: 6px;
  color: var(--gz-secondary);
  font-size: 12px;
  min-width: 160px;
}
.summary {
  background: var(--gz-bg);
  padding: 12px;
  display: flex;
  align-items: center;
  gap: 18px;
  flex-wrap: wrap;
  font-size: 12px;
}
.summary b {
  color: var(--gz-primary);
}
.drawer-body {
  padding: 8px 0 80px;
}
.drawer-body h3 {
  font-size: 14px;
  margin: 24px 0 12px;
}
.overdue {
  color: var(--gz-error);
}
.notice,
.item-toolbar {
  margin-bottom: 16px;
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
