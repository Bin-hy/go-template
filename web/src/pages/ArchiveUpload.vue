<script setup lang="ts">
import { ref } from 'vue'
import Card from '../components/ui/Card.vue'
import Input from '../components/ui/Input.vue'
import Button from '../components/ui/Button.vue'
import { uploadArchive, uploadArchiveInChunks } from '../services/archive'

const bucket = ref('example')
const fileRef = ref<File | null>(null)
const useChunks = ref(true)
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
  if (!fileRef.value) return alert('请选择压缩包（.zip 或 .7z）')
  if (!bucket.value) return alert('请输入 bucket')

  uploading.value = true
  errorMsg.value = ''
  result.value = null
  progress.value = { percent: 0, loadedBytes: 0, totalBytes: fileRef.value.size, currentChunk: 0, totalChunks: 0 }

  try {
    // 根据勾选决定是否使用分块上传（适合大文件，规避 Nginx 限制）
    if (useChunks.value) {
      const resp = await uploadArchiveInChunks({
        file: fileRef.value,
        bucket: bucket.value,
        chunkSize: Math.max(1, chunkSizeMB.value) * 1024 * 1024,
        onProgress: (p) => {
          progress.value = p
        },
      })
      result.value = resp
    } else {
      // 直接上传（适合中小文件）
      const resp = await uploadArchive(bucket.value, fileRef.value)
      result.value = resp
    }
  } catch (err: any) {
    errorMsg.value = err?.message || String(err)
  } finally {
    uploading.value = false
  }
}

function exampleBucket() {
  bucket.value = 'example'
}
</script>

<template>
  <div class="space-y-6">
    <Card>
      <template #header>
        <h3 class="text-base font-semibold">压缩包上传（支持 .zip / .7z，含分块上传）</h3>
      </template>
      <div class="grid gap-3">
        <div class="grid grid-cols-1 md:grid-cols-3 gap-3 items-center">
          <div>
            <label class="text-sm text-gray-600">Bucket</label>
            <div class="flex gap-2 items-center">
              <Input v-model="bucket" placeholder="bucket 名称" />
              <Button variant="secondary" size="sm" @click="exampleBucket">示例</Button>
            </div>
          </div>
          <div>
            <label class="text-sm text-gray-600">选择压缩包</label>
            <input type="file" accept=".zip,.7z" @change="onFileChange" />
          </div>
          <div class="flex items-center gap-3">
            <label class="text-sm text-gray-600">使用分块上传</label>
            <input type="checkbox" v-model="useChunks" />
          </div>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-3 gap-3 items-center" v-if="useChunks">
          <div>
            <label class="text-sm text-gray-600">分片大小 (MB)</label>
            <Input v-model.number="chunkSizeMB" type="number" min="1" />
          </div>
          <div class="md:col-span-2 text-sm text-gray-600">
            注意：分块上传适合大文件，后端在最后一个分片合并并解压，仅处理根目录及一级目录文件。
          </div>
        </div>

        <div class="flex gap-2">
          <Button :disabled="uploading" @click="startUpload">{{ uploading ? '上传中...' : '开始上传' }}</Button>
        </div>

        <div class="mt-2" v-if="useChunks">
          <div class="text-sm text-gray-600">进度：{{ progress.percent }}% （{{ progress.currentChunk }}/{{ progress.totalChunks }}）</div>
          <div class="h-2 bg-gray-100 rounded">
            <div class="h-2 bg-blue-500 rounded" :style="{ width: progress.percent + '%' }"></div>
          </div>
        </div>

        <div v-if="errorMsg" class="text-sm text-red-600">{{ errorMsg }}</div>

        <div v-if="result" class="mt-4">
          <div class="text-sm text-gray-600">后端返回：</div>
          <pre class="rounded-md border bg-gray-50 p-3 text-sm overflow-auto">{{ result }}</pre>
          <div class="mt-2 text-sm text-gray-600">
            后端会返回 uploaded / skipped 列表，包含已上传文件记录及被跳过的条目。
          </div>
        </div>
      </div>
    </Card>
  </div>
  
</template>