<script setup lang="ts">
import PageCard from './PageCard.vue'
defineProps<{ title: string; value: string | number; unit: string; sub: string; icon: string; note?: string; tone?: 'error' }>()
</script>
<template>
  <PageCard>
    <div class="top">
      <span>{{ title }}</span>
      <span class="icon">{{ icon }}</span>
    </div>
    <div class="value" :class="tone">
      {{ value }} <small>{{ unit }}</small>
    </div>
    <div class="bottom">
      <span>{{ sub }}</span>
      <span>{{ note }}</span>
    </div>
  </PageCard>
</template>
<style scoped>
.top,
.bottom {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  color: var(--gz-secondary);
}
.icon {
  background: var(--gz-bg);
  color: var(--gz-primary);
  border-radius: 50%;
  padding: 6px 10px;
}
.value {
  font-size: 28px;
  font-weight: 700;
  line-height: 36px;
  margin: 18px 0;
  font-variant-numeric: tabular-nums;
}
.value.error {
  color: var(--gz-error, #ff4d4f);
}
.value small {
  font-size: 12px;
  font-weight: 400;
  color: var(--gz-muted);
}
.bottom {
  border-top: 1px solid var(--gz-border);
  padding-top: 12px;
  font-size: 12px;
}
</style>
