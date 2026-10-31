<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import type { FormInstance, TablePaginationConfig } from 'ant-design-vue'
import { ReloadOutlined } from '@ant-design/icons-vue'
import dayjs, { type Dayjs } from 'dayjs'
import { ApiError, call, client, errorMessage, type Schemas } from '@/api/client'
import PageCard from '@/components/PageCard.vue'
import TableToolbar from '@/components/TableToolbar.vue'
import CodeText from '@/components/CodeText.vue'
import StatusTag from '@/components/StatusTag.vue'

type Plan = Schemas['Plan']
type PlanKind = Schemas['PlanKind']
type Summary = Schemas['PlanPage']['summary']
type Period = 'last' | 'this' | 'next' | 'custom'

const props = defineProps<{ kind: PlanKind }>()
const router = useRouter()

const titles: Record<PlanKind, [string, string]> = {
  sheet_week: ['钣金周计划', '按周安排钣金车间生产，进度按「钣金下线」扫码统计'],
  sheet_daily: ['钣金日计划', '按工序安排钣金日生产，进度按计划工序扫码统计'],
  spray_daily: ['喷涂日计划', '按喷涂线与班次安排喷涂，进度按「下挂冷却」扫码统计'],
}
const isSheet = computed(() => props.kind !== 'spray_daily')
const isSheetDaily = computed(() => props.kind === 'sheet_daily')
const isSpray = computed(() => props.kind === 'spray_daily')

const WORKSHOPS = ['钣金一车间', '钣金二车间'].map((v) => ({ value: v, label: v }))
const PAINT_LINES = ['1号线', '2号线'].map((v) => ({ value: v, label: v }))
const SHIFTS = ['早班', '中班', '晚班'].map((v) => ({ value: v, label: v }))
const URGENCY = [
  { value: 'normal', label: '普通' },
  { value: 'urgent', label: '加急' },
]
const AUDIT_OPTIONS = [
  { value: 'draft', label: '未审核' },
  { value: 'audited', label: '已审核' },
]
const STOP_OPTIONS = [
  { value: 'normal', label: '正常' },
  { value: 'paused', label: '暂停' },
  { value: 'stopped', label: '停止' },
]
const COMPLETED_OPTIONS = [
  { value: 'all', label: '全部' },
  { value: 'no', label: '未完成' },
  { value: 'yes', label: '已完成' },
]
const errorTexts: Record<string, string> = {
  duplicate_plan: '该批次在此时段已有计划',
  already_audited: '计划已审核，不可修改',
  not_audited: '计划尚未审核',
  invalid_transition: '当前状态不允许此操作',
  forbidden_factory: '无权操作该厂区的批次',
  not_found: '批次或计划不存在',
  invalid_request: '请求参数有误，请检查日期与必填项',
}
function explain(e: unknown) {
  return e instanceof ApiError && errorTexts[e.code] ? errorTexts[e.code] : errorMessage(e)
}

function monday(d: Dayjs) {
  return d.startOf('day').subtract((d.day() + 6) % 7, 'day')
}
function weekRange(offset: number): [string, string] {
  const start = monday(dayjs()).add(offset, 'week')
  return [start.format('YYYY-MM-DD'), start.add(6, 'day').format('YYYY-MM-DD')]
}
const periodOffsets: Record<Exclude<Period, 'custom'>, number> = { last: -1, this: 0, next: 1 }

function emptyFilters() {
  return {
    period: 'this' as Period,
    dates: weekRange(0) as string[],
    auditState: undefined as Schemas['AuditState'] | undefined,
    stopState: undefined as Schemas['StopState'] | undefined,
    workshop: undefined as string | undefined,
    jobId: undefined as number | undefined,
    paintLine: undefined as string | undefined,
    completed: 'all' as 'all' | 'yes' | 'no',
    keyword: '',
  }
}
const filters = reactive(emptyFilters())
const applied = reactive(emptyFilters())
const range = computed<[string, string] | undefined>(() => {
  if (applied.period !== 'custom') return weekRange(periodOffsets[applied.period])
  const [from, to] = applied.dates
  return from && to ? [from, to] : undefined
})
function onPeriodChange() {
  if (filters.period === 'custom') return
  filters.dates = weekRange(periodOffsets[filters.period])
  search()
}

const rows = ref<Plan[]>([])
const summary = ref<Summary>({ plans: 0, qty: 0, area: 0, doneQty: 0, doneArea: 0, overdue: 0 })
const jobs = ref<Schemas['Job'][]>([])
const batches = ref<Schemas['Batch'][]>([])
const pager = reactive({ current: 1, pageSize: 20, total: 0 })
const loading = ref(false)
const busyId = ref<number>()

const jobOptions = computed(() => jobs.value.map((j) => ({ value: j.id, label: `${j.code} ${j.name}` })))
const batchOptions = computed(() =>
  batches.value
    .filter((b) => b.status === 'scheduled' || b.status === 'in_production')
    .map((b) => ({ value: b.id, label: `${b.lotNo} · ${b.projectName}` })),
)

const columns = computed(() => [
  { title: '批次号', dataIndex: 'lotNo', width: 170, fixed: 'left' as const },
  { title: '工程 / 客户', dataIndex: 'projectName', width: 200, ellipsis: true },
  { title: '规格', dataIndex: 'thickness', width: 140 },
  { title: '件数 / 面积', dataIndex: 'itemQty', width: 110 },
  { title: '计划起止', dataIndex: 'startDate', width: 130 },
  ...(isSheetDaily.value ? [{ title: '工序', dataIndex: 'jobName', width: 90 }] : []),
  ...(isSheet.value ? [{ title: '车间', dataIndex: 'workshop', width: 90 }] : []),
  ...(isSpray.value
    ? [
        { title: '喷涂线', dataIndex: 'paintLine', width: 90 },
        { title: '班次', dataIndex: 'shift', width: 90 },
      ]
    : []),
  { title: '目标工序', dataIndex: 'targetJobName', width: 90 },
  { title: '进度', dataIndex: 'progress', width: 150, fixed: 'right' as const },
  { title: '紧急', dataIndex: 'urgency', width: 70, fixed: 'right' as const },
  { title: '审核', dataIndex: 'auditState', width: 80, fixed: 'right' as const },
  { title: '状态', dataIndex: 'stopState', width: 70, fixed: 'right' as const },
  { title: '操作', key: 'action', width: 200, fixed: 'right' as const },
])
const scrollX = computed(() => columns.value.reduce((n, c) => n + c.width, 0))

async function load() {
  loading.value = true
  try {
    const result = await call(
      client.GET('/plans', {
        params: {
          query: {
            kind: props.kind,
            from: range.value?.[0],
            to: range.value?.[1],
            auditState: applied.auditState,
            stopState: applied.stopState,
            workshop: isSheet.value ? applied.workshop : undefined,
            jobId: isSheetDaily.value ? applied.jobId : undefined,
            paintLine: isSpray.value ? applied.paintLine : undefined,
            completed: applied.completed === 'all' ? undefined : applied.completed === 'yes',
            keyword: applied.keyword.trim() || undefined,
            page: pager.current,
            size: pager.pageSize,
          },
        },
      }),
    )
    rows.value = result.items
    pager.total = result.total
    summary.value = result.summary
  } catch (e) {
    message.error(explain(e))
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
  Object.assign(filters, emptyFilters())
  search()
}
function onTableChange(p: TablePaginationConfig) {
  pager.current = p.current ?? 1
  pager.pageSize = p.pageSize ?? 20
  void load()
}
function openBatch(p: Plan) {
  void router.push({ path: '/scheduling/batches', query: { keyword: p.lotNo } })
}
function md(d: string) {
  return dayjs(d).format('MM-DD')
}
function rowClass(p: Plan) {
  return p.overdue ? 'row-overdue' : ''
}

async function run(p: Plan, ok: string, fn: () => Promise<unknown>) {
  busyId.value = p.id
  try {
    await fn()
    message.success(ok)
    await load()
    return true
  } catch (e) {
    message.error(explain(e))
    return false
  } finally {
    busyId.value = undefined
  }
}
function audit(p: Plan) {
  const action = p.auditState === 'audited' ? 'unaudit' : 'audit'
  return run(p, action === 'audit' ? '审核成功' : '已反审核', () =>
    call(client.POST('/plans/{id}/audit', { params: { path: { id: p.id } }, body: { action } })),
  )
}
function togglePause(p: Plan) {
  const action = p.stopState === 'paused' ? 'resume' : 'pause'
  return run(p, action === 'pause' ? '计划已暂停' : '计划已恢复', () =>
    call(client.POST('/plans/{id}/stop-state', { params: { path: { id: p.id } }, body: { action } })),
  )
}
function remove(p: Plan) {
  return run(p, '计划已删除', () => call(client.DELETE('/plans/{id}', { params: { path: { id: p.id } } })))
}

const stopTarget = ref<Plan>()
const stopReason = ref('')
const stopping = ref(false)
function openStop(p: Plan) {
  stopTarget.value = p
  stopReason.value = ''
}
function closeStop() {
  stopTarget.value = undefined
}
async function confirmStop() {
  const p = stopTarget.value
  if (!p) return
  stopping.value = true
  const ok = await run(p, '计划已停止', () =>
    call(
      client.POST('/plans/{id}/stop-state', {
        params: { path: { id: p.id } },
        body: { action: 'stop', reason: stopReason.value.trim() || undefined },
      }),
    ),
  )
  stopping.value = false
  if (ok) closeStop()
}

type FormState = {
  batchId?: number
  dates: string[]
  workshop?: string
  jobId?: number
  paintLine?: string
  shift?: string
  sprayType: string
  urgency: Schemas['Urgency']
  remark: string
}
const formOpen = ref(false)
const formRef = ref<FormInstance>()
const editing = ref<Plan>()
const saving = ref(false)
const form = ref<FormState>({ dates: [], sprayType: '', urgency: 'normal', remark: '' })
function openCreate() {
  const today = dayjs().format('YYYY-MM-DD')
  editing.value = undefined
  form.value = {
    dates: props.kind === 'sheet_week' ? weekRange(0) : [today, today],
    sprayType: '',
    urgency: 'normal',
    remark: '',
  }
  formOpen.value = true
}
function openEdit(p: Plan) {
  editing.value = p
  form.value = {
    batchId: p.batchId,
    dates: [p.startDate, p.endDate],
    workshop: p.workshop || undefined,
    jobId: p.jobId ?? undefined,
    paintLine: p.paintLine || undefined,
    shift: p.shift || undefined,
    sprayType: p.sprayType ?? '',
    urgency: p.urgency,
    remark: p.remark,
  }
  formOpen.value = true
}
async function saveForm() {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }
  const f = form.value
  const [startDate, endDate] = f.dates
  if (!startDate || !endDate) return
  const fields = {
    startDate,
    endDate,
    workshop: isSheet.value ? (f.workshop ?? '') : '',
    jobId: isSheetDaily.value ? (f.jobId ?? null) : null,
    paintLine: isSpray.value ? (f.paintLine ?? '') : '',
    shift: isSpray.value ? (f.shift ?? '') : '',
    sprayType: isSpray.value ? f.sprayType.trim() : '',
    urgency: f.urgency,
    remark: f.remark,
  }
  saving.value = true
  try {
    if (editing.value) {
      await call(client.PUT('/plans/{id}', { params: { path: { id: editing.value.id } }, body: fields }))
      message.success('计划已保存')
    } else {
      await call(client.POST('/plans', { body: { ...fields, kind: props.kind, batchId: f.batchId as number } }))
      message.success('计划已创建')
    }
    formOpen.value = false
    await load()
  } catch (e) {
    message.error(explain(e))
  } finally {
    saving.value = false
  }
}
function closeForm() {
  formOpen.value = false
}

onMounted(() => {
  void load()
  void (async () => {
    try {
      jobs.value = await call(client.GET('/jobs'))
    } catch (e) {
      message.error(errorMessage(e))
    }
  })()
  void (async () => {
    try {
      batches.value = (await call(client.GET('/batches', { params: { query: { size: 200 } } }))).items
    } catch (e) {
      message.error(errorMessage(e))
    }
  })()
})
</script>
<template>
  <div class="plans">
    <div class="heading">
      <div>
        <h1>{{ titles[kind][0] }}</h1>
        <span>{{ titles[kind][1] }}</span>
      </div>
      <a-button type="primary" data-testid="plan-new" @click="openCreate">新建计划</a-button>
    </div>
    <PageCard>
      <div class="filters">
        <label
          >时间段<a-radio-group v-model:value="filters.period" class="period" button-style="solid" @change="onPeriodChange"
            ><a-radio-button value="last">上周</a-radio-button><a-radio-button value="this">本周</a-radio-button
            ><a-radio-button value="next">下周</a-radio-button><a-radio-button value="custom">自定义</a-radio-button></a-radio-group
          ></label
        >
        <label v-if="filters.period === 'custom'">日期范围<a-range-picker v-model:value="filters.dates" value-format="YYYY-MM-DD" /></label>
        <label>审核状态<a-select v-model:value="filters.auditState" allow-clear placeholder="全部" :options="AUDIT_OPTIONS" /></label>
        <label>暂停状态<a-select v-model:value="filters.stopState" allow-clear placeholder="全部" :options="STOP_OPTIONS" /></label>
        <label v-if="isSheet"
          >车间<a-select v-model:value="filters.workshop" allow-clear placeholder="全部车间" :options="WORKSHOPS"
        /></label>
        <label v-if="isSheetDaily"
          >工序<a-select v-model:value="filters.jobId" allow-clear placeholder="全部工序" :options="jobOptions"
        /></label>
        <label v-if="isSpray"
          >喷涂线<a-select v-model:value="filters.paintLine" allow-clear placeholder="全部" :options="PAINT_LINES"
        /></label>
        <label>完成状态<a-select v-model:value="filters.completed" :options="COMPLETED_OPTIONS" /></label>
        <label
          >关键字<a-input v-model:value="filters.keyword" placeholder="批次号 / 工程 / 客户" allow-clear @press-enter="search"
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
          <div class="summary" data-testid="plan-summary">
            <span
              >计划 <b>{{ summary.plans }}</b> 条</span
            >
            <span
              >件数 <b>{{ summary.qty }}</b></span
            >
            <span
              >面积 <b>{{ Number(summary.area).toFixed(2) }}</b> m²</span
            >
            <span
              >已完成 <b>{{ summary.doneQty }}</b> 件 / <b>{{ Number(summary.doneArea).toFixed(2) }}</b> m²</span
            >
            <span class="danger"
              >逾期 <b>{{ summary.overdue }}</b> 条</span
            >
            <span v-if="range" class="muted">{{ range[0] }} ~ {{ range[1] }}</span>
          </div>
        </template>
        <template #right>
          <a-button aria-label="刷新" @click="load"><ReloadOutlined /></a-button>
        </template>
      </TableToolbar>
      <a-table
        row-key="id"
        size="middle"
        :columns="columns"
        :data-source="rows"
        :loading="loading"
        :scroll="{ x: scrollX }"
        :row-class-name="rowClass"
        :pagination="{ ...pager, showSizeChanger: true, pageSizeOptions: ['20', '50', '100'], showTotal: (n: number) => `共 ${n} 条` }"
        data-testid="plan-table"
        @change="onTableChange"
      >
        <template #emptyText><a-empty description="该时段暂无计划" /></template>
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'lotNo'">
            <a @click="openBatch(record)"><CodeText :value="record.lotNo" /></a>
          </template>
          <template v-else-if="column.dataIndex === 'projectName'">
            <strong class="truncate" :title="record.projectName">{{ record.projectName }}</strong>
            <div class="muted truncate" :title="record.customerName">{{ record.customerName }}</div>
          </template>
          <template v-else-if="column.dataIndex === 'thickness'">
            {{ record.thickness }} mm
            <div class="muted">{{ record.color || '—' }}</div>
          </template>
          <template v-else-if="column.dataIndex === 'itemQty'">
            {{ record.itemQty }} 件
            <div class="muted">{{ Number(record.itemArea).toFixed(2) }} m²</div>
          </template>
          <template v-else-if="column.dataIndex === 'startDate'">
            <span :class="{ danger: record.overdue }">{{ md(record.startDate) }} ~ {{ md(record.endDate) }}</span>
          </template>
          <template v-else-if="['jobName', 'workshop', 'paintLine', 'shift', 'targetJobName'].includes(String(column.dataIndex))">
            {{ record[column.dataIndex as string] || '—' }}
          </template>
          <template v-else-if="column.dataIndex === 'progress'">
            <a-progress :percent="record.progress" size="small" />
            <div class="muted">{{ record.doneQty }}/{{ record.labelQty }}</div>
          </template>
          <template v-else-if="column.dataIndex === 'urgency'">
            <StatusTag v-if="record.urgency === 'urgent'" tone="error" label="加急" />
            <StatusTag v-else tone="neutral" label="普通" />
          </template>
          <template v-else-if="column.dataIndex === 'auditState'">
            <StatusTag
              :tone="record.auditState === 'audited' ? 'success' : 'neutral'"
              :label="record.auditState === 'audited' ? '已审核' : '未审核'"
            />
          </template>
          <template v-else-if="column.dataIndex === 'stopState'">
            <StatusTag v-if="record.stopState === 'paused'" tone="warning" label="暂停" />
            <a-tooltip v-else-if="record.stopState === 'stopped'" :title="record.stopReason || undefined">
              <StatusTag tone="error" label="停止" />
            </a-tooltip>
          </template>
          <template v-else-if="column.key === 'action'">
            <a-button
              v-if="record.auditState === 'draft' && record.stopState !== 'stopped'"
              type="link"
              size="small"
              @click="openEdit(record)"
              >编辑</a-button
            >
            <a-popconfirm
              v-if="record.stopState !== 'stopped'"
              :title="record.auditState === 'audited' ? '确定反审核此计划？' : '确定审核此计划？'"
              @confirm="audit(record)"
            >
              <a-button type="link" size="small" :loading="busyId === record.id">{{
                record.auditState === 'audited' ? '反审核' : '审核'
              }}</a-button>
            </a-popconfirm>
            <a-popconfirm
              v-if="record.stopState !== 'stopped'"
              :title="record.stopState === 'paused' ? '确定恢复此计划？' : '确定暂停此计划？'"
              @confirm="togglePause(record)"
            >
              <a-button type="link" size="small">{{ record.stopState === 'paused' ? '恢复' : '暂停' }}</a-button>
            </a-popconfirm>
            <a-popconfirm v-if="record.stopState !== 'stopped'" title="停止后不可恢复，确定停止？" @confirm="openStop(record)">
              <a-button type="link" size="small" danger>停止</a-button>
            </a-popconfirm>
            <a-popconfirm
              v-if="record.auditState === 'draft' && record.stopState !== 'stopped'"
              title="确定删除此计划？"
              @confirm="remove(record)"
            >
              <a-button type="link" size="small" danger>删除</a-button>
            </a-popconfirm>
            <a-button v-if="record.stopState === 'stopped'" type="link" size="small" disabled>不可操作</a-button>
          </template>
        </template>
      </a-table>
    </PageCard>
    <a-modal
      :open="!!stopTarget"
      :title="`停止计划 · ${stopTarget?.lotNo ?? ''}`"
      :confirm-loading="stopping"
      ok-text="确认停止"
      :ok-button-props="{ danger: true }"
      @ok="confirmStop"
      @cancel="closeStop"
    >
      <label class="field">停止原因<a-input v-model:value="stopReason" :maxlength="200" placeholder="选填，最多 200 字" /></label>
    </a-modal>
    <a-modal
      v-model:open="formOpen"
      :title="editing ? `编辑计划 · ${editing.lotNo}` : `新建${titles[kind][0]}`"
      :width="620"
      :confirm-loading="saving"
      ok-text="保存"
      destroy-on-close
      @ok="saveForm"
      @cancel="closeForm"
    >
      <a-form ref="formRef" :model="form" :label-col="{ span: 5 }" data-testid="plan-form">
        <a-form-item label="批次" name="batchId" :rules="[{ required: true, message: '请选择批次' }]">
          <a-select
            v-model:value="form.batchId"
            show-search
            option-filter-prop="label"
            placeholder="已排产 / 生产中的批次"
            :disabled="!!editing"
            :options="editing ? [{ value: editing.batchId, label: `${editing.lotNo} · ${editing.projectName}` }] : batchOptions"
          />
        </a-form-item>
        <a-form-item label="计划起止" name="dates" :rules="[{ required: true, type: 'array', len: 2, message: '请选择起止日期' }]">
          <a-range-picker v-model:value="form.dates" value-format="YYYY-MM-DD" style="width: 100%" />
        </a-form-item>
        <a-form-item v-if="isSheet" label="车间" name="workshop">
          <a-select v-model:value="form.workshop" allow-clear placeholder="未指定" :options="WORKSHOPS" />
        </a-form-item>
        <a-form-item v-if="isSheetDaily" label="工序" name="jobId" :rules="[{ required: true, message: '请选择工序' }]">
          <a-select v-model:value="form.jobId" placeholder="请选择工序" :options="jobOptions" />
        </a-form-item>
        <template v-if="isSpray">
          <a-form-item label="喷涂线" name="paintLine" :rules="[{ required: true, message: '请选择喷涂线' }]">
            <a-select v-model:value="form.paintLine" placeholder="请选择喷涂线" :options="PAINT_LINES" />
          </a-form-item>
          <a-form-item label="班次" name="shift" :rules="[{ required: true, message: '请选择班次' }]">
            <a-select v-model:value="form.shift" placeholder="请选择班次" :options="SHIFTS" />
          </a-form-item>
          <a-form-item label="喷涂类型" name="sprayType">
            <a-input v-model:value="form.sprayType" :maxlength="50" placeholder="留空则沿用批次喷涂类型" />
          </a-form-item>
        </template>
        <a-form-item label="紧急度" name="urgency">
          <a-radio-group v-model:value="form.urgency" :options="URGENCY" />
        </a-form-item>
        <a-form-item label="备注" name="remark">
          <a-textarea v-model:value="form.remark" :maxlength="500" :rows="2" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>
<style scoped>
.plans {
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
.filters {
  display: flex;
  flex-wrap: wrap;
  align-items: end;
  gap: 12px;
}
.filters label,
.field {
  display: grid;
  gap: 6px;
  color: var(--gz-secondary);
  font-size: 12px;
  min-width: 150px;
}
.period {
  display: flex;
  white-space: nowrap;
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
.truncate {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.summary b {
  color: var(--gz-primary);
}
.summary .danger b,
.danger {
  color: var(--gz-error);
}
:deep(.row-overdue > td) {
  background: #fff1f0 !important;
}
</style>
