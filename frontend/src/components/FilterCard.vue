<script setup lang="ts">
import { ref } from 'vue'
import PageCard from './PageCard.vue'
const expanded = ref(false)
const emit = defineEmits<{ search: []; reset: [] }>()
</script>
<template>
  <PageCard>
    <div class="filters">
      <slot :expanded="expanded" />
      <div class="actions">
        <a-button type="primary" @click="emit('search')">查询</a-button>
        <a-button @click="emit('reset')">重置</a-button>
        <a-button type="link" @click="expanded = !expanded">{{ expanded ? '收起' : '展开更多' }}</a-button>
      </div>
    </div>
  </PageCard>
</template>
<style scoped>
.filters {
  display: flex;
  flex-wrap: wrap;
  align-items: end;
  gap: 12px;
}
.actions {
  display: flex;
  gap: 8px;
  margin-left: auto;
}
</style>
