<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { message } from 'ant-design-vue'
import dayjs from 'dayjs'
import { call, client, errorMessage, type Schemas } from '@/api/client'
import CodeText from '@/components/CodeText.vue'
import StatusTag from '@/components/StatusTag.vue'

type History = { time: string; barCode: string; job: string; success: boolean; result: string }
const jobs = ref<Schemas['Job'][]>([])
const machines = ref<Schemas['Machine'][]>([])
const machineId = ref<number>()
const jobCode = ref(localStorage.getItem('gz-scan-job') || '')
const barCode = ref('')
const input = ref<{ focus: () => void }>()
const busy = ref(false)
const invalid = ref(false)
const result = ref<Schemas['ScanResult']>()
const history = ref<History[]>([])
let timer: ReturnType<typeof setTimeout> | undefined
const currentJob = computed(() => jobs.value.find((j) => j.code === jobCode.value))
const jobMachines = computed(() => machines.value.filter((m) => m.jobId === currentJob.value?.id))
function selectMachine(id: number | undefined) {
  machineId.value = id
  if (currentJob.value) {
    const key = `gz-scan-machine-${currentJob.value.id}`
    if (id) localStorage.setItem(key, String(id))
    else localStorage.removeItem(key)
  }
}
const qcResult = ref<Schemas['QcResult']>('pass')
const defectJobId = ref<number>()
const defectCategory = ref('')
const defectReason = ref('')
/** Jobs a failed piece can be sent back to: anything before the current QC job. */
const defectJobOptions = computed(() =>
  jobs.value.filter((j) => currentJob.value && j.seq < currentJob.value.seq).map((j) => ({ value: j.id, label: `${j.code}-${j.name}` })),
)
function selectJob(code: string) {
  jobCode.value = code
  const job = jobs.value.find((j) => j.code === code)
  const stored = Number(localStorage.getItem(`gz-scan-machine-${job?.id}`))
  machineId.value = machines.value.some((m) => m.id === stored && m.jobId === job?.id) ? stored : undefined
  localStorage.setItem('gz-scan-job', code)
  qcResult.value = 'pass'
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
  const isQc = currentJob.value?.isQc ?? false
  const failed = isQc && qcResult.value === 'fail'
  try {
    result.value = await call(
      client.POST('/scans', {
        body: {
          barCode: code,
          jobCode: jobCode.value,
          device: 'web',
          machineId: machineId.value,
          ...(isQc ? { result: qcResult.value } : {}),
          ...(failed
            ? {
                defectJobId: defectJobId.value,
                defectCategory: defectCategory.value.trim(),
                defectReason: defectReason.value.trim(),
              }
            : {}),
        },
      }),
    )
    success = true
    text = failed ? '已记录不合格' : isQc ? '合格' : '扫描成功'
    if (failed) message.warning(text)
    else message.success(text)
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
    ;[jobs.value, machines.value] = await Promise.all([call(client.GET('/jobs')), call(client.GET('/machines'))])
    if (!jobs.value.some((j) => j.code === jobCode.value) && jobs.value[0]) selectJob(jobs.value[0].code)
    else selectJob(jobCode.value)
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
      <template v-if="jobMachines.length"
        ><label class="input-label">机台（可选）</label
        ><a-select
          :value="machineId"
          allow-clear
          size="large"
          show-search
          option-filter-prop="label"
          style="width: 100%"
          :options="jobMachines.map((m) => ({ value: m.id, label: `${m.code} ${m.name}` }))"
          data-testid="scan-machine"
          @change="selectMachine"
      /></template>
      <template v-if="currentJob?.isQc">
        <label class="input-label">判定结果</label>
        <a-radio-group v-model:value="qcResult" size="large" button-style="solid" class="verdict" data-testid="qc-result">
          <a-radio-button value="pass" class="pass">合格</a-radio-button>
          <a-radio-button value="fail" class="fail">不合格</a-radio-button>
        </a-radio-group>
        <div v-if="qcResult === 'fail'" class="defect">
          <a-select
            v-model:value="defectJobId"
            allow-clear
            placeholder="责任工序（返回到哪道工序）"
            :options="defectJobOptions"
            style="width: 100%"
            data-testid="defect-job"
          />
          <a-input v-model:value="defectCategory" placeholder="不良类别（如 外观 / 尺寸）" :maxlength="50" data-testid="defect-category" />
          <a-input v-model:value="defectReason" placeholder="不良原因（如 划伤 / 色差）" :maxlength="200" data-testid="defect-reason" />
        </div>
      </template>
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
      <a-button
        :type="currentJob?.isQc && qcResult === 'fail' ? 'default' : 'primary'"
        :danger="currentJob?.isQc && qcResult === 'fail'"
        size="large"
        block
        :loading="busy"
        class="submit"
        @click="scan"
        >{{ currentJob?.isQc ? (qcResult === 'fail' ? '记录不合格' : '确认合格') : '确认报工' }}</a-button
      >
      <template v-if="result">
        <a-divider>本次扫描</a-divider>
        <a-alert
          v-if="result.qc"
          :type="result.qc.result === 'pass' ? 'success' : 'error'"
          show-icon
          class="qc-alert"
          data-testid="qc-alert"
          :message="
            result.qc.result === 'pass'
              ? `${result.qc.jobName}：合格`
              : `${result.qc.jobName}：不合格，标签停留在 ${result.label.lastJobName ?? '未开工'}`
          "
          :description="
            result.qc.result === 'fail'
              ? [result.qc.defectJobName && `责任工序 ${result.qc.defectJobName}`, result.qc.defectCategory, result.qc.defectReason]
                  .filter(Boolean)
                  .join(' · ') || undefined
              : undefined
          "
        />
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
.verdict {
  display: flex;
}
.verdict :deep(.ant-radio-button-wrapper) {
  flex: 1;
  text-align: center;
}
.verdict :deep(.fail.ant-radio-button-wrapper-checked) {
  background: var(--ant-color-error, #cf1322);
  border-color: var(--ant-color-error, #cf1322);
}
.defect {
  display: grid;
  gap: 8px;
  margin-top: 8px;
}
.qc-alert {
  margin-bottom: 12px;
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
