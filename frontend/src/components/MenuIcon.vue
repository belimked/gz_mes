<script setup lang="ts">
import { computed, type Component } from 'vue'
import {
  AccountBookOutlined,
  AppstoreOutlined,
  AuditOutlined,
  BarcodeOutlined,
  BgColorsOutlined,
  BuildOutlined,
  CalculatorOutlined,
  CalendarOutlined,
  CarOutlined,
  ContactsOutlined,
  FileTextOutlined,
  HomeOutlined,
  ProjectOutlined,
  ScanOutlined,
  ScheduleOutlined,
  SettingOutlined,
  ShopOutlined,
  SwapOutlined,
  TeamOutlined,
} from '@ant-design/icons-vue'

const props = defineProps<{ name?: string }>()

const icons: Record<string, Component> = {
  'account-book': AccountBookOutlined,
  build: BuildOutlined,
  audit: AuditOutlined,
  barcode: BarcodeOutlined,
  'bg-colors': BgColorsOutlined,
  calculator: CalculatorOutlined,
  calendar: CalendarOutlined,
  car: CarOutlined,
  contacts: ContactsOutlined,
  'file-text': FileTextOutlined,
  home: HomeOutlined,
  project: ProjectOutlined,
  scan: ScanOutlined,
  schedule: ScheduleOutlined,
  setting: SettingOutlined,
  shop: ShopOutlined,
  swap: SwapOutlined,
  team: TeamOutlined,
  truck: CarOutlined,
}

const icon = computed(() => (props.name && icons[props.name]) || AppstoreOutlined)
</script>

<template>
  <component :is="icon" />
</template>
