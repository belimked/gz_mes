<script setup lang="ts">
import { computed, onUnmounted, ref } from 'vue'
import { call, client, errorMessage, type Schemas } from '@/api/client'
import BoardShell from '@/components/BoardShell.vue'
import { theme } from '@/theme'
const data = ref<Schemas['WorkOrderBoard']>({items:[],total:0,jobs:[]})
const refreshed=ref(''),error=ref(''),keyword=ref(''),status=ref(''),page=ref(1),interval=ref(10)
const pages=computed(() => Math.max(1,Math.ceil(data.value.total/10)))
let requestId=0
async function load() {const id=++requestId;try {const result=await call(client.GET('/boards/work-orders',{params:{query:{page:page.value,size:10,...(status.value ? {status:status.value as 'pending'|'scheduled'|'in_production'}:{}),...(keyword.value ? {keyword:keyword.value}:{})}}}));if(id===requestId){data.value=result;refreshed.value=new Date().toLocaleTimeString('zh-CN');error.value='';if(page.value>pages.value)page.value=1}}catch(e){if(id===requestId)error.value=errorMessage(e)}}
let elapsed=0
const carousel=setInterval(() => {if(++elapsed>=interval.value){elapsed=0;page.value=page.value%pages.value+1;void load()}},1000)
const refresh=setInterval(() => void load(),60000)
onUnmounted(() => {clearInterval(carousel);clearInterval(refresh)})
void load()
const primary=theme?.token?.colorPrimary ?? '#1677ff'
</script>
<template>
  <BoardShell title="工单执行看板" filter-label="" :rows="[]" :kpis="[]" :chart-labels="[]" :chart-planned="[]" :chart-done="[]" :refreshed="refreshed" custom @refresh="load">
    <div class="strip"><span>未完工批次 <b>{{ data.total }}</b></span><span>本页逾期 <b>{{ data.items.filter(x=>x.overdue).length }}</b></span><span>本页急单 <b>{{ data.items.filter(x=>x.urgency==='urgent').length }}</b></span>
      <input v-model="keyword" placeholder="项目 / 客户 / 批次" @change="page=1;load()" />
      <select v-model="status" @change="page=1;load()"><option value="">所有未完工</option><option value="pending">待排产</option><option value="scheduled">已排产</option><option value="in_production">生产中</option></select>
      <label>轮播秒数 <input v-model.number="interval" type="number" min="3" max="120" style="width:65px" /></label></div>
    <p v-if="error" role="alert">{{ error }}</p>
    <div class="panel"><div class="heading"><h2>批次工序进度</h2><span>第 {{ page }} / {{ pages }} 页 <button @click="page=page%pages+1;load()">下一页</button></span></div>
    <div class="scroll"><table><thead><tr><th>批次 / 项目</th><th>客户 / 车间</th><th>交期</th><th>状态</th><th>数量 / 面积</th><th>标签</th><th v-for="j in data.jobs" :key="j.id">{{ j.name }}</th></tr></thead>
    <tbody><tr v-for="r in data.items" :key="r.batchId" :class="{overdue:r.overdue}"><td>{{ r.lotNo }}<small>{{ r.projectName }}</small></td><td>{{ r.customerName }}<small>{{ r.workshop||'未分配' }}</small></td><td>{{ r.requiredDate||'—' }}<small>{{ r.overdue?'已逾期':r.urgency==='urgent'?'急单':'' }}</small></td><td>{{ r.status }}</td><td>{{ r.itemQty }} / {{ r.itemArea.toFixed(2) }}㎡</td><td>{{ r.labelQty }} · {{ r.printState }}</td><td v-for="j in r.jobs" :key="j.jobId"><div class="bar"><i :style="{width:Math.min(j.pct,100)+'%',background:primary}" /></div>{{ j.qty }} / {{ j.pct }}%</td></tr></tbody></table><p v-if="!data.items.length">暂无未完工批次</p></div></div>
  </BoardShell>
</template>
<style scoped>
.strip,.panel{background:#10243d;border:1px solid #284460;border-radius:8px;padding:16px;margin-top:16px;color:#e8f5ff}.strip{display:flex;align-items:center;flex-wrap:wrap;gap:24px}.strip b{font-size:28px;color:#25d4c8;margin-left:8px}input,select,button{background:#162e49;color:#e8f5ff;border:1px solid #345675;border-radius:4px;padding:6px}.heading{display:flex;justify-content:space-between}.scroll{overflow:auto}table{border-collapse:collapse;white-space:nowrap;width:100%}th,td{border-bottom:1px solid #284460;padding:12px;text-align:left}th,small{color:#93b2ce}small{display:block}.overdue{background:#58332766}.bar{width:66px;height:6px;background:#284460;border-radius:4px;overflow:hidden}.bar i{display:block;height:100%}
</style>
