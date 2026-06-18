export const AI_CHAT_MAX_IMAGES = 4
export const AI_CHAT_MAX_IMAGE_BYTES = 4 * 1024 * 1024

const ACCEPT_TYPES = new Set(['image/jpeg', 'image/png', 'image/gif', 'image/webp'])

export function isAiChatImageType(type: string) {
  return ACCEPT_TYPES.has(type)
}

export function readFileAsDataUrl(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result || ''))
    reader.onerror = () => reject(new Error('read failed'))
    reader.readAsDataURL(file)
  })
}

export async function fileToAiChatImage(file: File): Promise<string> {
  if (!isAiChatImageType(file.type)) {
    throw new Error('unsupportedType')
  }
  if (file.size > AI_CHAT_MAX_IMAGE_BYTES) {
    throw new Error('tooLarge')
  }
  const dataUrl = await readFileAsDataUrl(file)
  if (!dataUrl.startsWith('data:image/')) {
    throw new Error('invalid')
  }
  return dataUrl
}

export async function filesToAiChatImages(files: FileList | File[], currentCount = 0): Promise<string[]> {
  const list = Array.from(files)
  if (currentCount + list.length > AI_CHAT_MAX_IMAGES) {
    throw new Error('tooMany')
  }
  const out: string[] = []
  for (const file of list) {
    out.push(await fileToAiChatImage(file))
  }
  return out
}

export function extractClipboardImages(event: ClipboardEvent): File[] {
  const items = event.clipboardData?.items
  if (!items) return []
  const files: File[] = []
  for (const item of items) {
    if (item.kind === 'file' && item.type.startsWith('image/')) {
      const f = item.getAsFile()
      if (f) files.push(f)
    }
  }
  return files
}
