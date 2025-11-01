<script setup lang="ts">
import { ref } from 'vue'
import Card from '../components/ui/Card.vue'
import Input from '../components/ui/Input.vue'
import Button from '../components/ui/Button.vue'
import { uploadLargeFileInChunks } from '../services/chunk-upload'

const bucket = ref('example')
const fileRef = ref<File | null>(null)
const chunkSizeMB = ref(5)
const uploading = ref(false)
const progress = ref({ percent: 0, loadedBytes: 0, totalBytes: 0, currentChunk: 0, totalChunks: 0 })
const result = ref<any>(null)
const errorMsg = ref('')

function onFileChange(e: Event) {
  const input = e.target as HTMLInputElement
  fileRef.value = input.files?.[0] || null
}

async function startUpload() {
  if (!fileRef.value) return alert('请选择文件')
  if (!bucket.value) return alert('请输入 bucket')

  uploading.value = true
  errorMsg.value = ''
  result.value = null

  try {
    const resp = await uploadLargeFileInChunks({
      file: fileRef.value,
      bucket: bucket.value,
      chunkSize: Math.max(1, chunkSizeMB.value) * 1024 * 1024,
      onProgress: (p) => {
        progress.value = p
      },
    })
    result.value = resp
  } catch (err: any) {
    errorMsg.value = err?.message || String(err)
  } finally {
    uploading.value = false
  }
}
</script>

<template>
  <div class="space-y-6">
    <Card>
      <template #header>
        <h3 class="text-base font-semibold">大文件分块上传</h3>
      </template>
      <div class="grid gap-3">
        <div class="grid grid-cols-1 md:grid-cols-3 gap-3 items-center">
          <div>
            <label class="text-sm text-gray-600">Bucket</label>
            <Input v-model="bucket" placeholder="bucket 名称" />
          </div>
          <div>
            <label class="text-sm text-gray-600">选择文件</label>
            <input type="file" @change="onFileChange" />
          </div>
          <div>
            <label class="text-sm text-gray-600">分片大小 (MB)</label>
            <Input v-model.number="chunkSizeMB" type="number" min="1" />
          </div>
        </div>
        <div class="flex gap-2">
          <Button :disabled="uploading" @click="startUpload">{{ uploading ? '上传中...' : '开始上传' }}</Button>
        </div>

        <div class="mt-2">
          <div class="text-sm text-gray-600">进度：{{ progress.percent }}% （{{ progress.currentChunk }}/{{ progress.totalChunks }}）</div>
          <div class="h-2 bg-gray-100 rounded">
            <div class="h-2 bg-blue-500 rounded" :style="{ width: progress.percent + '%' }"></div>
          </div>
        </div>

        <div v-if="errorMsg" class="text-sm text-red-600">{{ errorMsg }}</div>

        <div v-if="result" class="mt-4">
          <div class="text-sm text-gray-600">后端返回：</div>
          <pre class="rounded-md border bg-gray-50 p-3 text-sm overflow-auto">{{ result }}</pre>
          <div class="mt-2" v-if="result?.data?.URL || result?.data?.url">
            <a :href="(result?.data?.URL || result?.data?.url)" target="_blank" class="text-blue-600 hover:underline break-all">下载地址</a>
          </div>
        </div>
      </div>
    </Card>
  </div>
  
</template>