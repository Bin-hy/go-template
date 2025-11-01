<script setup lang="ts">
import { ref } from 'vue'
import Input from '../ui/Input.vue'
import Button from '../ui/Button.vue'

const props = defineProps<{ bucket: string }>()
const emit = defineEmits<{
  (e: 'update:bucket', v: string): void
  (e: 'upload', payload: { bucket: string; file: File }): void
}>()

const file = ref<File | null>(null)

function onFileChange(e: Event) {
  const input = e.target as HTMLInputElement
  file.value = input.files?.[0] ?? null
}

function doUpload() {
  if (!file.value) {
    alert('请选择文件')
    return
  }
  emit('upload', { bucket: props.bucket, file: file.value })
}
</script>

<template>
  <div class="space-y-3">
    <div class="grid gap-2">
      <label class="text-sm text-gray-600">Bucket</label>
      <Input v-model="(props.bucket as any)" @update:modelValue="(v) => emit('update:bucket', String(v))" placeholder="bucket 名称" />
    </div>
    <div class="grid gap-2">
      <label class="text-sm text-gray-600">选择文件</label>
      <input type="file" @change="onFileChange" class="block w-full text-sm" />
    </div>
    <div>
      <Button @click="doUpload">上传</Button>
    </div>
  </div>
</template>