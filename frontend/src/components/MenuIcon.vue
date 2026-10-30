<script setup lang="ts">
import { computed, type Component } from 'vue'
import {
  AccountBookOutlined,
  AppstoreOutlined,
  BuildOutlined,
  CarOutlined,
  ScheduleOutlined,
  SettingOutlined,
  ShopOutlined,
} from '@ant-design/icons-vue'

const props = defineProps<{ name?: string }>()

const icons: Record<string, Component> = {
  'account-book': AccountBookOutlined,
  build: BuildOutlined,
  car: CarOutlined,
  schedule: ScheduleOutlined,
  setting: SettingOutlined,
  shop: ShopOutlined,
}

const icon = computed(() => (props.name && icons[props.name]) || AppstoreOutlined)
</script>

<template>
  <component :is="icon" />
</template>
