<script setup lang="ts">
withDefaults(defineProps<{ size?: number; bg?: string; fg?: string }>(), { size: 36, bg: '#1677ff', fg: '#fff' })
</script>

<template>
  <span class="logo" :style="{ width: `${size}px`, height: `${size}px`, fontSize: `${size * 0.55}px`, background: bg, color: fg }">M</span>
</template>

<style scoped>
.logo {
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  font-weight: 800;
}
</style>
