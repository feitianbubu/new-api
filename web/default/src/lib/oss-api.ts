/*
 * OSS file API helpers ported from web/classic/src/helpers/ossApi.js.
 * Used by the usage-logs OSS preview modal to list, fetch and abbreviate
 * raw request/response bodies that the gateway has persisted to object
 * storage. The /api/files endpoint is the same one the classic theme
 * talks to (route renamed in classic 63d591139).
 */

export type OSSFileEntry = {
  id: string
  filename?: string
  bytes?: number
  // Backend may include other fields (mime, last_modified, etc.) that
  // we forward unchanged.
  [key: string]: unknown
}

const DEFAULT_ABBREVIATE_STR_LEN = 80
const OSS_FILE_ID_PREFIX = 'file-'
const MAX_ABBREVIATE_DEPTH = 32

function generateFileId(objectKey: string): string {
  const encoder = new TextEncoder()
  const data = encoder.encode(objectKey)
  const base64 = btoa(String.fromCharCode(...data))
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
  return `${OSS_FILE_ID_PREFIX}${base64}`
}

function getOSSLogBasePath(): string {
  return 'logs'
}

export async function listOSSFiles(
  requestId: string
): Promise<OSSFileEntry[]> {
  if (!requestId) return []

  const date = requestId.substring(0, 8)
  const basePath = getOSSLogBasePath()
  const prefix = `${basePath}/${date}/${requestId}/`

  try {
    const response = await fetch(
      `/api/files?prefix=${encodeURIComponent(prefix)}`,
      {
        method: 'GET',
        headers: { 'Cache-Control': 'no-cache' },
      }
    )
    if (!response.ok) {
      if (response.status === 404) return []
      throw new Error(`Failed to list OSS files: ${response.status}`)
    }
    const result = (await response.json()) as {
      code?: number
      data?: { data?: OSSFileEntry[] }
    }
    if (result.code === 0 && result.data && result.data.data) {
      return result.data.data
    }
    return []
  } catch (_error) {
    // eslint-disable-next-line no-console
    console.error('Failed to list OSS files:', _error)
    return []
  }
}

export function buildOSSFileId(
  requestId: string,
  fileType: string
): string | null {
  if (!requestId) return null
  const dateStr = requestId.substring(0, 8)
  const basePath = getOSSLogBasePath()
  const objectKey = `${basePath}/${dateStr}/${requestId}/${fileType}.json`
  return generateFileId(objectKey)
}

export function buildOSSUrl(
  fileIdOrRequestId: string,
  fileType?: string
): string | null {
  if (
    typeof fileIdOrRequestId === 'string' &&
    fileIdOrRequestId.startsWith(OSS_FILE_ID_PREFIX)
  ) {
    return `/api/files/${fileIdOrRequestId}/content`
  }
  if (!fileType) return null
  const fileId = buildOSSFileId(fileIdOrRequestId, fileType)
  if (!fileId) return null
  return `/api/files/${fileId}/content`
}

export async function fetchOSSContent(
  url: string,
  rangeBytes?: number
): Promise<string> {
  if (!url) throw new Error('OSS URL is required')

  const headers: Record<string, string> = { 'Cache-Control': 'no-cache' }
  if (rangeBytes && rangeBytes > 0) {
    headers['Range'] = `bytes=0-${rangeBytes - 1}`
  }

  const response = await fetch(url, { method: 'GET', headers })
  if (!response.ok) {
    if (response.status === 404) throw new Error('OSS file not found')
    if (response.status === 403) throw new Error('Access denied to OSS file')
    if (response.status === 416) throw new Error('Range not satisfiable')
    throw new Error(`Failed to fetch OSS file: ${response.status}`)
  }
  return await response.text()
}

export function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes}B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)}KB`
  return `${(bytes / 1024 / 1024).toFixed(2)}MB`
}

function abbreviateString(str: string, maxLen: number): string {
  const dataUrlMatch = str.match(/^data:([^;,]+)(;base64)?,(.*)$/)
  if (dataUrlMatch) {
    const mime = dataUrlMatch[1] || 'unknown'
    const isBase64 = !!dataUrlMatch[2]
    const payload = dataUrlMatch[3] || ''
    const bytes = isBase64
      ? Math.floor((payload.length * 3) / 4)
      : payload.length
    return `[${mime}${isBase64 ? ';base64' : ''}, ${formatSize(bytes)}]`
  }
  if (str.length <= maxLen) return str
  return `${str.slice(0, maxLen)}...(${str.length} 字符)`
}

function abbreviateValue(
  value: unknown,
  maxLen: number,
  depth = 0
): unknown {
  if (depth > MAX_ABBREVIATE_DEPTH) return '[…too deep]'
  if (value === null || value === undefined) return value
  if (typeof value === 'string') return abbreviateString(value, maxLen)
  if (Array.isArray(value)) {
    return value.map((v) => abbreviateValue(v, maxLen, depth + 1))
  }
  if (typeof value === 'object') {
    const result: Record<string, unknown> = {}
    for (const key of Object.keys(value as Record<string, unknown>)) {
      result[key] = abbreviateValue(
        (value as Record<string, unknown>)[key],
        maxLen,
        depth + 1
      )
    }
    return result
  }
  return value
}

export type AbbreviateJsonOptions = { maxStrLen?: number }

export function abbreviateJsonContent(
  content: string,
  options: AbbreviateJsonOptions = {}
): string {
  if (!content) return ''
  const maxLen = options.maxStrLen || DEFAULT_ABBREVIATE_STR_LEN
  try {
    const parsed = JSON.parse(content)
    if (parsed && (parsed as Record<string, unknown>)._truncated === true) {
      return JSON.stringify(parsed, null, 2)
    }
    return JSON.stringify(abbreviateValue(parsed, maxLen), null, 2)
  } catch {
    return content
  }
}

export type BackendTruncatedInfo = { originalSize: number } | null

export function getBackendTruncatedInfo(
  content: string
): BackendTruncatedInfo {
  if (!content) return null
  try {
    const parsed = JSON.parse(content) as Record<string, unknown>
    if (parsed && parsed._truncated === true) {
      return { originalSize: Number(parsed._original_size) || 0 }
    }
    return null
  } catch {
    return null
  }
}

function formatSSEStream(sseContent: string): string {
  const lines = sseContent.split('\n')
  const chunks: Array<Record<string, unknown>> = []
  let finalMessage: Record<string, unknown> | null = null
  const allReasoningContent: string[] = []
  const allContent: string[] = []

  for (const line of lines) {
    const trimmedLine = line.trim()
    if (!trimmedLine || trimmedLine === 'data: [DONE]') continue
    if (trimmedLine.startsWith('data: ')) {
      const jsonStr = trimmedLine.substring(6)
      try {
        const chunk = JSON.parse(jsonStr) as Record<string, unknown>
        chunks.push(chunk)
        const choices = chunk.choices as
          | Array<{
              delta?: { reasoning_content?: string; content?: string }
              message?: unknown
            }>
          | undefined
        if (choices?.[0]?.delta?.reasoning_content) {
          allReasoningContent.push(choices[0].delta.reasoning_content)
        }
        if (choices?.[0]?.delta?.content) {
          allContent.push(choices[0].delta.content)
        }
        if (choices?.[0]?.message) {
          finalMessage = chunk
        }
      } catch {
        continue
      }
    }
  }

  let formattedOutput = ''

  if (finalMessage) {
    const choices = finalMessage.choices as
      | Array<{
          message: {
            reasoning_content?: string
            content?: string
          }
        }>
      | undefined
    const message = choices?.[0]?.message
    formattedOutput += '=== Stream Response Summary ===\n\n'
    if (message?.reasoning_content) {
      formattedOutput += '[Reasoning]\n'
      formattedOutput += message.reasoning_content + '\n\n'
    }
    if (message?.content) {
      formattedOutput += '[Response]\n'
      formattedOutput += message.content + '\n\n'
    }
    formattedOutput += '[Metadata]\n'
    formattedOutput += `Model: ${finalMessage.model ?? ''}\n`
    formattedOutput += `ID: ${finalMessage.id ?? ''}\n`
    const usage = finalMessage.usage as Record<string, unknown> | undefined
    if (usage) {
      formattedOutput += `Total Tokens: ${(usage.total_tokens as number) || 0}\n`
      formattedOutput += `Input Tokens: ${(usage.prompt_tokens as number) || 0}\n`
      formattedOutput += `Output Tokens: ${(usage.completion_tokens as number) || 0}\n`
      const compDetails = usage.completion_tokens_details as
        | Record<string, unknown>
        | undefined
      if (compDetails?.reasoning_tokens) {
        formattedOutput += `Reasoning Tokens: ${compDetails.reasoning_tokens as number}\n`
      }
    }
    formattedOutput += '\n'
  } else {
    if (allReasoningContent.length > 0) {
      formattedOutput += '[Response]\n'
      formattedOutput += allReasoningContent.join('') + '\n\n'
    }
    if (allContent.length > 0) {
      formattedOutput += '[Response]\n'
      formattedOutput += allContent.join('') + '\n\n'
    }
  }

  formattedOutput += '=== data ===\n\n'
  formattedOutput += chunks
    .map(
      (chunk, idx) =>
        `--- ${idx + 1} ---\n${JSON.stringify(chunk, null, 2)}`
    )
    .join('\n\n')

  return formattedOutput
}

export function formatOSSContent(content: string): string {
  if (!content) return ''
  if (content.includes('data: {')) return formatSSEStream(content)
  try {
    const parsed = JSON.parse(content)
    return JSON.stringify(parsed, null, 2)
  } catch {
    return content
  }
}

export function getOSSConfig(): string | null {
  try {
    const statusStr = localStorage.getItem('status')
    if (statusStr) {
      const status = JSON.parse(statusStr) as { oss_path?: string }
      return status?.oss_path || null
    }
  } catch (_error) {
    // eslint-disable-next-line no-console
    console.error('Failed to get OSS config:', _error)
  }
  return null
}
