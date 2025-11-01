<script setup lang="ts">
import Card from '../components/ui/Card.vue'
import Button from '../components/ui/Button.vue'
import { RouterLink } from 'vue-router'

const features = [
  {
    icon: '📁',
    title: '模型上传',
    desc: '将模型文件上传到服务器，支持基础上传流程。',
    to: '/files',
    action: '进入上传',
  },
  {
    icon: '🧩',
    title: '分块上传',
    desc: '针对大文件的分块上传示例，展示进度与分片信息。',
    to: '/upload-chunks',
    action: '开始分块上传',
  },
  {
    icon: '📦',
    title: '压缩包上传',
    desc: '上传 .zip 或 .7z 压缩包，后端自动解压并保存根/一级目录文件。支持分块上传。',
    to: '/archive-upload',
    action: '上传压缩包',
  },
  {
    icon: '🪣',
    title: 'Bucket 浏览',
    desc: '查看对象存储中的 Buckets 与文件列表，支持下载。',
    to: '/buckets',
    action: '查看 Buckets',
  },
  {
    icon: '📜',
    title: 'Swagger 文档',
    desc: 'API 接口的在线文档与调试入口。',
    to: '/swagger',
    action: '打开文档',
  },
  {
    icon: '🩺',
    title: '健康检查',
    desc: '查看后端健康状态与基本信息。',
    to: '/health',
    action: '查看健康状态',
  },
]
</script>

<template>
  <div class="space-y-6">
    <div>
      <h1 class="text-2xl font-semibold bg-gradient-to-r from-indigo-600 via-blue-600 to-cyan-500 bg-clip-text text-transparent">欢迎使用 AR 装配模型上传系统</h1>
      <p class="mt-1 text-sm text-gray-600">选择下方功能卡片快速开始使用。</p>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
      <Card v-for="f in features" :key="f.title">
        <template #header>
          <div class="flex items-center gap-2">
            <span class="text-2xl">{{ f.icon }}</span>
            <h3 class="text-base font-semibold">{{ f.title }}</h3>
          </div>
        </template>
        <div class="space-y-3">
          <p class="text-sm text-gray-700">{{ f.desc }}</p>
          <div>
            <Button :as="RouterLink" :to="f.to">{{ f.action }}</Button>
          </div>
        </div>
      </Card>
    </div>
  </div>
</template>