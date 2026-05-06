import { useCallback, useEffect, useRef, useState } from 'react'
import { Loader2, RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { api } from '@/lib/api'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

const POLL_INTERVAL_MS = 2000
const MAX_POLL_DURATION_MS = 10 * 60 * 1000

type CreateQrcodeResp = {
  success: boolean
  data?: { login_token?: string; qrcode_url?: string }
  message?: string
}

type LoginStatusResp = {
  success: boolean
  data?: { status?: string; auth_code?: string }
  message?: string
}

type LoginByCodeResp = {
  success: boolean
  data?: unknown
  oauth_callback?: string
  message?: string
}

type WeChatDirectQrDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  onAuthCode: (code: string) => Promise<void> | void
}

export function WeChatDirectQrDialog({
  open,
  onOpenChange,
  onAuthCode,
}: WeChatDirectQrDialogProps) {
  const { t } = useTranslation()
  const [qrcodeUrl, setQrcodeUrl] = useState('')
  const [loginToken, setLoginToken] = useState('')
  const [loading, setLoading] = useState(false)
  const pollTimer = useRef<ReturnType<typeof setInterval> | null>(null)
  const pollStartedAt = useRef(0)

  const stopPolling = useCallback(() => {
    if (pollTimer.current) {
      clearInterval(pollTimer.current)
      pollTimer.current = null
    }
  }, [])

  const generateQrcode = useCallback(async () => {
    setLoading(true)
    stopPolling()
    try {
      const res = await api.post('/api/wechat/create_login_qrcode')
      const payload = res?.data as CreateQrcodeResp | undefined
      if (!payload?.success || !payload.data?.login_token) {
        toast.error(payload?.message || t('Failed to load QR code'))
        return
      }
      setLoginToken(payload.data.login_token)
      setQrcodeUrl(payload.data.qrcode_url || '')
      pollStartedAt.current = Date.now()
    } catch (_error) {
      toast.error(t('Failed to load QR code'))
    } finally {
      setLoading(false)
    }
  }, [stopPolling, t])

  const pollLoginStatus = useCallback(
    async (token: string) => {
      try {
        const res = await api.get('/api/wechat/login_status', {
          params: { login_token: token },
        })
        const payload = res?.data as LoginStatusResp | undefined
        if (!payload?.success) return
        if (payload.data?.status === 'success' && payload.data?.auth_code) {
          stopPolling()
          await onAuthCode(payload.data.auth_code)
        }
      } catch (_error) {
        // Network blip — keep polling silently.
      }
    },
    [onAuthCode, stopPolling]
  )

  useEffect(() => {
    if (!open) {
      stopPolling()
      setQrcodeUrl('')
      setLoginToken('')
      return
    }
    void generateQrcode()
    return () => {
      stopPolling()
    }
  }, [open, generateQrcode, stopPolling])

  useEffect(() => {
    if (!open || !loginToken) return
    stopPolling()
    pollTimer.current = setInterval(() => {
      if (Date.now() - pollStartedAt.current > MAX_POLL_DURATION_MS) {
        stopPolling()
        toast.info(t('QR code expired, please refresh'))
        return
      }
      void pollLoginStatus(loginToken)
    }, POLL_INTERVAL_MS)
    return stopPolling
  }, [open, loginToken, pollLoginStatus, stopPolling, t])

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='max-w-sm'>
        <DialogHeader className='text-left'>
          <DialogTitle>{t('WeChat sign in')}</DialogTitle>
          <DialogDescription>
            {t('Scan the QR code with WeChat to sign in.')}
          </DialogDescription>
        </DialogHeader>

        <div className='flex flex-col items-center gap-3 py-2'>
          {loading || !qrcodeUrl ? (
            <div className='flex h-32 w-32 items-center justify-center rounded border'>
              <Loader2 className='size-5 animate-spin' />
            </div>
          ) : (
            <img
              src={qrcodeUrl}
              alt={t('WeChat login QR code')}
              className='h-32 w-32 rounded border object-contain'
            />
          )}
          <p className='text-muted-foreground text-xs'>
            {t('Waiting for scan...')}
          </p>
        </div>

        <DialogFooter className='gap-2 sm:justify-between'>
          <Button
            type='button'
            variant='ghost'
            onClick={generateQrcode}
            disabled={loading}
            className='gap-2'
          >
            <RefreshCw className='size-4' />
            {t('Refresh')}
          </Button>
          <Button
            type='button'
            variant='outline'
            onClick={() => onOpenChange(false)}
          >
            {t('Cancel')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

export type { LoginByCodeResp }
