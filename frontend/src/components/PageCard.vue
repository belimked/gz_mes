<template>
  <section class="card">
    <header v-if="title || $slots.extra || $slots.icon || $slots.subtitle" class="head">
      <div class="heading">
        <span v-if="$slots.icon" class="heading-icon"><slot name="icon" /></span>
        <div>
          <div class="heading-title">
            <strong>{{ title }}</strong
            ><slot name="subtitle" />
          </div>
        </div>
      </div>
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
.heading,
.heading-title {
  display: flex;
  align-items: center;
  gap: 8px;
}
.heading-icon {
  display: inline-flex;
  color: var(--gz-primary);
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
