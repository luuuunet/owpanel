import api from '@/api'

export const CHUNK_SIZE = 2 * 1024 * 1024 // 2 MiB

export type UploadBoardStatus = 'queued' | 'uploading' | 'done' | 'error' | 'paused'

export interface UploadBoardItem {
  key: string
  name: string
  size: number
  destDir: string
  status: UploadBoardStatus
  progress: number
  uploadId?: string
  error?: string
  file?: File
}

interface ChunkStatus {
  id: string
  filename: string
  size: number
  chunk_size: number
  total_chunks: number
  received_chunks: number[]
  received_count: number
  complete: boolean
  progress: number
}

function fingerprint(destDir: string, file: File): string {
  return `${destDir}|${file.name}|${file.size}|${file.lastModified}`
}

function loadStoredId(fp: string): string | undefined {
  try {
    return localStorage.getItem(`owpanel-upload:${fp}`) || undefined
  } catch {
    return undefined
  }
}

function storeId(fp: string, id: string) {
  try {
    localStorage.setItem(`owpanel-upload:${fp}`, id)
  } catch {
    /* ignore */
  }
}

function clearStoredId(fp: string) {
  try {
    localStorage.removeItem(`owpanel-upload:${fp}`)
  } catch {
    /* ignore */
  }
}

export function createUploadItem(file: File, destDir: string): UploadBoardItem {
  return {
    key: fingerprint(destDir, file),
    name: file.name,
    size: file.size,
    destDir,
    status: 'queued',
    progress: 0,
    file,
  }
}

export async function uploadFileResumable(
  item: UploadBoardItem,
  onProgress: (item: UploadBoardItem) => void,
  signal?: { cancelled: boolean },
): Promise<void> {
  const file = item.file
  if (!file) {
    item.status = 'error'
    item.error = 'missing file'
    onProgress({ ...item })
    return
  }

  item.status = 'uploading'
  item.error = undefined
  onProgress({ ...item })

  const fp = item.key
  const resumeId = item.uploadId || loadStoredId(fp)

  const initRes: any = await api.post('/files/upload/init', {
    path: item.destDir,
    filename: file.name,
    size: file.size,
    chunk_size: CHUNK_SIZE,
    upload_id: resumeId || '',
  })
  const st = initRes.data as ChunkStatus
  item.uploadId = st.id
  storeId(fp, st.id)

  const received = new Set(st.received_chunks || [])
  item.progress = st.progress || 0
  onProgress({ ...item })

  const total = st.total_chunks || Math.max(1, Math.ceil(file.size / (st.chunk_size || CHUNK_SIZE)))
  const chunkSize = st.chunk_size || CHUNK_SIZE

  for (let i = 0; i < total; i++) {
    if (signal?.cancelled) {
      item.status = 'paused'
      onProgress({ ...item })
      return
    }
    if (received.has(i)) {
      item.progress = Math.round(((i + 1) / total) * 100)
      onProgress({ ...item })
      continue
    }

    const start = i * chunkSize
    const end = Math.min(file.size, start + chunkSize)
    const blob = file.slice(start, end)
    const fd = new FormData()
    fd.append('upload_id', st.id)
    fd.append('index', String(i))
    fd.append('chunk', blob, `${file.name}.part${i}`)

    const chunkRes: any = await api.post('/files/upload/chunk', fd, {
      headers: { 'Content-Type': 'multipart/form-data' },
      timeout: 120000,
    })
    const cst = chunkRes.data as ChunkStatus
    item.progress = cst.progress ?? Math.round(((i + 1) / total) * 100)
    onProgress({ ...item })
  }

  if (signal?.cancelled) {
    item.status = 'paused'
    onProgress({ ...item })
    return
  }

  await api.post('/files/upload/complete', { upload_id: st.id })
  clearStoredId(fp)
  item.progress = 100
  item.status = 'done'
  item.uploadId = undefined
  onProgress({ ...item })
}

export async function cancelUploadSession(uploadId?: string) {
  if (!uploadId) return
  try {
    await api.delete(`/files/upload/${encodeURIComponent(uploadId)}`)
  } catch {
    /* ignore */
  }
}
