<script setup lang="ts">
// 新建生产计划向导（原型）：① 按优先级选批次 → ② 选材料并预填数量 → ③ 确认并保存。
// 材料 / 库存 / 结算 / 产能为 mock.ts 中的模拟数据，保存仍走现有 POST /plans，预定材料写入备注。
import { computed, ref, watch } from 'vue'
import { message } from 'ant-design-vue'
import dayjs from 'dayjs'
import type { EChartsOption } from 'echarts'
import { call, client, errorMessage, type Schemas } from '@/api/client'
import EChart from '@/components/EChart.vue'
import {
  FACTORS,
  LIMIT_TEXT,
  freeCapacity,
  overlaps,
  rankCandidates,
  suggestDates,
  suggestQty,
  type Candidate,
  type Limit,
  type Material,
  type PlanKind,
  type PlanRange,
  type Tone,
} from './mock'

type Option = { value: string | number; label: string }
const props = defineProps<{
  open: boolean
  kind: PlanKind
  title: string
  batches: Schemas['Batch'][]
  workshops: Option[]
  jobs: Option[]
  paintLines: Option[]
  shifts: Option[]
}>()
const emit = defineEmits<{ 'update:open': [boolean]; created: [] }>()

const step = ref(0)
const plansByBatch = ref(new Map<number, PlanRange[]>())
const candidates = computed(() => rankCandidates(props.batches, props.kind, plansByBatch.value))

/** 拉取本类型未停止、未结束的计划，用于标记「已有计划」并避开重复计划校验。 */
async function loadExisting() {
  try {
    const res = await call(client.GET('/plans', { params: { query: { kind: props.kind, from: dayjs().format('YYYY-MM-DD'), size: 200 } } }))
    const m = new Map<number, PlanRange[]>()
    for (const p of res.items) {
      if (p.stopState === 'stopped') continue
      const list = m.get(p.batchId) ?? []
      list.push({ start: p.startDate, end: p.endDate })
      m.set(p.batchId, list)
    }
    plansByBatch.value = m
  } catch (e) {
    message.error(errorMessage(e))
  }
}
const picked = ref<Candidate>()
const dates = ref<[string, string]>(['', ''])
const selected = ref<Record<string, { qty: number; limit: Limit }>>({})
const fields = ref({
  workshop: undefined as string | undefined,
  jobId: undefined as number | undefined,
  paintLine: undefined as string | undefined,
  shift: undefined as string | undefined,
  urgency: 'normal' as Schemas['Urgency'],
  remark: '',
})
const saving = ref(false)

watch(
  () => props.open,
  (o) => {
    if (!o) return
    step.value = 0
    picked.value = undefined
    selected.value = {}
    void loadExisting()
  },
)

const toneColor: Record<Tone, string> = { good: 'green', warn: 'orange', bad: 'red', info: 'default' }
const wan = (n: number) => `${(n / 10000).toFixed(1)} 万`
const md = (d: string | null) => (d ? dayjs(d).format('MM-DD') : '—')

// ---------- ① 批次优先级 ----------
const rankOption = computed<EChartsOption>(() => {
  const list = candidates.value
  return {
    grid: { left: 8, right: 56, top: 36, bottom: 8, containLabel: true },
    legend: { top: 0, data: FACTORS.map((f) => f.label) },
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: (p: unknown) => {
        const arr = p as { dataIndex: number }[]
        const c = list[arr[0]?.dataIndex ?? 0]
        if (!c) return ''
        const rows = FACTORS.map((f) => `${f.label}：<b>${c.scores[f.key]}</b> / ${f.max}`).join('<br/>')
        const why = c.reasons.map((r) => `· ${r.text}`).join('<br/>')
        return `<b>${c.batch.lotNo}</b> · ${c.batch.projectName}<br/>总分 <b>${c.total}</b>${c.blocked ? '（不建议排产）' : c.existing.length ? '（已有计划）' : ''}<br/>${rows}<hr style="margin:4px 0"/>${why}`
      },
    },
    xAxis: { type: 'value', max: 100, splitLine: { lineStyle: { type: 'dashed' } } },
    yAxis: {
      type: 'category',
      inverse: true,
      data: list.map((c) => `${c.batch.lotNo}  ${c.batch.projectName.slice(0, 10)}`),
      axisLabel: { color: (_v?: string | number, i?: number) => (list[i ?? 0]?.blocked ? '#aaa' : '#333') },
    },
    series: FACTORS.map((f, fi) => ({
      name: f.label,
      type: 'bar',
      stack: 'score',
      barWidth: 18,
      itemStyle: { color: f.color },
      cursor: 'pointer',
      data: list.map((c) => ({
        value: c.scores[f.key],
        itemStyle: {
          color: c.blocked ? '#d4d4d8' : f.color,
          opacity: picked.value && picked.value !== c ? 0.35 : c.existing.length ? 0.55 : 1,
        },
      })),
      ...(fi === FACTORS.length - 1
        ? {
            label: {
              show: true,
              position: 'right',
              color: '#333',
              fontWeight: 'bold',
              formatter: (p: { dataIndex: number }) => String(list[p.dataIndex]?.total ?? ''),
            },
          }
        : {}),
    })),
  } as EChartsOption
})

function pickCandidate(c: Candidate) {
  picked.value = c
  dates.value = suggestDates(c)
  selected.value = {}
  fields.value.urgency = c.batch.urgency
  fields.value.workshop = c.batch.workshop || undefined
}
function onRankClick(p: { dataIndex: number }) {
  const c = candidates.value[p.dataIndex]
  if (c) pickCandidate(c)
}

// ---------- ② 材料 ----------
const capacity = computed(() => (dates.value[0] ? freeCapacity(props.kind, dates.value) : { days: 0, total: 0, free: 0 }))
const materials = computed(() => picked.value?.materials ?? [])

function pct(v: number, m: Material) {
  return m.remaining > 0 ? Math.round((v / m.remaining) * 100) : 0
}
const matOption = computed<EChartsOption>(() => {
  const ms = materials.value
  const series: [string, string, (m: Material) => number][] = [
    ['剩余需求', '#475569', (m) => m.remaining],
    ['可用库存', '#2563eb', (m) => Math.max(0, m.available)],
    ['已被预定', '#f59e0b', (m) => m.reserved],
    ['在途', '#93c5fd', (m) => m.inTransit],
  ]
  return {
    grid: { left: 8, right: 16, top: 36, bottom: 8, containLabel: true },
    legend: { top: 0 },
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: (p: unknown) => {
        const m = ms[(p as { dataIndex: number }[])[0]?.dataIndex ?? 0]
        if (!m) return ''
        return `<b>${m.name}</b><br/>剩余需求 ${m.remaining}${m.unit}（总需 ${m.need}，已排 ${m.planned}）<br/>现有库存 ${m.onHand} · 已预定 ${m.reserved} · 可用 <b>${m.available}</b><br/>在途 ${m.inTransit}${m.transitEta ? `（预计 ${md(m.transitEta)} 到）` : ''}<br/>损耗率 ${(m.lossRate * 100).toFixed(0)}%<br/><i>点击选用并预填数量</i>`
      },
    },
    xAxis: { type: 'category', data: ms.map((m) => m.name), axisLabel: { interval: 0, width: 110, overflow: 'break' } },
    yAxis: { type: 'value', name: '相对剩余需求 %', axisLabel: { formatter: '{value}%' } },
    series: series.map(([name, color, get]) => ({
      name,
      type: 'bar',
      cursor: 'pointer',
      itemStyle: { color },
      data: ms.map((m) => pct(get(m), m)),
    })),
  } as EChartsOption
})

function status(m: Material): { tone: Tone; text: string } {
  if (m.available >= m.remaining) return { tone: 'good', text: '可建' }
  if (m.available + m.inTransit >= m.remaining) return { tone: 'warn', text: `待在途 ${md(m.transitEta)}` }
  return {
    tone: 'bad',
    text: `缺 ${(m.remaining - Math.max(0, m.available) - m.inTransit).toFixed(1)}${m.unit}，采购 ${m.supplier.leadDays} 天`,
  }
}
function toggle(m: Material) {
  if (selected.value[m.code]) {
    const rest = { ...selected.value }
    delete rest[m.code]
    selected.value = rest
  } else selected.value = { ...selected.value, [m.code]: suggestQty(m, capacity.value.free) }
}
function onMatClick(p: { dataIndex: number }) {
  const m = materials.value[p.dataIndex]
  if (m) toggle(m)
}
function fillAll() {
  const next: typeof selected.value = {}
  for (const m of materials.value) next[m.code] = suggestQty(m, capacity.value.free)
  selected.value = next
}
function qtyWarn(m: Material) {
  const s = selected.value[m.code]
  if (!s) return ''
  if (s.qty > Math.max(0, m.available)) return '超过可用库存'
  if (s.qty > m.remaining) return '超过剩余需求'
  if (s.qty / m.perArea > capacity.value.free) return '超过产线剩余产能'
  return ''
}
const plannedArea = computed(() => {
  const main = materials.value[0]
  const s = main && selected.value[main.code]
  return s && main ? Math.round((s.qty / main.perArea) * 10) / 10 : 0
})

// ---------- ③ 确认 ----------
const chosen = computed(() => materials.value.filter((m) => selected.value[m.code]))
const warnings = computed(() => {
  const c = picked.value
  if (!c) return []
  const w: string[] = []
  if (c.blocked) w.push(`客户逾期欠款 ${wan(c.settlement.overdue)}，超出授信，建议先催收`)
  const dup = overlaps(c, dates.value)
  if (dup) w.push(`与已有计划 ${md(dup.start)}~${md(dup.end)} 日期重叠，保存会被拒绝`)
  if (c.batch.requiredDate && dayjs(dates.value[1]).isAfter(dayjs(c.batch.requiredDate)))
    w.push(`计划结束 ${md(dates.value[1])} 晚于交期 ${md(c.batch.requiredDate)}`)
  for (const m of chosen.value) if (qtyWarn(m)) w.push(`${m.name}：${qtyWarn(m)}`)
  if (plannedArea.value > capacity.value.free) w.push(`计划面积 ${plannedArea.value}㎡ 超过所选日期剩余产能 ${capacity.value.free}㎡`)
  return w
})

const canNext = computed(() => (step.value === 0 ? !!picked.value : step.value === 1 ? chosen.value.length > 0 : true))

async function save() {
  const c = picked.value
  if (!c) return
  const f = fields.value
  if (props.kind === 'sheet_daily' && !f.jobId) return void message.warning('请选择工序')
  if (props.kind === 'spray_daily' && (!f.paintLine || !f.shift)) return void message.warning('请选择喷涂线和班次')
  const reserve = chosen.value.map((m) => `${m.name} ${selected.value[m.code]?.qty}${m.unit}`).join('；')
  saving.value = true
  try {
    await call(
      client.POST('/plans', {
        body: {
          kind: props.kind,
          batchId: c.batch.id,
          startDate: dates.value[0],
          endDate: dates.value[1],
          workshop: props.kind !== 'spray_daily' ? (f.workshop ?? '') : '',
          jobId: props.kind === 'sheet_daily' ? (f.jobId ?? null) : null,
          paintLine: props.kind === 'spray_daily' ? (f.paintLine ?? '') : '',
          shift: props.kind === 'spray_daily' ? (f.shift ?? '') : '',
          sprayType: '',
          urgency: f.urgency,
          remark: [f.remark.trim(), `[预定材料] ${reserve}`].filter(Boolean).join('\n').slice(0, 500),
        },
      }),
    )
    message.success('计划已创建（材料预定为原型演示，未写入库存）')
    emit('update:open', false)
    emit('created')
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <a-modal
    :open="open"
    :title="`新建${title} · 排产助手（原型）`"
    :width="1080"
    :footer="null"
    destroy-on-close
    data-testid="plan-wizard"
    @cancel="emit('update:open', false)"
  >
    <a-steps :current="step" size="small" class="steps" :items="[{ title: '选择批次' }, { title: '选择材料' }, { title: '确认计划' }]" />
    <a-alert
      type="info"
      show-icon
      class="proto"
      message="原型：批次、交期、紧急度为系统真实数据；材料、库存、预定、在途、结算、产能为模拟数据。"
    />

    <!-- ① 批次优先级 -->
    <section v-show="step === 0">
      <p class="hint">
        柱子越长越优先，颜色表示得分构成；浅色为已有计划（排在后面，可续排），灰色为不建议排产（客户逾期欠款超出授信）。<b>点击柱子</b>查看原因并选中。
      </p>
      <div class="legend-desc">
        <span v-for="f in FACTORS" :key="f.key"><i :style="{ background: f.color }" />{{ f.label }}（{{ f.max }}）：{{ f.desc }}</span>
      </div>
      <a-empty v-if="!candidates.length" description="没有已排程 / 生产中的批次" />
      <div v-else class="rank-scroll">
        <EChart :option="rankOption" :height="Math.max(220, candidates.length * 30 + 50)" @click="onRankClick" />
      </div>
      <div v-if="picked" class="picked" data-testid="wizard-picked">
        <div>
          <b>{{ picked.batch.lotNo }}</b> · {{ picked.batch.projectName }} · {{ picked.batch.customerName }}
          <span class="muted"
            >{{ picked.area.toFixed(1) }}㎡ · {{ picked.batch.thickness }}mm {{ picked.batch.colorNo || picked.batch.color }}</span
          >
        </div>
        <div class="tags">
          <a-tag v-for="r in picked.reasons" :key="r.text" :color="toneColor[r.tone]">{{ r.text }}</a-tag>
        </div>
      </div>
    </section>

    <!-- ② 材料 -->
    <section v-if="step === 1 && picked">
      <div class="cards">
        <div class="card">
          <h4>交付</h4>
          <p>
            交期 <b>{{ md(picked.batch.requiredDate) }}</b>
            <span v-if="picked.daysLeft != null" :class="{ danger: (picked.slack ?? 1) <= 0 }">（剩 {{ picked.daysLeft }} 天）</span>
          </p>
          <p>
            预计工期 {{ picked.estDays }} 天 · 最晚开工 <b>{{ md(picked.latestStart) }}</b>
          </p>
        </div>
        <div class="card" :class="{ bad: picked.blocked, warn: !picked.blocked && picked.settlement.overdue > 0 }">
          <h4>甲方结算 · {{ picked.batch.customerName }}</h4>
          <p>{{ picked.settlement.terms }}</p>
          <p>
            应收 {{ wan(picked.settlement.receivable) }} · 逾期
            <b :class="{ danger: picked.settlement.overdue > 0 }">{{ wan(picked.settlement.overdue) }}</b>
            <span v-if="picked.settlement.overdueDays">（{{ picked.settlement.overdueDays }} 天）</span>
          </p>
        </div>
        <div class="card">
          <h4>计划日期 / 产能</h4>
          <a-range-picker v-model:value="dates" value-format="YYYY-MM-DD" size="small" :allow-clear="false" style="width: 100%" />
          <p>
            剩余产能 <b>{{ capacity.free }}</b> / {{ capacity.total }}㎡（{{ capacity.days }} 天）
          </p>
        </div>
      </div>
      <p class="hint"><b>点击柱子或表格行</b>选用材料，系统按「剩余需求 / 可用库存 / 产能」三者最小值预填数量。</p>
      <EChart :option="matOption" :height="240" @click="onMatClick" />
      <div class="toolbar"><a-button size="small" @click="fillAll">全部按建议预填</a-button></div>
      <table class="mat" data-testid="wizard-materials">
        <thead>
          <tr>
            <th />
            <th>材料</th>
            <th>剩余需求</th>
            <th>可用 / 现有</th>
            <th>已预定</th>
            <th>在途</th>
            <th>状态</th>
            <th>供应商结算</th>
            <th>本次数量</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="m in materials" :key="m.code" :class="{ on: selected[m.code] }" @click="toggle(m)">
            <td><a-checkbox :checked="!!selected[m.code]" /></td>
            <td>
              <b>{{ m.name }}</b>
              <div class="muted">{{ m.code }}</div>
            </td>
            <td>{{ m.remaining }}{{ m.unit }}</td>
            <td>{{ m.available }} / {{ m.onHand }}</td>
            <td>{{ m.reserved }}</td>
            <td>
              {{ m.inTransit || '—' }}
              <div v-if="m.transitEta" class="muted">{{ md(m.transitEta) }} 到</div>
            </td>
            <td>
              <a-tag :color="toneColor[status(m).tone]">{{ status(m).text }}</a-tag>
            </td>
            <td>
              {{ m.supplier.name }}
              <div class="muted">{{ m.supplier.terms }} · 交货 {{ m.supplier.leadDays }} 天</div>
            </td>
            <td @click.stop>
              <template v-if="selected[m.code]">
                <a-input-number v-model:value="selected[m.code]!.qty" :min="0" size="small" style="width: 100px" :addon-after="m.unit" />
                <div class="muted">{{ LIMIT_TEXT[selected[m.code]!.limit] }}</div>
                <div v-if="qtyWarn(m)" class="warn-text">{{ qtyWarn(m) }}</div>
              </template>
              <span v-else class="muted">—</span>
            </td>
          </tr>
        </tbody>
      </table>
    </section>

    <!-- ③ 确认 -->
    <section v-if="step === 2 && picked" class="confirm">
      <a-form layout="vertical" class="form">
        <a-form-item label="批次">{{ picked.batch.lotNo }} · {{ picked.batch.projectName }}</a-form-item>
        <a-form-item label="计划起止"><a-range-picker v-model:value="dates" value-format="YYYY-MM-DD" :allow-clear="false" /></a-form-item>
        <a-form-item v-if="kind !== 'spray_daily'" label="车间"
          ><a-select v-model:value="fields.workshop" allow-clear :options="workshops"
        /></a-form-item>
        <a-form-item v-if="kind === 'sheet_daily'" label="工序" required
          ><a-select v-model:value="fields.jobId" :options="jobs"
        /></a-form-item>
        <template v-if="kind === 'spray_daily'">
          <a-form-item label="喷涂线" required><a-select v-model:value="fields.paintLine" :options="paintLines" /></a-form-item>
          <a-form-item label="班次" required><a-select v-model:value="fields.shift" :options="shifts" /></a-form-item>
        </template>
        <a-form-item label="紧急度">
          <a-radio-group
            v-model:value="fields.urgency"
            :options="[
              { value: 'normal', label: '普通' },
              { value: 'urgent', label: '加急' },
            ]"
          />
        </a-form-item>
        <a-form-item label="备注"><a-textarea v-model:value="fields.remark" :rows="2" :maxlength="300" /></a-form-item>
      </a-form>
      <div>
        <h4>本计划将预定</h4>
        <table class="mat">
          <thead>
            <tr>
              <th>材料</th>
              <th>预定</th>
              <th>预定后可用</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="m in chosen" :key="m.code">
              <td>{{ m.name }}</td>
              <td>{{ selected[m.code]?.qty }}{{ m.unit }}</td>
              <td :class="{ danger: m.available - (selected[m.code]?.qty ?? 0) < 0 }">
                {{ (m.available - (selected[m.code]?.qty ?? 0)).toFixed(1) }}{{ m.unit }}
              </td>
            </tr>
          </tbody>
        </table>
        <p class="muted">折合计划面积约 {{ plannedArea }}㎡ · 产能余 {{ capacity.free }}㎡</p>
        <a-alert v-if="warnings.length" type="warning" show-icon message="请注意">
          <template #description
            ><div v-for="w in warnings" :key="w">· {{ w }}</div></template
          >
        </a-alert>
        <a-alert v-else type="success" show-icon message="材料、产能、交期、回款均无风险" />
      </div>
    </section>

    <div class="footer">
      <a-button v-if="step > 0" @click="step--">上一步</a-button>
      <a-button @click="emit('update:open', false)">取消</a-button>
      <a-button v-if="step < 2" type="primary" :disabled="!canNext" data-testid="wizard-next" @click="step++">下一步</a-button>
      <a-button v-else type="primary" :loading="saving" data-testid="wizard-save" @click="save">保存计划</a-button>
    </div>
  </a-modal>
</template>

<style scoped>
.steps {
  margin: 8px 0 12px;
}
.proto {
  margin-bottom: 12px;
}
.hint {
  margin: 4px 0 8px;
  font-size: 13px;
}
.legend-desc {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 4px 16px;
  font-size: 12px;
  color: var(--gz-muted);
  margin-bottom: 8px;
}
.legend-desc i {
  display: inline-block;
  width: 10px;
  height: 10px;
  border-radius: 2px;
  margin-right: 6px;
}
.rank-scroll {
  max-height: 420px;
  overflow-y: auto;
}
.picked {
  margin-top: 8px;
  padding: 10px 12px;
  border: 1px solid var(--gz-border, #e5e7eb);
  border-radius: 6px;
  background: #f8fafc;
}
.tags {
  margin-top: 6px;
}
.cards {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
  margin-bottom: 8px;
}
.card {
  padding: 10px 12px;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  font-size: 13px;
}
.card h4 {
  margin: 0 0 6px;
  font-size: 13px;
}
.card p {
  margin: 4px 0 0;
}
.card.warn {
  border-color: #fcd34d;
  background: #fffbeb;
}
.card.bad {
  border-color: #fca5a5;
  background: #fef2f2;
}
.toolbar {
  display: flex;
  justify-content: flex-end;
  margin: 4px 0;
}
.mat {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.mat th,
.mat td {
  padding: 6px 8px;
  border-bottom: 1px solid #eef0f3;
  text-align: left;
  vertical-align: top;
}
.mat th {
  color: var(--gz-muted);
  font-weight: 500;
}
.mat tbody tr {
  cursor: pointer;
}
.mat tbody tr.on {
  background: #eff6ff;
}
.confirm {
  display: grid;
  grid-template-columns: 360px 1fr;
  gap: 24px;
}
.confirm h4 {
  margin: 0 0 8px;
}
.form :deep(.ant-form-item) {
  margin-bottom: 10px;
}
.muted {
  color: var(--gz-muted);
  font-size: 12px;
}
.danger {
  color: var(--gz-error, #dc2626);
}
.warn-text {
  color: #d97706;
  font-size: 12px;
}
.footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 16px;
}
</style>
