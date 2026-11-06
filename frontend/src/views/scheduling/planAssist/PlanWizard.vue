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
  lineDay,
  lineLoad,
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
const isSpray = computed(() => props.kind === 'spray_daily')
/** 可选产线：钣金计划用车间字典，喷涂计划用喷涂线字典；字典为空时给演示值。 */
const lineOptions = computed(() => {
  const opts = (isSpray.value ? props.paintLines : props.workshops).map((o) => String(o.value))
  if (opts.length) return opts
  return isSpray.value ? ['喷涂一线', '喷涂二线', '喷涂三线'] : ['钣金一车间', '钣金二车间']
})
const currentLine = computed(() => (isSpray.value ? fields.value.paintLine : fields.value.workshop) ?? '')
function setLine(l: string) {
  if (isSpray.value) fields.value.paintLine = l
  else fields.value.workshop = l
}

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
  const d = dates.value
  const best = [...lineOptions.value].sort((a, b) => lineLoad(props.kind, b, d).free - lineLoad(props.kind, a, d).free)[0]
  setLine(!isSpray.value && lineOptions.value.includes(c.batch.workshop) ? c.batch.workshop : (best ?? ''))
  if (isSpray.value && !fields.value.shift) fields.value.shift = props.shifts[0] ? String(props.shifts[0].value) : undefined
  if (props.kind === 'sheet_daily' && !fields.value.jobId) fields.value.jobId = props.jobs[0] ? Number(props.jobs[0].value) : undefined
}
function onRankClick(p: { dataIndex: number }) {
  const c = candidates.value[p.dataIndex]
  if (c) pickCandidate(c)
}

// ---------- ② 材料 ----------
const capacity = computed(() =>
  dates.value[0] && currentLine.value ? lineLoad(props.kind, currentLine.value, dates.value) : { days: 0, total: 0, used: 0, free: 0 },
)
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
  if (eqArea(m) > capacity.value.free) return '超过产线剩余产能'
  return ''
}
// ---------- ③ 组装计划 ----------
const chosen = computed(() => materials.value.filter((m) => selected.value[m.code]))
/** 某材料数量能支撑的产品面积：按该材料在本批次需求中的占比折算。 */
function eqArea(m: Material) {
  const qty = selected.value[m.code]?.qty ?? 0
  return m.need > 0 ? (qty / m.need) * (picked.value?.area ?? 0) : 0
}
/** 计划面积：各所选材料折算成产品面积，取最小值（木桶效应）。 */
const plannedArea = computed(() => {
  const eq = chosen.value.map(eqArea)
  return eq.length ? Math.round(Math.min(...eq) * 10) / 10 : 0
})
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
  else if (currentLine.value && planDays.value.length) {
    const perDay = plannedArea.value / planDays.value.length
    const tight = planDays.value.filter((d) => {
      const x = lineDay(props.kind, currentLine.value, d)
      return perDay > x.cap - x.used
    })
    if (tight.length) w.push(`${tight.map((d) => md(d)).join('、')} 当天空闲产能不足（每天约 ${perDay.toFixed(1)}㎡），建议换日期或换线`)
  }
  return w
})

const canNext = computed(() => (step.value === 0 ? !!picked.value : step.value === 1 ? chosen.value.length > 0 : true))

const WEEK = ['日', '一', '二', '三', '四', '五', '六']
const PLAN_BLUE = '#2563eb'

/** 材料饼图：每种材料一个饼，本计划拿走的那一块「切出来」。 */
const PIE_ROW = 150
const pieCols = computed(() => (chosen.value.length <= 1 ? 1 : 2))
const pieHeight = computed(() => Math.ceil(Math.max(1, chosen.value.length) / pieCols.value) * PIE_ROW + 30)
const pieOption = computed<EChartsOption>(() => {
  const ms = chosen.value
  const cols = pieCols.value
  const pos = (i: number) => ({ x: ((i % cols) + 0.5) / cols, y: 30 + Math.floor(i / cols) * PIE_ROW })
  return {
    legend: { top: 0, itemWidth: 10, itemHeight: 10, data: ['本计划', '缺口', '剩余可用', '他单已预定'] },
    tooltip: { trigger: 'item', formatter: '{a}<br/>{b}：{c}（{d}%）' },
    title: ms.map((m, i) => ({
      text: m.name.length > 14 ? `${m.name.slice(0, 14)}…` : m.name,
      subtext: `本计划 ${selected.value[m.code]?.qty}${m.unit}`,
      left: `${pos(i).x * 100}%`,
      top: pos(i).y + 2,
      textAlign: 'center',
      itemGap: 2,
      textStyle: { fontSize: 12, fontWeight: 600 },
      subtextStyle: { fontSize: 11, color: PLAN_BLUE },
    })),
    series: ms.map((m, i) => {
      const qty = selected.value[m.code]?.qty ?? 0
      const avail = Math.max(0, m.available)
      const fromStock = Math.min(qty, avail)
      const unit = m.unit
      return {
        name: `${m.name}（${unit}）`,
        type: 'pie',
        radius: 40,
        center: [`${pos(i).x * 100}%`, pos(i).y + 92],
        selectedMode: 'multiple',
        selectedOffset: 14,
        label: { show: false },
        data: [
          { name: '本计划', value: Math.round(fromStock * 10) / 10, selected: true, itemStyle: { color: PLAN_BLUE } },
          { name: '缺口', value: Math.round(Math.max(0, qty - avail) * 10) / 10, selected: true, itemStyle: { color: '#ef4444' } },
          { name: '剩余可用', value: Math.round((avail - fromStock) * 10) / 10, itemStyle: { color: '#cbd5e1' } },
          { name: '他单已预定', value: m.reserved, itemStyle: { color: '#fcd34d' } },
        ],
      }
    }),
  } as EChartsOption
})

/** 产线负荷图：全部产线在所选日期内的已占用 / 本计划 / 剩余，点击选线。 */
const lineStats = computed(() =>
  lineOptions.value.map((l) => {
    const s = dates.value[0] ? lineLoad(props.kind, l, dates.value) : { days: 0, total: 0, used: 0, free: 0 }
    const mine = l === currentLine.value ? plannedArea.value : 0
    return { line: l, ...s, mine: Math.min(mine, s.free), over: Math.max(0, mine - s.free), rest: Math.max(0, s.free - mine) }
  }),
)
const lineOption = computed<EChartsOption>(() => {
  const st = lineStats.value
  const bar = (name: string, color: string, get: (x: (typeof st)[number]) => number) => ({
    name,
    type: 'bar',
    stack: 'load',
    barWidth: 20,
    cursor: 'pointer',
    itemStyle: { color },
    data: st.map((x) => ({ value: Math.round(get(x)), itemStyle: { opacity: x.line === currentLine.value ? 1 : 0.55 } })),
  })
  return {
    grid: { left: 8, right: 16, top: 30, bottom: 4, containLabel: true },
    legend: { top: 0, itemWidth: 10, itemHeight: 10 },
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: (p: unknown) => {
        const x = st[(p as { dataIndex: number }[])[0]?.dataIndex ?? 0]
        if (!x) return ''
        return `<b>${x.line}</b>（${x.days} 天）<br/>产能 ${x.total}㎡ · 已占用 ${x.used}㎡<br/>剩余 <b>${x.free}</b>㎡${x.line === currentLine.value ? '' : '<br/><i>点击选用这条线</i>'}`
      },
    },
    xAxis: { type: 'value', name: '㎡', splitLine: { lineStyle: { type: 'dashed' } } },
    yAxis: {
      type: 'category',
      inverse: true,
      data: st.map((x) => x.line),
      axisLabel: {
        formatter: (v: string) => (v === currentLine.value ? `{on|▶ ${v}}` : v),
        rich: { on: { color: PLAN_BLUE, fontWeight: 'bold' } },
      },
    },
    series: [
      bar('已占用', '#94a3b8', (x) => x.used),
      bar('本计划', PLAN_BLUE, (x) => x.mine),
      bar('超出产能', '#ef4444', (x) => x.over),
      bar('剩余', '#bbf7d0', (x) => x.rest),
    ],
  } as EChartsOption
})
function onLineClick(p: { dataIndex: number }) {
  const x = lineStats.value[p.dataIndex]
  if (x) setLine(x.line)
}

/** 日历：6 周，颜色为所选产线当天的空闲比例；蓝框为计划日期，红框交期，灰框已有计划。 */
const pickingEnd = ref(false)
const calDays = computed(() => {
  const today = dayjs().startOf('day')
  const start = today.subtract((today.day() + 6) % 7, 'day')
  return Array.from({ length: 42 }, (_, i) => start.add(i, 'day').format('YYYY-MM-DD'))
})
const inRange = (d: string, a: string, b: string) => d >= a && d <= b
const calOption = computed<EChartsOption>(() => {
  const days = calDays.value
  const today = dayjs().format('YYYY-MM-DD')
  const line = currentLine.value
  const due = picked.value?.batch.requiredDate ?? ''
  const existing = picked.value?.existing ?? []
  const heat = days.map((d) => {
    if (d < today || !line) return [d, -1]
    const x = lineDay(props.kind, line, d)
    return [d, Math.round(((x.cap - x.used) / x.cap) * 100) / 100]
  })
  const frame = (list: string[], color: string, type: 'solid' | 'dashed' = 'solid') => ({
    type: 'scatter',
    coordinateSystem: 'calendar',
    symbol: 'rect',
    symbolSize: [42, 24],
    silent: true,
    itemStyle: { color: 'transparent', borderColor: color, borderWidth: 2, borderType: type },
    data: list.map((d) => [d, 1]),
    z: 5,
  })
  return {
    tooltip: {
      formatter: (p: unknown) => {
        const v = (p as { value: [string, number] }).value
        const d = v[0]
        const tags = [
          inRange(d, dates.value[0], dates.value[1]) ? '本计划' : '',
          d === due ? '交期' : '',
          existing.some((e) => inRange(d, e.start, e.end)) ? '已有计划' : '',
        ].filter(Boolean)
        if (d < today) return `${d}（已过）`
        const x = lineDay(props.kind, line, d)
        return `${d} 周${WEEK[dayjs(d).day()]}<br/>${line}：空闲 <b>${x.cap - x.used}</b> / ${x.cap}㎡${tags.length ? `<br/>${tags.join(' · ')}` : ''}`
      },
    },
    visualMap: {
      type: 'piecewise',
      seriesIndex: 0,
      orient: 'horizontal',
      top: 0,
      left: 'center',
      itemWidth: 12,
      itemHeight: 10,
      textStyle: { fontSize: 11 },
      pieces: [
        { min: 0.5, label: '空闲', color: '#86efac' },
        { min: 0.25, max: 0.5, label: '偏紧', color: '#fde68a' },
        { min: 0, max: 0.25, label: '快满', color: '#fca5a5' },
        { max: -0.5, label: '已过', color: '#f1f5f9' },
      ],
    },
    calendar: {
      orient: 'horizontal',
      range: [days[0] as string, days[days.length - 1] as string],
      cellSize: [46, 28],
      left: 34,
      top: 58,
      dayLabel: { firstDay: 1, nameMap: 'ZH', fontSize: 11 },
      monthLabel: { nameMap: 'ZH', fontSize: 11 },
      yearLabel: { show: false },
      splitLine: { show: false },
      itemStyle: { borderColor: '#fff', borderWidth: 2 },
    },
    series: [
      {
        type: 'heatmap',
        coordinateSystem: 'calendar',
        cursor: 'pointer',
        data: heat,
        label: {
          show: true,
          fontSize: 11,
          color: '#334155',
          formatter: (p: { value: [string, number] }) => String(dayjs(p.value[0]).date()),
        },
      },
      frame(
        existing.flatMap((e) => days.filter((d) => inRange(d, e.start, e.end))),
        '#64748b',
        'dashed',
      ),
      frame(due && days.includes(due) ? [due] : [], '#dc2626'),
      frame(dates.value[0] ? days.filter((d) => inRange(d, dates.value[0], dates.value[1])) : [], PLAN_BLUE),
    ],
  } as EChartsOption
})
function onCalClick(p: { seriesType?: string; value?: unknown }) {
  if (p.seriesType !== 'heatmap' || !Array.isArray(p.value)) return
  const d = String(p.value[0])
  if (d < dayjs().format('YYYY-MM-DD')) return
  if (pickingEnd.value && d >= dates.value[0]) {
    dates.value = [dates.value[0], d]
    pickingEnd.value = false
    return
  }
  const est = picked.value?.estDays ?? 1
  dates.value = [
    d,
    dayjs(d)
      .add(est - 1, 'day')
      .format('YYYY-MM-DD'),
  ]
  pickingEnd.value = true
}

/** 计划图：材料 → 计划 → 产线 → 每天，宽度为折算面积。 */
const planDays = computed(() => {
  if (!dates.value[0]) return []
  const out: string[] = []
  for (let d = dayjs(dates.value[0]); !d.isAfter(dayjs(dates.value[1])) && out.length < 31; d = d.add(1, 'day'))
    out.push(d.format('YYYY-MM-DD'))
  return out
})
const sankeyOption = computed<EChartsOption>(() => {
  const c = picked.value
  const area = plannedArea.value
  if (!c || !area || !currentLine.value) return {}
  const planNode = `计划 ${c.batch.lotNo}`
  const lineNode = currentLine.value
  const eqs = chosen.value.map(eqArea)
  const eqSum = eqs.reduce((a, b) => a + b, 0) || 1
  const perDay = Math.round((area / Math.max(1, planDays.value.length)) * 10) / 10
  const dayNodes = planDays.value.map((d) => {
    const x = lineDay(props.kind, lineNode, d)
    return { d, name: `${dayjs(d).format('MM-DD')} 周${WEEK[dayjs(d).day()]}`, free: x.cap - x.used, over: perDay > x.cap - x.used }
  })
  const matColors = ['#0ea5e9', '#6366f1', '#14b8a6', '#a855f7']
  return {
    tooltip: {
      trigger: 'item',
      formatter: (p: unknown) => {
        const q = p as { dataType: string; name: string; data: { source?: string; target?: string; value?: number; tip?: string } }
        if (q.data.tip) return q.data.tip
        if (q.dataType === 'edge') return `${q.data.source} → ${q.data.target}<br/>约 ${q.data.value}㎡`
        return q.name
      },
    },
    series: [
      {
        type: 'sankey',
        left: 8,
        right: 110,
        top: 24,
        bottom: 10,
        nodeWidth: 14,
        nodeGap: 8,
        layoutIterations: 0,
        draggable: false,
        emphasis: { focus: 'adjacency' },
        label: { fontSize: 12, color: '#1e293b' },
        lineStyle: { color: 'gradient', opacity: 0.35, curveness: 0.5 },
        data: [
          ...chosen.value.map((m, i) => ({
            name: m.name,
            depth: 0,
            itemStyle: { color: matColors[i % matColors.length] },
            tip: `${m.name}<br/>本计划 ${selected.value[m.code]?.qty}${m.unit}`,
          })),
          {
            name: planNode,
            depth: 1,
            itemStyle: { color: PLAN_BLUE },
            label: { position: 'top', fontWeight: 'bold' },
            tip: `${planNode}<br/>${c.batch.projectName}<br/>约 ${area}㎡`,
          },
          {
            name: lineNode,
            depth: 2,
            itemStyle: { color: '#0f766e' },
            label: { position: 'top', fontWeight: 'bold' },
            tip: `${lineNode}<br/>剩余产能 ${capacity.value.free}㎡`,
          },
          ...dayNodes.map((x) => ({
            name: x.name,
            depth: 3,
            itemStyle: { color: x.over ? '#ef4444' : '#22c55e' },
            label: { color: x.over ? '#dc2626' : '#1e293b' },
            tip: `${x.d}<br/>安排 ${perDay}㎡ · 当天空闲 ${x.free}㎡${x.over ? '<br/><b style="color:#dc2626">超出当天产能</b>' : ''}`,
          })),
        ],
        links: [
          ...chosen.value.map((m, i) => ({
            source: m.name,
            target: planNode,
            value: Math.round(((area * (eqs[i] ?? 0)) / eqSum) * 10) / 10,
          })),
          { source: planNode, target: lineNode, value: area },
          ...dayNodes.map((x) => ({ source: lineNode, target: x.name, value: perDay })),
        ],
      },
    ],
  } as EChartsOption
})

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
    <a-steps :current="step" size="small" class="steps" :items="[{ title: '选择批次' }, { title: '选择材料' }, { title: '组装计划' }]" />
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

    <!-- ③ 组装计划：材料饼图 + 产线 + 日历 → 计划图 -->
    <section v-if="step === 2 && picked" class="compose" data-testid="wizard-compose">
      <div class="panel">
        <h4><span class="no">1</span>材料 · 从库存里切出本计划的一块</h4>
        <EChart :option="pieOption" :height="pieHeight" />
      </div>
      <div class="panel">
        <h4>
          <span class="no">2</span>{{ isSpray ? '喷涂线' : '车间' }} · 点击柱子选一条线
          <span class="muted">（所选日期内负荷）</span>
        </h4>
        <EChart :option="lineOption" :height="Math.max(110, lineStats.length * 34 + 44)" data-testid="wizard-lines" @click="onLineClick" />
        <div class="chips">
          <template v-if="isSpray">
            <span class="muted">班次</span>
            <a-radio-group v-model:value="fields.shift" size="small" button-style="solid">
              <a-radio-button v-for="s in shifts" :key="s.value" :value="s.value">{{ s.label }}</a-radio-button>
            </a-radio-group>
          </template>
          <template v-if="kind === 'sheet_daily'">
            <span class="muted">工序</span>
            <a-radio-group v-model:value="fields.jobId" size="small" button-style="solid">
              <a-radio-button v-for="j in jobs.slice(0, 4)" :key="j.value" :value="j.value">{{ j.label }}</a-radio-button>
            </a-radio-group>
          </template>
        </div>
      </div>
      <div class="panel">
        <h4><span class="no">3</span>日期 · {{ pickingEnd ? '再点一天作为结束日（或点比开始更早的一天重选）' : '点击一天作为开始日' }}</h4>
        <EChart :option="calOption" :height="270" data-testid="wizard-calendar" @click="onCalClick" />
        <div class="cal-legend muted">
          <span><i class="box plan" />本计划</span><span><i class="box due" />交期 {{ md(picked.batch.requiredDate) }}</span
          ><span><i class="box exist" />已有计划</span><span>颜色：{{ currentLine }} 当天空闲产能</span>
        </div>
      </div>

      <div class="panel result">
        <h4><span class="no done">=</span>新生产计划</h4>
        <div class="plan-head">
          <b>{{ picked.batch.lotNo }}</b> · {{ picked.batch.projectName }}
          <div class="muted">
            {{ currentLine || '未选产线' }}<template v-if="isSpray && fields.shift"> · {{ fields.shift }}</template> · {{ md(dates[0]) }} ~
            {{ md(dates[1]) }}（{{ planDays.length }} 天）· 约 <b>{{ plannedArea }}</b
            >㎡
          </div>
        </div>
        <EChart
          v-if="plannedArea && currentLine"
          :option="sankeyOption"
          :height="Math.max(220, Math.max(chosen.length, planDays.length) * 30 + 30)"
          data-testid="wizard-sankey"
        />
        <a-empty v-else description="选择材料和产线后生成计划图" />
        <a-alert v-if="warnings.length" type="warning" show-icon message="请注意" class="gap">
          <template #description
            ><div v-for="w in warnings" :key="w">· {{ w }}</div></template
          >
        </a-alert>
        <a-alert v-else type="success" show-icon message="材料、产能、交期、回款均无风险" class="gap" />
        <div class="extra">
          <a-radio-group
            v-model:value="fields.urgency"
            size="small"
            :options="[
              { value: 'normal', label: '普通' },
              { value: 'urgent', label: '加急' },
            ]"
          />
          <a-input v-model:value="fields.remark" size="small" :maxlength="300" placeholder="备注（选填）" />
        </div>
      </div>
    </section>

    <div class="footer">
      <a-button v-if="step > 0" @click="step--">上一步</a-button>
      <a-button @click="emit('update:open', false)">取消</a-button>
      <a-button v-if="step < 2" type="primary" :disabled="!canNext" data-testid="wizard-next" @click="step++">下一步</a-button>
      <a-button v-else type="primary" :loading="saving" :disabled="!plannedArea || !currentLine" data-testid="wizard-save" @click="save"
        >确认生成计划</a-button
      >
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
.compose {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}
.panel {
  padding: 10px 12px;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  min-width: 0;
}
.panel h4 {
  margin: 0 0 6px;
  font-size: 13px;
  display: flex;
  align-items: center;
  gap: 6px;
}
.no {
  display: inline-grid;
  place-items: center;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: #e2e8f0;
  font-size: 11px;
}
.no.done {
  background: #2563eb;
  color: #fff;
}
.result {
  grid-column: 2;
  grid-row: 1 / span 3;
  border-color: #bfdbfe;
  background: #f8fbff;
}
.plan-head {
  font-size: 13px;
  margin-bottom: 4px;
}
.chips {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-top: 4px;
}
.cal-legend {
  display: flex;
  flex-wrap: wrap;
  gap: 14px;
}
.box {
  display: inline-block;
  width: 12px;
  height: 9px;
  margin-right: 4px;
  border: 2px solid;
  vertical-align: middle;
}
.box.plan {
  border-color: #2563eb;
}
.box.due {
  border-color: #dc2626;
}
.box.exist {
  border-color: #64748b;
  border-style: dashed;
}
.gap {
  margin-top: 8px;
}
.extra {
  display: grid;
  grid-template-columns: auto 1fr;
  align-items: center;
  gap: 10px;
  margin-top: 10px;
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
