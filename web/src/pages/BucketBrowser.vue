<template>
  <div class="space-y-6">
    <h1 class="text-2xl font-semibold bg-gradient-to-r from-indigo-600 via-blue-600 to-cyan-500 bg-clip-text text-transparent">Bucket 浏览</h1>

    <div class="flex items-center gap-3">
      <label class="text-sm text-gray-100 bg-blue-600/70 px-2 py-1 rounded">选择 Bucket</label>
      <select
        class="border rounded px-3 py-2 bg-white shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
        v-model="selectedBucket"
        @change="onBucketChange"
      >
        <option v-for="b in buckets" :key="b.name" :value="b.name">
          {{ b.name }}
        </option>
      </select>
      <button class="ml-2 px-3 py-1 border rounded bg-white hover:bg-gray-50 shadow-sm" @click="refreshBuckets" :disabled="loadingBuckets">
        {{ loadingBuckets ? '加载中...' : '刷新 Bucket' }}
      </button>
    </div>

    <div v-if="selectedBucket" class="space-y-2">
      <div class="flex items-center justify-between">
        <h2 class="text-lg font-medium">{{ selectedBucket }} 的文件</h2>
        <button class="px-3 py-1 rounded bg-gradient-to-r from-indigo-600 via-blue-600 to-cyan-500 text-white shadow-sm hover:opacity-90" @click="loadFiles" :disabled="loadingFiles">
          {{ loadingFiles ? '读取中...' : '刷新文件列表' }}
        </button>
      </div>
      <div v-if="files.length === 0 && !loadingFiles" class="text-sm text-gray-600">暂无文件或 Bucket 为空。</div>
      <div class="overflow-x-auto border rounded bg-white/90 shadow-sm">
        <table class="min-w-full text-sm">
          <thead>
            <tr class="bg-gradient-to-r from-gray-100 to-gray-50 text-left">
              <th class="px-3 py-2">ID</th>
              <th class="px-3 py-2">对象名</th>
              <th class="px-3 py-2">原始文件名</th>
              <th class="px-3 py-2">大小</th>
              <th class="px-3 py-2">类型</th>
              <th class="px-3 py-2">创建时间</th>
              <th class="px-3 py-2">下载</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="f in files" :key="f.ID" class="border-t hover:bg-gray-50">
              <td class="px-3 py-2">{{ f.ID }}</td>
              <td class="px-3 py-2">{{ f.ObjectName }}</td>
              <td class="px-3 py-2">{{ f.OriginalName || '-' }}</td>
              <td class="px-3 py-2">{{ formatSize(f.Size) }}</td>
              <td class="px-3 py-2"><span class="inline-block px-2 py-0.5 rounded bg-blue-50 text-blue-700 border border-blue-200">{{ f.MimeType || '-' }}</span></td>
              <td class="px-3 py-2">{{ formatDate(f.CreatedAt) }}</td>
              <td class="px-3 py-2">
                <a class="text-blue-600 hover:underline" :href="f.URL" target="_blank">服务器下载</a>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { listBuckets, listFilesByBucket } from '../services/api'

type BucketItem = { name: string; createdAt?: string }
type FileItem = {
  ID: number
  Bucket: string
  ObjectName: string
  OriginalName?: string | null
  URL: string
  Size?: number | null
  MimeType?: string | null
  CreatedAt: string
}

const buckets = ref<BucketItem[]>([])
const selectedBucket = ref<string>('')
const files = ref<FileItem[]>([])
const loadingBuckets = ref(false)
const loadingFiles = ref(false)

onMounted(async () => {
  await refreshBuckets()
  const first = buckets.value[0]
  if (first) {
    selectedBucket.value = first.name
    await loadFiles()
  }
})

async function refreshBuckets() {
  loadingBuckets.value = true
  try {
    const res = await listBuckets()
    buckets.value = Array.isArray(res.data) ? res.data : []
  } catch (e) {
    console.error('加载 buckets 失败', e)
  } finally {
    loadingBuckets.value = false
  }
}

function onBucketChange() {
  loadFiles()
}

async function loadFiles() {
  if (!selectedBucket.value) return
  loadingFiles.value = true
  try {
    const res = await listFilesByBucket(selectedBucket.value)
    files.value = Array.isArray(res.data) ? res.data : []
  } catch (e) {
    console.error('加载文件列表失败', e)
  } finally {
    loadingFiles.value = false
  }
}

function formatSize(v?: number | null) {
  if (!v || v <= 0) return '-'
  const units = ['B', 'KB', 'MB', 'GB']
  let size = v
  let idx = 0
  while (size >= 1024 && idx < units.length - 1) {
    size /= 1024
    idx++
  }
  return `${size.toFixed(2)} ${units[idx]}`
}

function formatDate(s: string) {
  try {
    const d = new Date(s)
    return d.toLocaleString()
  } catch {
    return s
  }
}
</script>

<style scoped>
</style>