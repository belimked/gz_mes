<script setup lang="ts">
import { computed } from 'vue'
import { PROCESS_STEPS, statusTone, type Tone } from '@/theme'
import type { Schemas } from '@/api/client'
import { fmtH } from '@/utils/time'
import StatusTag from './StatusTag.vue'

const props = defineProps<{ items: Schemas['JobLoad'][] }>()
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
    return {
      seq,
      name,
      qty,
      area: item?.area ?? 0,
      cycleSamples: item?.cycleSamples ?? 0,
      cycleHours: item?.cycleHours ?? null,
      cycleP90Hours: item?.cycleP90Hours ?? null,
      dwellHours: item?.dwellHours ?? null,
      dwellMaxHours: item?.dwellMaxHours ?? null,
      tone,
      label,
    }
  })
})
const groups = computed(() => [
  { name: '钣金工段', color: '#1677ff', steps: steps.value.slice(0, 4) },
  { name: '喷涂工段', color: '#13c2c2', steps: steps.value.slice(4, 8) },
  { name: '成品工段', color: '#52c41a', steps: steps.value.slice(8, 11) },
])
</script>
<template>
  <div class="pipeline-scroll">
    <div class="pipeline">
      <section v-for="group in groups" :key="group.name" class="group" :class="{ finished: group.steps.length === 3 }">
        <div class="group-header">
          <strong><i class="group-mark" :style="{ background: group.color }" />{{ group.name }}</strong>
          <span
            >{{ group.steps.length }} 个工步 · {{ group.steps.reduce((total, step) => total + step.qty, 0) }} 件{{
              group.name === '成品工段' ? '待转' : '在制'
            }}</span
          >
        </div>
        <div class="group-steps">
          <div
            v-for="step in group.steps"
            :key="step.seq"
            class="step"
            :style="{ background: statusTone[step.tone].bg, borderColor: statusTone[step.tone].border }"
          >
            <div class="step-header">
              <strong>{{ String(step.seq).padStart(2, '0') }}. {{ step.name }}</strong>
              <StatusTag :tone="step.tone" :label="step.label" />
            </div>
            <div class="step-value">{{ step.qty }}<span> 件</span></div>
            <div class="step-area">{{ step.area.toFixed(1) }} m²</div>
            <div class="step-times">
              <a-tooltip :title="`近 14 天 ${step.cycleSamples} 件，从上一道扫码到本道扫码的平均/90 分位耗时`">
                <div>
                  周期 ø {{ step.cycleHours == null ? '—' : fmtH(step.cycleHours) }} · P90
                  {{ step.cycleP90Hours == null ? '—' : fmtH(step.cycleP90Hours) }}
                </div>
              </a-tooltip>
              <a-tooltip v-if="step.qty > 0" :title="`当前停在本道的 ${step.qty} 件已等待时长`">
                <div :class="{ 'dwell-alert': (step.dwellMaxHours ?? 0) > 24 }">
                  停留 ø {{ step.dwellHours == null ? '—' : fmtH(step.dwellHours) }} · 最长
                  {{ step.dwellMaxHours == null ? '—' : fmtH(step.dwellMaxHours) }}
                </div>
              </a-tooltip>
              <div v-if="!step.qty" class="step-idle">目前无在制工件</div>
            </div>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>
<style scoped>
.pipeline-scroll {
  overflow-x: auto;
  padding-bottom: 4px;
}
.pipeline {
  display: flex;
  gap: 10px;
  min-width: 1100px;
}
.group {
  flex: 4 0 0;
  min-width: 0;
  padding: 10px;
  border: 1px solid var(--gz-border);
  border-radius: var(--gz-radius);
  background: var(--gz-fill, #f5f7fa);
}
.group.finished {
  flex: 3 0 0;
}
.group-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 6px;
  margin-bottom: 10px;
  font-size: 12px;
}
.group-header strong {
  display: inline-flex;
  align-items: center;
  white-space: nowrap;
}
.group-header span {
  color: var(--gz-muted);
  white-space: nowrap;
  font-size: 11px;
}
.group-mark {
  width: 4px;
  height: 15px;
  border-radius: 2px;
  margin-right: 6px;
}
.group-steps {
  display: flex;
  gap: 7px;
}
.step {
  flex: 1 0 0;
  min-width: 0;
  height: 170px;
  border: 1px solid;
  border-radius: var(--gz-radius);
  padding: 8px 6px;
  display: flex;
  flex-direction: column;
}
.step-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 2px;
  font-size: 11px;
  line-height: 1.25;
  min-height: 28px;
}
.step-header strong {
  min-width: 0;
  overflow-wrap: anywhere;
}
.step-header :deep(.status) {
  font-size: 10px;
  padding: 0 3px;
  line-height: 17px;
}
.step-value {
  margin-top: 8px;
  font-size: 26px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  line-height: 1.1;
}
.step-value span {
  font-size: 11px;
  font-weight: 400;
}
.step-area {
  font-size: 11px;
  color: var(--gz-secondary);
  margin-top: 3px;
}
.step-times {
  margin-top: auto;
  padding-top: 7px;
  border-top: 1px solid var(--gz-border);
  font-size: 10px;
  color: var(--gz-secondary);
  line-height: 1.4;
  overflow-wrap: anywhere;
}
.step-idle {
  margin-top: 3px;
  color: var(--gz-muted);
  font-size: 11px;
}
.dwell-alert {
  color: var(--gz-error);
}
</style>
