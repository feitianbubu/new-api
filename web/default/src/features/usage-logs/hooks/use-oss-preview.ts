import { useCallback, useMemo, useState } from 'react'
import { getOSSConfig } from '@/lib/oss-api'

export function useOSSPreview() {
  const [open, setOpen] = useState(false)
  const [requestId, setRequestId] = useState('')
  const [title, setTitle] = useState('')

  const ossPath = useMemo(() => getOSSConfig(), [])

  const hasOSSFiles = useCallback(
    (rid: string) => Boolean(rid && ossPath),
    [ossPath]
  )

  const openPreview = useCallback((rid: string, label?: string) => {
    setRequestId(rid)
    setTitle(label || '')
    setOpen(true)
  }, [])

  return {
    open,
    setOpen,
    requestId,
    title,
    hasOSSFiles,
    openPreview,
  }
}
