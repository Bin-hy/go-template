<script setup lang="ts">
// 简单地提供后端 Swagger 文档的跳转或 iframe 嵌入
const swaggerUrl = (import.meta.env.VITE_API_BASE || 'http://localhost:8080') + '/swagger/index.html'
</script>

<template>
  <div class="space-y-4">
    <h2 class="text-lg font-semibold">Swagger 文档</h2>
    <a class="text-blue-600 hover:underline" :href="swaggerUrl" target="_blank">在新窗口打开 Swagger 文档</a>
    <div class="rounded-lg border">
      <iframe :src="swaggerUrl" class="w-full h-[70vh]" />
    </div>
  </div>
</template>