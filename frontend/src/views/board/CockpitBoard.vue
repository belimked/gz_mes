<script setup lang="ts">
import { computed, onUnmounted, ref } from 'vue'
import type { EChartsOption } from 'echarts'
import { call, client, errorMessage, type Schemas } from '@/api/client'
import BoardShell from '@/components/BoardShell.vue'
import EChart from '@/components/EChart.vue'
import { theme } from '@/theme'
const data=ref<Schemas['CockpitBoard']>()
const refreshed=ref(''),error=ref('')
async function load(){try{data.value=await call(client.GET('/boards/cockpit'));refreshed.value=new Date().toLocaleTimeString('zh-CN');error.value=''}catch(e){error.value=errorMessage(e)}}
void load()
const refresh=setInterval(() => void load(),60000)
onUnmounted(() => clearInterval(refresh))
const kpis=computed(() => {const k=data.value?.kpi;return k ? [['在产批次',k.inProductionBatches],['未完工批次',k.openBatches],['逾期批次',k.overdueBatches],['今日扫码',k.todayScanQty],['今日扫码面积',k.todayScanArea],['今日发货',k.todayShipQty],['今日发货面积',k.todayShipArea],['半成品库存',k.semiStockQty],['半成品面积',k.semiStockArea],['成品库存',k.finishedStockQty],['成品面积',k.finishedStockArea]] as const : []})
const primary=theme?.token?.colorPrimary ?? '#1677ff'
function chart(rows:Schemas['BoardDailyMetric'][] | undefined):EChartsOption {return {backgroundColor:'transparent',grid:{left:42,right:12,top:20,bottom:28},tooltip:{trigger:'axis'},xAxis:{type:'category',data:rows?.map(r=>r.date.slice(5)),axisLabel:{color:'#b9d7f0'}},yAxis:{type:'value',axisLabel:{color:'#b9d7f0'},splitLine:{lineStyle:{color:'#284460'}}},series:[{type:'bar',name:'件数',data:rows?.map(r=>r.qty),itemStyle:{color:primary}},{type:'line',name:'面积',data:rows?.map(r=>r.area),itemStyle:{color:'#25d4c8'}}]}}
const output=computed<EChartsOption>(() => ({grid:{left:80,right:20,top:15,bottom:25},tooltip:{trigger:'axis'},xAxis:{type:'value',axisLabel:{color:'#b9d7f0'}},yAxis:{type:'category',data:data.value?.jobOutput.map(x=>x.name).reverse(),axisLabel:{color:'#b9d7f0'}},series:[{type:'bar',data:data.value?.jobOutput.map(x=>x.area).reverse(),itemStyle:{color:primary}}]}))
</script>
<template>
<BoardShell title="办公室驾驶舱" filter-label="" :rows="[]" :kpis="[]" :chart-labels="[]" :chart-planned="[]" :chart-done="[]" :refreshed="refreshed" custom @refresh="load">
<p v-if="error" role="alert">{{ error }}</p>
<div class="kpis"><article v-for="[name,value] in kpis" :key="name"><small>{{ name }}</small><b>{{ value.toLocaleString('zh-CN',{maximumFractionDigits:2}) }}</b></article></div>
<div class="grid trends"><section v-for="plot in [{title:'近七日钣金下线 · 20',rows:data?.weeklySheet},{title:'近七日下挂 · 40',rows:data?.weeklySpray},{title:'近七日发货',rows:data?.weeklyShipment}]" :key="plot.title"><h2>{{ plot.title }}</h2><EChart :option="chart(plot.rows)" :height="180" /></section></div>
<div class="grid details"><section><h2>工序今日产出 · ㎡</h2><EChart :option="output" :height="340" /></section><section><h2>机台产能</h2><div class="scroll"><p v-for="m in data?.machineCapacity" :key="m.machineId">{{ m.name }} <strong>{{ m.qty }} 件 / {{ m.area.toFixed(2) }} ㎡</strong></p><p v-if="!data?.machineCapacity.length">暂无运行机台</p></div></section><section><h2>钣金日计划完成</h2><div class="ring">{{ (data?.sheetDaily.areaRate ?? 0).toFixed(1) }}%</div><p>数量 {{ data?.sheetDaily.doneQty ?? 0 }} / {{ data?.sheetDaily.allQty ?? 0 }} · {{ (data?.sheetDaily.qtyRate ?? 0).toFixed(1) }}%</p><p>面积 {{ (data?.sheetDaily.doneArea ?? 0).toFixed(2) }} / {{ (data?.sheetDaily.allArea ?? 0).toFixed(2) }} ㎡</p></section></div>
<div class="grid bottom"><section><h2>车间待钣金下线</h2><p v-for="w in data?.workshopBacklog" :key="w.workshop">{{ w.workshop }} <strong>{{ w.noScanQty }} 件 · {{ w.noScanArea.toFixed(2) }} ㎡</strong></p></section><section><h2>未完工项目 · 工序 55 完成</h2><div class="scroll"><table><thead><tr><th>项目</th><th>批次</th><th>计划</th><th>完成</th><th>进度</th></tr></thead><tbody><tr v-for="p in data?.projects" :key="p.projectName"><td>{{ p.projectName }}</td><td>{{ p.lotNos }}</td><td>{{ p.planQty }} 件 / {{ p.planArea.toFixed(2) }} ㎡</td><td>{{ p.finishQty }} 件 / {{ p.finishArea.toFixed(2) }} ㎡</td><td>{{ p.pct }}%</td></tr></tbody></table></div></section></div>
</BoardShell>
</template>
<style scoped>
.kpis{display:grid;grid-template-columns:repeat(auto-fit,minmax(140px,1fr));gap:12px;margin-top:16px}.kpis article,section{background:#10243d;border:1px solid #284460;border-radius:8px;padding:16px;color:#e8f5ff}.kpis article{border-top:3px solid #25d4c8}.kpis small{display:block;color:#93b2ce}.kpis b{display:block;font-size:25px}.grid{display:grid;gap:14px;margin-top:14px}.trends,.details{grid-template-columns:repeat(3,minmax(0,1fr))}.bottom{grid-template-columns:1fr 2fr}h2{font-size:17px;color:#e8f5ff}p{border-bottom:1px solid #284460;padding:8px 0}strong{float:right;color:#25d4c8}.scroll{overflow:auto;max-height:350px}table{width:100%;white-space:nowrap;border-collapse:collapse}td,th{padding:8px;border-bottom:1px solid #284460;text-align:left}.ring{border:12px solid #25d4c8;border-radius:50%;width:155px;height:155px;display:grid;place-items:center;font-size:30px;margin:auto}@media(max-width:1000px){.trends,.details,.bottom{grid-template-columns:1fr}}
</style>
