<script setup lang="ts">
import { computed, ref } from 'vue'
import { message } from 'ant-design-vue'
import { useRouter } from 'vue-router'
import { isDemo } from '@/api/client'
import { resetDemo } from '@/api/demo'
import { kpis, pipeline, activeBatches, alerts } from '@/api/demo/dashboard'
import { PROCESS_STEPS } from '@/theme'
import PageCard from '@/components/PageCard.vue'
import KpiCard from '@/components/KpiCard.vue'
import StatusTag from '@/components/StatusTag.vue'
import ProcessPipeline from '@/components/ProcessPipeline.vue'
import CodeText from '@/components/CodeText.vue'
const router = useRouter()
const tab = ref('全部')
const filtered = computed(() =>
  activeBatches.filter(
    (b) =>
      tab.value === '全部' ||
      (tab.value === '加急'
        ? b.urgent
        : tab.value === '钣金工段'
          ? b.current < 4
          : tab.value === '喷涂工段'
            ? b.current >= 4 && b.current < 8
            : b.current >= 8),
  ),
)
const columns = [
  { title: '批次编号', dataIndex: 'id', width: 175 },
  { title: '工程项目', dataIndex: 'project', width: 180 },
  { title: '规格材质', dataIndex: 'material', width: 120 },
  { title: '件数 / 展开面积', dataIndex: 'quantity', width: 130, align: 'right' as const },
  { title: '当前工序', dataIndex: 'current', width: 115 },
  { title: '责任班组', dataIndex: 'team', width: 100 },
  { title: '完工进度', dataIndex: 'progress', width: 125 },
  { title: '质检状态', dataIndex: 'qc', width: 100 },
  { title: '操作', key: 'action', width: 85 },
]
function reset() {
  resetDemo()
  message.success('演示数据已重置')
}
</script>
<template>
  <div class="home">
    <PageCard class="title-card">
      <div class="title-row">
        <div>
          <h1><span class="dot" />铝单板生产执行看板</h1>
          <StatusTag tone="processing" label="生产中 12 个批次" />
          <StatusTag tone="success" label="达成率 94.2%" />
        </div>
        <div class="buttons">
          <a-button type="primary" @click="message.info('演示模式：扫码报工')">一键扫码报工</a-button>
          <a-button @click="router.push('/scheduling/batches')">新建生产批次</a-button>
          <a-button @click="message.info('演示模式：打印标签')">打印标签 (QR)</a-button>
          <a-button @click="message.info('演示模式：导出日报')">导出日报</a-button>
          <a-button v-if="isDemo" size="small" @click="reset">重置演示数据</a-button>
        </div>
      </div>
    </PageCard>
    <a-row :gutter="[16, 16]">
      <a-col v-for="item in kpis" :key="item.title" :xs="24" :sm="12" :xl="6">
        <KpiCard v-bind="item" />
      </a-col>
    </a-row>
    <PageCard title="铝单板 11 大工序流转全景看板">
      <template #extra>
        <div class="legend">
          <i class="dot-legend" style="background: var(--gz-primary)" /> 在制运转
          <i class="dot-legend" style="background: var(--gz-warning)" /> 工序堆积
          <i class="dot-legend" style="background: var(--gz-success)" /> 质检完成
        </div>
      </template>
      <ProcessPipeline :items="pipeline" />
    </PageCard>
    <a-row :gutter="[16, 16]">
      <a-col :xs="24" :xl="16">
        <PageCard title="在制批次与一物一码跟踪">
          <a-tabs v-model:active-key="tab" size="small">
            <a-tab-pane v-for="name in ['全部', '钣金工段', '喷涂工段', '包装发货', '加急']" :key="name" :tab="name" />
          </a-tabs>
          <a-table :columns="columns" :data-source="filtered" row-key="id" size="middle" :pagination="false" :scroll="{ x: 1050 }">
            <template #bodyCell="{ column, record }">
              <template v-if="column.dataIndex === 'id'">
                <a @click="router.push('/scheduling/batches')">
                  <CodeText :value="record.id" />
                </a>
              </template>
              <template v-else-if="column.dataIndex === 'project'">
                <strong>{{ record.project }}</strong>
                <div class="muted">{{ record.section }}</div>
              </template>
              <template v-else-if="column.dataIndex === 'material'">
                {{ record.thickness }}
                <div class="muted">{{ record.material }}</div>
              </template>
              <template v-else-if="column.dataIndex === 'quantity'">
                {{ record.quantity }} 件
                <div class="muted">{{ record.area }} m²</div>
              </template>
              <template v-else-if="column.dataIndex === 'current'">
                <StatusTag tone="processing" :label="PROCESS_STEPS[record.current] ?? '—'" />
              </template>
              <template v-else-if="column.dataIndex === 'progress'">
                <a-progress :percent="record.progress" size="small" />
              </template>
              <template v-else-if="column.dataIndex === 'qc'">
                <StatusTag :tone="record.qc === '合格' ? 'success' : 'warning'" :label="record.qc" />
              </template>
              <template v-else-if="column.key === 'action'">
                <a @click="router.push('/scheduling/batches')">明细</a>
              </template>
            </template>
          </a-table>
        </PageCard>
      </a-col>
      <a-col :xs="24" :xl="8">
        <PageCard title="异常预警与工单提醒">
          <div v-for="alert in alerts" :key="alert.title" class="alert">
            <StatusTag :tone="alert.tone" :label="alert.title" />
            <p>{{ alert.detail }}</p>
          </div>
          <a-button block @click="message.info('暂无更多预警')">查看全部预警</a-button>
        </PageCard>
        <PageCard class="shipping" title="今日装车发货计划">
          <div v-for="(name, i) in ['演示·滨湖科创中心', '演示·云谷产业园二期', '演示·江畔文化馆']" :key="name" class="shipping-row">
            <strong>{{ name }}</strong>
            <span>{{ 9 + i * 3 }}:30 · {{ 80 + i * 24 }} 件</span>
          </div>
        </PageCard>
      </a-col>
    </a-row>
  </div>
</template>
<style scoped>
.home {
  display: grid;
  gap: 16px;
}
.title-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}
.title-row h1 {
  display: inline-block;
  font-size: 20px;
  margin: 0 14px 0 0;
}
.dot {
  display: inline-block;
  width: 9px;
  height: 9px;
  margin-right: 10px;
  border-radius: 50%;
  background: var(--gz-primary);
}
.buttons {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.legend,
.muted {
  color: var(--gz-muted);
  font-size: 12px;
}
.legend {
  display: flex;
  align-items: center;
  gap: 6px;
}
.dot-legend {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  margin-left: 8px;
}
.dot-legend:first-child {
  margin-left: 0;
}
.alert {
  padding: 12px;
  margin-bottom: 10px;
  background: var(--gz-bg);
  border-radius: var(--gz-radius);
}
.alert p {
  color: var(--gz-secondary);
  margin: 8px 0 0;
  font-size: 12px;
}
.shipping {
  margin-top: 16px;
}
.shipping-row {
  display: flex;
  justify-content: space-between;
  border-bottom: 1px solid var(--gz-border);
  padding: 12px 0;
  gap: 8px;
}
.shipping-row span {
  color: var(--gz-muted);
  white-space: nowrap;
}
</style>
