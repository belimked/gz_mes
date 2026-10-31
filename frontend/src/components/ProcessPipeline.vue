<script setup lang="ts">
import { computed } from 'vue'
import { PROCESS_STEPS, statusTone, type Tone } from '@/theme'
import StatusTag from './StatusTag.vue'

const props = defineProps<{ items: { jobCode: string; jobName: string; seq: number; qty: number; area: number }[] }>()
const steps = computed(() => {
  const maxQty = Math.max(0, ...props.items.map((item) => item.qty))
  return PROCESS_STEPS.map((name, index) => {
    const seq = index + 1
    const item = props.items.find((entry) => entry.seq === seq)
    const qty = item?.qty ?? 0
    let tone: Tone = 'neutral'
    let label = '空闲'
    if (seq === 11) {
      tone = 'success'
      label = qty > 0 ? '待对账' : '已清'
    } else if (qty > 0 && qty === maxQty) {
      tone = 'warning'
      label = '堆积'
    } else if (qty > 0) {
      tone = 'processing'
      label = '在制'
    }
    return { seq, name, qty, area: item?.area ?? 0, tone, label }
  })
})
</script>
<template>
  <div class="pipeline-scroll">
    <div class="pipeline">
      <div class="group sheet">钣金工段</div>
      <div class="group coating">喷涂工段</div>
      <div class="group finished">成品工段</div>
      <div
        v-for="step in steps"
        :key="step.seq"
        class="step"
        :style="{ background: statusTone[step.tone].bg, borderColor: statusTone[step.tone].border }"
      >
        <div class="step-name">{{ String(step.seq).padStart(2, '0') }}. {{ step.name }}</div>
        <div class="step-value">{{ step.qty }}<span> 件</span></div>
        <div class="step-area">{{ step.area.toFixed(1) }} m²</div>
        <StatusTag :tone="step.tone" :label="step.label" />
      </div>
    </div>
  </div>
</template>
<style scoped>
.pipeline-scroll {
  overflow-x: auto;
  padding-bottom: 4px;
}
.pipeline {
  display: grid;
  grid-template-columns: repeat(11, minmax(0, 1fr));
  gap: 10px;
  min-width: 1100px;
}
.group {
  background: #f5f7fa;
  color: var(--gz-secondary);
  text-align: center;
  font-size: 12px;
  padding: 6px;
  border-radius: var(--gz-radius);
}
.sheet {
  grid-column: span 4;
}
.coating {
  grid-column: span 4;
}
.finished {
  grid-column: span 3;
}
.step {
  position: relative;
  min-height: 150px;
  border: 1px solid;
  border-radius: var(--gz-radius);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: space-between;
  text-align: center;
  padding: 10px 4px;
}
.step:not(:last-child)::after {
  content: '›';
  position: absolute;
  right: -10px;
  top: 50%;
  transform: translate(50%, -50%);
  color: var(--gz-muted);
  font-size: 16px;
  line-height: 1;
}
.step-name,
.step-area {
  font-size: 12px;
  color: var(--gz-secondary);
}
.step-value {
  font-size: 26px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}
.step-value span {
  font-size: 12px;
  font-weight: 400;
}
</style>
