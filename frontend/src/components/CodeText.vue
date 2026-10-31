<script setup lang="ts">
import { message } from 'ant-design-vue'
import { CopyOutlined } from '@ant-design/icons-vue'
defineProps<{ value: string }>()
async function copy(value: string) {
  await navigator.clipboard.writeText(value)
  message.success('已复制')
}
</script>
<template>
  <span class="code">
    <slot>{{ value }}</slot>
    <button aria-label="复制编号" @click.stop="copy(value)">
      <CopyOutlined />
    </button>
  </span>
</template>
<style scoped>
.code {
  font-family: 'JetBrains Mono', SFMono-Regular, Consolas, monospace;
  font-variant-numeric: tabular-nums;
  color: var(--gz-primary);
}
.code button {
  border: 0;
  background: none;
  color: var(--gz-muted);
  cursor: pointer;
}
</style>
