import { createFileRoute } from '@tanstack/react-router'
import { useStatus } from '@/hooks/use-status'

const DEFAULT_DOCS_LINK = '/swag.html'

export const Route = createFileRoute('/_authenticated/docs')({
  component: DocsPage,
})

function DocsPage() {
  const { status } = useStatus()
  const docsLink =
    (status as { docs_link?: string } | null)?.docs_link ||
    (typeof window !== 'undefined' && window.localStorage.getItem('docs_link')) ||
    DEFAULT_DOCS_LINK

  return (
    <div className='h-[calc(100vh-56px)] min-h-[600px] w-full'>
      <iframe
        src={docsLink}
        title='API Documentation'
        sandbox='allow-same-origin allow-scripts allow-popups allow-forms'
        className='h-full min-h-[600px] w-full border-none'
      />
    </div>
  )
}
