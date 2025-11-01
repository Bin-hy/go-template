import axios from 'axios'

const baseURL = import.meta.env.VITE_API_BASE || 'http://localhost:8080'

export const api = axios.create({
  baseURL,
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json',
  },
})

// 简单的健康检查
export async function getHealth() {
  const { data } = await api.get('/healthz')
  return data
}

// 文件上传（multipart）
export async function uploadFile(bucket: string, file: File) {
  const form = new FormData()
  form.append('bucket', bucket)
  form.append('file', file)
  const { data } = await api.post('/api/v1/files', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return data
}

// 删除（软删除）
export async function deleteFile(id: string | number) {
  const { data } = await api.delete(`/api/v1/files/${id}`)
  return data
}

// 硬删除
export async function hardDeleteFile(id: string | number) {
  const { data } = await api.delete(`/api/v1/files/${id}/hard-delete`)
  return data
}

// 获取所有 Bucket 列表
export async function listBuckets() {
  const { data } = await api.get('/api/v1/files/buckets')
  return data
}

// 根据 Bucket 获取文件列表
export async function listFilesByBucket(bucket: string) {
  const { data } = await api.get(`/api/v1/files/bucket/${encodeURIComponent(bucket)}`)
  return data
}