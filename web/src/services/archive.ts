import { api } from './api'

// 直接上传压缩包（zip 或 7z），后端接口：POST /api/v1/files/archive
export async function uploadArchive(bucket: string, file: File) {
  const form = new FormData()
  form.append('bucket', bucket)
  // 后端当前实现使用字段名 `zip` 或检测文件类型，这里统一按 `zip` 字段提交
  form.append('zip', file)
  const { data } = await api.post('/api/v1/files/archive', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
    maxBodyLength: Infinity,
    maxContentLength: Infinity,
  })
  return data
}

// 分块上传初始化：POST /api/v1/files/archive/multipart/init
export async function initArchiveChunkUpload(bucket: string, filename: string, mimeType?: string) {
  const form = new FormData()
  form.append('bucket', bucket)
  form.append('filename', filename)
  if (mimeType) form.append('mime_type', mimeType)
  const { data } = await api.post('/api/v1/files/archive/multipart/init', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return data
}

// 分块上传：POST /api/v1/files/archive/multipart/chunk
export async function uploadArchiveChunk(payload: {
  upload_id: string
  chunk_index: number
  total_chunks: number
  chunk: Blob
  bucket?: string
  filename?: string
}) {
  const form = new FormData()
  form.append('upload_id', payload.upload_id)
  form.append('chunk_index', String(payload.chunk_index))
  form.append('total_chunks', String(payload.total_chunks))
  form.append('chunk', payload.chunk)
  if (payload.bucket) form.append('bucket', payload.bucket)
  if (payload.filename) form.append('filename', payload.filename)
  const { data } = await api.post('/api/v1/files/archive/multipart/chunk', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
    maxBodyLength: Infinity,
    maxContentLength: Infinity,
  })
  return data
}

// 前端分片调度：将 File 切片并按序上传，最后一个分片返回后端处理结果（uploaded/ skipped 列表）
export async function uploadArchiveInChunks(opts: {
  file: File
  bucket: string
  chunkSize?: number // bytes, default ~5MB
  onProgress?: (p: { loadedBytes: number; totalBytes: number; percent: number; currentChunk: number; totalChunks: number }) => void
  onChunkUploaded?: (i: number) => void
}) {
  const { file, bucket } = opts
  const chunkSize = opts.chunkSize ?? 5 * 1024 * 1024
  const totalBytes = file.size
  const totalChunks = Math.ceil(totalBytes / chunkSize)

  // 1) 初始化会话
  const initResp = await initArchiveChunkUpload(bucket, file.name, file.type)
  const uploadId = initResp?.data?.upload_id || initResp?.data?.uploadId || initResp?.upload_id
  if (!uploadId) throw new Error('初始化分块上传失败：缺少 upload_id')

  let loadedBytes = 0

  for (let i = 0; i < totalChunks; i++) {
    const start = i * chunkSize
    const end = Math.min(start + chunkSize, totalBytes)
    const blob = file.slice(start, end)
    const chunkIndex = i + 1

    const resp = await uploadArchiveChunk({
      upload_id: uploadId,
      chunk_index: chunkIndex,
      total_chunks: totalChunks,
      chunk: blob,
      bucket,
      filename: file.name,
    })

    loadedBytes = end
    const percent = Math.round((loadedBytes / totalBytes) * 100)
    opts.onProgress?.({ loadedBytes, totalBytes, percent, currentChunk: chunkIndex, totalChunks })
    opts.onChunkUploaded?.(chunkIndex)

    // 最后一个分片返回后端处理结果
    if (chunkIndex === totalChunks) {
      return resp
    }
  }

  // 正常不会走到这里
  return { code: 0, msg: 'upload finished', data: null }
}