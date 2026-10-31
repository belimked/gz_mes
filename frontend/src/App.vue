<script setup lang="ts">
import zhCN from 'ant-design-vue/es/locale/zh_CN'
import dayjs from 'dayjs'
import 'dayjs/locale/zh-cn'
import { theme } from './theme'

dayjs.locale('zh-cn')
</script>

<template>
  <a-config-provider :locale="zhCN" :theme="theme">
    <router-view />
  </a-config-provider>
</template>

<style>
:root {
  --gz-primary: #1677ff;
  --gz-success: #52c41a;
  --gz-warning: #faad14;
  --gz-error: #ff4d4f;
  --gz-bg: #f5f7fa;
  --gz-card: #ffffff;
  --gz-border: #f0f0f0;
  --gz-text: #1f1f1f;
  --gz-muted: #8c8c8c;
  --gz-secondary: #595959;
  --gz-sider: #001529;
  --gz-submenu: #000c17;
  --gz-radius: 8px;
  --gz-shadow: 0 1px 2px rgb(0 0 0 / 3%), 0 1px 6px -1px rgb(0 0 0 / 2%), 0 2px 4px rgb(0 0 0 / 2%);
}
html,
body,
#app {
  height: 100%;
  margin: 0;
}
</style>
