import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react'
import {
  Download,
  Loader2,
  Maximize2,
  Minimize2,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import {
  abbreviateJsonContent,
  buildOSSUrl,
  fetchOSSContent,
  formatOSSContent,
  formatSize,
  getBackendTruncatedInfo,
  listOSSFiles,
  type OSSFileEntry,
} from '@/lib/oss-api'

const PREVIEW_THRESHOLD = 100 * 1024 // 100 KB
const PREVIEW_BYTES = 1024 // 1 KB initial preview

const IMAGE_EXTS = ['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'bmp']
const AUDIO_EXTS = ['mp3', 'wav', 'ogg', 'm4a', 'aac', 'flac']
const VIDEO_EXTS = ['mp4', 'webm', 'ogg', 'mov', 'avi', 'mkv']
const JSON_EXTS = ['json']

type FileKind = 'image' | 'audio' | 'video' | 'json' | 'text'

function getExt(filename: string): string {
  const parts = filename.split('.')
  return (parts[parts.length - 1] || '').toLowerCase()
}

function detectKind(filename: string): FileKind {
  const ext = getExt(filename)
  if (IMAGE_EXTS.includes(ext)) return 'image'
  if (AUDIO_EXTS.includes(ext)) return 'audio'
  if (VIDEO_EXTS.includes(ext)) return 'video'
  if (JSON_EXTS.includes(ext)) return 'json'
  return 'text'
}

function truncateMiddle(text: string, max = 11): string {
  if (text.length <= max) return text
  const head = Math.ceil((max - 1) / 2)
  const tail = Math.floor((max - 1) / 2)
  return `${text.slice(0, head)}…${text.slice(text.length - tail)}`
}

type ContentMode = 'preview' | 'full' | 'structure'

type ContentState = {
  display: string
  rawPreview: string | null // first PREVIEW_BYTES bytes when range-loaded
  rawFull: string | null // full content once fetched
  showStructure: boolean
  loading: boolean
  loadingMode: ContentMode | null
  error: string
}

type OSSPreviewModalProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  requestId: string
  title?: string
}

export function OSSPreviewModal({
  open,
  onOpenChange,
  requestId,
  title,
}: OSSPreviewModalProps) {
  const { t } = useTranslation()
  const [files, setFiles] = useState<OSSFileEntry[]>([])
  const [loadingList, setLoadingList] = useState(false)
  const [activeId, setActiveId] = useState<string>('')
  const [fullscreen, setFullscreen] = useState(false)
  const cache = useRef<Record<string, ContentState>>({})
  const [, forceRender] = useState(0)

  const setContent = useCallback((id: string, next: Partial<ContentState>) => {
    cache.current[id] = {
      display: '',
      rawPreview: null,
      rawFull: null,
      showStructure: false,
      loading: false,
      loadingMode: null,
      error: '',
      ...cache.current[id],
      ...next,
    }
    forceRender((n) => n + 1)
  }, [])

  useEffect(() => {
    if (!open) {
      cache.current = {}
      setFiles([])
      setActiveId('')
      setFullscreen(false)
      return
    }
    if (!requestId) return
    let cancelled = false
    setLoadingList(true)
    listOSSFiles(requestId)
      .then((result) => {
        if (cancelled) return
        setFiles(result)
        if (result.length > 0) setActiveId(result[0].id)
      })
      .finally(() => {
        if (!cancelled) setLoadingList(false)
      })
    return () => {
      cancelled = true
    }
  }, [open, requestId])

  const activeFile = useMemo(
    () => files.find((f) => f.id === activeId),
    [files, activeId]
  )

  const applyRaw = useCallback(
    (
      file: OSSFileEntry,
      raw: string,
      opts: { isTruncated: boolean; showStructure: boolean }
    ) => {
      const display = opts.showStructure
        ? abbreviateJsonContent(raw)
        : formatOSSContent(raw)
      const prev = cache.current[file.id] || ({} as ContentState)
      setContent(file.id, {
        display,
        rawPreview: opts.isTruncated ? raw : (prev.rawPreview ?? null),
        rawFull: !opts.isTruncated ? raw : (prev.rawFull ?? null),
        showStructure: opts.showStructure,
        loading: false,
        loadingMode: null,
      })
    },
    [setContent]
  )

  const loadContent = useCallback(
    async (file: OSSFileEntry, mode: ContentMode = 'preview') => {
      const existing = cache.current[file.id]
      // Already have full content cached — switch view without refetching
      if (
        existing?.rawFull != null &&
        (mode === 'full' || mode === 'structure')
      ) {
        applyRaw(file, existing.rawFull, {
          isTruncated: false,
          showStructure: mode === 'structure',
        })
        return
      }

      setContent(file.id, {
        loading: true,
        loadingMode: mode === 'preview' ? null : mode,
        error: '',
      })

      try {
        const url = buildOSSUrl(file.id)
        if (!url) throw new Error('Invalid OSS URL')
        const kind = detectKind(file.filename || file.id)
        const isTextish = kind === 'text' || kind === 'json'
        const totalBytes = file.bytes ?? 0
        const shouldPreview =
          mode === 'preview' && isTextish && totalBytes > PREVIEW_THRESHOLD
        const rangeBytes = shouldPreview ? PREVIEW_BYTES : undefined
        const raw = await fetchOSSContent(url, rangeBytes)
        applyRaw(file, raw, {
          isTruncated: rangeBytes != null,
          showStructure: mode === 'structure',
        })
      } catch (error) {
        setContent(file.id, {
          loading: false,
          loadingMode: null,
          error: error instanceof Error ? error.message : String(error),
        })
      }
    },
    [applyRaw, setContent]
  )

  useEffect(() => {
    if (!activeFile) return
    const kind = detectKind(activeFile.filename || activeFile.id)
    if (kind === 'image' || kind === 'audio' || kind === 'video') return
    if (cache.current[activeFile.id]) return
    void loadContent(activeFile, 'preview')
  }, [activeFile, loadContent])

  const handleDownload = useCallback(() => {
    if (!activeFile) return
    const url = buildOSSUrl(activeFile.id)
    if (!url) return
    const a = document.createElement('a')
    a.href = url
    a.download = activeFile.filename || activeFile.id
    a.target = '_blank'
    a.rel = 'noopener noreferrer'
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
  }, [activeFile])

  const renderActive = () => {
    if (!activeFile) {
      if (loadingList) {
        return (
          <div className='flex h-40 items-center justify-center'>
            <Loader2 className='size-5 animate-spin' />
          </div>
        )
      }
      return (
        <div className='text-muted-foreground flex h-40 items-center justify-center text-sm'>
          {t('No files found')}
        </div>
      )
    }
    const url = buildOSSUrl(activeFile.id) || ''
    const kind = detectKind(activeFile.filename || activeFile.id)
    if (kind === 'image') {
      return (
        <div className='flex min-h-0 flex-1 items-center justify-center'>
          <img
            src={url}
            alt={activeFile.filename || activeFile.id}
            className='max-h-full max-w-full object-contain'
          />
        </div>
      )
    }
    if (kind === 'audio') {
      return <audio controls src={url} className='w-full' />
    }
    if (kind === 'video') {
      return <video controls src={url} className='max-h-full w-full' />
    }
    const state = cache.current[activeFile.id]
    if (!state || (state.loading && !state.display)) {
      return (
        <div className='flex h-40 items-center justify-center'>
          <Loader2 className='size-5 animate-spin' />
        </div>
      )
    }
    if (state.error) {
      return (
        <div className='text-destructive py-8 text-center text-sm'>
          {state.error}
        </div>
      )
    }
    const isJsonish = kind === 'json' || kind === 'text'
    const truncatedInfo = isJsonish
      ? getBackendTruncatedInfo(state.display)
      : null
    const isRangeTruncated =
      state.rawFull == null && state.rawPreview != null
    return (
      <div className='flex min-h-0 flex-1 flex-col gap-2'>
        <div className='text-muted-foreground flex flex-wrap items-center gap-x-2 gap-y-1 text-xs'>
          <span>
            {t('Filename')}: {activeFile.filename || activeFile.id}
            {' | '}
            {t('Size')}: {formatSize(activeFile.bytes ?? 0)}
          </span>
          {isJsonish && truncatedInfo && (
            <span className='text-amber-600 dark:text-amber-400'>
              [
              {t('Backend summary, original {{size}}', {
                size: formatSize(truncatedInfo.originalSize),
              })}
              ]
            </span>
          )}
          {isJsonish && !truncatedInfo && isRangeTruncated && (
            <span className='text-amber-600 dark:text-amber-400'>
              [{t('Showing first 1KB preview')}
              {state.loadingMode === 'full' ? (
                ' ...'
              ) : (
                <button
                  type='button'
                  className='hover:text-primary ml-1 underline'
                  onClick={() => loadContent(activeFile, 'full')}
                >
                  {state.showStructure ? t('Show raw') : t('Show all')}
                </button>
              )}
              {kind === 'json' && (
                <>
                  {' | '}
                  {state.loadingMode === 'structure' ? (
                    '...'
                  ) : (
                    <button
                      type='button'
                      className='hover:text-primary underline'
                      onClick={() => loadContent(activeFile, 'structure')}
                    >
                      {t('Structure preview')}
                    </button>
                  )}
                </>
              )}
              ]
            </span>
          )}
          {isJsonish &&
            !truncatedInfo &&
            !isRangeTruncated &&
            state.showStructure && (
              <span className='text-amber-600 dark:text-amber-400'>
                [{t('Structure preview (values abbreviated)')}
                {state.loadingMode === 'full' ? (
                  ' ...'
                ) : (
                  <button
                    type='button'
                    className='hover:text-primary ml-1 underline'
                    onClick={() => loadContent(activeFile, 'full')}
                  >
                    {t('Show raw')}
                  </button>
                )}
                ]
              </span>
            )}
          {kind === 'json' &&
            !truncatedInfo &&
            !isRangeTruncated &&
            !state.showStructure && (
              <span>
                [
                {state.loadingMode === 'structure' ? (
                  '...'
                ) : (
                  <button
                    type='button'
                    className='hover:text-primary underline'
                    onClick={() => loadContent(activeFile, 'structure')}
                  >
                    {t('Structure preview')}
                  </button>
                )}
                ]
              </span>
            )}
        </div>
        <pre className='bg-muted/30 min-h-0 flex-1 overflow-auto rounded border p-3 text-xs leading-relaxed whitespace-pre-wrap break-all'>
          {state.display}
        </pre>
      </div>
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className={cn(
          'flex flex-col overflow-hidden p-0',
          fullscreen
            ? 'h-[calc(100vh-2rem)] max-h-[calc(100vh-2rem)] w-[calc(100vw-2rem)] !max-w-[calc(100vw-2rem)] sm:!max-w-[calc(100vw-2rem)]'
            : 'h-[640px] max-h-[calc(100vh-2rem)] w-full !max-w-3xl sm:!max-w-3xl'
        )}
      >
        <DialogHeader className='flex flex-row items-center justify-between border-b px-4 py-3'>
          <DialogTitle className='text-base'>
            {title || t('OSS Preview')}{' '}
            {requestId && (
              <span className='text-muted-foreground font-mono text-xs'>
                {requestId}
              </span>
            )}
          </DialogTitle>
          <div className='flex items-center gap-1'>
            <Button
              type='button'
              variant='ghost'
              size='sm'
              onClick={handleDownload}
              disabled={!activeFile}
              className='gap-1'
            >
              <Download className='size-4' />
            </Button>
            <Button
              type='button'
              variant='ghost'
              size='sm'
              onClick={() => setFullscreen((v) => !v)}
              className='gap-1'
            >
              {fullscreen ? (
                <Minimize2 className='size-4' />
              ) : (
                <Maximize2 className='size-4' />
              )}
            </Button>
          </div>
        </DialogHeader>

        <div className='flex min-h-0 flex-1 flex-col gap-3 overflow-hidden p-4'>
          {files.length > 0 && (
            <Tabs value={activeId} onValueChange={setActiveId}>
              <TabsList className='flex flex-wrap'>
                {files.map((file) => (
                  <TabsTrigger
                    key={file.id}
                    value={file.id}
                    title={file.filename || file.id}
                  >
                    {truncateMiddle(file.filename || file.id, 11)}
                  </TabsTrigger>
                ))}
              </TabsList>
            </Tabs>
          )}

          <div className='flex-1 overflow-auto'>{renderActive()}</div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
