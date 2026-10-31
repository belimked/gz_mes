<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { message } from 'ant-design-vue'
import dayjs from 'dayjs'
import { call, client, errorMessage, type Schemas } from '@/api/client'
import CodeText from '@/components/CodeText.vue'
import StatusTag from '@/components/StatusTag.vue'

type History = { time: string; barCode: string; job: string; success: boolean; result: string }
const jobs = ref<Schemas['Job'][]>([])
const jobCode = ref(localStorage.getItem('gz-scan-job') || '')
const barCode = ref('')
const input = ref<{ focus: () => void }>()
const busy = ref(false)
const invalid = ref(false)
const result = ref<Schemas['ScanResult']>()
const history = ref<History[]>([])
let timer: ReturnType<typeof setTimeout> | undefined
const currentJob = computed(() => jobs.value.find((j) => j.code === jobCode.value))
function selectJob(code: string) {
  jobCode.value = code
  localStorage.setItem('gz-scan-job', code)
  void nextTick(() => input.value?.focus())
}
async function scan() {
  const code = barCode.value.trim()
  if (!code || busy.value) return
  if (!jobCode.value) {
    message.warning('请先选择工序')
    input.value?.focus()
    return
  }
  busy.value = true
  let success = false
  let text = '扫描成功'
  try {
    result.value = await call(client.POST('/scans', { body: { barCode: code, jobCode: jobCode.value, device: 'web' } }))
    success = true
    message.success(text)
  } catch (e) {
    text = errorMessage(e)
    message.error(text)
    invalid.value = true
    if (timer) clearTimeout(timer)
    timer = setTimeout(() => {
      invalid.value = false
    }, 300)
  } finally {
    history.value.unshift({
      time: dayjs().format('HH:mm:ss'),
      barCode: code,
      job: currentJob.value?.name ?? jobCode.value,
      success,
      result: text,
    })
    history.value = history.value.slice(0, 10)
    barCode.value = ''
    busy.value = false
    await nextTick()
    input.value?.focus()
  }
}
onMounted(async () => {
  input.value?.focus()
  try {
    jobs.value = await call(client.GET('/jobs'))
    if (!jobs.value.some((j) => j.code === jobCode.value) && jobs.value[0]) selectJob(jobs.value[0].code)
  } catch (e) {
    message.error(errorMessage(e))
  }
})
onUnmounted(() => {
  if (timer) clearTimeout(timer)
})
</script>
<template>
  <div class="scan-page">
    <a-card class="handheld" title="扫码报工">
      <label>当前工序</label>
      <a-select
        :value="jobCode"
        size="large"
        show-search
        option-filter-prop="label"
        style="width: 100%"
        :options="jobs.map((j) => ({ value: j.code, label: `${j.code}-${j.name}` }))"
        @change="selectJob"
      />
      <label class="input-label">扫描条码</label>
      <a-input
        ref="input"
        v-model:value="barCode"
        autofocus
        size="large"
        class="barcode"
        :status="invalid ? 'error' : undefined"
        data-testid="scan-input"
        placeholder="请扫描或输入条码并回车"
        :disabled="busy"
        @press-enter="scan"
      />
      <a-button type="primary" size="large" block :loading="busy" class="submit" @click="scan">确认报工</a-button>
      <template v-if="result">
        <a-divider>本次扫描</a-divider>
        <a-descriptions bordered size="small" :column="2">
          <a-descriptions-item label="批次号">{{ result.label.lotNo }}</a-descriptions-item
          ><a-descriptions-item label="工程">{{ result.label.projectName }}</a-descriptions-item>
          <a-descriptions-item label="编号 / 第几件"
            >{{ result.label.pieceNo }} · {{ result.label.pieceIndex }} / {{ result.label.qty }}</a-descriptions-item
          ><a-descriptions-item label="颜色">{{ result.label.color || '—' }}</a-descriptions-item>
        </a-descriptions>
        <h3>
          批次工序进度 <small>标签 {{ result.progress.labelQty }} / 共 {{ result.progress.totalQty }} 件</small>
        </h3>
        <div v-for="job in result.progress.jobs" :key="job.jobId" class="step" :class="{ active: job.jobCode === jobCode }">
          <div>
            <b>{{ String(job.seq).padStart(2, '0') }}. {{ job.jobName }}</b
            ><span>{{ job.scannedQty }} / {{ result.progress.totalQty }}</span>
          </div>
          <a-progress
            :percent="result.progress.totalQty ? Math.round((job.scannedQty / result.progress.totalQty) * 100) : 0"
            size="small"
          />
        </div>
      </template>
      <a-divider>最近扫描</a-divider>
      <a-empty v-if="!history.length" description="本会话暂无扫描记录" />
      <div v-for="(entry, i) in history" :key="i" class="history">
        <span>{{ entry.time }}</span
        ><CodeText :value="entry.barCode" /><span>{{ entry.job }}</span
        ><StatusTag :tone="entry.success ? 'success' : 'error'" :label="entry.success ? '成功' : entry.result" />
      </div>
    </a-card>
  </div>
</template>
<style scoped>
.scan-page {
  display: flex;
  justify-content: center;
  padding: 20px 0 48px;
}
.handheld {
  width: 100%;
  max-width: 560px;
}
label {
  display: block;
  color: var(--gz-secondary);
  margin: 4px 0 8px;
}
.input-label {
  margin-top: 24px;
}
.barcode {
  font-size: 22px;
  height: 56px;
  letter-spacing: 1px;
}
.submit {
  margin-top: 16px;
}
h3 {
  margin: 24px 0 12px;
}
h3 small {
  font-size: 12px;
  color: var(--gz-muted);
  font-weight: normal;
}
.step {
  padding: 5px 10px;
  border-radius: 6px;
}
.step.active {
  background: var(--gz-bg);
  color: var(--gz-primary);
}
.step > div:first-child {
  display: flex;
  justify-content: space-between;
}
.history {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 0;
  border-bottom: 1px solid var(--gz-border);
  font-size: 12px;
}
</style>
