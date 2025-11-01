<template>
  <nav class="flex items-center justify-between px-4 py-3 bg-white border-b">
    <div class="font-semibold">AR 裝配模型上传系统</div>
    <ul class="flex gap-4 list-none m-0 p-0 text-sm">
      <li><RouterLink class="hover:underline" to="/">首页</RouterLink></li>
      <!-- <li><RouterLink class="hover:underline" to="/health">健康检查</RouterLink></li> -->
      <li><RouterLink class="hover:underline" to="/files">文件示例</RouterLink></li>
      <li><RouterLink class="hover:underline" to="/swagger">Swagger 文档</RouterLink></li>
    </ul>
  </nav>
</template>

<script setup lang="ts">
import { RouterLink } from 'vue-router'
</script>