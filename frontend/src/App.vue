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
  --gz-primary: #1677FF; --gz-success: #52C41A; --gz-warning: #FAAD14; --gz-error: #FF4D4F; --gz-bg: #F5F7FA; --gz-card: #FFFFFF; --gz-border: #F0F0F0;
  --gz-text: #1F1F1F; --gz-muted: #8C8C8C; --gz-secondary: #595959;
  --gz-sider: #001529; --gz-submenu: #000C17; --gz-radius: 8px;
  --gz-shadow: 0 1px 2px rgb(0 0 0 / 3%), 0 1px 6px -1px rgb(0 0 0 / 2%), 0 2px 4px rgb(0 0 0 / 2%);
}
html,
body,
#app {
  height: 100%;
  margin: 0;
}
</style>
