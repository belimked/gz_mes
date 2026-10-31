<script setup lang="ts">
import { computed } from 'vue'
import { statusTone, type Tone } from '@/theme'
const props = withDefaults(defineProps<{ tone?: Tone; label: string }>(), { tone: 'neutral' })
const color = computed(() => statusTone[props.tone])
</script>
<template>
  <a-tag class="status" :style="{ background: color.bg, borderColor: color.border, color: color.text }">{{ label }}</a-tag>
</template>
<style scoped>
.status {
  margin: 0;
  border-radius: 9999px;
  white-space: normal;
}
</style>
