<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { message } from 'ant-design-vue'
import { ReloadOutlined, SettingOutlined } from '@ant-design/icons-vue'
import { batches, type Batch } from '@/api/demo/batches'
import { PROCESS_STEPS } from '@/theme'
import { useAuthStore } from '@/stores/auth'
import PageCard from '@/components/PageCard.vue'
import FilterCard from '@/components/FilterCard.vue'
import TableToolbar from '@/components/TableToolbar.vue'
import CodeText from '@/components/CodeText.vue'
import StatusTag from '@/components/StatusTag.vue'
import ProcessTimeline from '@/components/ProcessTimeline.vue'
const auth = useAuthStore()
const form = reactive({ keyword: '', project: '', status: '', date: [] as string[], material: '', color: '' })
const query = reactive({ ...form })
const selected = ref<Batch>()
const checked = ref<string[]>([])
const tab = ref('基本信息')
const loading = ref(false)
const page = ref(1)
const size = ref(20)
const visible = computed(() =>
  batches.filter(
    (b) =>
      (auth.user?.isAdmin || b.factoryId === 2) &&
      (!query.keyword || b.id.toLowerCase().includes(query.keyword.toLowerCase())) &&
      (!query.project || b.project.includes(query.project)) &&
      (!query.status || b.status === query.status) &&
      (!query.material || b.thickness.includes(query.material)) &&
      (!query.color || b.color.includes(query.color)) &&
      (!query.date?.length || (b.due >= query.date[0]! && b.due <= query.date[1]!)),
  ),
)
const totals = computed(() => ({
  quantity: visible.value.reduce((n, b) => n + b.quantity, 0),
  area: visible.value.reduce((n, b) => n + b.area, 0).toFixed(2),
  price: visible.value.reduce((n, b) => n + b.priceArea, 0).toFixed(2),
}))
const columns = [
  { title: '批次编号', dataIndex: 'id', width: 155 },
  { title: '工程项目', dataIndex: 'project', width: 150 },
  { title: '规格 / 材质', dataIndex: 'material', width: 110 },
  { title: '件数', dataIndex: 'quantity', width: 65, align: 'right' as const },
  { title: '展开面积', dataIndex: 'area', width: 85, align: 'right' as const },
  { title: '当前工序', dataIndex: 'current', width: 105 },
  { title: '生产进度', dataIndex: 'progress', width: 110 },
  { title: '交付日期', dataIndex: 'due', width: 110 },
  { title: '状态', dataIndex: 'status', width: 95 },
  { title: '操作', key: 'action', width: 70, fixed: 'right' as const },
]
function search() {
  Object.assign(query, form)
  page.value = 1
  loading.value = true
  setTimeout(() => (loading.value = false), 250)
}
function reset() {
  Object.assign(form, { keyword: '', project: '', status: '', date: [], material: '', color: '' })
  search()
}
function open(b: Batch) {
  selected.value = b
  tab.value = '基本信息'
}
const notify = () => message.info('演示模式：此操作暂不保存')
</script>
<template>
  <div class="batches">
    <div class="heading">
      <div>
        <h1>生产批次查询</h1>
        <span>查询排产与工序执行进度，追溯构件全生命周期</span>
      </div>
      <StatusTag tone="processing" :label="`共 ${visible.length} 个批次`" />
    </div>
    <FilterCard @search="search" @reset="reset">
      <template #default="{ expanded }">
        <label>批次号<a-input v-model:value="form.keyword" placeholder="PC-20241018-01" allow-clear /> </label>
        <label>工程项目<a-input v-model:value="form.project" placeholder="输入项目名称" allow-clear /> </label>
        <label
          >生产状态<a-select
            v-model:value="form.status"
            allow-clear
            placeholder="全部状态"
            :options="['生产中', '已完工', '加急逾期', '待排产'].map((x) => ({ label: x, value: x }))"
          />
        </label>
        <label>交付日期<a-range-picker v-model:value="form.date" value-format="YYYY-MM-DD" /> </label>
        <template v-if="expanded">
          <label>材质厚度<a-input v-model:value="form.material" placeholder="如 3.0" /> </label>
          <label>颜色色号<a-input v-model:value="form.color" placeholder="如 RAL 7016" /> </label>
        </template>
      </template>
    </FilterCard>
    <PageCard>
      <TableToolbar>
        <template #left>
          <a-button type="primary" @click="notify">＋ 新建生产批次</a-button>
          <a-button :disabled="!checked.length" @click="notify">批量审核</a-button>
          <a-button @click="notify">导入清单</a-button>
          <a-button @click="notify">导出 ▾</a-button>
          <StatusTag v-if="query.status" tone="processing" :label="query.status" />
        </template>
        <template #right>
          <a-button aria-label="列设置" @click="notify">
            <SettingOutlined />
          </a-button>
          <a-button @click="size = size === 20 ? 50 : 20">密度</a-button>
          <a-button aria-label="刷新" @click="search">
            <ReloadOutlined />
          </a-button>
        </template>
      </TableToolbar>
      <a-skeleton v-if="loading" active :paragraph="{ rows: 7 }" />
      <a-table
        v-else
        row-key="id"
        size="middle"
        :columns="columns"
        :data-source="visible"
        :scroll="{ x: 1150 }"
        :row-selection="{ selectedRowKeys: checked, onChange: (keys: string[]) => (checked = keys) }"
        :pagination="{
          current: page,
          pageSize: size,
          total: visible.length,
          showSizeChanger: true,
          pageSizeOptions: ['20', '50', '100'],
          showQuickJumper: true,
          showTotal: (n: number) => `共 ${n} 条`,
        }"
        @change="
          (p: any) => {
            page = p.current ?? 1
            size = p.pageSize ?? 20
          }
        "
      >
        <template #emptyText>
          <a-empty description="暂无匹配的生产批次数据">
            <a-button type="primary" @click="reset">重置查询条件</a-button>
          </a-empty>
        </template>
        <template #expandedRowRender="{ record }">
          <div class="sub-title">构件展开明细</div>
          <a-table
            :columns="[
              { title: '构件编码', dataIndex: 'code' },
              { title: '规格尺寸', dataIndex: 'spec' },
              { title: '数量', dataIndex: 'quantity' },
            ]"
            :data-source="record.components"
            row-key="code"
            size="small"
            :pagination="false"
          />
        </template>
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'id'">
            <a @click="open(record)">
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
          <template v-else-if="column.dataIndex === 'current'">
            <StatusTag tone="processing" :label="PROCESS_STEPS[record.current] ?? '—'" />
          </template>
          <template v-else-if="column.dataIndex === 'progress'">
            <a-progress :percent="record.progress" size="small" />
          </template>
          <template v-else-if="column.dataIndex === 'status'">
            <StatusTag :tone="record.tone" :label="record.status" />
          </template>
          <template v-else-if="column.key === 'action'">
            <a @click="open(record)">查看</a>
          </template>
        </template>
      </a-table>
      <div class="summary">
        <b>当前页数据汇总合计:</b>
        <span>当前筛选: {{ visible.length }} 个批次</span>
        <span
          >总件数: <b>{{ totals.quantity }}</b> 件</span
        >
        <span
          >总展开面积: <b>{{ totals.area }}</b> m²</span
        >
        <span
          >总计价面积: <b>{{ totals.price }}</b> m²</span
        >
        <span
          >生产中 {{ visible.filter((x) => x.status === '生产中').length }} · 已完工
          {{ visible.filter((x) => x.status === '已完工').length }} · 加急逾期 {{ visible.filter((x) => x.urgent).length }}</span
        >
        <span>数据更新: 10:48</span>
      </div>
    </PageCard>
    <a-drawer :open="!!selected" :width="720" placement="right" :closable="true" @close="selected = undefined">
      <template #title>
        <span>批次详情: {{ selected?.id }}</span>
        <StatusTag v-if="selected" :tone="selected.tone" :label="selected.status" />
      </template>
      <template v-if="selected">
        <a-tabs v-model:active-key="tab">
          <a-tab-pane key="基本信息" tab="基本信息" />
          <a-tab-pane key="11道工序全景" tab="11道工序全景 ●" />
          <a-tab-pane key="一物一码构件清单" :tab="`一物一码构件清单(${selected.quantity})`" />
          <a-tab-pane key="质检记录" tab="质检记录 (QC)" />
          <a-tab-pane key="发货物流" tab="发货物流" />
        </a-tabs>
        <div v-if="tab === '基本信息'" class="drawer-body">
          <div class="barcode">
            <div>
              <small>车间 MES 工单追溯条码 / UID:</small>
              <div class="uid">{{ selected.uid }}</div>
              <small>派工时间: {{ selected.start }} 09:30 · 责任班组: {{ selected.team }}</small>
            </div>
            <div class="qr">▦<small>扫码报工</small></div>
          </div>
          <h3>批次工程要素信息</h3>
          <a-descriptions bordered :column="2" size="small">
            <a-descriptions-item label="所属项目">{{ selected.project }}</a-descriptions-item>
            <a-descriptions-item label="安装标段">{{ selected.section }}</a-descriptions-item>
            <a-descriptions-item label="材质厚度">{{ selected.material }} · {{ selected.thickness }}</a-descriptions-item>
            <a-descriptions-item label="涂层体系">氟碳三涂二烤</a-descriptions-item>
            <a-descriptions-item label="颜色色号">{{ selected.color }}</a-descriptions-item>
            <a-descriptions-item label="排产件数">{{ selected.quantity }} 件</a-descriptions-item>
            <a-descriptions-item label="展开面积">{{ selected.area }} m²</a-descriptions-item>
            <a-descriptions-item label="计价面积">{{ selected.priceArea }} m²</a-descriptions-item>
            <a-descriptions-item label="计划开工">{{ selected.start }}</a-descriptions-item>
            <a-descriptions-item label="最迟交付">
              <span :class="{ overdue: selected.urgent }">{{ selected.due }}{{ selected.urgent ? ' (逾期风险)' : '' }}</span>
            </a-descriptions-item>
          </a-descriptions>
          <h3>
            11 道工序作业流转进度 <span class="muted">综合完工率 {{ selected.progress }}%</span>
          </h3>
          <ProcessTimeline :current="selected.current" :quantity="selected.quantity" />
          <h3>最近生产异常与调度备注</h3>
          <a-alert show-icon type="warning" message="工艺提醒：折弯成型后需复核板面尺寸与角度，误差控制在 ±1.5mm 内。" />
        </div>
        <div v-else-if="tab === '11道工序全景'" class="drawer-body">
          <ProcessTimeline :current="selected.current" :quantity="selected.quantity" />
        </div>
        <div v-else-if="tab === '一物一码构件清单'" class="drawer-body">
          <a-list bordered :data-source="selected.components">
            <template #renderItem="{ item }">
              <a-list-item> <CodeText :value="item.code" /> · {{ item.spec }} · × {{ item.quantity }} 件 </a-list-item>
            </template>
          </a-list>
        </div>
        <div v-else class="drawer-body">
          <a-list
            bordered
            :data-source="
              tab === '质检记录' ? ['钣金质检：待复核', '喷涂膜厚：合格', '颜色色差：待检'] : ['发货计划：待装车', '装架：待扫码交接']
            "
          >
            <template #renderItem="{ item }">
              <a-list-item>{{ item }}</a-list-item>
            </template>
          </a-list>
        </div>
        <div class="drawer-footer">
          <span>操作员: 演示操作员</span>
          <div>
            <a-button @click="notify">打印批次流转卡</a-button>
            <a-button @click="notify">生成件标签</a-button>
            <a-button type="primary" @click="notify">去工位报工</a-button>
          </div>
        </div>
      </template>
    </a-drawer>
  </div>
</template>
<style scoped>
.batches {
  display: grid;
  gap: 16px;
}
.heading {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.heading h1 {
  margin: 0 0 4px;
  font-size: 20px;
}
.heading span,
.muted {
  color: var(--gz-muted);
  font-size: 12px;
}
label {
  display: grid;
  gap: 6px;
  color: var(--gz-secondary);
  font-size: 12px;
  min-width: 160px;
}
.sub-title {
  font-weight: 600;
  margin-bottom: 8px;
}
.summary {
  background: var(--gz-bg);
  padding: 12px;
  display: flex;
  align-items: center;
  gap: 18px;
  flex-wrap: wrap;
  font-size: 12px;
}
.summary b {
  color: var(--gz-primary);
}
.drawer-body {
  padding: 8px 0 80px;
}
.drawer-body h3 {
  font-size: 14px;
  margin: 24px 0 12px;
}
.barcode {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  background: var(--gz-bg);
  border: 1px solid var(--gz-border);
  border-radius: var(--gz-radius);
  gap: 12px;
}
.barcode small {
  color: var(--gz-muted);
}
.uid {
  font:
    600 17px 'JetBrains Mono',
    Consolas,
    monospace;
  margin: 8px 0;
  word-break: break-all;
}
.qr {
  display: grid;
  text-align: center;
  font-size: 38px;
  background: var(--gz-card);
  padding: 4px 12px;
}
.qr small {
  font-size: 11px;
}
.overdue {
  color: var(--gz-error);
}
.drawer-footer {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  padding: 12px 20px;
  background: var(--gz-card);
  border-top: 1px solid var(--gz-border);
  font-size: 12px;
  color: var(--gz-secondary);
}
.drawer-footer > div {
  display: flex;
  gap: 8px;
}
@media (max-width: 768px) {
  .drawer-footer {
    flex-wrap: wrap;
  }
}
</style>
