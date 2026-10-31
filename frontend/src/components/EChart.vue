<script setup lang="ts">
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { PieChart, BarChart, LineChart } from 'echarts/charts'
import { TitleComponent, TooltipComponent, LegendComponent, GridComponent } from 'echarts/components'
import type { EChartsOption } from 'echarts'

use([CanvasRenderer, PieChart, BarChart, LineChart, TitleComponent, TooltipComponent, LegendComponent, GridComponent])

defineProps<{ option: EChartsOption; height?: number }>()
</script>
<template>
  <VChart :option="option" :style="{ height: `${height ?? 280}px`, width: '100%' }" autoresize />
</template>
