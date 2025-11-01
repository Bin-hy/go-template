<script setup lang="ts">
import PageShell from './components/layout/PageShell.vue'
</script>

<template>
  <PageShell>
    <RouterView />
  </PageShell>
  
</template>

<style scoped>
/* 页面外层布局已由 PageShell 统一提供，这里无需额外样式 */
</style>

