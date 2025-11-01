<template>
  <div class="min-h-screen bg-gradient-to-br from-slate-50 via-white to-slate-100 text-gray-900">
    <header class="sticky top-0 z-10 border-b bg-gradient-to-r from-indigo-600 via-blue-600 to-cyan-500 text-white shadow-md">
      <div class="container mx-auto px-4 h-14 flex items-center justify-between">
        <div class="font-semibold tracking-wide">AR 裝配模型上传系统</div>
        <nav class="flex items-center gap-2 text-sm">
          <RouterLink class="px-3 py-1 rounded-md hover:bg-white/10" to="/">首页</RouterLink>
          <RouterLink class="px-3 py-1 rounded-md hover:bg-white/10" to="/files">模型上传</RouterLink>
          <RouterLink class="px-3 py-1 rounded-md hover:bg-white/10" to="/upload-chunks">模型分块上传</RouterLink>
          <RouterLink class="px-3 py-1 rounded-md hover:bg-white/10" to="/buckets">Bucket 列表</RouterLink>
          <RouterLink class="px-3 py-1 rounded-md hover:bg-white/10" to="/swagger">Swagger 文档</RouterLink>
        </nav>
      </div>
    </header>
    <main class="container mx-auto px-4 py-6">
      <slot />
    </main>
  </div>
  
</template>

<script setup lang="ts">
import { RouterLink } from 'vue-router'
</script>