<script setup lang="ts">
import { useRoute } from 'vue-router'

const route = useRoute()
</script>

<template>
  <a-empty :description="`「${route.meta.title ?? ''}」开发中`" />
</template>
