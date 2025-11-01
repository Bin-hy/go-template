<script setup lang="ts">
import { ref } from 'vue'
import { uploadFile, deleteFile, hardDeleteFile } from '../services/api'
import { useAsync } from '../composables/useAsync'
import Card from '../components/ui/Card.vue'
import Input from '../components/ui/Input.vue'
import Button from '../components/ui/Button.vue'
import UploadForm from '../components/common/UploadForm.vue'

const bucket = ref('example')
const uploadResp = ref<any>(null)
const modelUrl = ref<string>('')
const delId = ref('')
const delResp = ref<any>(null)
const hardDelResp = ref<any>(null)

const { loading: uploading, run: runUpload } = useAsync(uploadFile)
const { loading: softing, run: runSoftDelete } = useAsync(deleteFile)
const { loading: harding, run: runHardDelete } = useAsync(hardDeleteFile)

async function handleUpload(payload: { bucket: string; file: File }) {
  uploadResp.value = await runUpload(payload.bucket, payload.file)
  // 根据后端返回结构提取模型地址（URL 字段）
  const url = uploadResp.value?.data?.URL || uploadResp.value?.data?.url
  modelUrl.value = typeof url === 'string' ? url : ''
}

async function doDelete() {
  if (!delId.value) return alert('请输入文件记录ID')
  delResp.value = await runSoftDelete(delId.value)
}

async function doHardDelete() {
  if (!delId.value) return alert('请输入文件记录ID')
  hardDelResp.value = await runHardDelete(delId.value)
}
</script>

<template>
  <div class="space-y-6">
    <Card>
      <template #header>
        <h3 class="text-base font-semibold">上传文件</h3>
      </template>
      <UploadForm :bucket="bucket" @update:bucket="(v) => (bucket = v)" @upload="handleUpload" />
      <div class="mt-4">
        <Button variant="secondary" size="sm" :disabled="uploading">{{ uploading ? '上传中...' : '重新上传' }}</Button>
      </div>
      <pre v-if="uploadResp" class="mt-4 rounded-md border bg-gray-50 p-3 text-sm overflow-auto">{{ uploadResp }}</pre>
      <div v-if="modelUrl" class="mt-4 grid gap-2">
        <div class="text-sm text-gray-600">模型地址（基于 bucket 手动区分）：</div>
        <a :href="modelUrl" target="_blank" class="text-blue-600 hover:underline break-all">{{ modelUrl }}</a>

      </div>
    </Card>

    <Card>
      <template #header>
        <h3 class="text-base font-semibold">删除文件</h3>
      </template>
      <div class="grid gap-3">
        <Input v-model="delId" placeholder="文件记录ID" />
        <div class="flex gap-2">
          <Button @click="doDelete" :disabled="softing">{{ softing ? '软删除中...' : '软删除' }}</Button>
          <Button variant="destructive" @click="doHardDelete" :disabled="harding">{{ harding ? '硬删除中...' : '硬删除' }}</Button>
        </div>
        <pre v-if="delResp" class="rounded-md border bg-gray-50 p-3 text-sm overflow-auto">{{ delResp }}</pre>
        <pre v-if="hardDelResp" class="rounded-md border bg-gray-50 p-3 text-sm overflow-auto">{{ hardDelResp }}</pre>
      </div>
    </Card>
  </div>
</template>