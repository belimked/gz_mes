<template>
  <section class="card">
    <header v-if="title || $slots.extra" class="head">
      <strong>{{ title }}</strong>
      <slot name="extra" />
    </header>
    <slot />
  </section>
</template>
<script setup lang="ts">
defineProps<{ title?: string }>()
</script>
<style scoped>
.card {
  background: var(--gz-card);
  border: 1px solid var(--gz-border);
  border-radius: var(--gz-radius);
  box-shadow: var(--gz-shadow);
  padding: 16px;
  min-width: 0;
}
.head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
  font-size: 16px;
}
</style>
