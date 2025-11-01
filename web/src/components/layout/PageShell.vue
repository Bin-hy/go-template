<template>
  <div class="min-h-screen bg-gray-50 text-gray-900">
    <header class="sticky top-0 z-10 bg-white border-b">
      <div class="container mx-auto px-4 h-14 flex items-center justify-between">
        <div class="font-semibold">AR 裝配模型上传系统</div>
        <nav class="flex items-center gap-4 text-sm">
          <RouterLink class="hover:underline" to="/">首页</RouterLink>
          <!-- <RouterLink class="hover:underline" to="/health">健康检查</RouterLink> -->
          <RouterLink class="hover:underline" to="/files">文件上传</RouterLink>
          <RouterLink class="hover:underline" to="/upload-chunks">分块上传示例</RouterLink>
          <RouterLink class="hover:underline" to="/swagger">Swagger 文档</RouterLink>
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