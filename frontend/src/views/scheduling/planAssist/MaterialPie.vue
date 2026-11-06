<script setup lang="ts">
// 可拖动的材料饼图：整饼 = 可用库存 + 他单已预定；蓝色切片为本计划用量，拖动切片边缘的圆点调整，
// 拖动范围 0 ~ 可用库存；超出库存的数量只能在输入框里填，超出部分以外圈红弧表示缺口。
import { computed, ref } from 'vue'
import type { Material } from './mock'

const props = defineProps<{ material: Material; hint?: string }>()
const qty = defineModel<number>({ required: true })

const R = 52
const C = 70
const SIZE = 140
const OFFSET = 7
const PLAN = '#2563eb'

const avail = computed(() => Math.max(0, props.material.available))
const total = computed(() => Math.max(0.1, avail.value + props.material.reserved))
const inStock = computed(() => Math.min(Math.max(0, qty.value || 0), avail.value))
const short = computed(() => Math.max(0, (qty.value || 0) - avail.value))
const frac = (v: number) => Math.min(1, Math.max(0, v / total.value))

/** 从 12 点方向顺时针的角度（弧度）对应的坐标。 */
function pt(a: number, r = R, cx = C, cy = C) {
  return [cx + r * Math.sin(a), cy - r * Math.cos(a)] as const
}
function arc(a0: number, a1: number, r = R, cx = C, cy = C) {
  if (a1 - a0 <= 1e-4) return ''
  if (a1 - a0 >= Math.PI * 2 - 1e-4) return `M${cx},${cy - r}A${r},${r} 0 1 1 ${cx - 0.01},${cy - r}Z`
  const [x0, y0] = pt(a0, r, cx, cy)
  const [x1, y1] = pt(a1, r, cx, cy)
  return `M${cx},${cy}L${x0},${y0}A${r},${r} 0 ${a1 - a0 > Math.PI ? 1 : 0} 1 ${x1},${y1}Z`
}

const aPlan = computed(() => frac(inStock.value) * Math.PI * 2)
const aAvail = computed(() => frac(avail.value) * Math.PI * 2)
const aNeed = computed(() => frac(props.material.remaining) * Math.PI * 2)
/** 本计划切片沿角平分线向外移出，形成「切出来」的效果。 */
const planShift = computed(() => {
  const mid = aPlan.value / 2
  return [OFFSET * Math.sin(mid), -OFFSET * Math.cos(mid)] as const
})
const planPath = computed(() => arc(0, aPlan.value, R, C + planShift.value[0], C + planShift.value[1]))
const restPath = computed(() => arc(aPlan.value, aAvail.value))
const reservedPath = computed(() => arc(aAvail.value, Math.PI * 2))
const handle = computed(() => pt(aPlan.value, R, C + planShift.value[0], C + planShift.value[1]))
const needTick = computed(() => [pt(aNeed.value, R - 6), pt(aNeed.value, R + 9)] as const)
/** 缺口外圈红弧：按缺口占剩余需求的比例画。 */
const shortArc = computed(() => {
  if (!short.value) return ''
  const a = Math.min(Math.PI * 2 - 0.01, (short.value / Math.max(0.1, props.material.remaining)) * Math.PI * 2)
  const r = R + 6
  const [x0, y0] = pt(0, r)
  const [x1, y1] = pt(a, r)
  return `M${x0},${y0}A${r},${r} 0 ${a > Math.PI ? 1 : 0} 1 ${x1},${y1}`
})

const svg = ref<SVGSVGElement>()
const dragging = ref(false)
function setFromPointer(e: PointerEvent) {
  const rect = svg.value?.getBoundingClientRect()
  if (!rect) return
  const x = ((e.clientX - rect.left) / rect.width) * SIZE - C
  const y = ((e.clientY - rect.top) / rect.height) * SIZE - C
  let a = Math.atan2(x, -y)
  if (a < 0) a += Math.PI * 2
  // 拖过 12 点回绕时贴边，避免数量从最大跳到 0
  const prev = aPlan.value
  if (prev > Math.PI * 1.5 && a < Math.PI / 2) a = Math.PI * 2
  if (prev < Math.PI / 2 && a > Math.PI * 1.5) a = 0
  let v = (a / (Math.PI * 2)) * total.value
  // 靠近剩余需求刻度时吸附
  if (Math.abs(v - props.material.remaining) < total.value * 0.025) v = props.material.remaining
  qty.value = Math.round(Math.min(avail.value, Math.max(0, v)) * 10) / 10
}
function down(e: PointerEvent) {
  dragging.value = true
  ;(e.currentTarget as Element).setPointerCapture(e.pointerId)
  setFromPointer(e)
}
function move(e: PointerEvent) {
  if (dragging.value) setFromPointer(e)
}
function up() {
  dragging.value = false
}
</script>

<template>
  <div class="pie" :class="{ dragging }">
    <div class="name" :title="material.name">{{ material.name }}</div>
    <svg
      ref="svg"
      :viewBox="`0 0 ${SIZE} ${SIZE}`"
      class="svg"
      data-testid="material-pie"
      @pointerdown="down"
      @pointermove="move"
      @pointerup="up"
      @pointercancel="up"
    >
      <path :d="reservedPath" fill="#fcd34d" />
      <path :d="restPath" fill="#cbd5e1" />
      <path :d="planPath" :fill="PLAN" />
      <path v-if="shortArc" :d="shortArc" fill="none" stroke="#ef4444" stroke-width="4" stroke-linecap="round" />
      <line
        v-if="material.remaining > 0 && material.remaining <= total"
        :x1="needTick[0][0]"
        :y1="needTick[0][1]"
        :x2="needTick[1][0]"
        :y2="needTick[1][1]"
        stroke="#0f172a"
        stroke-width="1.5"
        stroke-dasharray="2 2"
      >
        <title>剩余需求 {{ material.remaining }}{{ material.unit }}</title>
      </line>
      <circle :cx="handle[0]" :cy="handle[1]" r="7" fill="#fff" :stroke="PLAN" stroke-width="2.5" class="knob" />
    </svg>
    <a-input-number
      v-model:value="qty"
      :min="0"
      :step="1"
      :precision="1"
      size="small"
      class="input"
      :addon-after="material.unit"
      data-testid="material-qty"
    />
    <div class="meta">
      可用 {{ material.available }} · 需 {{ material.remaining }}
      <span v-if="short" class="danger">· 缺 {{ short.toFixed(1) }}</span>
    </div>
    <div v-if="hint" class="hint">{{ hint }}</div>
  </div>
</template>

<style scoped>
.pie {
  display: grid;
  justify-items: center;
  gap: 2px;
  min-width: 0;
}
.name {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  font-weight: 600;
}
.svg {
  width: 120px;
  height: 120px;
  touch-action: none;
  cursor: grab;
  overflow: visible;
}
.dragging .svg {
  cursor: grabbing;
}
.knob {
  filter: drop-shadow(0 1px 2px rgb(0 0 0 / 0.3));
}
.input {
  width: 120px;
}
.meta {
  font-size: 11px;
  color: var(--gz-muted);
}
.hint {
  font-size: 11px;
  color: #d97706;
}
.danger {
  color: var(--gz-error, #dc2626);
}
</style>
