<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { message } from 'ant-design-vue'
import dayjs, { type Dayjs } from 'dayjs'
import { call, client, errorMessage, fileRequest, type Schemas } from '@/api/client'
import { downloadCsv, exportFilename, fetchAllPages } from '@/utils/export'
import PageCard from '@/components/PageCard.vue'

type Batch = Schemas['DrawingBatch']
type Drawing = Schemas['Drawing']
type Kind = Drawing['kind']
const kinds: { key: Kind; label: string }[] = [
  { key: 'unfold', label: '展开加工图' },
  { key: 'forming', label: '成型加工图' },
  { key: 'nesting', label: '套料图' },
]
const dates = ref<[Dayjs, Dayjs] | undefined>()
const state = ref<'none' | 'partial' | 'complete' | undefined>()
const keyword = ref('')
const applied = reactive<{ beginDate?: string; endDate?: string; uploadState?: 'none' | 'partial' | 'complete'; keyword?: string }>({})
const rows = ref<Batch[]>([])
const pager = reactive({ current: 1, pageSize: 20, total: 0 })
const loading = ref(false)
const exporting = ref(false)
const selected = ref<Batch>()
const drawings = ref<Drawing[]>([])
const tab = ref<Kind>('unfold')
const drawingNo = ref('')
const remark = ref('')
const uploading = ref(false)
const previewURL = ref('')
const previewType = ref('')
const previewOpen = ref(false)
const visibleDrawings = computed(() => drawings.value.filter((d) => d.kind === tab.value))
const columns = [
  { title: '批次号', dataIndex: 'lotNo' },
  { title: '工程名称', dataIndex: 'projectName' },
  { title: '客户', dataIndex: 'customerName' },
  { title: '展开图数', dataIndex: 'unfoldCount' },
  { title: '成型图数', dataIndex: 'formingCount' },
  { title: '套料图数', dataIndex: 'nestingCount' },
  { title: '上传状态', key: 'state' },
  { title: '最后上传时间', key: 'lastUploadedAt' },
  { title: '操作', key: 'actions' },
]
function stateOf(b: Batch) {
  if (b.unfoldCount && b.formingCount) return 'complete'
  if (b.unfoldCount || b.formingCount || b.nestingCount) return 'partial'
  return 'none'
}
const stateName = { none: '未上传', partial: '部分上传', complete: '已齐全' }
const stateColor = { none: 'default', partial: 'orange', complete: 'green' }
function fetchPage(page: number, size: number) {
  return call(client.GET('/drawing-batches', { params: { query: { ...applied, page, size } } }))
}
async function load() {
  loading.value = true
  try {
    const data = await fetchPage(pager.current, pager.pageSize)
    rows.value = data.items
    pager.total = data.total
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    loading.value = false
  }
}
function search() {
  applied.beginDate = dates.value?.[0]?.format('YYYY-MM-DD')
  applied.endDate = dates.value?.[1]?.format('YYYY-MM-DD')
  applied.uploadState = state.value
  applied.keyword = keyword.value.trim() || undefined
  pager.current = 1
  void load()
}
async function open(b: Batch) {
  selected.value = b
  tab.value = 'unfold'
  await refresh()
}
async function refresh() {
  if (!selected.value) return
  try {
    drawings.value = await call(client.GET('/batches/{id}/drawings', { params: { path: { id: selected.value.id } } }))
  } catch (e) {
    message.error(errorMessage(e))
  }
}
async function upload({ file, onSuccess, onError }: { file: File; onSuccess?: (data: unknown) => void; onError?: (error: Error) => void }) {
  if (!selected.value) return
  uploading.value = true
  try {
    const form = new FormData()
    form.append('file', file)
    form.append('kind', tab.value)
    form.append('drawingNo', drawingNo.value)
    form.append('remark', remark.value)
    const response = await fileRequest(`/batches/${selected.value.id}/drawings`, { method: 'POST', body: form })
    onSuccess?.(await response.json())
    drawingNo.value = ''
    remark.value = ''
    message.success('上传成功')
    await Promise.all([refresh(), load()])
  } catch (e) {
    message.error(errorMessage(e))
    onError?.(e instanceof Error ? e : new Error(String(e)))
  } finally {
    uploading.value = false
  }
}
async function download(d: Drawing) {
  try {
    const response = await fileRequest(`/drawings/${d.id}/file?download=1`)
    const u = URL.createObjectURL(await response.blob())
    const a = document.createElement('a')
    a.href = u
    a.download = d.originalName
    a.click()
    setTimeout(() => URL.revokeObjectURL(u), 1000)
  } catch (e) {
    message.error(errorMessage(e))
  }
}
function closePreview() {
  previewOpen.value = false
  if (previewURL.value) URL.revokeObjectURL(previewURL.value)
  previewURL.value = ''
}
onBeforeUnmount(closePreview)
async function preview(d: Drawing) {
  if (!d.contentType.startsWith('image/') && d.contentType !== 'application/pdf') {
    message.info('CAD 文件请下载后用 CAD 软件打开')
    return
  }
  try {
    closePreview()
    const response = await fileRequest(`/drawings/${d.id}/file`)
    previewType.value = d.contentType
    previewURL.value = URL.createObjectURL(await response.blob())
    previewOpen.value = true
  } catch (e) {
    message.error(errorMessage(e))
  }
}
async function remove(d: Drawing) {
  try {
    await call(client.DELETE('/drawings/{id}', { params: { path: { id: d.id } } }))
    message.success('已删除')
    await Promise.all([refresh(), load()])
  } catch (e) {
    message.error(errorMessage(e))
  }
}
async function exportRows() {
  exporting.value = true
  try {
    const all = await fetchAllPages(fetchPage)
    downloadCsv(
      exportFilename('图纸管理'),
      [
        { title: '批次号', value: (b) => b.lotNo },
        { title: '工程名称', value: (b) => b.projectName },
        { title: '客户', value: (b) => b.customerName },
        { title: '展开图数', value: (b) => b.unfoldCount },
        { title: '成型图数', value: (b) => b.formingCount },
        { title: '套料图数', value: (b) => b.nestingCount },
        { title: '上传状态', value: (b) => stateName[stateOf(b)] },
        { title: '最后上传时间', value: (b) => b.lastUploadedAt ?? '' },
      ],
      all,
    )
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    exporting.value = false
  }
}
onMounted(load)
</script>

<template>
  <PageCard title="图纸查询">
    <a-space wrap style="margin-bottom: 16px">
      <a-range-picker v-model:value="dates" aria-label="批次创建日期" />
      <a-select
        v-model:value="state"
        allow-clear
        placeholder="上传状态"
        style="width: 150px"
        :options="[
          { value: 'none', label: '未上传' },
          { value: 'partial', label: '部分上传' },
          { value: 'complete', label: '已齐全' },
        ]"
      />
      <a-input v-model:value="keyword" placeholder="批次号 / 工程 / 合同号" style="width: 230px" @press-enter="search" />
      <a-button type="primary" @click="search">查询</a-button>
      <a-button :loading="exporting" @click="exportRows">导出 CSV</a-button>
    </a-space>
    <a-table
      :columns="columns"
      :data-source="rows"
      :loading="loading"
      :row-key="(r: Batch) => r.id"
      :pagination="{ current: pager.current, pageSize: pager.pageSize, total: pager.total, showSizeChanger: true }"
      @change="
        (p: { current?: number; pageSize?: number }) => {
          pager.current = p.current ?? 1
          pager.pageSize = p.pageSize ?? 20
          load()
        }
      "
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'state'"
          ><a-tag :color="stateColor[stateOf(record)]">{{ stateName[stateOf(record)] }}</a-tag></template
        >
        <template v-else-if="column.key === 'lastUploadedAt'">{{
          record.lastUploadedAt ? dayjs(record.lastUploadedAt).format('YYYY-MM-DD HH:mm') : '—'
        }}</template>
        <template v-else-if="column.key === 'actions'"><a-button type="link" @click="open(record)">管理</a-button></template>
      </template>
    </a-table>
  </PageCard>
  <a-drawer :open="!!selected" :title="`图纸管理 · ${selected?.lotNo ?? ''}`" :width="760" @close="selected = undefined">
    <a-tabs v-model:active-key="tab">
      <a-tab-pane v-for="kind in kinds" :key="kind.key" :tab="`${kind.label} (${drawings.filter((d) => d.kind === kind.key).length})`" />
    </a-tabs>
    <a-space wrap style="margin-bottom: 16px">
      <a-input v-model:value="drawingNo" placeholder="图号 / 方案号" style="width: 170px" />
      <a-input v-model:value="remark" placeholder="备注" style="width: 170px" />
      <a-upload :show-upload-list="false" :custom-request="upload" accept=".pdf,.dwg,.dxf,.png,.jpg,.jpeg">
        <a-button type="primary" :loading="uploading">上传{{ kinds.find((k) => k.key === tab)?.label }}</a-button>
      </a-upload>
    </a-space>
    <a-list :data-source="visibleDrawings" bordered>
      <template #renderItem="{ item }">
        <a-list-item>
          <template #actions>
            <a-button type="link" @click="preview(item)">预览</a-button>
            <a-button type="link" @click="download(item)">下载</a-button>
            <a-popconfirm title="确定删除图纸？" @confirm="remove(item)"><a-button type="link" danger>删除</a-button></a-popconfirm>
          </template>
          <a-list-item-meta
            :title="item.originalName"
            :description="`图号：${item.drawingNo || '—'} · ${(item.sizeBytes / 1024).toFixed(1)} KB · ${item.uploaderName} · ${dayjs(item.uploadedAt).format('YYYY-MM-DD HH:mm')}${item.remark ? ` · ${item.remark}` : ''}`"
          />
        </a-list-item>
      </template>
    </a-list>
  </a-drawer>
  <a-modal :open="previewOpen" title="图纸预览" width="80%" :footer="null" @cancel="closePreview">
    <img v-if="previewType.startsWith('image/')" :src="previewURL" alt="图纸预览" style="max-width: 100%; max-height: 75vh" />
    <iframe v-else :src="previewURL" title="PDF 预览" style="width: 100%; height: 75vh; border: 0" />
  </a-modal>
</template>
